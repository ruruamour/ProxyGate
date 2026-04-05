package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
	"goproxy/config"
	"goproxy/storage"
)

type Server struct {
	storage          *storage.Storage
	cfg              *config.Config
	sessions         *SessionManager
	sessionNamespace string
	mode             string // "random" 或 "lowest-latency"
	port             string
	clientCache      sync.Map
}

func New(s *storage.Storage, cfg *config.Config, sessions *SessionManager, mode string, port string) *Server {
	return &Server{
		storage:          s,
		cfg:              cfg,
		sessions:         sessions,
		sessionNamespace: "http-" + mode,
		mode:             mode,
		port:             port,
	}
}

func (s *Server) Start() error {
	modeDesc := "随机轮换"
	if s.mode == "lowest-latency" {
		modeDesc = "最低延迟"
	}
	authStatus := "无认证"
	if s.cfg.ProxyAuthEnabled {
		authStatus = fmt.Sprintf("需认证 (用户: %s)", s.cfg.ProxyAuthUsername)
	}
	log.Printf("proxy server listening on %s [%s] [%s]", s.port, modeDesc, authStatus)
	server := &http.Server{
		Addr:              s.port,
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	return server.ListenAndServe()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	opts := RequestOptions{}

	// 认证检查（如果启用）
	if s.cfg.ProxyAuthEnabled {
		var ok bool
		opts, ok = s.parseAuth(r)
		if !ok {
			w.Header().Set("Proxy-Authenticate", `Basic realm="GoProxy"`)
			http.Error(w, "Proxy Authentication Required", http.StatusProxyAuthRequired)
			return
		}
	}

	if r.Method == http.MethodConnect {
		s.handleTunnel(w, r, opts)
	} else {
		s.handleHTTP(w, r, opts)
	}
}

// parseAuth 验证代理 Basic Auth，并解析扩展会话参数
func (s *Server) parseAuth(r *http.Request) (RequestOptions, bool) {
	auth := r.Header.Get("Proxy-Authorization")
	if auth == "" {
		return RequestOptions{}, false
	}

	// 解析 Basic Auth
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return RequestOptions{}, false
	}

	decoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return RequestOptions{}, false
	}

	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		return RequestOptions{}, false
	}

	username := credentials[0]
	password := credentials[1]

	// 验证用户名和密码
	passwordHash := fmt.Sprintf("%x", sha256.Sum256([]byte(password)))
	passwordMatch := subtle.ConstantTimeCompare([]byte(passwordHash), []byte(s.cfg.ProxyAuthPasswordHash)) == 1
	if !passwordMatch {
		return RequestOptions{}, false
	}

	opts, err := parseUsernameOptions(s.cfg.ProxyAuthUsername, username, s.sessionNamespace)
	if err != nil {
		return RequestOptions{}, false
	}

	return opts, true
}

// selectProxy 根据使用模式和选择策略获取代理
func (s *Server) selectProxy(tried []string, lowestLatency bool, opts RequestOptions) (*storage.Proxy, error) {
	cfg := s.cfg
	if sticky := selectExistingStickyProxy(s.storage, s.sessions, "", tried, opts); sticky != nil {
		return sticky, nil
	}
	sourceFilter := sourceFilterFromMode(cfg.CustomProxyMode)

	// 混用 + 优先模式：先尝试优先源，无可用则 fallback 全部
	if cfg.CustomProxyMode == "mixed" && (cfg.CustomPriority || cfg.CustomFreePriority) {
		preferSource := "custom"
		if cfg.CustomFreePriority {
			preferSource = "free"
		}
		var p *storage.Proxy
		var err error
		p, err = selectFromPool(s.storage, s.sessions, preferSource, s.sessionNamespace, "", tried, lowestLatency, opts)
		if err == nil {
			return p, nil
		}
		// fallback 到全部
		return selectFromPool(s.storage, s.sessions, "", s.sessionNamespace, "", tried, lowestLatency, opts)
	}

	return selectFromPool(s.storage, s.sessions, sourceFilter, s.sessionNamespace, "", tried, lowestLatency, opts)
}

// removeOrDisableProxy 根据代理来源决定删除或禁用
func removeOrDisableProxy(store *storage.Storage, p *storage.Proxy) {
	if p.Source == "custom" {
		store.DisableProxy(p.Address)
	} else {
		store.Delete(p.Address)
	}
}

// sourceFilterFromMode 根据使用模式返回来源过滤值
func sourceFilterFromMode(mode string) string {
	switch mode {
	case "custom_only":
		return "custom"
	case "free_only":
		return "free"
	default:
		return "" // mixed
	}
}

// handleHTTP 处理普通 HTTP 请求（带自动重试）
func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request, opts RequestOptions) {
	var tried []string
	bodyReplayable := requestReplayable(r)
	maxAttempts := s.cfg.MaxRetry
	if !bodyReplayable {
		maxAttempts = 0
	}

	var bodyBytes []byte
	if hasRequestBody(r) && bodyReplayable {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		_ = r.Body.Close()
	}

	for attempt := 0; attempt <= maxAttempts; attempt++ {
		p, err := s.selectProxy(tried, s.mode == "lowest-latency", opts)
		if err != nil {
			http.Error(w, "no available proxy", http.StatusServiceUnavailable)
			return
		}

		tried = append(tried, p.Address)

		client, err := s.buildClient(p)
		if err != nil {
			removeOrDisableProxy(s.storage, p)
			continue
		}

		req, err := buildOutgoingRequest(r, bodyBytes, bodyReplayable)
		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[proxy] %s via %s failed, removing", r.RequestURI, p.Address)
			s.storage.RecordProxyUse(p.Address, false)
			if opts.SessionKey != "" {
				s.sessions.Delete(opts.SessionKey)
			}
			removeOrDisableProxy(s.storage, p)
			continue
		}
		defer resp.Body.Close()

		// 写回响应
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		s.storage.RecordProxyUse(p.Address, true)
		if resp.StatusCode == 429 {
			log.Printf("[proxy] ⚠️  429 %s via %s (protocol=%s)", r.RequestURI, p.Address, p.Protocol)
		} else {
			log.Printf("[proxy] %s via %s -> %d", r.RequestURI, p.Address, resp.StatusCode)
		}
		return
	}

	http.Error(w, "all proxies failed", http.StatusBadGateway)
}

