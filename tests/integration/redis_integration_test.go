package integration

import (
	"context"
	"testing"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
)

// TestRedisBackendIntegration Redis 后端集成测试
// 需要运行中的 Redis 实例
func TestRedisBackendIntegration(t *testing.T) {
	// 检查是否跳过测试
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &backend.RedisConfig{
		Addr:       "localhost:6379",
		DefaultTTL: 5 * time.Second,
		MaxTTL:     1 * time.Minute,
	}

	b, err := backend.NewRedisBackend(config)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer b.Close()

	ctx := context.Background()

	// Test Set and Get
	t.Run("SetAndGet", func(t *testing.T) {
		err := b.Set(ctx, "test:key1", "value1", 5*time.Second)
		if err != nil {
			t.Fatalf("Failed to set: %v", err)
		}

		value, found, err := b.Get(ctx, "test:key1")
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if !found {
			t.Fatal("Expected key to be found")
		}
		if value != "value1" {
			t.Errorf("Expected value1, got %v", value)
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		err := b.Set(ctx, "test:key2", "value2", 5*time.Second)
		if err != nil {
			t.Fatalf("Failed to set: %v", err)
		}

		err = b.Delete(ctx, "test:key2")
		if err != nil {
			t.Fatalf("Failed to delete: %v", err)
		}

		_, found, _ := b.Get(ctx, "test:key2")
		if found {
			t.Error("Expected key to be deleted")
		}
	})

	// Test Expiration
	t.Run("Expiration", func(t *testing.T) {
		err := b.Set(ctx, "test:expire", "expiring", 1*time.Second)
		if err != nil {
			t.Fatalf("Failed to set: %v", err)
		}

		// Should exist immediately
		_, found, _ := b.Get(ctx, "test:expire")
		if !found {
			t.Error("Expected key to exist immediately after set")
		}

		// Wait for expiration
		time.Sleep(2 * time.Second)

		_, found, _ = b.Get(ctx, "test:expire")
		if found {
			t.Error("Expected key to be expired")
		}
	})

	// Test Stats
	t.Run("Stats", func(t *testing.T) {
		stats := b.Stats()
		if stats == nil {
			t.Fatal("Expected stats to be non-nil")
		}
		t.Logf("Redis Stats: Hits=%d, Misses=%d, Sets=%d", stats.Hits, stats.Misses, stats.Sets)
	})
}

// TestRedisBackendConcurrent 并发访问测试
func TestRedisBackendConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &backend.RedisConfig{
		Addr:       "localhost:6379",
		DefaultTTL: 10 * time.Second,
	}

	b, err := backend.NewRedisBackend(config)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	const goroutines = 100
	const operations = 100

	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < operations; j++ {
				key := "test:concurrent:" + string(rune(id)) + ":" + string(rune(j))
				_ = b.Set(ctx, key, "value", 5*time.Second)
				_, _, _ = b.Get(ctx, key)
			}
			done <- true
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}

	stats := b.Stats()
	t.Logf("Concurrent test completed: Sets=%d, Hits=%d, Misses=%d", stats.Sets, stats.Hits, stats.Misses)
}
