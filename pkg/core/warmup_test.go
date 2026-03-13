package core

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheWarmer_Basic(t *testing.T) {
	manager := NewCacheManager()
	defer manager.Close()

	warmer := NewCacheWarmer(manager)

	ctx := context.Background()
	entries := []WarmUpEntry{
		{
			Cache: "test-cache",
			Key:   "warm:key1",
			Loader: func() (interface{}, error) {
				return "warmed-value-1", nil
			},
			TTL: 10 * time.Minute,
		},
		{
			Cache: "test-cache",
			Key:   "warm:key2",
			Loader: func() (interface{}, error) {
				return "warmed-value-2", nil
			},
			TTL: 10 * time.Minute,
		},
	}

	// Execute warm up
	err := warmer.WarmUp(ctx, entries)
	if err != nil {
		t.Fatalf("WarmUp failed: %v", err)
	}

	// Verify values are cached
	cache, _ := manager.GetCache("test-cache")
	
	val1, found, _ := cache.Get(ctx, "warm:key1")
	if !found {
		t.Fatal("Expected key1 to be warmed")
	}
	if val1 != "warmed-value-1" {
		t.Fatalf("Expected 'warmed-value-1', got '%v'", val1)
	}

	val2, found, _ := cache.Get(ctx, "warm:key2")
	if !found {
		t.Fatal("Expected key2 to be warmed")
	}
	if val2 != "warmed-value-2" {
		t.Fatalf("Expected 'warmed-value-2', got '%v'", val2)
	}

	// Check stats
	stats := warmer.GetStats()
	if stats.Success != 2 {
		t.Errorf("Expected 2 successful warmups, got %d", stats.Success)
	}
	if stats.Failed != 0 {
		t.Errorf("Expected 0 failures, got %d", stats.Failed)
	}
}

func TestCacheWarmer_FailedLoader(t *testing.T) {
	manager := NewCacheManager()
	defer manager.Close()

	warmer := NewCacheWarmer(manager)

	ctx := context.Background()
	entries := []WarmUpEntry{
		{
			Cache: "test-cache",
			Key:   "fail:key1",
			Loader: func() (interface{}, error) {
				return nil, errors.New("loader error")
			},
			TTL: 10 * time.Minute,
		},
		{
			Cache: "test-cache",
			Key:   "success:key1",
			Loader: func() (interface{}, error) {
				return "success", nil
			},
			TTL: 10 * time.Minute,
		},
	}

	// Execute warm up (should not fail completely)
	err := warmer.WarmUp(ctx, entries)
	if err != nil {
		t.Logf("WarmUp returned error (expected): %v", err)
	}

	// Check stats
	stats := warmer.GetStats()
	if stats.Success != 1 {
		t.Errorf("Expected 1 successful warmup, got %d", stats.Success)
	}
	if stats.Failed != 1 {
		t.Errorf("Expected 1 failure, got %d", stats.Failed)
	}
}

func TestCacheWarmer_Concurrent(t *testing.T) {
	manager := NewCacheManager()
	defer manager.Close()

	warmer := NewCacheWarmer(manager)

	ctx := context.Background()
	
	// Create 20 entries
	entries := make([]WarmUpEntry, 20)
	for i := 0; i < 20; i++ {
		idx := i
		entries[i] = WarmUpEntry{
			Cache: "test-cache",
			Key:   "concurrent:" + string(rune('0'+idx)),
			Loader: func() (interface{}, error) {
				time.Sleep(10 * time.Millisecond) // Simulate work
				return "value-" + string(rune('0'+idx)), nil
			},
			TTL: 10 * time.Minute,
		}
	}

	// Warm up with concurrency
	err := warmer.WarmUpConcurrent(ctx, entries, 4)
	if err != nil {
		t.Fatalf("WarmUpConcurrent failed: %v", err)
	}

	// Verify all entries
	cache, _ := manager.GetCache("test-cache")
	for i := 0; i < 20; i++ {
		key := "concurrent:" + string(rune('0'+i))
		_, found, _ := cache.Get(ctx, key)
		if !found {
			t.Errorf("Expected key '%s' to be warmed", key)
		}
	}

	// Check stats
	stats := warmer.GetStats()
	if stats.Success != 20 {
		t.Errorf("Expected 20 successful warmups, got %d", stats.Success)
	}
}

