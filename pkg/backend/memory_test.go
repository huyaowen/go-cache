package backend

import (
	"context"
	"testing"
	"time"
)

func TestMemoryBackend(t *testing.T) {
	ctx := context.Background()

	t.Run("NewMemoryBackend", func(t *testing.T) {
		config := DefaultCacheConfig("test")
		backend, err := NewMemoryBackend(config)
		if err != nil {
			t.Fatalf("Failed to create MemoryBackend: %v", err)
		}
		defer backend.Close()

		if backend == nil {
			t.Fatal("MemoryBackend is nil")
		}
	})

	t.Run("Set and Get", func(t *testing.T) {
		config := DefaultCacheConfig("test")
		backend, _ := NewMemoryBackend(config)
		defer backend.Close()

		// Set
		err := backend.Set(ctx, "key1", "value1", 5*time.Minute)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Get
		value, found, err := backend.Get(ctx, "key1")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if !found {
			t.Error("Expected to find key1")
		}
		if value != "value1" {
			t.Errorf("Expected value1, got %v", value)
		}
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		config := DefaultCacheConfig("test")
		backend, _ := NewMemoryBackend(config)
		defer backend.Close()

		value, found, err := backend.Get(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if found {
			t.Error("Expected not to find nonexistent key")
		}
		if value != nil {
			t.Errorf("Expected nil value, got %v", value)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		config := DefaultCacheConfig("test")
		backend, _ := NewMemoryBackend(config)
		defer backend.Close()

		backend.Set(ctx, "key1", "value1", 5*time.Minute)
		err := backend.Delete(ctx, "key1")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, found, _ := backend.Get(ctx, "key1")
		if found {
			t.Error("Expected key1 to be deleted")
		}
	})

	t.Run("Expiration", func(t *testing.T) {
		config := DefaultCacheConfig("test")
		config.DefaultTTL = 100 * time.Millisecond
		backend, _ := NewMemoryBackend(config)
		defer backend.Close()

		backend.Set(ctx, "expire_key", "expire_value", 100*time.Millisecond)

		// Should exist immediately
		_, found, _ := backend.Get(ctx, "expire_key")
		if !found {
			t.Error("Expected key to exist immediately")
		}

		// Wait for expiration
		time.Sleep(200 * time.Millisecond)

		// Should be expired
		_, found, _ = backend.Get(ctx, "expire_key")
		if found {
			t.Error("Expected key to be expired")
		}
	})

	t.Run("Stats", func(t *testing.T) {
		config := DefaultCacheConfig("test")
		backend, _ := NewMemoryBackend(config)
		defer backend.Close()

		backend.Set(ctx, "key1", "value1", 5*time.Minute)
		backend.Set(ctx, "key2", "value2", 5*time.Minute)
		backend.Get(ctx, "key1") // hit
		backend.Get(ctx, "key1") // hit
		backend.Get(ctx, "key3") // miss

		stats := backend.Stats()
		if stats.Sets != 2 {
			t.Errorf("Expected 2 sets, got %d", stats.Sets)
		}
		if stats.Hits != 2 {
			t.Errorf("Expected 2 hits, got %d", stats.Hits)
		}
		if stats.Misses != 1 {
			t.Errorf("Expected 1 miss, got %d", stats.Misses)
		}
		if stats.Size != 2 {
			t.Errorf("Expected size 2, got %d", stats.Size)
		}
	})

	t.Run("Eviction LRU", func(t *testing.T) {
		config := DefaultCacheConfig("test")
		config.MaxSize = 3
		config.EvictionPolicy = "lru"
		backend, _ := NewMemoryBackend(config)
		defer backend.Close()

		// Add 3 items
		backend.Set(ctx, "key1", "value1", 0)
		backend.Set(ctx, "key2", "value2", 0)
		backend.Set(ctx, "key3", "value3", 0)

		// Access key1 to make it recently used
		backend.Get(ctx, "key1")

		// Add 4th item, should evict key2 (least recently used)
		backend.Set(ctx, "key4", "value4", 0)

		// key1 should still exist
		_, found, _ := backend.Get(ctx, "key1")
		if !found {
			t.Error("Expected key1 to exist")
		}

		// key2 should be evicted
		_, found, _ = backend.Get(ctx, "key2")
		if found {
			t.Error("Expected key2 to be evicted")
		}
	})

	t.Run("Close", func(t *testing.T) {
		config := DefaultCacheConfig("test")
		backend, _ := NewMemoryBackend(config)

		err := backend.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}

		// Double close should not panic
		err = backend.Close()
		if err != nil {
			t.Fatalf("Double close failed: %v", err)
		}
	})
}

func TestMemoryBackendValidation(t *testing.T) {
	t.Run("Empty name", func(t *testing.T) {
		config := &CacheConfig{
			Name: "",
		}
		_, err := NewMemoryBackend(config)
		if err == nil {
			t.Error("Expected error for empty name")
		}
	})

	t.Run("Invalid max size", func(t *testing.T) {
		config := &CacheConfig{
			Name:    "test",
			MaxSize: 0,
		}
		_, err := NewMemoryBackend(config)
		if err == nil {
			t.Error("Expected error for invalid max size")
		}
	})
}

func TestDefaultKeyBuilder(t *testing.T) {
	t.Run("Build with prefix", func(t *testing.T) {
		kb := NewDefaultKeyBuilder(":", "cache")
		key := kb.Build("user", "123")
		if key != "cache:user:123" {
			t.Errorf("Expected 'cache:user:123', got '%s'", key)
		}
	})

	t.Run("Build without prefix", func(t *testing.T) {
		kb := NewDefaultKeyBuilder(":", "")
		key := kb.Build("user", "123")
		if key != "user:123" {
			t.Errorf("Expected 'user:123', got '%s'", key)
		}
	})

	t.Run("Build single part", func(t *testing.T) {
		kb := NewDefaultKeyBuilder(":", "cache")
		key := kb.Build("single")
		if key != "cache:single" {
			t.Errorf("Expected 'cache:single', got '%s'", key)
		}
	})
}

func TestTTLManager(t *testing.T) {
	t.Run("Normalize - zero TTL", func(t *testing.T) {
		mgr := NewTTLManager(30*time.Minute, 24*time.Hour)
		result := mgr.Normalize(0)
		if result != 30*time.Minute {
			t.Errorf("Expected default TTL, got %v", result)
		}
	})

	t.Run("Normalize - within range", func(t *testing.T) {
		mgr := NewTTLManager(30*time.Minute, 24*time.Hour)
		result := mgr.Normalize(1 * time.Hour)
		if result != 1*time.Hour {
			t.Errorf("Expected 1h, got %v", result)
		}
	})

	t.Run("Normalize - exceeds max", func(t *testing.T) {
		mgr := NewTTLManager(30*time.Minute, 24*time.Hour)
		result := mgr.Normalize(48 * time.Hour)
		if result != 24*time.Hour {
			t.Errorf("Expected max TTL (24h), got %v", result)
		}
	})
}

func TestBackendRegistry(t *testing.T) {
	t.Run("Register and Get", func(t *testing.T) {
		factory := func(config *CacheConfig) (CacheBackend, error) {
			return NewMemoryBackend(config)
		}
		Register("test-backend", factory)

		retrieved, exists := GetFactory("test-backend")
		if !exists {
			t.Error("Expected to find test-backend")
		}
		if retrieved == nil {
			t.Error("Expected non-nil factory")
		}
	})

	t.Run("Get non-existent", func(t *testing.T) {
		_, exists := GetFactory("nonexistent")
		if exists {
			t.Error("Expected not to find nonexistent backend")
		}
	})
}

func BenchmarkMemoryBackend(b *testing.B) {
	config := DefaultCacheConfig("bench")
	backend, _ := NewMemoryBackend(config)
	defer backend.Close()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "key" + string(rune(i%1000))
		backend.Set(ctx, key, i, 5*time.Minute)
		backend.Get(ctx, key)
	}
}
