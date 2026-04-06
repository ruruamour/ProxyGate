package validator

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"goproxy/config"
	"goproxy/fetcher"
	"goproxy/storage"
)

const exitInfoTimeoutCap = 4 * time.Second

type Validator struct {
	concurrency int
	timeout     time.Duration
	validateURL string
	cfg         *config.Config
}

func concurrencyBuffer(total, concurrency int) int {
	if total < concurrency*10 {
		return total
	}
	return concurrency * 10
}

func New(concurrency, timeoutSec int, validateURL string) *Validator {
	return &Validator{
		concurrency: concurrency,
		timeout:     time.Duration(timeoutSec) * time.Second,
		validateURL: validateURL,
		cfg:         config.Get(),
	}
}

type Result struct {
	Proxy        storage.Proxy
	Valid        bool
	Latency      time.Duration
	ExitIP       string
	ExitLocation string
}

// HTTPS 测试目标列表，随机选一个验证代理的 CONNECT 隧道能力
var httpsTestTargets = []string{
	"https://www.google.com",
	"https://www.github.com",
	"https://www.cloudflare.com",
	"https://www.mozilla.org",
	"https://www.microsoft.com",
	"https://httpbin.org/ip",
}

// checkHTTPSConnect 通过 HTTP 代理实际访问一个随机 HTTPS 网站，验证 CONNECT 隧道是否可用
// 首次失败会换一个目标重试一次，避免目标网站偶尔抽风导致误杀
func checkHTTPSConnect(proxyAddr string, timeout time.Duration) bool {
	proxyURL, err := url.Parse(fmt.Sprintf("http://%s", proxyAddr))
	if err != nil {
		return false
	}

	transport := newProbeTransport(timeout)
	transport.Proxy = http.ProxyURL(proxyURL)
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
	defer transport.CloseIdleConnections()

	// 随机起始索引
	start := int(time.Now().UnixNano() % int64(len(httpsTestTargets)))

	for attempt := 0; attempt < 2; attempt++ {
		idx := (start + attempt) % len(httpsTestTargets)
		resp, err := client.Get(httpsTestTargets[idx])
		if err != nil {
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		// 2xx/3xx/4xx 都说明 CONNECT 隧道已经打通；
		// 这里只验证代理转发能力，不把目标站点策略误判成代理故障。
		if resp.StatusCode >= 200 && resp.StatusCode < 500 {
			return true
		}
	}

	return false
}

func checkReachability(client *http.Client, targets []string, attempts int, accept func(int) bool) (bool, time.Duration) {
	if len(targets) == 0 {
		return false, 0
	}
	if attempts < 1 {
		attempts = 1
	}
	if attempts > len(targets) {
		attempts = len(targets)
	}

	start := int(time.Now().UnixNano() % int64(len(targets)))
	for attempt := 0; attempt < attempts; attempt++ {
		target := targets[(start+attempt)%len(targets)]
		began := time.Now()
		resp, err := client.Get(target)
		latency := time.Since(began)
		if err != nil {
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if accept(resp.StatusCode) {
			return true, latency
		}
	}

	return false, 0
}

func probeFallbackTargets(cfg *config.Config) []string {
	if cfg == nil || len(cfg.ValidateFallbackURLs) == 0 {
		return nil
	}
	targets := make([]string, 0, len(cfg.ValidateFallbackURLs))
	seen := make(map[string]struct{}, len(cfg.ValidateFallbackURLs))
	for _, raw := range cfg.ValidateFallbackURLs {
		target := strings.TrimSpace(raw)
		if target == "" {
			continue
		}
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		targets = append(targets, target)
	}
	return targets
}

func validationTargets(primary string, cfg *config.Config) []string {
	targets := make([]string, 0, 1+len(probeFallbackTargets(cfg)))
	seen := make(map[string]struct{})

	appendTarget := func(raw string) {
		target := strings.TrimSpace(raw)
		if target == "" {
			return
		}
		if _, ok := seen[target]; ok {
			return
		}
		seen[target] = struct{}{}
		targets = append(targets, target)
	}

	appendTarget(primary)
	for _, target := range probeFallbackTargets(cfg) {
		appendTarget(target)
	}
	return targets
}

func validationAttempts(targets []string) int {
	switch n := len(targets); {
	case n <= 1:
		return n
	case n <= 4:
		return 2
	default:
		return 3
	}
}

func cloneClientWithTimeout(client *http.Client, timeout time.Duration) *http.Client {
	if client == nil {
		return nil
	}
	cloned := *client
	if timeout > 0 && (cloned.Timeout == 0 || cloned.Timeout > timeout) {
		cloned.Timeout = timeout
	}
	return &cloned
}

// ValidateAll 并发验证所有代理，返回验证结果
func (v *Validator) ValidateAll(proxies []storage.Proxy) []Result {
	var results []Result
	for r := range v.ValidateStreamContext(context.Background(), proxies) {
		results = append(results, r)
	}
	return results
}

// ValidateStream 并发验证，边验证边通过 channel 返回结果
func (v *Validator) ValidateStream(proxies []storage.Proxy) <-chan Result {
	return v.ValidateStreamContext(context.Background(), proxies)
}

// ValidateStreamContext 并发验证，支持外部取消，避免调用方提前退出后遗留 worker。
func (v *Validator) ValidateStreamContext(ctx context.Context, proxies []storage.Proxy) <-chan Result {
	ch := make(chan Result, concurrencyBuffer(len(proxies), v.concurrency))
	sem := make(chan struct{}, v.concurrency)
	var wg sync.WaitGroup

	go func() {
		for _, p := range proxies {
			select {
			case <-ctx.Done():
				wg.Wait()
				close(ch)
				return
			case sem <- struct{}{}:
			}

			wg.Add(1)
			go func(px storage.Proxy) {
				defer wg.Done()
				defer func() { <-sem }()

				valid, latency, exitIP, exitLocation := v.ValidateOne(px)
				result := Result{Proxy: px, Valid: valid, Latency: latency, ExitIP: exitIP, ExitLocation: exitLocation}

				select {
				case <-ctx.Done():
					return
				case ch <- result:
				}
			}(p)
		}
		wg.Wait()
		close(ch)
	}()

	return ch
}

// ValidateOne 验证单个代理是否可用，返回是否有效、延迟、出口IP和地理位置
func (v *Validator) ValidateOne(p storage.Proxy) (bool, time.Duration, string, string) {
	var client *http.Client
	var cleanup func()
	var err error

	switch p.Protocol {
	case "http":
		client, cleanup, err = newHTTPClient(p.Address, v.timeout)
	case "socks5":
		client, cleanup, err = newSOCKS5Client(p.Address, v.timeout)
	default:
		log.Printf("unknown protocol %s for %s", p.Protocol, p.Address)
		return false, 0, "", ""
	}

	if err != nil {
		return false, 0, "", ""
	}
	if cleanup != nil {
		defer cleanup()
	}

	targets := validationTargets(v.validateURL, v.cfg)
	ok, latency := checkReachability(client, targets, validationAttempts(targets), func(code int) bool {
		return code >= 200 && code < 500
	})
	if !ok {
		return false, latency, "", ""
	}

	// 获取出口 IP 和地理位置（仅在验证通过时）
	exitClient := cloneClientWithTimeout(client, exitInfoTimeoutCap)
	exitIP, exitLocation := fetcher.GetExitIPInfo(exitClient)

	if strings.TrimSpace(exitLocation) == "" {
		exitLocation = "UNKNOWN"
	}
	if strings.TrimSpace(exitIP) == "" {
		exitIP = "UNKNOWN"
	}

	// HTTP 代理额外检测：必须支持 HTTPS CONNECT 隧道
	if p.Protocol == "http" {
		if !checkHTTPSConnect(p.Address, v.timeout) {
			return false, latency, exitIP, exitLocation
		}
	}

	return true, latency, exitIP, exitLocation
}
