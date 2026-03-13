package core

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// LoaderFunc 数据加载函数类型
type LoaderFunc func() (interface{}, error)

// WarmUpEntry 预热条目
type WarmUpEntry struct {
	Cache string        // 缓存名称
	Key   string        // 缓存键
	Loader LoaderFunc   // 数据加载函数
	TTL   time.Duration // TTL
}

// CacheWarmer 缓存预热器
type CacheWarmer struct {
	manager CacheManager
	mu      sync.RWMutex
	entries []WarmUpEntry
	stats   *WarmerStats
}

// WarmerStats 预热器统计
type WarmerStats struct {
	Total     int64 // 总条目数
	Success   int64 // 成功预热数
	Failed    int64 // 失败预热数
	Skipped   int64 // 跳过数（已存在）
	Duration  time.Duration // 总耗时
}

// WarmerConfig 预热器配置
type WarmerConfig struct {
	Concurrency int           // 并发数（默认 1）
	Timeout     time.Duration // 单个条目超时（默认 10 秒）
	Retries     int           // 重试次数（默认 0）
	LogFailed   bool          // 记录失败日志（默认 true）
	SkipExisting bool         // 跳过已存在的缓存（默认 false）
}

// DefaultWarmerConfig 默认预热器配置
func DefaultWarmerConfig() *WarmerConfig {
	return &WarmerConfig{
		Concurrency:  1,
		Timeout:      10 * time.Second,
		Retries:      0,
		LogFailed:    true,
		SkipExisting: false,
	}
}

// NewCacheWarmer 创建缓存预热器
func NewCacheWarmer(manager CacheManager) *CacheWarmer {
	return &CacheWarmer{
		manager: manager,
		stats:   &WarmerStats{},
	}
}

// AddEntry 添加预热条目
func (w *CacheWarmer) AddEntry(entry WarmUpEntry) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.entries = append(w.entries, entry)
}

// AddEntries 批量添加预热条目
func (w *CacheWarmer) AddEntries(entries []WarmUpEntry) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.entries = append(w.entries, entries...)
}

// WarmUp 执行缓存预热
func (w *CacheWarmer) WarmUp(ctx context.Context, entries []WarmUpEntry) error {
	return w.WarmUpWithConfig(ctx, entries, DefaultWarmerConfig())
}

// WarmUpWithConfig 使用配置执行缓存预热
func (w *CacheWarmer) WarmUpWithConfig(ctx context.Context, entries []WarmUpEntry, config *WarmerConfig) error {
	if config == nil {
		config = DefaultWarmerConfig()
	}

	startTime := time.Now()
	w.mu.Lock()
	w.stats = &WarmerStats{Total: int64(len(entries))}
	w.mu.Unlock()

	if config.Concurrency <= 0 {
		config.Concurrency = 1
	}

	// 创建信号量控制并发
	sem := make(chan struct{}, config.Concurrency)
	var wg sync.WaitGroup
	errChan := make(chan error, len(entries))

	for _, entry := range entries {
		wg.Add(1)
		go func(e WarmUpEntry) {
			defer wg.Done()
			
			// 获取信号量
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}

			// 执行预热
			if err := w.warmUpEntry(ctx, e, config); err != nil {
				errChan <- err
			}
		}(entry)
	}

	// 等待所有 goroutine 完成
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// 收集错误
	var firstErr error
	for err := range errChan {
		if firstErr == nil && err != nil {
			firstErr = err
		}
	}

	w.mu.Lock()
	w.stats.Duration = time.Since(startTime)
	w.mu.Unlock()

	return firstErr
}

// warmUpEntry 预热单个条目
func (w *CacheWarmer) warmUpEntry(ctx context.Context, entry WarmUpEntry, config *WarmerConfig) error {
	cache, err := w.manager.GetCache(entry.Cache)
	if err != nil {
		w.mu.Lock()
		w.stats.Failed++
		w.mu.Unlock()
		
		if config.LogFailed {
			log.Printf("[CacheWarmer] Failed to get cache '%s': %v", entry.Cache, err)
		}
		return err
	}

	// 检查是否已存在（如果配置了跳过）
	if config.SkipExisting {
		if _, exists, _ := cache.Get(ctx, entry.Key); exists {
			w.mu.Lock()
			w.stats.Skipped++
			w.mu.Unlock()
			return nil
		}
	}

	// 执行加载函数
	value, err := entry.Loader()
	if err != nil {
		w.mu.Lock()
		w.stats.Failed++
		w.mu.Unlock()
		
		if config.LogFailed {
			log.Printf("[CacheWarmer] Failed to load key '%s' for cache '%s': %v", entry.Key, entry.Cache, err)
		}
		return err
	}

	// 设置缓存
	ttl := entry.TTL
	if ttl <= 0 {
		ttl = 30 * time.Minute // 默认 TTL
	}

	if err := cache.Set(ctx, entry.Key, value, ttl); err != nil {
		w.mu.Lock()
		w.stats.Failed++
		w.mu.Unlock()
		
		if config.LogFailed {
			log.Printf("[CacheWarmer] Failed to set key '%s' for cache '%s': %v", entry.Key, entry.Cache, err)
		}
		return err
	}

	w.mu.Lock()
	w.stats.Success++
	w.mu.Unlock()

	return nil
}

