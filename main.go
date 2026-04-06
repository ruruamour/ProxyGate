package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"goproxy/checker"
	"goproxy/config"
	"goproxy/custom"
	"goproxy/fetcher"
	"goproxy/logger"
	"goproxy/optimizer"
	"goproxy/pool"
	"goproxy/proxy"
	"goproxy/storage"
	"goproxy/validator"
	"goproxy/webui"
)

var fetchRunning atomic.Bool

const maxHTTPProbeConcurrency = 96
const (
	refillCandidateMin          = 256
	refillCandidateMax          = 1536
	refillShortageMultiplier    = 64
	emergencyCandidateMin       = 512
	emergencyCandidateMax       = 3072
	emergencyShortageMultiplier = 96
)

func main() {
	// 初始化日志收集器
	logger.Init()

	// 本地开发时自动加载仓库根目录 .env（已存在的环境变量优先）
	if _, err := os.Stat(".env"); err == nil {
		if err := config.LoadDotEnv(".env"); err != nil {
			log.Printf("[main] ⚠️ 加载 .env 失败: %v", err)
		} else {
			log.Println("[main] 已加载本地 .env 配置")
		}
	}

	// 加载配置
	cfg := config.Load()

	// 提示密码信息
	if os.Getenv("WEBUI_PASSWORD") == "" {
		log.Printf("[main] WebUI 使用默认密码: %s（可通过环境变量 WEBUI_PASSWORD 自定义）", config.DefaultPassword)
	} else {
		log.Println("[main] WebUI 密码已通过环境变量 WEBUI_PASSWORD 设置")
	}

	log.Printf("[main] 🎯 智能代理池配置: 容量=%d HTTP=%.0f%% SOCKS5=%.0f%% 延迟标准=%dms",
		cfg.PoolMaxSize, cfg.PoolHTTPRatio*100, (1-cfg.PoolHTTPRatio)*100, cfg.MaxLatencyMs)

	// 初始化存储
	store, err := storage.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("init storage: %v", err)
	}
	defer store.Close()

	// 初始化限流器
	fetcher.InitIPQueryLimiter(cfg.IPQueryRateLimit)

	// 初始化核心模块
	sourceMgr := fetcher.NewSourceManager(store.GetDB())
	fetch := fetcher.New(cfg.HTTPSourceURL, cfg.SOCKS5SourceURL, sourceMgr)
	validate := validator.New(cfg.ValidateConcurrency, cfg.ValidateTimeout, cfg.ValidateURL)
	poolMgr := pool.NewManager(store, cfg)
	healthChecker := checker.NewHealthChecker(store, validate, cfg, poolMgr)
	opt := optimizer.NewOptimizer(store, fetch, validate, poolMgr, cfg)

	// 创建 HTTP 代理服务器：随机轮换 + 最低延迟
	sessionMgr := proxy.NewSessionManager()
	randomServer := proxy.New(store, cfg, sessionMgr, "random", cfg.ProxyPort)
	stableServer := proxy.New(store, cfg, sessionMgr, "lowest-latency", cfg.StableProxyPort)

	// 创建 SOCKS5 代理服务器：随机轮换 + 最低延迟
	socks5RandomServer := proxy.NewSOCKS5(store, cfg, sessionMgr, "random", cfg.SOCKS5Port)
	socks5StableServer := proxy.NewSOCKS5(store, cfg, sessionMgr, "lowest-latency", cfg.StableSOCKS5Port)

	// 初始化订阅管理器
	customMgr := custom.NewManager(store, validate, cfg)

	// 优雅退出时顺手停止 sing-box 子进程，避免本地开发重启后残留僵尸/占口。
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("[main] 收到退出信号 %s，正在停止后台组件...", sig)
		customMgr.Stop()
		os.Exit(0)
	}()

	// 配置变更通知 channel
	configChanged := make(chan struct{}, 1)

	// 启动 WebUI（传递池子管理器和订阅管理器）
	ui := webui.New(store, cfg, poolMgr, customMgr, func() {
		go smartFetchAndFill(fetch, validate, store, poolMgr)
	}, configChanged)
	ui.Start()

	// 首次智能填充（清理后立即触发）
	go func() {
		log.Println("[main] 🚀 启动初始化填充...")
		smartFetchAndFill(fetch, validate, store, poolMgr)
	}()

	// 启动状态监控协程
	go startStatusMonitor(poolMgr, fetch, validate, store)

	// 启动健康检查器
	healthChecker.StartBackground()

	// 启动优化轮换器
	opt.StartBackground()

	// 启动订阅管理器
	go customMgr.Start()

	// 监听配置变更
	go watchConfigChanges(configChanged, poolMgr)

	// 启动 HTTP 稳定代理服务（最低延迟模式）
	go func() {
		if err := stableServer.Start(); err != nil {
			log.Fatalf("stable http proxy server: %v", err)
		}
	}()

	// 启动 SOCKS5 稳定代理服务（最低延迟模式）
	go func() {
		if err := socks5StableServer.Start(); err != nil {
			log.Fatalf("stable socks5 proxy server: %v", err)
		}
	}()

	// 启动 SOCKS5 随机代理服务
	go func() {
		if err := socks5RandomServer.Start(); err != nil {
			log.Fatalf("random socks5 proxy server: %v", err)
		}
	}()

	// 启动 HTTP 随机代理服务（阻塞）
	if err := randomServer.Start(); err != nil {
		log.Fatalf("random http proxy server: %v", err)
	}
}

