package integration

import (
	"context"
	"testing"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
	"github.com/coderiser/go-cache/pkg/core"
)

// TestEndToEnd 端到端测试 - 测试完整的缓存管理器流程
func TestEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 创建缓存管理器
	manager := core.NewCacheManager()
	defer manager.Close()

	ctx := context.Background()

	// 获取或创建缓存
	cache, err := manager.GetCache("e2e-test")
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	// Test 1: Basic Set and Get
	t.Run("BasicSetGet", func(t *testing.T) {
		err := cache.Set(ctx, "key1", "value1", 5*time.Second)
		if err != nil {
			t.Fatalf("Failed to set: %v", err)
		}

		value, found, err := cache.Get(ctx, "key1")
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

	// Test 2: Cache Miss
	t.Run("CacheMiss", func(t *testing.T) {
		_, found, err := cache.Get(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if found {
			t.Error("Expected key to not be found")
		}
	})

	// Test 3: Expiration
	t.Run("Expiration", func(t *testing.T) {
		err := cache.Set(ctx, "expire-key", "expire-value", 1*time.Second)
		if err != nil {
			t.Fatalf("Failed to set: %v", err)
		}

		// Should exist immediately
		_, found, _ := cache.Get(ctx, "expire-key")
		if !found {
			t.Error("Expected key to exist immediately")
		}

		// Wait for expiration
		time.Sleep(2 * time.Second)

		_, found, _ = cache.Get(ctx, "expire-key")
		if found {
			t.Error("Expected key to be expired")
		}
	})

	// Test 4: Delete
	t.Run("Delete", func(t *testing.T) {
		err := cache.Set(ctx, "delete-key", "delete-value", 5*time.Second)
		if err != nil {
			t.Fatalf("Failed to set: %v", err)
		}

		err = cache.Delete(ctx, "delete-key")
		if err != nil {
			t.Fatalf("Failed to delete: %v", err)
		}

		_, found, _ := cache.Get(ctx, "delete-key")
		if found {
			t.Error("Expected key to be deleted")
		}
	})

	// Test 5: Stats
	t.Run("Stats", func(t *testing.T) {
		stats := cache.Stats()
		if stats == nil {
			t.Fatal("Expected stats to be non-nil")
		}
		t.Logf("Cache Stats: Hits=%d, Misses=%d, Sets=%d, HitRate=%.2f%%",
			stats.Hits, stats.Misses, stats.Sets, stats.HitRate*100)
	})
}

// TestEndToEndMemoryBackend 内存后端端到端测试
func TestEndToEndMemoryBackend(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 创建内存后端配置
	config := &backend.CacheConfig{
		Name:         "e2e-memory",
		MaxSize:      1000,
		DefaultTTL:   5 * time.Minute,
		MaxTTL:       1 * time.Hour,
		EvictionPolicy: "lru",
	}

	// 创建后端
	b, err := backend.NewMemoryBackend(config)
	if err != nil {
		t.Fatalf("Failed to create memory backend: %v", err)
	}
	defer b.Close()

	ctx := context.Background()

	// Test LRU Eviction
	t.Run("LRUEviction", func(t *testing.T) {
		smallConfig := &backend.CacheConfig{
			Name:         "e2e-lru",
			MaxSize:      10, // Small size for testing eviction
			DefaultTTL:   5 * time.Minute,
			EvictionPolicy: "lru",
		}

		smallBackend, err := backend.NewMemoryBackend(smallConfig)
		if err != nil {
			t.Fatalf("Failed to create small backend: %v", err)
		}
		defer smallBackend.Close()

		// Fill the cache beyond capacity
		for i := 0; i < 15; i++ {
			key := "key" + string(rune(i))
			err := smallBackend.Set(ctx, key, "value"+string(rune(i)), 5*time.Minute)
			if err != nil {
				t.Fatalf("Failed to set key %d: %v", i, err)
			}
		}

		stats := smallBackend.Stats()
		if stats.Size > smallConfig.MaxSize {
			t.Errorf("Cache size %d exceeds max size %d", stats.Size, smallConfig.MaxSize)
		}
		t.Logf("LRU eviction test: Size=%d, Evictions=%d", stats.Size, stats.Evictions)
	})
}

// TestEndToEndConcurrent 并发端到端测试
func TestEndToEndConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	manager := core.NewCacheManager()
	defer manager.Close()

	cache, err := manager.GetCache("concurrent-test")
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	ctx := context.Background()
	const goroutines = 50
	const operations = 50

	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < operations; j++ {
				key := "concurrent:" + string(rune(id)) + ":" + string(rune(j))
				_ = cache.Set(ctx, key, "value", 5*time.Second)
				_, _, _ = cache.Get(ctx, key)
			}
			done <- true
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}

	stats := cache.Stats()
	t.Logf("Concurrent e2e test: Sets=%d, Hits=%d, Misses=%d, HitRate=%.2f%%",
		stats.Sets, stats.Hits, stats.Misses, stats.HitRate*100)
}
