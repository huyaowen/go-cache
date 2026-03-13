package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
	"github.com/coderiser/go-cache/pkg/core"
)

func main() {
	fmt.Println("=== Go-Cache Framework 缓存异常保护示例 ===")
	fmt.Println()

	// 示例 1: 基本保护配置
	example1BasicProtection()

	// 示例 2: 缓存穿透保护
	example2PenetrationProtection()

	// 示例 3: 缓存击穿保护
	example3BreakdownProtection()

	// 示例 4: 缓存雪崩保护
	example4AvalancheProtection()

	// 示例 5: 完整集成示例
	example5FullIntegration()
}

// 示例 1: 基本保护配置
func example1BasicProtection() {
	fmt.Println("【示例 1】基本保护配置")
	fmt.Println("-----------------------------------")

	// 创建缓存管理器（默认启用所有保护）
	manager := core.NewCacheManager()

	// 获取保护器
	protection := manager.GetProtection()
	config := protection.GetProtectionConfig()

	fmt.Printf("穿透保护：%v (空值 TTL: %v)\n", config.EnablePenetrationProtection, config.EmptyValueTTL)
	fmt.Printf("击穿保护：%v\n", config.EnableBreakdownProtection)
	fmt.Printf("雪崩保护：%v (抖动因子：%.1f%%)\n", config.EnableAvalancheProtection, config.TTLJitterFactor*100)
	fmt.Println()
}

// 示例 2: 缓存穿透保护
func example2PenetrationProtection() {
	fmt.Println("【示例 2】缓存穿透保护（空值缓存）")
	fmt.Println("-----------------------------------")

	ctx := context.Background()
	cache, _ := backend.NewMemoryBackend(backend.DefaultCacheConfig("users"))

	// 模拟查询不存在的数据
	dbCallCount := 0
	queryUser := func() (interface{}, error) {
		dbCallCount++
		fmt.Printf("  [DB] 查询用户 (第%d次查询)\n", dbCallCount)
		// 模拟用户不存在
		return nil, nil
	}

	protection := core.NewCacheProtection(core.DefaultProtectionConfig())

	// 第一次查询 - 缓存未命中，查询 DB
	fmt.Println("第一次查询（缓存未命中）:")
	result1, _ := protection.ProtectedGet(
		ctx,
		"user:non-existent",
		func() (interface{}, bool, error) {
			return cache.Get(ctx, "user:non-existent")
		},
		queryUser,
		func(value interface{}, ttl time.Duration) error {
			fmt.Printf("  [Cache] 写入空值标记，TTL=%v\n", ttl)
			return cache.Set(ctx, "user:non-existent", value, ttl)
		},
	)
	fmt.Printf("  结果：%v\n\n", result1)

	// 第二次查询 - 从缓存获取空值标记，不再查询 DB
	fmt.Println("第二次查询（缓存命中空值）:")
	result2, _ := protection.ProtectedGet(
		ctx,
		"user:non-existent",
		func() (interface{}, bool, error) {
			return cache.Get(ctx, "user:non-existent")
		},
		queryUser,
		func(value interface{}, ttl time.Duration) error {
			return cache.Set(ctx, "user:non-existent", value, ttl)
		},
	)
	fmt.Printf("  结果：%v\n", result2)
	fmt.Printf("  DB 查询总次数：%d（节省了 %d 次查询）\n\n", dbCallCount, dbCallCount-1)
}