// WarmUpAll 预热所有已添加的条目
func (w *CacheWarmer) WarmUpAll(ctx context.Context) error {
	w.mu.RLock()
	entries := make([]WarmUpEntry, len(w.entries))
	copy(entries, w.entries)
	w.mu.RUnlock()

	return w.WarmUp(ctx, entries)
}

// WarmUpConcurrent 并发预热（推荐用于生产环境）
func (w *CacheWarmer) WarmUpConcurrent(ctx context.Context, entries []WarmUpEntry, concurrency int) error {
	config := DefaultWarmerConfig()
	config.Concurrency = concurrency
	if concurrency <= 0 {
		config.Concurrency = 4 // 默认 4 并发
	}
	return w.WarmUpWithConfig(ctx, entries, config)
}

// GetStats 获取预热器统计信息
func (w *CacheWarmer) GetStats() *WarmerStats {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.stats
}

// Clear 清空预热条目列表
func (w *CacheWarmer) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.entries = nil
}

// Example: 使用示例
// ```go
// // 创建预热器
// warmer := NewCacheWarmer(manager)
//
// // 添加热点数据
// warmer.AddEntries([]WarmUpEntry{
//     {
//         Cache: "users",
//         Key:   "hot:1",
//         Loader: func() (interface{}, error) {
//             return db.GetHotUser("1")
//         },
//         TTL: 1 * time.Hour,
//     },
//     {
//         Cache: "products",
//         Key:   "featured",
//         Loader: func() (interface{}, error) {
//             return db.GetFeaturedProducts()
//         },
//         TTL: 30 * time.Minute,
//     },
// })
//
// // 执行预热
// ctx := context.Background()
// if err := warmer.WarmUpConcurrent(ctx, nil, 4); err != nil {
//     log.Printf("WarmUp failed: %v", err)
// }
//
// // 查看统计
// stats := warmer.GetStats()
// fmt.Printf("Warmed up %d/%d entries in %v\n", stats.Success, stats.Total, stats.Duration)
// ```

// WarmUpBuilder 预热器构建器（流式 API）
type WarmUpBuilder struct {
	warmer *CacheWarmer
	entries []WarmUpEntry
}

// NewWarmUpBuilder 创建预热器构建器
func NewWarmUpBuilder(manager CacheManager) *WarmUpBuilder {
	return &WarmUpBuilder{
		warmer: NewCacheWarmer(manager),
	}
}

// Add 添加预热条目
func (b *WarmUpBuilder) Add(cache, key string, loader LoaderFunc, ttl time.Duration) *WarmUpBuilder {
	b.entries = append(b.entries, WarmUpEntry{
		Cache:  cache,
		Key:    key,
		Loader: loader,
		TTL:    ttl,
	})
	return b
}

// AddWithTimeout 添加带超时的预热条目
func (b *WarmUpBuilder) AddWithTimeout(cache, key string, loader LoaderFunc, ttl, timeout time.Duration) *WarmUpBuilder {
	b.entries = append(b.entries, WarmUpEntry{
		Cache: cache,
		Key:   key,
		Loader: func() (interface{}, error) {
			done := make(chan struct{})
			var result interface{}
			var err error

			go func() {
				result, err = loader()
				close(done)
			}()

			select {
			case <-done:
				return result, err
			case <-time.After(timeout):
				return nil, fmt.Errorf("loader timeout after %v", timeout)
			}
		},
		TTL: ttl,
	})
	return b
}

// Build 构建并返回预热器
func (b *WarmUpBuilder) Build() *CacheWarmer {
	b.warmer.AddEntries(b.entries)
	return b.warmer
}

// Execute 执行预热
func (b *WarmUpBuilder) Execute(ctx context.Context, concurrency int) error {
	warmer := b.Build()
	return warmer.WarmUpConcurrent(ctx, b.entries, concurrency)
}

// Example: 流式 API 示例
// ```go
// err := NewWarmUpBuilder(manager).
//     Add("users", "hot:1", func() (interface{}, error) {
//         return db.GetUser("1")
//     }, 1*time.Hour).
//     Add("products", "featured", func() (interface{}, error) {
//         return db.GetFeatured()
//     }, 30*time.Minute).
//     Execute(context.Background(), 4)
// ```
