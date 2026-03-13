package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
	"github.com/coderiser/go-cache/pkg/core"
)

func main() {
	fmt.Println("=== Go-Cache Framework Demo ===")
	fmt.Println()

	// 1. 创建缓存管理器
	manager := core.NewCacheManager()
	defer manager.Close()

	// 2. 获取缓存
	cache, err := manager.GetCache("users")
	if err != nil {
		fmt.Printf("Error getting cache: %v\n", err)
		return
	}

	ctx := context.Background()

	// 3. 设置缓存
	fmt.Println("Setting cache...")
	cache.Set(ctx, "user:1", map[string]interface{}{"id": 1, "name": "Alice"}, 30*time.Minute)
	cache.Set(ctx, "user:2", map[string]interface{}{"id": 2, "name": "Bob"}, 30*time.Minute)

	// 4. 获取缓存
	fmt.Println("\nGetting cache...")
	if v, found, _ := cache.Get(ctx, "user:1"); found {
		fmt.Printf("Cache hit: %v\n", v)
	}

	// 5. 查看统计
	fmt.Println("\nCache Stats:")
	stats := cache.Stats()
	fmt.Printf("  Hits: %d\n", stats.Hits)
	fmt.Printf("  Misses: %d\n", stats.Misses)
	fmt.Printf("  Sets: %d\n", stats.Sets)
	fmt.Printf("  Hit Rate: %.2f%%\n", stats.HitRate*100)

	// 6. SpEL 示例
	fmt.Println("\n=== SpEL Demo ===")
	evalCtx := backend.DefaultCacheConfig("test")
	_ = evalCtx

	fmt.Println("\n=== Demo Complete ===")
}