// 示例 3: 缓存击穿保护
func example3BreakdownProtection() {
	fmt.Println("【示例 3】缓存击穿保护（Singleflight）")
	fmt.Println("-----------------------------------")

	ctx := context.Background()
	protection := core.NewCacheProtection(core.DefaultProtectionConfig())

	dbCallCount := 0
	slowQuery := func() (interface{}, error) {
		dbCallCount++
		fmt.Printf("  [DB] 执行慢查询 (第%d次)\n", dbCallCount)
		time.Sleep(100 * time.Millisecond) // 模拟慢查询
		return "hot-data", nil
	}

	// 模拟 10 个并发请求
	numRequests := 10
	var wg sync.WaitGroup
	results := make([]interface{}, numRequests)

	fmt.Printf("启动 %d 个并发请求...\n", numRequests)
	start := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			result, _, shared := protection.ApplyBreakdownProtection(ctx, "hot-key", slowQuery)
			results[idx] = result
			if shared {
				fmt.Printf("  [请求%d] 共享结果（等待）\n", idx)
			} else {
				fmt.Printf("  [请求%d] 执行查询\n", idx)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("\n总耗时：%v\n", elapsed)
	fmt.Printf("DB 查询次数：%d（节省了 %d 次查询）\n", dbCallCount, numRequests-dbCallCount)
	fmt.Printf("所有结果一致：%v\n\n", allEqual(results))
}

// 示例 4: 缓存雪崩保护
func example4AvalancheProtection() {
	fmt.Println("【示例 4】缓存雪崩保护（TTL 随机抖动）")
	fmt.Println("-----------------------------------")

	protection := core.NewCacheProtection(core.DefaultProtectionConfig())
	baseTTL := 30 * time.Minute

	fmt.Printf("基础 TTL: %v\n", baseTTL)
	fmt.Println("应用雪崩保护后的实际 TTL:")

	// 计算 10 次 TTL，展示随机性
	ttls := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		ttls[i] = protection.ApplyAvalancheProtection(baseTTL)
		fmt.Printf("  %2d. %v\n", i+1, ttls[i])
	}

	// 计算范围
	minTTL := ttls[0]
	maxTTL := ttls[0]
	for _, ttl := range ttls {
		if ttl < minTTL {
			minTTL = ttl
		}
		if ttl > maxTTL {
			maxTTL = ttl
		}
	}

	fmt.Printf("\nTTL 范围：%v ~ %v\n", minTTL, maxTTL)
	fmt.Printf("抖动范围：%.1f%% ~ %.1f%%\n\n",
		float64(minTTL-baseTTL)/float64(baseTTL)*100,
		float64(maxTTL-baseTTL)/float64(baseTTL)*100,
	)
}

// 示例 5: 完整集成示例
func example5FullIntegration() {
	fmt.Println("【示例 5】完整集成示例")
	fmt.Println("-----------------------------------")

	// 创建缓存管理器
	manager := core.NewCacheManager()

	// 自定义保护配置
	customConfig := &core.ProtectionConfig{
		EnablePenetrationProtection: true,
		EmptyValueTTL:               10 * time.Minute,
		EnableBreakdownProtection:   true,
		EnableAvalancheProtection:   true,
		TTLJitterFactor:             0.15, // 15% 抖动
	}

	err := manager.SetProtectionConfig(customConfig)
	if err != nil {
		log.Fatalf("设置保护配置失败：%v", err)
	}

	fmt.Println("✓ 缓存管理器已初始化")
	fmt.Println("✓ 保护配置已应用")

	// 获取保护器
	protection := manager.GetProtection()
	config := protection.GetProtectionConfig()

	fmt.Printf("\n当前配置:\n")
	fmt.Printf("  - 穿透保护：%v\n", config.EnablePenetrationProtection)
	fmt.Printf("  - 空值 TTL: %v\n", config.EmptyValueTTL)
	fmt.Printf("  - 击穿保护：%v\n", config.EnableBreakdownProtection)
	fmt.Printf("  - 雪崩保护：%v (%.0f%% 抖动)\n", config.EnableAvalancheProtection, config.TTLJitterFactor*100)
	fmt.Println("\n✓ 所有保护机制已启用，可以安全使用缓存！")
}

// 辅助函数：检查所有元素是否相等
func allEqual(results []interface{}) bool {
	if len(results) == 0 {
		return true
	}
	first := results[0]
	for _, r := range results[1:] {
		if r != first {
			return false
		}
	}
	return true
}
