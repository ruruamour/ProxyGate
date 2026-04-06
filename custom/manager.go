package custom

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"goproxy/config"
	"goproxy/storage"
	"goproxy/validator"
)

var subscriptionUserAgents = []string{
	"v2rayN",
	"Mihomo Meta v1.19.14 linux amd64 with go1.24.7",
	"clash-verge/v2.0.0",
	"Clash.Meta",
}

// Manager 订阅管理器
type Manager struct {
	storage   *storage.Storage
	validator *validator.Validator
	singbox   *SingBoxProcess
	nodeCache map[int64][]ParsedNode
	stopCh    chan struct{}
	refreshMu sync.Mutex // 防止并发刷新
}

type preparedSubscription struct {
	sub   storage.Subscription
	nodes []ParsedNode
}

// NewManager 创建订阅管理器
func NewManager(store *storage.Storage, v *validator.Validator, cfg *config.Config) *Manager {
	dataDir := ""
	if d := os.Getenv("DATA_DIR"); d != "" {
		dataDir = d
	}

	return &Manager{
		storage:   store,
		validator: v,
		singbox:   NewSingBoxProcess(cfg.SingBoxPath, dataDir, cfg.SingBoxBasePort),
		nodeCache: make(map[int64][]ParsedNode),
		stopCh:    make(chan struct{}),
	}
}

// Start 启动后台循环
func (m *Manager) Start() {
	log.Println("[custom] 订阅管理器启动")

	// 启动时立即刷新所有订阅
	go m.initialRefresh()

	// 订阅刷新循环
	go m.refreshLoop()

	// 探测唤醒循环
	go m.probeLoop()
}

// Stop 停止管理器
func (m *Manager) Stop() {
	close(m.stopCh)
	m.singbox.Stop()
	log.Println("[custom] 订阅管理器已停止")
}

// initialRefresh 启动时刷新所有活跃订阅
func (m *Manager) initialRefresh() {
	time.Sleep(3 * time.Second) // 等待其他模块初始化
	subs, err := m.storage.GetSubscriptions()
	if err != nil || len(subs) == 0 {
		return
	}

	activeSubs := 0
	for _, sub := range subs {
		if sub.Status == "active" {
			activeSubs++
		}
	}
	if activeSubs == 0 {
		return
	}

	log.Printf("[custom] 启动刷新，共 %d 个活跃订阅", activeSubs)
	m.RefreshAll()
}

// refreshLoop 订阅刷新循环
func (m *Manager) refreshLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkAndRefresh()
		}
	}
}

// checkAndRefresh 检查并刷新到期的订阅 + 清理长期无可用节点的订阅
func (m *Manager) checkAndRefresh() {
	// 清理连续 7 天无可用节点的订阅
	m.cleanupStaleSubscriptions()

	subs, err := m.storage.GetSubscriptions()
	if err != nil {
		log.Printf("[custom] 获取订阅列表失败: %v", err)
		return
	}

	var dueSubs []*storage.Subscription
	for i := range subs {
		sub := &subs[i]
		if sub.Status != "active" {
			continue
		}
		// 检查是否到刷新时间
		if !sub.LastFetch.IsZero() && time.Since(sub.LastFetch) < time.Duration(sub.RefreshMin)*time.Minute {
			continue
		}
		dueSubs = append(dueSubs, sub)
	}

	if len(dueSubs) == 0 {
		return
	}

	for _, sub := range dueSubs {
		log.Printf("[custom] 🔄 订阅 [%s] 到期，加入本轮批量刷新", sub.Name)
	}

	m.refreshMu.Lock()
	defer m.refreshMu.Unlock()

	pendingValidation := m.refreshSubscriptionsLocked(dueSubs, false)
	for subID, proxies := range pendingValidation {
		if len(proxies) == 0 {
			continue
		}
		m.validateCustomProxies(proxies, subID)
	}
}

