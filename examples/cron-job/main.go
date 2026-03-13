package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coderiser/go-cache/pkg/core"
	"github.com/coderiser/go-cache/examples/cron-job/job"
	"github.com/coderiser/go-cache/examples/cron-job/service"
	cached "github.com/coderiser/go-cache/examples/cron-job/service/.cache-gen"
)

// 示例：零配置使用缓存服务
// 1. 添加注解到 service 方法
// 2. 执行 go generate ./...
// 3. 使用 cached.NewProductService() 直接使用

var (
	// productService 全局商品服务实例（使用接口类型）
	// 方案 G: 零配置，直接使用
	productService service.ProductServiceInterface
)

func main() {
	// 命令行参数
	interval := flag.Duration("interval", 5*time.Minute, "Cache refresh interval")
	warmup := flag.Bool("warmup", true, "Enable cache warmup on startup")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("=== Go-Cache Framework: Cron Job Example ===")
	log.Println()

	// 1. 创建缓存管理器
	manager := core.NewCacheManager()
	defer func() {
		log.Println("[INFO] Closing cache manager...")
		manager.Close()
	}()

	// 2. 创建商品服务（方案 G: 零配置）
	// 使用代码生成的 NewProductService()，自动使用全局 Manager
	productService = cached.NewProductService()

	// 3. 创建缓存预热任务
	warmupJob := job.NewCacheWarmupJob(manager, productService)

	// 4. 创建缓存刷新任务
	refreshJob := job.NewCacheRefreshJob(manager, productService, *interval)

	// 5. 设置信号监听
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 6. 启动时预热缓存
	if *warmup {
		log.Println()
		if err := warmupJob.WarmupCacheSimple(ctx); err != nil {
			log.Printf("[WARN] Cache warmup completed with warnings: %v", err)
		}
	}

	// 7. 启动定时刷新任务（后台 goroutine）
	go func() {
		config := &job.RefreshConfig{
			Interval:          *interval,
			WarmupHotProducts: true,
			RefreshConfig:     true,
			LogStats:          true,
			StatsInterval:     6, // 每 6 次（30 分钟）输出一次统计
		}
		refreshJob.Start(ctx, config)
	}()

	// 8. 展示缓存统计（每 10 秒输出一次）
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				printCacheStats(manager)
			case <-ctx.Done():
				return
			}
		}
	}()

	// 9. 等待退出信号
	log.Println()
	log.Println("[INFO] Application running. Press Ctrl+C to exit...")
	log.Println()

	<-sigChan
	log.Println("[INFO] Shutting down...")

	// 10. 优雅关闭
	cancel()
	time.Sleep(500 * time.Millisecond) // 等待 goroutine 完成

	// 输出最终统计
	log.Println()
	log.Println("=== Final Cache Stats ===")
	printCacheStats(manager)

	// 输出刷新任务统计
	if stats := refreshJob.GetStats(); stats != nil {
		log.Println()
		log.Println("=== Refresh Job Stats ===")
		log.Printf("  Total Runs:   %d", stats.TotalRuns)
		log.Printf("  Success:      %d", stats.SuccessRuns)
		log.Printf("  Failed:       %d", stats.FailedRuns)
		log.Printf("  Total Time:   %v", stats.TotalTime)
	}

	log.Println()
	log.Println("=== Application Exited ===")
}

// printCacheStats 输出缓存统计
func printCacheStats(manager core.CacheManager) {
	caches := []string{"products", "hot_products", "config"}

	for _, cacheName := range caches {
		cache, err := manager.GetCache(cacheName)
		if err != nil {
			continue
		}

		stats := cache.Stats()
		if stats.Hits+stats.Misses > 0 {
			log.Printf("[STATS] %s: Hits=%d, Misses=%d, Sets=%d, HitRate=%.1f%%",
				cacheName,
				stats.Hits,
				stats.Misses,
				stats.Sets,
				stats.HitRate*100)
		}
	}
}