func cappedHTTPProbeConcurrency(total int) int {
	if total <= 0 {
		return 1
	}
	if total > maxHTTPProbeConcurrency {
		return maxHTTPProbeConcurrency
	}
	return total
}

func protocolShortage(status *pool.PoolStatus, protocol string) int {
	if status == nil {
		return 0
	}

	switch protocol {
	case "http":
		if missing := status.HTTPSlots - status.HTTP; missing > 0 {
			return missing
		}
	case "socks5":
		if missing := status.SOCKS5Slots - status.SOCKS5; missing > 0 {
			return missing
		}
	}

	return 0
}

func candidateBudget(mode string, shortage int) int {
	if shortage <= 0 {
		return 0
	}

	switch mode {
	case "refill":
		budget := shortage * refillShortageMultiplier
		if budget < refillCandidateMin {
			budget = refillCandidateMin
		}
		if budget > refillCandidateMax {
			budget = refillCandidateMax
		}
		return budget
	case "emergency":
		budget := shortage * emergencyShortageMultiplier
		if budget < emergencyCandidateMin {
			budget = emergencyCandidateMin
		}
		if budget > emergencyCandidateMax {
			budget = emergencyCandidateMax
		}
		return budget
	default:
		return 0
	}
}

func limitCandidatesForProtocol(mode string, status *pool.PoolStatus, protocol string, candidates []storage.Proxy) []storage.Proxy {
	if len(candidates) == 0 {
		return nil
	}
	if mode != "refill" && mode != "emergency" {
		return candidates
	}

	shortage := protocolShortage(status, protocol)
	if shortage <= 0 {
		return nil
	}

	budget := candidateBudget(mode, shortage)
	if budget <= 0 || len(candidates) <= budget {
		return candidates
	}

	limited := append([]storage.Proxy(nil), candidates...)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(limited), func(i, j int) {
		limited[i], limited[j] = limited[j], limited[i]
	})
	return limited[:budget]
}