// cleanupStaleSubscriptions 清理连续 7 天无可用节点的订阅
func (m *Manager) cleanupStaleSubscriptions() {
	staleSubs, err := m.storage.GetStaleSubscriptions(7)
	if err != nil || len(staleSubs) == 0 {
		return
	}

	for _, sub := range staleSubs {
		deleted, _ := m.storage.DeleteBySubscriptionID(sub.ID)
		m.storage.DeleteSubscription(sub.ID)
		log.Printf("[custom] 🗑️ 自动移除订阅 [%s]：连续 7 天无可用节点（清理 %d 个代理）", sub.Name, deleted)
	}

	// 重建 sing-box 配置
	if len(staleSubs) > 0 {
		m.RefreshAll()
	}
}

// probeLoop 探测唤醒循环
func (m *Manager) probeLoop() {
	// 等待初始化完成
	time.Sleep(5 * time.Second)

	for {
		cfg := config.Get()
		interval := time.Duration(cfg.CustomProbeInterval) * time.Minute
		if interval < time.Minute {
			interval = 10 * time.Minute
		}

		select {
		case <-m.stopCh:
			return
		case <-time.After(interval):
			m.probeDisabled()
		}
	}
}

// probeDisabled 探测被禁用的订阅代理
func (m *Manager) probeDisabled() {
	disabled, err := m.storage.GetDisabledCustomProxies()
	if err != nil || len(disabled) == 0 {
		return
	}

	log.Printf("[custom] 🔍 探测 %d 个禁用的订阅代理", len(disabled))

	recovered := 0
	recoveredSubs := make(map[int64]bool)
	for _, proxy := range disabled {
		valid, latency, exitIP, exitLocation := m.validator.ValidateOne(proxy)
		if valid {
			latencyMs := int(latency.Milliseconds())
			m.storage.EnableProxy(proxy.Address)
			m.storage.UpdateExitInfo(proxy.Address, exitIP, exitLocation, latencyMs)
			recovered++
			recoveredSubs[proxy.SubscriptionID] = true
			log.Printf("[custom] ✅ 代理 %s 恢复可用 (%dms %s)", proxy.Address, latency.Milliseconds(), exitLocation)
		}
	}
	// 有恢复的代理则更新对应订阅的 last_success
	for subID := range recoveredSubs {
		if subID > 0 {
			m.storage.UpdateSubscriptionSuccess(subID)
		}
	}

	if recovered > 0 {
		log.Printf("[custom] 探测完成：%d/%d 恢复可用", recovered, len(disabled))
	}
}

// RefreshSubscription 刷新单个订阅
func (m *Manager) RefreshSubscription(subID int64) error {
	m.refreshMu.Lock()
	defer m.refreshMu.Unlock()
	proxies, err := m.refreshSubscriptionLocked(subID)
	if err != nil {
		return err
	}
	if len(proxies) > 0 {
		m.validateCustomProxies(proxies, subID)
	}
	return nil
}

func (m *Manager) refreshSubscriptionLocked(subID int64) ([]storage.Proxy, error) {
	sub, err := m.storage.GetSubscription(subID)
	if err != nil {
		return nil, fmt.Errorf("获取订阅失败: %w", err)
	}
	if sub.Status != "active" {
		delete(m.nodeCache, subID)
		if deleted, _ := m.storage.DeleteBySubscriptionID(subID); deleted > 0 {
			log.Printf("[custom] 🧹 清理已暂停订阅 [%s] 旧代理 %d 个", sub.Name, deleted)
		}
		if err := m.reloadAllTunnelNodesLocked(); err != nil {
			return nil, fmt.Errorf("重载 sing-box 失败: %w", err)
		}
		return nil, nil
	}

	plan, err := m.prepareActiveSubscriptionLocked(sub)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, nil
	}

	reloadOK := true
	if err := m.reloadAllTunnelNodesLocked(); err != nil {
		log.Printf("[custom] ❌ sing-box 重载失败: %v", err)
		reloadOK = false
	}

	return m.applyPreparedSubscriptionLocked(plan, reloadOK), nil
}