// handleTunnel 处理 HTTPS CONNECT 隧道（带自动重试）
func (s *Server) handleTunnel(w http.ResponseWriter, r *http.Request, opts RequestOptions) {
	var tried []string
	for attempt := 0; attempt <= s.cfg.MaxRetry; attempt++ {
		p, err := s.selectProxy(tried, s.mode == "lowest-latency", opts)
		if err != nil {
			http.Error(w, "no available proxy", http.StatusServiceUnavailable)
			return
		}

		tried = append(tried, p.Address)

		conn, err := s.dialViaProxy(p, r.Host)
		if err != nil {
			log.Printf("[tunnel] dial %s via %s failed, removing", r.Host, p.Address)
			s.storage.RecordProxyUse(p.Address, false)
			if opts.SessionKey != "" {
				s.sessions.Delete(opts.SessionKey)
			}
			removeOrDisableProxy(s.storage, p)
			continue
		}

		s.storage.RecordProxyUse(p.Address, true)

		// 告知客户端隧道建立
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			conn.Close()
			http.Error(w, "hijack not supported", http.StatusInternalServerError)
			return
		}
		clientConn, _, err := hijacker.Hijack()
		if err != nil {
			conn.Close()
			return
		}

		fmt.Fprintf(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")
		log.Printf("[tunnel] %s via %s established", r.Host, p.Address)

		// 双向转发
		go transfer(conn, clientConn)
		go transfer(clientConn, conn)
		return
	}

	http.Error(w, "all proxies failed", http.StatusBadGateway)
}

func (s *Server) dialViaProxy(p *storage.Proxy, host string) (net.Conn, error) {
	timeout := time.Duration(s.cfg.ValidateTimeout) * time.Second
	switch p.Protocol {
	case "http":
		return dialHTTPConnect(p.Address, host, timeout)
	case "socks5":
		dialer, err := proxy.SOCKS5("tcp", p.Address, nil, proxy.Direct)
		if err != nil {
			return nil, err
		}
		return dialer.Dial("tcp", host)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", p.Protocol)
	}
}

func (s *Server) buildClient(p *storage.Proxy) (*http.Client, error) {
	cacheKey := p.Protocol + "|" + p.Address
	if cached, ok := s.clientCache.Load(cacheKey); ok {
		return cached.(*http.Client), nil
	}

	timeout := time.Duration(s.cfg.ValidateTimeout) * time.Second
	transport := &http.Transport{
		MaxIdleConns:        256,
		MaxIdleConnsPerHost: 32,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: timeout,
	}

	var client *http.Client
	switch p.Protocol {
	case "http":
		proxyURL, err := url.Parse(fmt.Sprintf("http://%s", p.Address))
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
		client = &http.Client{Transport: transport, Timeout: timeout}
	case "socks5":
		dialer, err := proxy.SOCKS5("tcp", p.Address, nil, proxy.Direct)
		if err != nil {
			return nil, err
		}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
		client = &http.Client{Transport: transport, Timeout: timeout}
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", p.Protocol)
	}

	actual, _ := s.clientCache.LoadOrStore(cacheKey, client)
	return actual.(*http.Client), nil
}

func hasRequestBody(r *http.Request) bool {
	return r.Body != nil && r.Body != http.NoBody
}

func requestReplayable(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return !hasRequestBody(r)
	}
}

func buildOutgoingRequest(r *http.Request, bodyBytes []byte, bodyReplayable bool) (*http.Request, error) {
	var body io.Reader
	if len(bodyBytes) > 0 {
		body = bytes.NewReader(bodyBytes)
	} else if !bodyReplayable {
		body = r.Body
	}

	req, err := http.NewRequest(r.Method, r.URL.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header = r.Header.Clone()
	req.Header.Del("Proxy-Connection")
	req.Host = r.Host
	req.TransferEncoding = append([]string(nil), r.TransferEncoding...)

	if len(bodyBytes) == 0 && hasRequestBody(r) {
		req.ContentLength = r.ContentLength
	}

	if len(bodyBytes) > 0 {
		req.ContentLength = int64(len(bodyBytes))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	return req, nil
}

func dialHTTPConnect(proxyAddress string, host string, timeout time.Duration) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", proxyAddress, timeout)
	if err != nil {
		return nil, err
	}
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		conn.Close()
		return nil, err
	}

	req, err := http.NewRequest(http.MethodConnect, "http://"+host, nil)
	if err != nil {
		conn.Close()
		return nil, err
	}
	req.Host = host

	if err := req.Write(conn); err != nil {
		conn.Close()
		return nil, err
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		conn.Close()
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
		conn.Close()
		return nil, fmt.Errorf("upstream proxy connect failed: %s", resp.Status)
	}

	if err := conn.SetDeadline(time.Time{}); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func transfer(dst io.WriteCloser, src io.ReadCloser) {
	defer dst.Close()
	defer src.Close()
	io.Copy(dst, src)
}
