package checker

import (
	"context"
	"log"
	"time"

	"proxygate/config"
	"proxygate/pool"
	"proxygate/storage"
	"proxygate/validator"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	storage   *storage.Storage
	validator *validator.Validator
	cfg       *config.Config
	poolMgr   *pool.Manager
}

func NewHealthChecker(s *storage.Storage, v *validator.Validator, cfg *config.Config, pm *pool.Manager) *HealthChecker {
	return &HealthChecker{
		storage:   s,
		validator: v,
		cfg:       cfg,
		poolMgr:   pm,
	}
}

func (hc *HealthChecker) currentConfig() *config.Config {
	if cfg := config.Get(); cfg != nil {
		return cfg
	}
	return hc.cfg
}

// RunOnce 执行一次健康检查
func (hc *HealthChecker) RunOnce() {
	start := time.Now()
	log.Println("[health] 开始健康检查...")
	cfg := hc.currentConfig()
	validate := validator.New(cfg.HealthCheckConcurrency, cfg.ValidateTimeout, cfg.ValidateURL)
	if cfg.CustomProxyMode == "custom_only" {
		log.Println("[health] custom_only 模式，跳过免费池健康检查")
		return
	}

	// 获取池子状态
	status, err := hc.poolMgr.GetStatus()
	if err != nil {
		log.Printf("[health] 获取状态失败: %v", err)
		return
	}

	// 健康状态且S级占比高时，跳过S级代理检查
	skipSGrade := status.State == "healthy"
	dist, _ := hc.storage.GetQualityDistribution()
	sGradeCount := dist["S"]
	totalCount := status.Total
	if totalCount > 0 && float64(sGradeCount)/float64(totalCount) > 0.3 {
		skipSGrade = true
	}

	// 批量获取需要检查的代理
	proxies, err := hc.storage.GetBatchForHealthCheck(cfg.HealthCheckBatchSize, skipSGrade, "free")
	if err != nil {
		log.Printf("[health] 获取检查批次失败: %v", err)
		return
	}

	if len(proxies) == 0 {
		log.Println("[health] 无需检查的代理")
		return
	}

	log.Printf("[health] 检查 %d 个免费代理（跳过S级=%v）", len(proxies), skipSGrade)

	// 执行验证
	validCount := 0
	removeCount := 0
	updateCount := 0

	for result := range validate.ValidateStream(proxies) {
		if result.Valid {
			validCount++
			// 更新延迟和质量等级
			latencyMs := int(result.Latency.Milliseconds())
			if err := hc.storage.UpdateExitInfo(result.Proxy.Address, result.ExitIP, result.ExitLocation, latencyMs); err == nil {
				updateCount++
			}
		} else {
			// 失败次数+1
			hc.storage.IncrementFailCount(result.Proxy.Address)
			// 如果失败次数 >= 3
			if result.Proxy.FailCount+1 >= 3 {
				if result.Proxy.Source == "custom" {
					// 订阅代理：禁用而非删除
					hc.storage.DisableProxy(result.Proxy.Address)
				} else {
					hc.storage.Delete(result.Proxy.Address)
				}
				removeCount++
			}
		}
	}

	elapsed := time.Since(start)
	log.Printf("[health] 完成: 验证%d 有效%d 更新%d 移除%d 耗时%v",
		len(proxies), validCount, updateCount, removeCount, elapsed)
}

// StartBackground 后台定时健康检查
func (hc *HealthChecker) StartBackground(ctx context.Context) {
	go func() {
		for {
			cfg := hc.currentConfig()
			interval := time.Duration(cfg.HealthCheckInterval) * time.Minute
			if interval <= 0 {
				interval = 5 * time.Minute
			}
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				hc.RunOnce()
			}
		}
	}()
	log.Printf("[health] 健康检查器已启动，间隔 %d 分钟", hc.currentConfig().HealthCheckInterval)
}
