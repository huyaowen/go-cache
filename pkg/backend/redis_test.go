package backend

import (
	"context"
	"testing"
	"time"
)

// TestRedisBackendBasic 测试 Redis 后端基本功能
// 注意：此测试需要运行中的 Redis 实例
func TestRedisBackendBasic(t *testing.T) {
	// 跳过需要 Redis 的测试（CI 环境可能没有 Redis）
	t.Skip("Skipping Redis test - requires running Redis instance")

	config := &RedisConfig{
		Addr:       "localhost:6379",
		Prefix:     "test",
		DefaultTTL: 5 * time.Second,
		MaxTTL:     10 * time.Second,
		PoolSize:   5,
	}

	backend, err := NewRedisBackend(config)
	if err != nil {
		t.Fatalf("Failed to create Redis backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()

	// 测试 Set 和 Get
	t.Run("SetAndGet", func(t *testing.T) {
		key := "test:key1"
		value := "hello world"

		err := backend.Set(ctx, key, value, 5*time.Second)
		if err != nil {
			t.Fatalf("Failed to set: %v", err)
		}

		got, found, err := backend.Get(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if !found {
			t.Fatal("Expected to find key")
		}
		if got != value {
			t.Errorf("Expected %v, got %v", value, got)
		}
	})

	// 测试 Delete
	t.Run("Delete", func(t *testing.T) {
		key := "test:key2"
		value := "to be deleted"

		err := backend.Set(ctx, key, value, 5*time.Second)
		if err != nil {
			t.Fatalf("Failed to set: %v", err)
		}

		err = backend.Delete(ctx, key)
		if err != nil {
			t.Fatalf("Failed to delete: %v", err)
		}

		_, found, _ := backend.Get(ctx, key)
		if found {
			t.Fatal("Expected key to be deleted")
		}
	})

	// 测试 TTL 过期
	t.Run("TTLExpiration", func(t *testing.T) {
		key := "test:key3"
		value := "expires soon"

		err := backend.Set(ctx, key, value, 1*time.Second)
		if err != nil {
			t.Fatalf("Failed to set: %v", err)
		}

		// 立即获取应该成功
		_, found, err := backend.Get(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if !found {
			t.Fatal("Expected to find key before expiration")
		}

		// 等待过期
		time.Sleep(2 * time.Second)

		_, found, _ = backend.Get(ctx, key)
		if found {
			t.Fatal("Expected key to be expired")
		}
	})

	// 测试 Stats
	t.Run("Stats", func(t *testing.T) {
		stats := backend.Stats()
		if stats == nil {
			t.Fatal("Expected stats to be non-nil")
		}
		if stats.Sets < 2 {
			t.Errorf("Expected at least 2 sets, got %d", stats.Sets)
		}
		if stats.Hits < 1 {
			t.Errorf("Expected at least 1 hit, got %d", stats.Hits)
		}
	})
}

// TestRedisBackendNilValue 测试空值缓存（穿透保护）
func TestRedisBackendNilValue(t *testing.T) {
	t.Skip("Skipping Redis test - requires running Redis instance")

	config := DefaultRedisConfig()
	config.Prefix = "test_nil"

	backend, err := NewRedisBackend(config)
	if err != nil {
		t.Fatalf("Failed to create Redis backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()

	// 测试缓存 nil 值
	key := "nil:key"
	err = backend.Set(ctx, key, nil, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to set nil: %v", err)
	}

	// 获取应该返回 false（模拟缓存穿透保护）
	val, found, err := backend.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}
	if found {
		t.Error("Expected nil value to not be found")
	}
	if val != nil {
		t.Error("Expected nil value")
	}
}

// TestRedisBackendJitter 测试 TTL 随机偏移（雪崩保护）
func TestRedisBackendJitter(t *testing.T) {
	t.Skip("Skipping Redis test - requires running Redis instance")

	config := DefaultRedisConfig()
	config.Prefix = "test_jitter"

	backend, err := NewRedisBackend(config)
	if err != nil {
		t.Fatalf("Failed to create Redis backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()

	// 多次设置同一个 key，TTL 应该有微小差异
	baseTTL := 10 * time.Second
	key := "jitter:key"

	for i := 0; i < 5; i++ {
		err := backend.SetWithJitter(ctx, key, "value", baseTTL, 0.2)
		if err != nil {
			t.Fatalf("Failed to set with jitter: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 验证值仍然存在
	val, found, err := backend.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}
	if !found {
		t.Fatal("Expected to find key")
	}
	if val != "value" {
		t.Errorf("Expected 'value', got %v", val)
	}
}

// TestRedisBackendConnectionPool 测试连接池配置
func TestRedisBackendConnectionPool(t *testing.T) {
	t.Skip("Skipping Redis test - requires running Redis instance")

	config := &RedisConfig{
		Addr:         "localhost:6379",
		PoolSize:     20,
		MinIdleConns: 10,
		MaxRetries:   5,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	}

	backend, err := NewRedisBackend(config)
	if err != nil {
		t.Fatalf("Failed to create Redis backend: %v", err)
	}
	defer backend.Close()

	// 验证配置
	if backend.config.PoolSize != 20 {
		t.Errorf("Expected PoolSize 20, got %d", backend.config.PoolSize)
	}
	if backend.config.MinIdleConns != 10 {
		t.Errorf("Expected MinIdleConns 10, got %d", backend.config.MinIdleConns)
	}
}

// TestRedisBackendPrefix 测试 Key 前缀
func TestRedisBackendPrefix(t *testing.T) {
	t.Skip("Skipping Redis test - requires running Redis instance")

	config := &RedisConfig{
		Addr:     "localhost:6379",
		Prefix:   "myprefix",
		DefaultTTL: 5 * time.Second,
	}

	backend, err := NewRedisBackend(config)
	if err != nil {
		t.Fatalf("Failed to create Redis backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()

	key := "testkey"
	value := "prefixed value"

	err = backend.Set(ctx, key, value, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// 验证前缀是否正确添加
	fullKey := backend.buildKey(key)
	expectedFullKey := "myprefix:testkey"
	if fullKey != expectedFullKey {
		t.Errorf("Expected full key %s, got %s", expectedFullKey, fullKey)
	}

	got, found, err := backend.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}
	if !found {
		t.Fatal("Expected to find key")
	}
	if got != value {
		t.Errorf("Expected %v, got %v", value, got)
	}
}

// TestRedisBackendClose 测试关闭
func TestRedisBackendClose(t *testing.T) {
	t.Skip("Skipping Redis test - requires running Redis instance")

	config := DefaultRedisConfig()
	config.Prefix = "test_close"

	backend, err := NewRedisBackend(config)
	if err != nil {
		t.Fatalf("Failed to create Redis backend: %v", err)
	}

	// 第一次关闭应该成功
	err = backend.Close()
	if err != nil {
		t.Fatalf("Failed to close: %v", err)
	}

	// 第二次关闭应该无错误（幂等）
	err = backend.Close()
	if err != nil {
		t.Fatalf("Second close should not error: %v", err)
	}

	// 关闭后操作应该失败
	ctx := context.Background()
	_, _, err = backend.Get(ctx, "key")
	if err == nil {
		t.Error("Expected error when getting from closed backend")
	}
}

// BenchmarkRedisBackend 性能基准测试
func BenchmarkRedisBackend(b *testing.B) {
	b.Skip("Skipping Redis benchmark - requires running Redis instance")

	config := DefaultRedisConfig()
	config.Prefix = "bench"

	backend, err := NewRedisBackend(config)
	if err != nil {
		b.Fatalf("Failed to create Redis backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := "bench:set:" + string(rune(i))
			_ = backend.Set(ctx, key, "value", 5*time.Second)
		}
	})

	b.Run("Get_Hit", func(b *testing.B) {
		// 先设置值
		key := "bench:get:hit"
		_ = backend.Set(ctx, key, "value", 5*time.Second)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = backend.Get(ctx, key)
		}
	})

	b.Run("Get_Miss", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := "bench:get:miss:" + string(rune(i))
			_, _, _ = backend.Get(ctx, key)
		}
	})
}