// RefreshAll 刷新所有活跃订阅
func (m *Manager) RefreshAll() {
	m.refreshMu.Lock()
	defer m.refreshMu.Unlock()

	subs, err := m.storage.GetSubscriptions()
	if err != nil {
		log.Printf("[custom] 获取订阅列表失败: %v", err)
		return
	}

	liveSubs := make(map[int64]bool, len(subs))
	activeCount := 0
	reloadNeeded := false
	for _, sub := range subs {
		liveSubs[sub.ID] = true
		if sub.Status != "active" {
			delete(m.nodeCache, sub.ID)
			if deleted, _ := m.storage.DeleteBySubscriptionID(sub.ID); deleted > 0 {
				log.Printf("[custom] 🧹 清理非活跃订阅 [%s] 代理 %d 个", sub.Name, deleted)
				reloadNeeded = true
			}
			continue
		}
		activeCount++
	}
	for subID := range m.nodeCache {
		if !liveSubs[subID] {
			delete(m.nodeCache, subID)
		}
	}

	if activeCount == 0 {
		if deleted, _ := m.storage.DeleteBySource("custom"); deleted > 0 {
			log.Printf("[custom] 🧹 已清理全部订阅代理 %d 个", deleted)
		}
		if err := m.singbox.Reload(nil); err != nil {
			log.Printf("[custom] ❌ 清空 sing-box 失败: %v", err)
		}
		return
	}

	activeSubs := make([]*storage.Subscription, 0, activeCount)
	for _, sub := range subs {
		if sub.Status != "active" {
			continue
		}
		subCopy := sub
		activeSubs = append(activeSubs, &subCopy)
	}

	pendingValidation := m.refreshSubscriptionsLocked(activeSubs, reloadNeeded)
	for subID, proxies := range pendingValidation {
		if len(proxies) == 0 {
			continue
		}
		m.validateCustomProxies(proxies, subID)
	}
}

func (m *Manager) refreshSubscriptionsLocked(subs []*storage.Subscription, forceReload bool) map[int64][]storage.Proxy {
	pendingValidation := make(map[int64][]storage.Proxy, len(subs))
	prepared := make(map[int64]*preparedSubscription, len(subs))
	reloadNeeded := forceReload

	for _, sub := range subs {
		if sub == nil || sub.Status != "active" {
			continue
		}
		plan, err := m.prepareActiveSubscriptionLocked(sub)
		if err != nil {
			log.Printf("[custom] ❌ 订阅 [%s] 刷新失败: %v", sub.Name, err)
			continue
		}
		if plan == nil {
			continue
		}
		prepared[sub.ID] = plan
		reloadNeeded = true
	}

	reloadOK := true
	if reloadNeeded {
		if err := m.reloadAllTunnelNodesLocked(); err != nil {
			log.Printf("[custom] ❌ sing-box 重载失败: %v", err)
			reloadOK = false
		}
	}

	for subID, plan := range prepared {
		pendingValidation[subID] = m.applyPreparedSubscriptionLocked(plan, reloadOK)
	}

	return pendingValidation
}

func (m *Manager) prepareActiveSubscriptionLocked(sub *storage.Subscription) (*preparedSubscription, error) {
	data, err := m.fetchSubscriptionData(sub)
	if err != nil {
		return nil, fmt.Errorf("拉取订阅内容失败: %w", err)
	}

	nodes, err := Parse(data, sub.Format)
	if err != nil {
		return nil, fmt.Errorf("解析订阅内容失败: %w", err)
	}

	if len(nodes) == 0 {
		log.Printf("[custom] ⚠️ 订阅 [%s] 无有效节点", sub.Name)
		return nil, nil
	}

	log.Printf("[custom] 订阅 [%s] 解析到 %d 个节点", sub.Name, len(nodes))
	m.nodeCache[sub.ID] = append([]ParsedNode(nil), nodes...)

	return &preparedSubscription{
		sub:   *sub,
		nodes: append([]ParsedNode(nil), nodes...),
	}, nil
}

