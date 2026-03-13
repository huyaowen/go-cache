package job

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/coderiser/go-cache/pkg/core"
	"github.com/coderiser/go-cache/examples/cron-job/service"
)

// CacheRefreshJob 缓存刷新任务
type CacheRefreshJob struct {
	manager  core.CacheManager
	service  service.ProductServiceInterface
	interval time.Duration
	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
	stats    *RefreshStats
}

// RefreshStats 刷新统计
type RefreshStats struct {
	TotalRuns   int64         // 总运行次数
	SuccessRuns int64         // 成功运行次数
	FailedRuns  int64         // 失败运行次数
	LastRun     time.Time     // 上次运行时间
	NextRun     time.Time     // 下次运行时间
	TotalTime   time.Duration // 总耗时
}

// RefreshConfig 刷新配置
type RefreshConfig struct {
	// 刷新间隔
	Interval time.Duration
	// 是否预热热点数据
	WarmupHotProducts bool
	// 是否刷新配置缓存
	RefreshConfig bool
	// 是否记录统计
	LogStats bool
	// 统计输出间隔（每 N 次输出一次）
	StatsInterval int64
}

// DefaultRefreshConfig 默认刷新配置
func DefaultRefreshConfig() *RefreshConfig {
	return &RefreshConfig{
		Interval:          5 * time.Minute,
		WarmupHotProducts: true,
		RefreshConfig:     true,
		LogStats:          true,
		StatsInterval:     6, // 每 30 分钟输出一次统计（5m * 6）
	}
}

// NewCacheRefreshJob 创建缓存刷新任务
func NewCacheRefreshJob(manager core.CacheManager, service service.ProductServiceInterface, interval time.Duration) *CacheRefreshJob {
	return &CacheRefreshJob{
		manager:  manager,
		service:  service,
		interval: interval,
		stopChan: make(chan struct{}),
		stats:    &RefreshStats{},
	}
}

// Start 启动定时刷新任务
func (j *CacheRefreshJob) Start(ctx context.Context, config *RefreshConfig) {
	if config == nil {
		config = DefaultRefreshConfig()
	}

	j.mu.Lock()
	if j.running {
		j.mu.Unlock()
		log.Println("[WARN] Cache refresh job is already running")
		return
	}
	j.running = true
	j.mu.Unlock()

	log.Printf("[INFO] Scheduled cache refresh every %v", config.Interval)

	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	// 立即执行一次
	j.refreshOnce(ctx, config)

	for {
		select {
		case <-ticker.C:
			j.refreshOnce(ctx, config)
		case <-j.stopChan:
			log.Println("[INFO] Cache refresh job stopped")
			j.mu.Lock()
			j.running = false
			j.mu.Unlock()
			return
		case <-ctx.Done():
			log.Println("[INFO] Cache refresh job cancelled by context")
			j.mu.Lock()
			j.running = false
			j.mu.Unlock()
			return
		}
	}
}

// refreshOnce 执行一次刷新
func (j *CacheRefreshJob) refreshOnce(ctx context.Context, config *RefreshConfig) {
	startTime := time.Now()

	j.mu.Lock()
	j.stats.TotalRuns++
	j.stats.LastRun = startTime
	j.stats.NextRun = startTime.Add(config.Interval)
	j.mu.Unlock()

	log.Printf("[INFO] Running cache refresh #%d...", j.stats.TotalRuns)

	var success bool
	defer func() {
		j.mu.Lock()
		if success {
			j.stats.SuccessRuns++
		} else {
			j.stats.FailedRuns++
		}
		j.stats.TotalTime += time.Since(startTime)
		j.mu.Unlock()
	}()

	// 1. 刷新热点商品
	if config.WarmupHotProducts {
		if err := j.refreshHotProducts(ctx); err != nil {
			log.Printf("[ERROR] Failed to refresh hot products: %v", err)
			success = false
			return
		}
	}

	// 2. 刷新配置缓存
	if config.RefreshConfig {
		if err := j.refreshConfig(ctx); err != nil {
			log.Printf("[ERROR] Failed to refresh config: %v", err)
			success = false
			return
		}
	}

	success = true
	log.Printf("[INFO] Cache refresh completed in %v", time.Since(startTime))

	// 输出统计
	if config.LogStats && j.stats.TotalRuns%config.StatsInterval == 0 {
		j.printStats()
	}
}

// refreshHotProducts 刷新热点商品缓存
func (j *CacheRefreshJob) refreshHotProducts(ctx context.Context) error {
	cache, err := j.manager.GetCache("hot_products")
	if err != nil {
		return err
	}

	products, err := j.service.GetHotProducts()
	if err != nil {
		return err
	}

	if err := cache.Set(ctx, "list", products, 5*time.Minute); err != nil {
		return err
	}

	log.Printf("[DEBUG] Refreshed %d hot products", len(products))
	return nil
}

// refreshConfig 刷新配置缓存（示例）
func (j *CacheRefreshJob) refreshConfig(ctx context.Context) error {
	cache, err := j.manager.GetCache("config")
	if err != nil {
		return err
	}

	// 模拟配置数据
	config := map[string]interface{}{
		"version":     "1.0.0",
		"updated_at":  time.Now().Unix(),
		"max_price":   999.99,
		"min_stock":   10,
	}

	if err := cache.Set(ctx, "app_config", config, 10*time.Minute); err != nil {
		return err
	}

	log.Printf("[DEBUG] Refreshed config cache")
	return nil
}

// Stop 停止刷新任务
func (j *CacheRefreshJob) Stop() {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.running {
		return
	}

	close(j.stopChan)
	log.Println("[INFO] Stopping cache refresh job...")
}

// IsRunning 检查是否正在运行
func (j *CacheRefreshJob) IsRunning() bool {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.running
}

// GetStats 获取统计信息
func (j *CacheRefreshJob) GetStats() *RefreshStats {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.stats
}

// printStats 输出统计信息
func (j *CacheRefreshJob) printStats() {
	j.mu.RLock()
	defer j.mu.RUnlock()

	avgTime := time.Duration(0)
	if j.stats.TotalRuns > 0 {
		avgTime = j.stats.TotalTime / time.Duration(j.stats.TotalRuns)
	}

	log.Println("=== Cache Refresh Stats ===")
	log.Printf("  Total Runs:   %d", j.stats.TotalRuns)
	log.Printf("  Success:      %d (%.1f%%)", j.stats.SuccessRuns, 
		float64(j.stats.SuccessRuns)/float64(j.stats.TotalRuns)*100)
	log.Printf("  Failed:       %d", j.stats.FailedRuns)
	log.Printf("  Avg Duration: %v", avgTime)
	log.Printf("  Last Run:     %v", j.stats.LastRun.Format("2006-01-02 15:04:05"))
	log.Printf("  Next Run:     %v", j.stats.NextRun.Format("2006-01-02 15:04:05"))
	log.Println("===========================")
}

// RunWithGracefulShutdown 运行并支持优雅关闭
func (j *CacheRefreshJob) RunWithGracefulShutdown(ctx context.Context, config *RefreshConfig) {
	// 创建带信号监听的上下文
	sigCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 监听信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("[INFO] Received signal %v, initiating graceful shutdown...", sig)
		cancel()
	}()

	// 启动刷新任务
	j.Start(sigCtx, config)
}
