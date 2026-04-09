package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
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
	"proxygate/config"
	"proxygate/storage"
)

type Server struct {
	storage          *storage.Storage
	cfg              *config.Config
	sessions         *SessionManager
	sessionNamespace string
	mode             string // "random" 或 "lowest-latency"
	port             string
	clientCache      sync.Map
	serverMu         sync.Mutex
	server           *http.Server
}

type cachedHTTPClient struct {
	client    *http.Client
	transport *http.Transport
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

func (s *Server) currentConfig() *config.Config {
	if cfg := config.Get(); cfg != nil {
		return cfg
	}
	return s.cfg
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
	s.serverMu.Lock()
	s.server = server
	s.serverMu.Unlock()
	defer func() {
		s.serverMu.Lock()
		if s.server == server {
			s.server = nil
		}
		s.serverMu.Unlock()
	}()
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.serverMu.Lock()
	server := s.server
	s.serverMu.Unlock()
	if server == nil {
		return nil
	}
	err := server.Shutdown(ctx)
	s.clientCache.Range(func(_, value any) bool {
		if entry, ok := value.(*cachedHTTPClient); ok {
			entry.transport.CloseIdleConnections()
		}
		return true
	})
	return err
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	opts := RequestOptions{}
	cfg := s.currentConfig()

	// 认证检查（如果启用）
	if cfg.ProxyAuthEnabled {
		authHeaderPresent := strings.TrimSpace(r.Header.Get("Proxy-Authorization")) != ""
		switch {
		case authHeaderPresent:
			var ok bool
			opts, ok = s.parseAuth(r)
			if !ok {
				w.Header().Set("Proxy-Authenticate", `Basic realm="ProxyGate"`)
				http.Error(w, "Proxy Authentication Required", http.StatusProxyAuthRequired)
				return
			}
		case canBypassProxyAuth(cfg, r.RemoteAddr):
			// Local loopback clients can skip auth so browsers can use SOCKS/HTTP
			// without embedding credentials. Remote clients still require auth.
		default:
			w.Header().Set("Proxy-Authenticate", `Basic realm="ProxyGate"`)
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
	cfg := s.currentConfig()
	passwordMatch := subtle.ConstantTimeCompare([]byte(passwordHash), []byte(cfg.ProxyAuthPasswordHash)) == 1
	if !passwordMatch {
		return RequestOptions{}, false
	}

	opts, err := parseUsernameOptions(cfg.ProxyAuthUsername, username, s.sessionNamespace)
	if err != nil {
		return RequestOptions{}, false
	}

	return opts, true
}

// selectProxy 根据使用模式和选择策略获取代理
func (s *Server) selectProxy(tried []string, lowestLatency bool, opts RequestOptions) (*storage.Proxy, error) {
	cfg := s.currentConfig()
	if sticky := selectExistingStickyProxy(s.storage, s.sessions, "", tried, opts); sticky != nil {
		return sticky, nil
	}

	// "mixed" 模式下直接从统一候选池选择，允许 HTTP/SOCKS5 与 free/custom 真正混合。
	if cfg.CustomProxyMode == "mixed" {
		return selectFromPool(s.storage, s.sessions, "", s.sessionNamespace, "", tried, lowestLatency, opts)
	}

	sourceFilter := sourceFilterFromMode(cfg.CustomProxyMode)
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
	target := r.Host
	if r.URL != nil && r.URL.Host != "" {
		target = r.URL.Host
	}
	if !shouldPenalizeProxyForTarget(target) {
		log.Printf("[proxy] skip non-public target %s without penalizing proxy", target)
		http.Error(w, "target is not routable through upstream proxy", http.StatusBadGateway)
		return
	}

	var tried []string
	bodyReplayable := requestReplayable(r)
	maxAttempts := s.currentConfig().MaxRetry
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
			s.evictClient(p)
			removeOrDisableProxy(s.storage, p)
			continue
		}

		req, err := buildOutgoingRequest(r, bodyBytes, bodyReplayable)
		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[proxy] %s %s via %s failed, removing", r.Method, target, p.Address)
			s.storage.RecordProxyUse(p.Address, false)
			if opts.SessionKey != "" {
				s.sessions.Delete(opts.SessionKey)
			}
			s.evictClient(p)
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
			log.Printf("[proxy] ⚠️  429 %s %s via %s (protocol=%s)", r.Method, target, p.Address, p.Protocol)
		} else if resp.StatusCode >= 400 {
			log.Printf("[proxy] %s %s via %s -> %d", r.Method, target, p.Address, resp.StatusCode)
		} else if seq, ok := sampledSuccessLog(&httpSuccessLogSeq); ok {
			log.Printf("[proxy] sampled_success total=%d %s %s via %s -> %d", seq, r.Method, target, p.Address, resp.StatusCode)
		}
		return
	}

	http.Error(w, "all proxies failed", http.StatusBadGateway)
}