func TestCacheWarmer_SkipExisting(t *testing.T) {
	manager := NewCacheManager()
	defer manager.Close()

	ctx := context.Background()
	
	// Pre-populate cache
	cache, _ := manager.GetCache("test-cache")
	cache.Set(ctx, "existing:key", "existing-value", 10*time.Minute)

	warmer := NewCacheWarmer(manager)
	
	entries := []WarmUpEntry{
		{
			Cache: "test-cache",
			Key:   "existing:key",
			Loader: func() (interface{}, error) {
				return "new-value", nil
			},
			TTL: 10 * time.Minute,
		},
	}

	// Warm up with SkipExisting
	config := DefaultWarmerConfig()
	config.SkipExisting = true
	
	err := warmer.WarmUpWithConfig(ctx, entries, config)
	if err != nil {
		t.Fatalf("WarmUpWithConfig failed: %v", err)
	}

	// Verify original value is still there
	val, found, _ := cache.Get(ctx, "existing:key")
	if !found {
		t.Fatal("Expected key to exist")
	}
	if val != "existing-value" {
		t.Fatalf("Expected 'existing-value', got '%v'", val)
	}

	// Check stats
	stats := warmer.GetStats()
	if stats.Skipped != 1 {
		t.Errorf("Expected 1 skipped, got %d", stats.Skipped)
	}
}

func TestCacheWarmer_AddEntry(t *testing.T) {
	manager := NewCacheManager()
	defer manager.Close()

	warmer := NewCacheWarmer(manager)

	// Add entries one by one
	warmer.AddEntry(WarmUpEntry{
		Cache: "test-cache",
		Key:   "add:key1",
		Loader: func() (interface{}, error) {
			return "value1", nil
		},
		TTL: 10 * time.Minute,
	})

	warmer.AddEntry(WarmUpEntry{
		Cache: "test-cache",
		Key:   "add:key2",
		Loader: func() (interface{}, error) {
			return "value2", nil
		},
		TTL: 10 * time.Minute,
	})

	// Warm up all
	ctx := context.Background()
	err := warmer.WarmUpAll(ctx)
	if err != nil {
		t.Fatalf("WarmUpAll failed: %v", err)
	}

	// Verify
	cache, _ := manager.GetCache("test-cache")
	
	val1, found, _ := cache.Get(ctx, "add:key1")
	if !found || val1 != "value1" {
		t.Errorf("Key1 not warmed correctly")
	}

	val2, found, _ := cache.Get(ctx, "add:key2")
	if !found || val2 != "value2" {
		t.Errorf("Key2 not warmed correctly")
	}
}

func TestWarmUpBuilder(t *testing.T) {
	manager := NewCacheManager()
	defer manager.Close()

	ctx := context.Background()

	var loadCount int32
	
	// Use builder pattern
	err := NewWarmUpBuilder(manager).
		Add("test-cache", "builder:key1", func() (interface{}, error) {
			atomic.AddInt32(&loadCount, 1)
			return "builder-value-1", nil
		}, 10*time.Minute).
		Add("test-cache", "builder:key2", func() (interface{}, error) {
			atomic.AddInt32(&loadCount, 1)
			return "builder-value-2", nil
		}, 10*time.Minute).
		Execute(ctx, 2)

	if err != nil {
		t.Fatalf("Builder Execute failed: %v", err)
	}

	// Verify
	cache, _ := manager.GetCache("test-cache")
	
	val1, found, _ := cache.Get(ctx, "builder:key1")
	if !found || val1 != "builder-value-1" {
		t.Errorf("Key1 not warmed correctly")
	}

	val2, found, _ := cache.Get(ctx, "builder:key2")
	if !found || val2 != "builder-value-2" {
		t.Errorf("Key2 not warmed correctly")
	}

	if loadCount != 2 {
		t.Errorf("Expected 2 loads, got %d", loadCount)
	}
}

func TestWarmUpBuilder_WithTimeout(t *testing.T) {
	manager := NewCacheManager()
	defer manager.Close()

	ctx := context.Background()

	// Fast loader
	err := NewWarmUpBuilder(manager).
		AddWithTimeout("test-cache", "fast:key", func() (interface{}, error) {
			return "fast-value", nil
		}, 10*time.Minute, 1*time.Second).
		Execute(ctx, 1)

	if err != nil {
		t.Fatalf("Fast loader failed: %v", err)
	}

	// Verify
	cache, _ := manager.GetCache("test-cache")
	val, found, _ := cache.Get(ctx, "fast:key")
	if !found || val != "fast-value" {
		t.Errorf("Fast key not warmed correctly")
	}
}

func TestCacheWarmer_Clear(t *testing.T) {
	manager := NewCacheManager()
	defer manager.Close()

	warmer := NewCacheWarmer(manager)

	// Add entries
	warmer.AddEntry(WarmUpEntry{
		Cache: "test-cache",
		Key:   "clear:key",
		Loader: func() (interface{}, error) {
			return "value", nil
		},
		TTL: 10 * time.Minute,
	})

	// Clear
	warmer.Clear()

	// WarmUpAll should do nothing
	ctx := context.Background()
	err := warmer.WarmUpAll(ctx)
	if err != nil {
		t.Fatalf("WarmUpAll after clear failed: %v", err)
	}

	stats := warmer.GetStats()
	if stats.Total != 0 {
		t.Errorf("Expected 0 total after clear, got %d", stats.Total)
	}
}
