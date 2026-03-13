package job

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/coderiser/go-cache/pkg/core"
	"github.com/coderiser/go-cache/examples/cron-job/service"
)

// CacheWarmupJob 缓存预热任务
type CacheWarmupJob struct {
	manager core.CacheManager
	service service.ProductServiceInterface
}

// NewCacheWarmupJob 创建缓存预热任务
func NewCacheWarmupJob(manager core.CacheManager, service service.ProductServiceInterface) *CacheWarmupJob {
	return &CacheWarmupJob{
		manager: manager,
		service: service,
	}
}

// WarmupConfig 预热配置
type WarmupConfig struct {
	// 并发数
	Concurrency int
	// 单个条目超时时间
	Timeout time.Duration
	// 是否跳过已存在的缓存
	SkipExisting bool
	// 预热商品 ID 列表
	ProductIDs []int64
}

// DefaultWarmupConfig 默认预热配置
func DefaultWarmupConfig() *WarmupConfig {
	return &WarmupConfig{
		Concurrency:  4,
		Timeout:      10 * time.Second,
		SkipExisting: false,
		ProductIDs:   []int64{1, 2, 3, 4, 5},
	}
}

// WarmupCache 执行缓存预热
func (j *CacheWarmupJob) WarmupCache(ctx context.Context, config *WarmupConfig) error {
	if config == nil {
		config = DefaultWarmupConfig()
	}

	log.Println("[INFO] Starting cache warmup...")
	startTime := time.Now()

	// 创建预热器
	warmer := core.NewCacheWarmer(j.manager)

	// 准备预热条目
	entries := make([]core.WarmUpEntry, 0)

	// 1. 预热热点商品列表
	entries = append(entries, core.WarmUpEntry{
		Cache: "hot_products",
		Key:   "list",
		Loader: func() (interface{}, error) {
			return j.service.GetHotProducts()
		},
		TTL: 5 * time.Minute,
	})

	// 2. 预热指定商品
	for _, id := range config.ProductIDs {
		productID := id // 闭包变量捕获
		entries = append(entries, core.WarmUpEntry{
			Cache: "products",
			Key:   fmt.Sprintf("%d", productID),
			Loader: func() (interface{}, error) {
				product := j.service.GetProductByID(productID)
				if product == nil {
					return nil, fmt.Errorf("product %d not found", productID)
				}
				return product, nil
			},
			TTL: 1 * time.Hour,
		})
	}

	// 执行预热
	warmupConfig := &core.WarmerConfig{
		Concurrency:  config.Concurrency,
		Timeout:      config.Timeout,
		SkipExisting: config.SkipExisting,
		LogFailed:    true,
	}

	if err := warmer.WarmUpWithConfig(ctx, entries, warmupConfig); err != nil {
		log.Printf("[WARN] Cache warmup completed with errors: %v", err)
	}

	// 输出统计
	stats := warmer.GetStats()
	duration := time.Since(startTime)

	log.Printf("[INFO] Cache warmup completed in %v", duration)
	log.Printf("[INFO] Warmed up %d/%d entries (Success: %d, Failed: %d, Skipped: %d)",
		stats.Success, stats.Total, stats.Success, stats.Failed, stats.Skipped)

	return nil
}

// WarmupCacheSimple 简单预热（无配置）
func (j *CacheWarmupJob) WarmupCacheSimple(ctx context.Context) error {
	return j.WarmupCache(ctx, DefaultWarmupConfig())
}

// WarmupCacheWithBuilder 使用构建器预热
func (j *CacheWarmupJob) WarmupCacheWithBuilder(ctx context.Context) error {
	log.Println("[INFO] Starting cache warmup with builder...")
	startTime := time.Now()

	err := core.NewWarmUpBuilder(j.manager).
		Add("hot_products", "list", func() (interface{}, error) {
			return j.service.GetHotProducts()
		}, 5*time.Minute).
		Add("products", "1", func() (interface{}, error) {
			return j.service.GetProductByID(1), nil
		}, 1*time.Hour).
		Add("products", "2", func() (interface{}, error) {
			return j.service.GetProductByID(2), nil
		}, 1*time.Hour).
		Execute(ctx, 4)

	if err != nil {
		log.Printf("[WARN] Cache warmup completed with errors: %v", err)
	}

	log.Printf("[INFO] Cache warmup completed in %v", time.Since(startTime))
	return err
}