// handleTunnel 处理 HTTPS CONNECT 隧道（带自动重试）
func (s *Server) handleTunnel(w http.ResponseWriter, r *http.Request, opts RequestOptions) {
	if !shouldPenalizeProxyForTarget(r.Host) {
		log.Printf("[tunnel] skip non-public target %s without penalizing proxy", r.Host)
		http.Error(w, "target is not routable through upstream proxy", http.StatusBadGateway)
		return
	}

	var tried []string
	maxRetries := s.currentConfig().MaxRetry
	for attempt := 0; attempt <= maxRetries; attempt++ {
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
		if seq, ok := sampledSuccessLog(&tunnelEstablishedSeq); ok {
			log.Printf("[tunnel] sampled_established total=%d %s via %s", seq, r.Host, p.Address)
		}

		go func(proxyAddress string, upstreamConn, downstreamConn net.Conn, sessionKey string) {
			outcome := relayTunnel(downstreamConn, upstreamConn)
			if tunnelLooksHealthy(outcome) {
				s.storage.RecordProxyUse(proxyAddress, true)
				return
			}

			log.Printf(
				"[tunnel] %s via %s closed before upstream response, keep proxy (duration=%s client_bytes=%d upstream_bytes=%d)",
				r.Host,
				proxyAddress,
				outcome.duration.Truncate(time.Millisecond),
				outcome.clientBytes,
				outcome.upstreamBytes,
			)
		}(p.Address, conn, clientConn, opts.SessionKey)
		return
	}

	http.Error(w, "all proxies failed", http.StatusBadGateway)
}

func (s *Server) dialViaProxy(p *storage.Proxy, host string) (net.Conn, error) {
	timeout := time.Duration(s.currentConfig().ValidateTimeout) * time.Second
	switch p.Protocol {
	case "http":
		return dialHTTPConnect(p.Address, host, timeout)
	case "socks5":
		return dialSOCKS5Connect(p.Address, host, timeout)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", p.Protocol)
	}
}

func (s *Server) buildClient(p *storage.Proxy) (*http.Client, error) {
	timeout := time.Duration(s.currentConfig().ValidateTimeout) * time.Second
	cacheKey := fmt.Sprintf("%s|%s|%s", p.Protocol, p.Address, timeout)
	if cached, ok := s.clientCache.Load(cacheKey); ok {
		return cached.(*cachedHTTPClient).client, nil
	}

	baseDialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
	}
	transport := &http.Transport{
		DialContext:           baseDialer.DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          64,
		MaxIdleConnsPerHost:   4,
		MaxConnsPerHost:       8,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: time.Second,
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
		dialer, err := proxy.SOCKS5("tcp", p.Address, nil, baseDialer)
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

	entry := &cachedHTTPClient{client: client, transport: transport}
	actual, loaded := s.clientCache.LoadOrStore(cacheKey, entry)
	if loaded {
		transport.CloseIdleConnections()
		return actual.(*cachedHTTPClient).client, nil
	}
	return client, nil
}

func (s *Server) evictClient(p *storage.Proxy) {
	if p == nil {
		return
	}
	cacheKeyPrefix := p.Protocol + "|" + p.Address + "|"
	s.clientCache.Range(func(key, value any) bool {
		keyStr, ok := key.(string)
		if !ok || !strings.HasPrefix(keyStr, cacheKeyPrefix) {
			return true
		}
		s.clientCache.Delete(key)
		value.(*cachedHTTPClient).transport.CloseIdleConnections()
		return true
	})
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
