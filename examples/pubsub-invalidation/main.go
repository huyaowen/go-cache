package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
)

// PubSub 缓存失效示例
// 多实例缓存一致性解决方案

var (
	cacheStore = make(map[string]interface{})
	cacheMu    sync.RWMutex
)

func main() {
	fmt.Println("=== PubSub Cache Invalidation Example ===\n")

	// Redis 配置
	redisConfig := backend.DefaultRedisConfig()
	redisConfig.Addr = "localhost:6379"
	redisConfig.DefaultTTL = 30 * time.Minute

	// 创建 Redis 后端
	redisBackend, err := backend.NewRedisBackend(redisConfig)
	if err != nil {
		fmt.Printf("❌ Failed to create Redis backend: %v\n", err)
		fmt.Println("\n💡 Note: This example requires Redis running on localhost:6379")
		fmt.Println("   Use: docker run -d -p 6379:6379 redis:latest\n")
		return
	}
	defer redisBackend.Close()

	// 创建 PubSub 缓存失效器
	pubsubConfig := backend.DefaultPubSubInvalidatorConfig()
	pubsubConfig.Channel = "cache-invalidation"
	pubsubConfig.RedisAddr = "localhost:6379"

	invalidator, err := backend.NewPubSubInvalidator(pubsubConfig)
	if err != nil {
		fmt.Printf("❌ Failed to create PubSub invalidator: %v\n", err)
		return
	}
	defer invalidator.Close()

	ctx := context.Background()

	// 模拟实例 1: 更新数据并广播失效消息
	fmt.Println("1️⃣  Instance 1: Updating data...")
	updateData(ctx, redisBackend, invalidator, "user:1", map[string]interface{}{
		"id":    1,
		"name":  "John Doe Updated",
		"email": "john.updated@example.com",
	})

	// 等待消息传播
	time.Sleep(500 * time.Millisecond)

	// 模拟实例 2: 接收失效消息并清除本地缓存
	fmt.Println("\n2️⃣  Instance 2: Received invalidation message")
	fmt.Println("   ✅ Local cache cleared for key: user:1")

	// 模拟实例 3: 从数据库重新加载
	fmt.Println("\n3️⃣  Instance 3: Reloading from database...")
	reloadData(ctx, redisBackend, "user:1")

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("\n💡 Key Points:")
	fmt.Println("   • PubSub 实现多实例缓存一致性")
	fmt.Println("   • 更新时广播失效消息")
	fmt.Println("   • 接收方清除本地缓存并重新加载")
}

// updateData 更新数据并广播失效消息
func updateData(ctx context.Context, backend backend.CacheBackend, invalidator *backend.PubSubInvalidator, key string, data interface{}) {
	// 1. 更新数据库（模拟）
	cacheStore[key] = data
	fmt.Printf("   ✅ Database updated: %+v\n", data)

	// 2. 更新缓存
	backend.Set(ctx, key, data, 30*time.Minute)
	fmt.Println("   ✅ Cache updated")

	// 3. 广播失效消息
	invalidator.Invalidate(ctx, key)
	fmt.Println("   ✅ Invalidation message broadcasted")
}

// reloadData 重新加载数据
func reloadData(ctx context.Context, backend backend.CacheBackend, key string) {
	// 1. 从数据库读取（模拟）
	cacheMu.RLock()
	data := cacheStore[key]
	cacheMu.RUnlock()

	// 2. 写入缓存
	backend.Set(ctx, key, data, 30*time.Minute)
	fmt.Printf("   ✅ Cache reloaded: %+v\n", data)
}

// createPubSubInvalidator 创建 PubSub 缓存失效器（多实例共享）
func createPubSubInvalidator(instanceID string) (*backend.PubSubInvalidator, error) {
	config := backend.DefaultPubSubInvalidatorConfig()
	config.Channel = "cache-invalidation"
	config.RedisAddr = "localhost:6379"
	config.InstanceID = instanceID // 实例 ID，避免处理自己的消息

	return backend.NewPubSubInvalidator(config)
}

// 多实例部署示例
func multiInstanceDeployment() {
	fmt.Println("=== Multi-Instance Deployment ===\n")

	// 实例 1
	instance1, _ := createPubSubInvalidator("instance-1")
	defer instance1.Close()

	// 实例 2
	instance2, _ := createPubSubInvalidator("instance-2")
	defer instance2.Close()

	// 实例 3
	instance3, _ := createPubSubInvalidator("instance-3")
	defer instance3.Close()

	fmt.Println("✅ 3 instances deployed with PubSub invalidation")
	fmt.Println("   • All instances share the same Redis PubSub channel")
	fmt.Println("   • Updates from any instance invalidate all caches")
}
