package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
)

// Redis Cluster 缓存示例
// 适用于大规模分布式缓存场景

func main() {
	fmt.Println("=== Redis Cluster Cache Example ===\n")

	// Redis Cluster 配置
	clusterConfig := backend.DefaultRedisClusterConfig()
	clusterConfig.Addrs = []string{
		"localhost:7000",
		"localhost:7001",
		"localhost:7002",
	}
	clusterConfig.DefaultTTL = 30 * time.Minute
	clusterConfig.MaxTTL = 2 * time.Hour
	clusterConfig.PoolSize = 20
	clusterConfig.MinIdleConns = 5

	// 创建 Redis Cluster 缓存后端
	clusterBackend, err := backend.NewRedisClusterBackend(clusterConfig)
	if err != nil {
		fmt.Printf("❌ Failed to create Redis Cluster backend: %v\n", err)
		fmt.Println("\n💡 Note: This example requires Redis Cluster running on localhost:7000-7002")
		fmt.Println("   Use docker-compose or manual setup for Redis Cluster.\n")
		fmt.Println("📚 See README.md for setup instructions.\n")
		return
	}
	defer clusterBackend.Close()

	ctx := context.Background()

	// 示例 1: 设置缓存
	fmt.Println("1️⃣  Setting cache...")
	user := map[string]interface{}{
		"id":       1,
		"name":     "John Doe",
		"email":    "john@example.com",
		"age":      30,
		"location": "Beijing",
	}
	err = clusterBackend.Set(ctx, "user:1", user, 1*time.Hour)
	if err != nil {
		fmt.Printf("   ❌ Set failed: %v\n", err)
	} else {
		fmt.Println("   ✅ Set success")
	}

	// 示例 2: 获取缓存
	fmt.Println("\n2️⃣  Getting cache...")
	val, found, err := clusterBackend.Get(ctx, "user:1")
	if err != nil {
		fmt.Printf("   ❌ Get failed: %v\n", err)
	} else if found {
		fmt.Printf("   ✅ Get success: %+v\n", val)
	} else {
		fmt.Println("   ⚠️  Cache miss")
	}

	// 示例 3: 批量操作
	fmt.Println("\n3️⃣  Batch operations...")
	for i := 2; i <= 5; i++ {
		key := fmt.Sprintf("user:%d", i)
		userData := map[string]interface{}{
			"id":   i,
			"name": fmt.Sprintf("User %d", i),
		}
		clusterBackend.Set(ctx, key, userData, 30*time.Minute)
		fmt.Printf("   ✅ Set %s\n", key)
	}

	// 示例 4: 查看统计
	fmt.Println("\n4️⃣  Cache Statistics:")
	stats := clusterBackend.Stats()
	fmt.Printf("   Hits: %d\n", stats.Hits)
	fmt.Printf("   Misses: %d\n", stats.Misses)
	fmt.Printf("   Sets: %d\n", stats.Sets)
	fmt.Printf("   Deletes: %d\n", stats.Deletes)
	fmt.Printf("   Hit Rate: %.2f%%\n", stats.HitRate*100)
	fmt.Printf("   Size: %d\n", stats.Size)

	fmt.Println("\n=== Example Complete ===")
}

// createClusterCache 创建 Redis Cluster 缓存管理器
func createClusterCache() (*backend.RedisClusterBackend, error) {
	config := backend.DefaultRedisClusterConfig()
	
	// 生产环境配置示例
	config.Addrs = []string{
		"redis-cluster-node1:6379",
		"redis-cluster-node2:6379",
		"redis-cluster-node3:6379",
		"redis-cluster-node4:6379",
		"redis-cluster-node5:6379",
		"redis-cluster-node6:6379",
	}
	config.Password = "" // 如果有密码，设置此处
	config.DefaultTTL = 30 * time.Minute
	config.MaxTTL = 24 * time.Hour
	config.PoolSize = 50
	config.MinIdleConns = 10
	config.MaxRetries = 3
	config.DialTimeout = 5 * time.Second
	config.ReadTimeout = 3 * time.Second
	config.WriteTimeout = 3 * time.Second

	return backend.NewRedisClusterBackend(config)
}
