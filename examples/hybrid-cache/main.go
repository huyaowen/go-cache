package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
	"github.com/coderiser/go-cache/pkg/core"
)

// Hybrid 缓存示例 - L1 + L2 两级缓存
// L1: 本地 Memory 缓存（快速访问）
// L2: Redis 缓存（分布式共享）

func main() {
	fmt.Println("=== Hybrid Cache Example ===\n")

	// 创建 Hybrid 缓存配置
	hybridConfig := backend.DefaultHybridConfig()
	hybridConfig.L1Config.Name = "products"
	hybridConfig.L1Config.MaxSize = 1000
	hybridConfig.L1Config.DefaultTTL = 5 * time.Minute
	hybridConfig.L1Config.MaxTTL = 30 * time.Minute
	hybridConfig.L1Config.EvictionPolicy = "lru"

	hybridConfig.L2Config.Addr = "localhost:6379"
	hybridConfig.L2Config.DefaultTTL = 30 * time.Minute
	hybridConfig.L2Config.MaxTTL = 2 * time.Hour
	hybridConfig.L1WriteBackTTL = 5 * time.Minute // L1 回写 TTL

	// 创建 Hybrid 缓存后端
	hybridBackend, err := backend.NewHybridBackend(hybridConfig)
	if err != nil {
		fmt.Printf("❌ Failed to create hybrid backend: %v\n", err)
		fmt.Println("\n💡 Note: This example requires Redis running on localhost:6379")
		fmt.Println("   If Redis is not available, the example will use L1 (Memory) cache only.\n")
		// 降级使用纯内存缓存
		useMemoryOnly()
		return
	}
	defer hybridBackend.Close()

	ctx := context.Background()

	// 示例 1: 设置缓存
	fmt.Println("1️⃣  Setting cache...")
	product := map[string]interface{}{
		"id":    1,
		"name":  "iPhone 15 Pro",
		"price": 7999.00,
	}
	err = hybridBackend.Set(ctx, "product:1", product, 30*time.Minute)
	if err != nil {
		fmt.Printf("   ❌ Set failed: %v\n", err)
	} else {
		fmt.Println("   ✅ Set success")
	}

	// 示例 2: 获取缓存（L1 命中）
	fmt.Println("\n2️⃣  Getting cache (L1 hit)...")
	val, found, err := hybridBackend.Get(ctx, "product:1")
	if err != nil {
		fmt.Printf("   ❌ Get failed: %v\n", err)
	} else if found {
		fmt.Printf("   ✅ Get success: %+v\n", val)
	} else {
		fmt.Println("   ⚠️  Cache miss")
	}

	// 示例 3: 获取缓存（模拟 L1 miss, L2 hit）
	fmt.Println("\n3️⃣  Getting cache (L1 miss, L2 fallback)...")
	// 清除 L1 缓存模拟 miss
	hybridBackend.GetL1().Delete(ctx, "product:1")
	
	val, found, err = hybridBackend.Get(ctx, "product:1")
	if err != nil {
		fmt.Printf("   ❌ Get failed: %v\n", err)
	} else if found {
		fmt.Printf("   ✅ Get success from L2: %+v\n", val)
	} else {
		fmt.Println("   ⚠️  Cache miss (Redis not available)")
	}

	// 示例 4: 查看统计
	fmt.Println("\n4️⃣  Cache Statistics:")
	stats := hybridBackend.Stats()
	fmt.Printf("   Total Hits: %d\n", stats.Hits)
	fmt.Printf("   Total Misses: %d\n", stats.Misses)
	fmt.Printf("   Sets: %d\n", stats.Sets)
	fmt.Printf("   Hit Rate: %.2f%%\n", stats.HitRate*100)
	fmt.Println("   (Use GetHybridStats() for detailed L1/L2 stats)")

	fmt.Println("\n=== Example Complete ===")
}

// useMemoryOnly 降级使用纯内存缓存（当 Redis 不可用时）
func useMemoryOnly() {
	fmt.Println("=== Fallback to Memory Cache ===\n")

	config := backend.DefaultCacheConfig("products")
	config.MaxSize = 1000
	config.DefaultTTL = 30 * time.Minute

	memoryBackend, err := backend.NewMemoryBackend(config)
	if err != nil {
		fmt.Printf("❌ Failed to create memory backend: %v\n", err)
		return
	}
	defer memoryBackend.Close()

	ctx := context.Background()

	// 设置缓存
	product := map[string]interface{}{
		"id":    1,
		"name":  "iPhone 15 Pro",
		"price": 7999.00,
	}
	memoryBackend.Set(ctx, "product:1", product, 30*time.Minute)
	fmt.Println("   ✅ Set success (Memory)")

	// 获取缓存
	val, found, _ := memoryBackend.Get(ctx, "product:1")
	if found {
		fmt.Printf("   ✅ Get success: %+v\n", val)
	}

	// 查看统计
	stats := memoryBackend.Stats()
	fmt.Printf("\n   Hits: %d, Misses: %d, Hit Rate: %.2f%%\n",
		stats.Hits, stats.Misses, stats.HitRate*100)

	fmt.Println("\n=== Example Complete ===")
}

// createCacheManager 使用 Hybrid 缓存创建 CacheManager
func createCacheManager() core.CacheManager {
	manager := core.NewCacheManager()

	// 注册 Hybrid 缓存配置
	hybridConfig := backend.DefaultHybridConfig()
	hybridConfig.L1Config.Name = "users"
	hybridConfig.L2Config.Addr = "localhost:6379"

	// 注册 hybrid 后端工厂
	manager.RegisterBackend("hybrid", func(cfg *backend.CacheConfig) (backend.CacheBackend, error) {
		return backend.NewHybridBackend(hybridConfig)
	})

	// GetCache 会自动使用注册的 backend 创建缓存
	_, err := manager.GetCache("users")
	if err != nil {
		fmt.Printf("Warning: Failed to create hybrid cache: %v\n", err)
	}

	return manager
}
