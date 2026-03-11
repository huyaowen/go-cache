package core

import (
	"context"
	"testing"
	"time"

	"github.com/go-cache-framework/pkg/backend"
)

func TestCacheManager(t *testing.T) {
	t.Run("NewCacheManager", func(t *testing.T) {
		manager := NewCacheManager()
		defer manager.Close()

		if manager == nil {
			t.Fatal("Expected non-nil manager")
		}
	})

	t.Run("GetCache - first time creates", func(t *testing.T) {
		manager := NewCacheManager()
		defer manager.Close()

		cache, err := manager.GetCache("test-cache")
		if err != nil {
			t.Fatalf("GetCache failed: %v", err)
		}
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
	})

	t.Run("GetCache - returns same instance", func(t *testing.T) {
		manager := NewCacheManager()
		defer manager.Close()

		cache1, _ := manager.GetCache("test-cache")
		cache2, _ := manager.GetCache("test-cache")

		if cache1 != cache2 {
			t.Error("Expected same cache instance")
		}
	})

	t.Run("DefaultCacheConfig", func(t *testing.T) {
		config := DefaultCacheConfig("test")

		if config.Name != "test" {
			t.Errorf("Expected name 'test', got '%s'", config.Name)
		}
		if config.MaxSize != 10000 {
			t.Errorf("Expected MaxSize 10000, got %d", config.MaxSize)
		}
		if config.DefaultTTL != 30*time.Minute {
			t.Errorf("Expected DefaultTTL 30m, got %v", config.DefaultTTL)
		}
		if config.MaxTTL != 24*time.Hour {
			t.Errorf("Expected MaxTTL 24h, got %v", config.MaxTTL)
		}
	})

	t.Run("RegisterBackend", func(t *testing.T) {
		manager := NewCacheManager()
		defer manager.Close()

		customFactory := func(config *CacheConfig) (CacheBackend, error) {
			return backend.NewMemoryBackend(config)
		}

		err := manager.RegisterBackend("custom", customFactory)
		if err != nil {
			t.Fatalf("RegisterBackend failed: %v", err)
		}

		cache, err := manager.GetCache("custom-cache")
		if err != nil {
			t.Fatalf("GetCache with custom backend failed: %v", err)
		}
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
	})

	t.Run("RegisterBackend - invalid input", func(t *testing.T) {
		manager := NewCacheManager()
		defer manager.Close()

		err := manager.RegisterBackend("", nil)
		if err == nil {
			t.Error("Expected error for invalid input")
		}
	})

	t.Run("CacheConfig via GetCache", func(t *testing.T) {
		manager := NewCacheManager()
		defer manager.Close()

		// GetCache creates cache with default config if not registered
		cache, err := manager.GetCache("auto-created")
		if err != nil {
			t.Fatalf("GetCache failed: %v", err)
		}
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}

		// Verify it's the same instance on second call
		cache2, err := manager.GetCache("auto-created")
		if err != nil {
			t.Fatalf("GetCache failed: %v", err)
		}
		if cache != cache2 {
			t.Error("Expected same cache instance")
		}
	})

	t.Run("Execute - Cacheable", func(t *testing.T) {
		manager := NewCacheManager()
		defer manager.Close()

		ctx := context.Background()
		meta := &MethodMeta{
			CacheType: "cacheable",
			CacheName: "test",
			KeyExpr:   "'test-key'",
		}

		// First call - cache miss
		result, err := manager.Execute(ctx, meta, nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result on cache miss, got %v", result)
		}

		// Set value in cache
		cache, _ := manager.GetCache("test")
		cache.Set(ctx, "test-key", "cached-value", 5*time.Minute)

		// Second call - cache hit
		result, err = manager.Execute(ctx, meta, nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		if result != "cached-value" {
			t.Errorf("Expected 'cached-value', got %v", result)
		}
	})

	t.Run("Execute - CacheEvict", func(t *testing.T) {
		manager := NewCacheManager()
		defer manager.Close()

		ctx := context.Background()
		cache, _ := manager.GetCache("evict-test")
		cache.Set(ctx, "evict-key", "value", 5*time.Minute)

		// Verify value exists
		if v, found, _ := cache.Get(ctx, "evict-key"); !found || v != "value" {
			t.Fatal("Expected value to exist before eviction")
		}

		meta := &MethodMeta{
			CacheType: "cacheevict",
			CacheName: "evict-test",
			KeyExpr:   "'evict-key'",
		}

		_, err := manager.Execute(ctx, meta, nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Verify value is evicted
		if _, found, _ := cache.Get(ctx, "evict-key"); found {
			t.Error("Expected value to be evicted")
		}
	})

	t.Run("Execute - unknown type", func(t *testing.T) {
		manager := NewCacheManager()
		defer manager.Close()

		ctx := context.Background()
		meta := &MethodMeta{
			CacheType: "unknown",
		}

		_, err := manager.Execute(ctx, meta, nil)
		if err == nil {
			t.Error("Expected error for unknown cache type")
		}
	})

	t.Run("Close", func(t *testing.T) {
		manager := NewCacheManager()

		// Create some caches
		manager.GetCache("cache1")
		manager.GetCache("cache2")

		err := manager.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	})

	t.Run("GetEvaluator", func(t *testing.T) {
		manager := NewCacheManager()
		defer manager.Close()

		impl, ok := manager.(*cacheManagerImpl)
		if !ok {
			t.Fatal("Expected cacheManagerImpl")
		}

		evaluator := impl.GetEvaluator()
		if evaluator == nil {
			t.Fatal("Expected non-nil evaluator")
		}
	})
}

func TestCacheManagerConcurrency(t *testing.T) {
	manager := NewCacheManager()
	defer manager.Close()

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			cacheName := "concurrent-cache"
			cache, err := manager.GetCache(cacheName)
			if err != nil {
				t.Errorf("GetCache failed: %v", err)
				done <- false
				return
			}

			ctx := context.Background()
			key := "key-" + string(rune('0'+id))
			cache.Set(ctx, key, id, 5*time.Minute)

			val, found, _ := cache.Get(ctx, key)
			if !found || val != id {
				t.Errorf("Expected %d, got %v", id, val)
			}

			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestStatsCounter(t *testing.T) {
	counter := NewStatsCounter(1000)

	counter.RecordHit()
	counter.RecordHit()
	counter.RecordMiss()
	counter.RecordSet()
	counter.RecordDelete()
	counter.RecordEviction()
	counter.SetSize(50)

	snapshot := counter.Snapshot()

	if snapshot.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", snapshot.Hits)
	}
	if snapshot.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", snapshot.Misses)
	}
	if snapshot.Sets != 1 {
		t.Errorf("Expected 1 set, got %d", snapshot.Sets)
	}
	if snapshot.Deletes != 1 {
		t.Errorf("Expected 1 delete, got %d", snapshot.Deletes)
	}
	if snapshot.Evictions != 1 {
		t.Errorf("Expected 1 eviction, got %d", snapshot.Evictions)
	}
	if snapshot.Size != 50 {
		t.Errorf("Expected size 50, got %d", snapshot.Size)
	}
	if snapshot.MaxSize != 1000 {
		t.Errorf("Expected maxSize 1000, got %d", snapshot.MaxSize)
	}
}

