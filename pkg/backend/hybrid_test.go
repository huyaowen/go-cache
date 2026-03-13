package backend

import (
	"context"
	"testing"
	"time"
)

func TestHybridBackend_Basic(t *testing.T) {
	// Skip if Redis is not available
	redisAddr := "localhost:6379"
	
	config := &HybridConfig{
		L1Config: &CacheConfig{
			Name:           "test-hybrid",
			MaxSize:        100,
			DefaultTTL:     5 * time.Minute,
			MaxTTL:         30 * time.Minute,
			EvictionPolicy: "lru",
		},
		L2Config: &RedisConfig{
			Addr:       redisAddr,
			DefaultTTL: 10 * time.Minute,
			MaxTTL:     1 * time.Hour,
		},
		L1WriteBackTTL: 5 * time.Minute,
	}

	// Try to create hybrid backend (may fail if Redis is not running)
	hybrid, err := NewHybridBackend(config)
	if err != nil {
		t.Skipf("Redis not available, skipping hybrid backend test: %v", err)
	}
	defer hybrid.Close()

	ctx := context.Background()

	// Test Set
	err = hybrid.Set(ctx, "test:key1", "value1", 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Test Get
	val, found, err := hybrid.Get(ctx, "test:key1")
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}
	if !found {
		t.Fatal("Expected to find value")
	}
	if val != "value1" {
		t.Fatalf("Expected 'value1', got '%v'", val)
	}

	// Test Delete
	err = hybrid.Delete(ctx, "test:key1")
	if err != nil {
		t.Fatalf("Failed to delete value: %v", err)
	}

	// Verify deletion
	_, found, _ = hybrid.Get(ctx, "test:key1")
	if found {
		t.Fatal("Expected value to be deleted")
	}
}

func TestHybridBackend_L1L2Flow(t *testing.T) {
	redisAddr := "localhost:6379"
	
	config := &HybridConfig{
		L1Config: &CacheConfig{
			Name:    "test-hybrid-flow",
			MaxSize: 100,
		},
		L2Config: &RedisConfig{
			Addr: redisAddr,
		},
		L1WriteBackTTL: 5 * time.Minute,
	}

	hybrid, err := NewHybridBackend(config)
	if err != nil {
		t.Skipf("Redis not available, skipping: %v", err)
	}
	defer hybrid.Close()

	ctx := context.Background()

	// Set value in L2 only (bypass L1)
	err = hybrid.l2.Set(ctx, "test:flow", "l2value", 10*time.Minute)
	if err != nil {
		t.Skipf("Failed to set L2 value: %v", err)
	}

	// Get should hit L2 and backfill L1
	val, found, err := hybrid.Get(ctx, "test:flow")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find value in L2")
	}
	if val != "l2value" {
		t.Fatalf("Expected 'l2value', got '%v'", val)
	}

	// Verify L1 was backfilled
	_, found, _ = hybrid.l1.Get(ctx, "test:flow")
	if !found {
		t.Fatal("Expected L1 to be backfilled from L2")
	}

	// Check stats
	stats := hybrid.GetHybridStats()
	if stats.getL2Fallbacks() < 1 {
		t.Fatal("Expected at least 1 L2 fallback")
	}
	if stats.getL1Backfills() < 1 {
		t.Fatal("Expected at least 1 L1 backfill")
	}
}

func TestHybridBackend_Stats(t *testing.T) {
	config := &HybridConfig{
		L1Config: &CacheConfig{
			Name:    "test-hybrid-stats",
			MaxSize: 100,
		},
		L2Config: &RedisConfig{
			Addr: "localhost:6379",
		},
	}

	hybrid, err := NewHybridBackend(config)
	if err != nil {
		t.Skipf("Redis not available, skipping: %v", err)
	}
	defer hybrid.Close()

	ctx := context.Background()

	// Set and get some values
	_ = hybrid.Set(ctx, "key1", "val1", 5*time.Minute)
	_, _, _ = hybrid.Get(ctx, "key1")
	_, _, _ = hybrid.Get(ctx, "key2") // Miss

	stats := hybrid.Stats()
	if stats.Sets < 1 {
		t.Errorf("Expected at least 1 set, got %d", stats.Sets)
	}
	if stats.Size < 1 {
		t.Errorf("Expected size >= 1, got %d", stats.Size)
	}
}

func TestHybridBackend_Close(t *testing.T) {
	config := &HybridConfig{
		L1Config: &CacheConfig{
			Name:    "test-hybrid-close",
			MaxSize: 100,
		},
		L2Config: &RedisConfig{
			Addr: "localhost:6379",
		},
	}

	hybrid, err := NewHybridBackend(config)
	if err != nil {
		t.Skipf("Redis not available, skipping: %v", err)
	}

	// Close should succeed
	err = hybrid.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Double close should be safe
	err = hybrid.Close()
	if err != nil {
		t.Fatalf("Double close failed: %v", err)
	}

	// Operations after close should be safe
	ctx := context.Background()
	_, found, err := hybrid.Get(ctx, "key")
	if err != nil || found {
		t.Log("Get after close returned:", err, found)
	}
}

func TestHybridBackend_Concurrent(t *testing.T) {
	config := &HybridConfig{
		L1Config: &CacheConfig{
			Name:    "test-hybrid-concurrent",
			MaxSize: 1000,
		},
		L2Config: &RedisConfig{
			Addr: "localhost:6379",
		},
	}

	hybrid, err := NewHybridBackend(config)
	if err != nil {
		t.Skipf("Redis not available, skipping: %v", err)
	}
	defer hybrid.Close()

	ctx := context.Background()
	done := make(chan bool, 10)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(idx int) {
			key := "concurrent:" + string(rune('0'+idx))
			_ = hybrid.Set(ctx, key, idx, 5*time.Minute)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(idx int) {
			key := "concurrent:" + string(rune('0'+idx))
			_, _, _ = hybrid.Get(ctx, key)
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}
}