func (m *Manager) applyPreparedSubscriptionLocked(plan *preparedSubscription, reloadOK bool) []storage.Proxy {
	if plan == nil {
		return nil
	}

	sub := plan.sub
	nodes := plan.nodes

	oldDeleted, _ := m.storage.DeleteBySubscriptionID(sub.ID)
	if oldDeleted > 0 {
		log.Printf("[custom] 🧹 清理订阅 [%s] 旧代理 %d 个", sub.Name, oldDeleted)
	}

	var directNodes []ParsedNode
	var tunnelNodes []ParsedNode
	for _, node := range nodes {
		if node.IsDirect() {
			directNodes = append(directNodes, node)
		} else {
			tunnelNodes = append(tunnelNodes, node)
		}
	}

	var allProxies []storage.Proxy
	for _, node := range directNodes {
		addr := node.DirectAddress()
		proto := node.DirectProtocol()
		m.storage.AddProxyWithSource(addr, proto, "custom", sub.ID)
		allProxies = append(allProxies, storage.Proxy{Address: addr, Protocol: proto, Source: "custom"})
	}
	if len(directNodes) > 0 {
		log.Printf("[custom] 📥 %d 个 HTTP/SOCKS5 节点直接入池", len(directNodes))
	}

	if reloadOK {
		portMap := m.singbox.GetPortMap()
		for _, node := range tunnelNodes {
			key := node.NodeKey()
			if port, ok := portMap[key]; ok {
				addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
				m.storage.AddProxyWithSource(addr, "socks5", "custom", sub.ID)
				allProxies = append(allProxies, storage.Proxy{Address: addr, Protocol: "socks5", Source: "custom"})
			}
		}
		if len(tunnelNodes) > 0 {
			log.Printf("[custom] 📥 %d 个加密节点通过 sing-box 转换入池", len(tunnelNodes))
		}
	} else if len(tunnelNodes) > 0 {
		log.Printf("[custom] ⚠️ 订阅 [%s] 的 %d 个隧道节点本轮未入池（sing-box 未就绪）", sub.Name, len(tunnelNodes))
	}

	m.storage.UpdateSubscriptionFetch(sub.ID, len(allProxies))
	log.Printf("[custom] ✅ 订阅 [%s] 刷新完成，解析 %d 节点，入池 %d 个", sub.Name, len(nodes), len(allProxies))
	return allProxies
}

func (m *Manager) reloadAllTunnelNodesLocked() error {
	subs, err := m.storage.GetSubscriptions()
	if err != nil {
		return fmt.Errorf("获取订阅列表失败: %w", err)
	}

	var allNodes []ParsedNode
	for _, sub := range subs {
		if sub.Status != "active" {
			delete(m.nodeCache, sub.ID)
			continue
		}
		nodes, ok := m.nodeCache[sub.ID]
		if !ok {
			log.Printf("[custom] ⚠️ 订阅 [%s] 尚无缓存节点，等待本次刷新完成后再并入 sing-box", sub.Name)
			continue
		}
		for _, node := range nodes {
			if !node.IsDirect() {
				allNodes = append(allNodes, node)
			}
		}
	}
	return m.singbox.Reload(allNodes)
}