func TestCacheItem(t *testing.T) {
	item := &CacheItem{
		Value:     "test-value",
		ExpiresAt: time.Now().Add(1 * time.Second),
	}

	if item.IsExpired() {
		t.Error("Expected item to not be expired")
	}

	// Create expired item
	expiredItem := &CacheItem{
		Value:     "expired-value",
		ExpiresAt: time.Now().Add(-1 * time.Second),
	}

	if !expiredItem.IsExpired() {
		t.Error("Expected item to be expired")
	}

	// Item with zero ExpiresAt should not be expired
	noExpiryItem := &CacheItem{
		Value:     "no-expiry-value",
		ExpiresAt: time.Time{},
	}

	if noExpiryItem.IsExpired() {
		t.Error("Expected item with zero ExpiresAt to not be expired")
	}
}

func BenchmarkCacheManager(b *testing.B) {
	manager := NewCacheManager()
	defer manager.Close()

	ctx := context.Background()
	meta := &MethodMeta{
		CacheType: "cacheable",
		CacheName: "bench",
		KeyExpr:   "'bench-key'",
	}

	// Warm up
	manager.GetCache("bench")
	cache, _ := manager.GetCache("bench")
	cache.Set(ctx, "bench-key", "value", 5*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.Execute(ctx, meta, nil)
	}
}