// smartFetchAndFill 智能抓取和填充
func smartFetchAndFill(fetch *fetcher.Fetcher, validate *validator.Validator, store *storage.Storage, poolMgr *pool.Manager) {
	// 防止并发执行
	if !fetchRunning.CompareAndSwap(false, true) {
		log.Println("[main] 抓取已在运行，跳过")
		return
	}
	defer fetchRunning.Store(false)
	currentCfg := config.Get()
	httpProbeConcurrency := 0
	socks5ProbeConcurrency := 0

	// 获取池子状态
	status, err := poolMgr.GetStatus()
	if err != nil {
		log.Printf("[main] 获取池子状态失败: %v", err)
		return
	}

	log.Printf("[main] 📊 池子状态: %s | HTTP=%d/%d SOCKS5=%d/%d 总计=%d/%d",
		status.State, status.HTTP, status.HTTPSlots, status.SOCKS5, status.SOCKS5Slots,
		status.Total, config.Get().PoolMaxSize)

	// 判断是否需要抓取
	needFetch, mode, preferredProtocol := poolMgr.NeedsFetch(status)
	if !needFetch {
		log.Println("[main] 池子健康，无需抓取")
		return
	}

	log.Printf("[main] 🔍 智能抓取: 模式=%s 协议偏好=%s", mode, preferredProtocol)

	// 智能抓取
	candidates, err := fetch.FetchSmart(mode, preferredProtocol)
	if err != nil {
		log.Printf("[main] 抓取失败: %v", err)
		return
	}

	// 按协议分组
	var httpCandidates, socks5Candidates []storage.Proxy
	for _, c := range candidates {
		if c.Protocol == "http" {
			httpCandidates = append(httpCandidates, c)
		} else {
			socks5Candidates = append(socks5Candidates, c)
		}
	}

	rawHTTPCount := len(httpCandidates)
	rawSOCKS5Count := len(socks5Candidates)
	httpCandidates = limitCandidatesForProtocol(mode, status, "http", httpCandidates)
	socks5Candidates = limitCandidatesForProtocol(mode, status, "socks5", socks5Candidates)
	scheduledCount := len(httpCandidates) + len(socks5Candidates)
	if scheduledCount == 0 {
		log.Printf("[main] 当前没有需要补位的候选协议，跳过本轮验证（模式=%s）", mode)
		return
	}
	if rawHTTPCount != len(httpCandidates) {
		log.Printf("[main] HTTP 候选按缺口限流: %d -> %d", rawHTTPCount, len(httpCandidates))
	}
	if rawSOCKS5Count != len(socks5Candidates) {
		log.Printf("[main] SOCKS5 候选按缺口限流: %d -> %d", rawSOCKS5Count, len(socks5Candidates))
	}
	log.Printf("[main] 抓取到 %d 个候选代理（SOCKS5=%d HTTP=%d），本轮验证 %d 个（SOCKS5=%d HTTP=%d）...",
		len(candidates), rawSOCKS5Count, rawHTTPCount, scheduledCount, len(socks5Candidates), len(httpCandidates))

	switch {
	case len(httpCandidates) > 0 && len(socks5Candidates) > 0:
		httpProbeConcurrency = cappedHTTPProbeConcurrency(currentCfg.ValidateConcurrency / 2)
		socks5ProbeConcurrency = currentCfg.ValidateConcurrency - httpProbeConcurrency
		if socks5ProbeConcurrency < 1 {
			socks5ProbeConcurrency = 1
		}
	case len(httpCandidates) > 0:
		httpProbeConcurrency = cappedHTTPProbeConcurrency(currentCfg.ValidateConcurrency)
	case len(socks5Candidates) > 0:
		socks5ProbeConcurrency = currentCfg.ValidateConcurrency
	}

	log.Printf("[main] 验证并发预算: total=%d http=%d socks5=%d",
		currentCfg.ValidateConcurrency, httpProbeConcurrency, socks5ProbeConcurrency)

	// 共享计数器
	var addedCount atomic.Int32
	var validCount atomic.Int32
	var rejectedFull atomic.Int32

	// 入池处理函数（两个协程共用）
	processResult := func(result validator.Result) {
		if !result.Valid {
			return
		}

		validCount.Add(1)
		latencyMs := int(result.Latency.Milliseconds())
		exitIP := result.ExitIP
		if exitIP == "" {
			exitIP = "UNKNOWN"
		}
		exitLocation := result.ExitLocation
		if exitLocation == "" {
			exitLocation = "UNKNOWN"
		}

		proxyToAdd := storage.Proxy{
			Address:      result.Proxy.Address,
			Protocol:     result.Proxy.Protocol,
			ExitIP:       exitIP,
			ExitLocation: exitLocation,
			Latency:      latencyMs,
		}

		if added, reason := poolMgr.TryAddProxy(proxyToAdd); added {
			addedCount.Add(1)
		} else if reason == "slots_full" {
			rejectedFull.Add(1)
		}
	}

	// 池子是否已满的检查函数
	poolFilled := func() bool {
		currentStatus, _ := poolMgr.GetStatus()
		return !poolMgr.NeedsFetchQuick(currentStatus)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// SOCKS5 协程：验证快，优先填充
	if len(socks5Candidates) > 0 {
		socks5Validator := validator.New(socks5ProbeConcurrency, currentCfg.ValidateTimeout, currentCfg.ValidateURL)
		wg.Add(1)
		go func() {
			defer wg.Done()
			count := 0
			stopped := false
			for result := range socks5Validator.ValidateStreamContext(ctx, socks5Candidates) {
				if stopped {
					continue
				}
				processResult(result)
				count++
				if count%20 == 0 && poolFilled() {
					log.Println("[main] ✅ SOCKS5 验证中检测到池子已满，停止")
					stopped = true
					cancel()
				}
			}
			log.Printf("[main] SOCKS5 验证完成，处理 %d 个", count)
		}()
	}

	// HTTP 协程：有额外 HTTPS 检测，较慢
	if len(httpCandidates) > 0 {
		httpValidator := validator.New(httpProbeConcurrency, currentCfg.ValidateTimeout, currentCfg.ValidateURL)
		wg.Add(1)
		go func() {
			defer wg.Done()
			count := 0
			stopped := false
			for result := range httpValidator.ValidateStreamContext(ctx, httpCandidates) {
				if stopped {
					continue
				}
				processResult(result)
				count++
				if count%20 == 0 && poolFilled() {
					log.Println("[main] ✅ HTTP 验证中检测到池子已满，停止")
					stopped = true
					cancel()
				}
			}
			log.Printf("[main] HTTP 验证完成，处理 %d 个", count)
		}()
	}

	wg.Wait()

	// 最终状态
	finalStatus, _ := poolMgr.GetStatus()
	log.Printf("[main] 填充完成: 验证%d 通过%d 入池%d | 拒绝[满:%d] | 最终: %s HTTP=%d SOCKS5=%d",
		scheduledCount, validCount.Load(), addedCount.Load(),
		rejectedFull.Load(),
		finalStatus.State, finalStatus.HTTP, finalStatus.SOCKS5)
}

// startStatusMonitor 状态监控协程
func startStatusMonitor(poolMgr *pool.Manager, fetch *fetcher.Fetcher, validate *validator.Validator, store *storage.Storage) {
	ticker := time.NewTicker(30 * time.Second)
	log.Println("[monitor] 📡 状态监控器已启动（每30秒检查）")

	for range ticker.C {
		status, err := poolMgr.GetStatus()
		if err != nil {
			continue
		}

		// 每分钟检查池子状态
		needFetch, mode, preferredProtocol := poolMgr.NeedsFetch(status)
		if needFetch {
			log.Printf("[monitor] ⚠️  检测到池子需求: 状态=%s 模式=%s 协议=%s",
				status.State, mode, preferredProtocol)
			// 触发智能填充
			go smartFetchAndFill(fetch, validate, store, poolMgr)
		}
	}
}

// watchConfigChanges 监听配置变更
func watchConfigChanges(configChanged <-chan struct{}, poolMgr *pool.Manager) {
	var oldSize int
	var oldRatio float64

	cfg := config.Get()
	oldSize = cfg.PoolMaxSize
	oldRatio = cfg.PoolHTTPRatio

	for range configChanged {
		newCfg := config.Get()
		if newCfg.PoolMaxSize != oldSize || newCfg.PoolHTTPRatio != oldRatio {
			log.Printf("[config] 🔧 配置变更检测: 容量 %d→%d 比例 %.2f→%.2f",
				oldSize, newCfg.PoolMaxSize, oldRatio, newCfg.PoolHTTPRatio)
			poolMgr.AdjustForConfigChange(oldSize, oldRatio)
			oldSize = newCfg.PoolMaxSize
			oldRatio = newCfg.PoolHTTPRatio
		}
	}
}