// fetchSubscriptionData 获取订阅数据
func (m *Manager) fetchSubscriptionData(sub *storage.Subscription) ([]byte, error) {
	// 优先使用本地文件
	if sub.FilePath != "" {
		data, err := os.ReadFile(sub.FilePath)
		if err != nil {
			return nil, fmt.Errorf("读取文件 %s 失败: %w", sub.FilePath, err)
		}
		return data, nil
	}

	// 从 URL 拉取
	if sub.URL == "" {
		return nil, fmt.Errorf("订阅未配置 URL 或文件路径")
	}

	// 尝试拉取（直连 → 代理）
	data, err := m.fetchWithRetry(sub.URL)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// fetchWithRetry 使用多个常见 User-Agent 直连拉取订阅。
// 订阅 URL 往往携带 token，绝不能转交给池内代理，避免泄露认证信息。
func (m *Manager) fetchWithRetry(urlStr string) ([]byte, error) {
	return m.fetchURL(urlStr)
}

// fetchURL 直连拉取 URL 内容。
func (m *Manager) fetchURL(urlStr string) ([]byte, error) {
	transport := &http.Transport{}
	client := &http.Client{Timeout: 30 * time.Second, Transport: transport}
	var lastErr error
	for _, ua := range subscriptionUserAgents {
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", ua)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}
		if len(strings.TrimSpace(string(body))) == 0 {
			lastErr = fmt.Errorf("empty response with user-agent %q", ua)
			continue
		}

		if ua != subscriptionUserAgents[0] {
			log.Printf("[custom] 订阅 URL 使用备用 User-Agent 成功: %s", ua)
		}
		return body, nil
	}

	return nil, lastErr
}

// validateCustomProxies 验证订阅代理，返回可用数
func (m *Manager) validateCustomProxies(proxies []storage.Proxy, subID int64) int {
	if len(proxies) == 0 {
		return 0
	}

	log.Printf("[custom] 🔍 开始验证 %d 个订阅代理", len(proxies))

	resultCh := m.validator.ValidateStream(proxies)
	valid, invalid := 0, 0
	for result := range resultCh {
		if result.Valid {
			latencyMs := int(result.Latency.Milliseconds())
			m.storage.UpdateExitInfo(result.Proxy.Address, result.ExitIP, result.ExitLocation, latencyMs)
			m.storage.EnableProxy(result.Proxy.Address)
			valid++
		} else {
			invalid++
			m.storage.DisableProxy(result.Proxy.Address)
		}
	}

	// 有可用节点则更新 last_success
	if valid > 0 && subID > 0 {
		m.storage.UpdateSubscriptionSuccess(subID)
	}

	log.Printf("[custom] 验证完成：%d 可用，%d 不可用", valid, invalid)
	return valid
}
// GetStatus 获取订阅管理器状态
func (m *Manager) GetStatus() map[string]interface{} {
	customCount, _ := m.storage.CountBySource("custom")
	disabled, _ := m.storage.GetDisabledCustomProxies()
	subs, _ := m.storage.GetSubscriptions()

	return map[string]interface{}{
		"singbox_running":    m.singbox.IsRunning(),
		"singbox_nodes":      m.singbox.GetNodeCount(),
		"custom_count":       customCount,
		"disabled_count":     len(disabled),
		"subscription_count": len(subs),
	}
}

// ValidateSubscription 验证订阅能否解析出节点（不入库，仅检查）
func (m *Manager) ValidateSubscription(url, filePath string) (int, error) {
	var data []byte
	var err error

	if filePath != "" {
		data, err = os.ReadFile(filePath)
		if err != nil {
			return 0, fmt.Errorf("读取文件失败: %w", err)
		}
	} else if url != "" {
		data, err = m.fetchWithRetry(url)
		if err != nil {
			return 0, err
		}
	} else {
		return 0, fmt.Errorf("未提供 URL 或文件")
	}

	nodes, err := Parse(data, "auto")
	if err != nil {
		return 0, err
	}
	if len(nodes) == 0 {
		return 0, fmt.Errorf("解析结果为空，未找到有效代理节点")
	}

	return len(nodes), nil
}

// GetSingBox 获取 sing-box 进程管理器
func (m *Manager) GetSingBox() *SingBoxProcess {
	return m.singbox
}
