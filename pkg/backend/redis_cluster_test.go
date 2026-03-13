package backend

import (
	"context"
	"testing"
	"time"
)

func TestRedisClusterBackend_Basic(t *testing.T) {
	t.Skip("Skipping test that requires Redis Cluster")

	config := DefaultRedisClusterConfig()
	config.Addrs = []string{"localhost:7000", "localhost:7001", "localhost:7002"}

	backend, err := NewRedisClusterBackend(config)
	if err != nil {
		t.Fatalf("Failed to create cluster backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()

	err = backend.Set(ctx, "test-key", "test-value", 5*time.Minute)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	val, found, err := backend.Get(ctx, "test-key")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if !found {
		t.Error("Key should be found")
	}
	if val != "test-value" {
		t.Errorf("Expected test-value, got %v", val)
	}

	err = backend.Delete(ctx, "test-key")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	_, found, _ = backend.Get(ctx, "test-key")
	if found {
		t.Error("Key should not be found after deletion")
	}
}

func TestRedisClusterBackend_NilValue(t *testing.T) {
	t.Skip("Skipping test that requires Redis Cluster")

	config := DefaultRedisClusterConfig()
	config.Addrs = []string{"localhost:7000"}

	backend, err := NewRedisClusterBackend(config)
	if err != nil {
		t.Fatalf("Failed to create cluster backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()

	_, found, err := backend.Get(ctx, "non-existent-key")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if found {
		t.Error("Non-existent key should not be found")
	}
}

func TestRedisClusterBackend_NilMarker(t *testing.T) {
	t.Skip("Skipping test that requires Redis Cluster")

	config := DefaultRedisClusterConfig()
	config.Addrs = []string{"localhost:7000"}

	backend, err := NewRedisClusterBackend(config)
	if err != nil {
		t.Fatalf("Failed to create cluster backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()

	err = backend.Set(ctx, "nil-key", nil, 5*time.Minute)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	_, found, err := backend.Get(ctx, "nil-key")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if found {
		t.Error("Nil marker should be treated as not found")
	}
}

func TestRedisClusterBackend_SetWithJitter(t *testing.T) {
	t.Skip("Skipping test that requires Redis Cluster")

	config := DefaultRedisClusterConfig()
	config.Addrs = []string{"localhost:7000"}

	backend, err := NewRedisClusterBackend(config)
	if err != nil {
		t.Fatalf("Failed to create cluster backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()

	err = backend.SetWithJitter(ctx, "jitter-key", "value", 5*time.Minute, 0.1)
	if err != nil {
		t.Errorf("SetWithJitter failed: %v", err)
	}

	val, found, err := backend.Get(ctx, "jitter-key")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if !found {
		t.Error("Key should be found")
	}
	if val != "value" {
		t.Errorf("Expected value, got %v", val)
	}
}

func TestRedisClusterBackend_Stats(t *testing.T) {
	t.Skip("Skipping test that requires Redis Cluster")

	config := DefaultRedisClusterConfig()
	config.Addrs = []string{"localhost:7000"}

	backend, err := NewRedisClusterBackend(config)
	if err != nil {
		t.Fatalf("Failed to create cluster backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()

	backend.Set(ctx, "key1", "value1", 5*time.Minute)
	backend.Set(ctx, "key2", "value2", 5*time.Minute)
	backend.Get(ctx, "key1")
	backend.Get(ctx, "key2")
	backend.Get(ctx, "non-existent")
	backend.Delete(ctx, "key1")

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
	if stats.Deletes != 1 {
		t.Errorf("Expected 1 delete, got %d", stats.Deletes)
	}
}

func TestRedisClusterBackend_Ping(t *testing.T) {
	t.Skip("Skipping test that requires Redis Cluster")

	config := DefaultRedisClusterConfig()
	config.Addrs = []string{"localhost:7000"}

	backend, err := NewRedisClusterBackend(config)
	if err != nil {
		t.Fatalf("Failed to create cluster backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()

	err = backend.Ping(ctx)
	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestRedisClusterBackend_Close(t *testing.T) {
	t.Skip("Skipping test that requires Redis Cluster")

	config := DefaultRedisClusterConfig()
	config.Addrs = []string{"localhost:7000"}

	backend, err := NewRedisClusterBackend(config)
	if err != nil {
		t.Fatalf("Failed to create cluster backend: %v", err)
	}

	err = backend.Close()
	if err != nil {
		t.Errorf("First close failed: %v", err)
	}

	err = backend.Close()
	if err != nil {
		t.Errorf("Second close should be no-op: %v", err)
	}

	ctx := context.Background()
	err = backend.Set(ctx, "key", "value", 5*time.Minute)
	if err == nil {
		t.Error("Set after close should fail")
	}
}

func TestRedisClusterBackend_Prefix(t *testing.T) {
	t.Skip("Skipping test that requires Redis Cluster")

	config := DefaultRedisClusterConfig()
	config.Addrs = []string{"localhost:7000"}
	config.Prefix = "test-prefix"

	backend, err := NewRedisClusterBackend(config)
	if err != nil {
		t.Fatalf("Failed to create cluster backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()

	err = backend.Set(ctx, "mykey", "myvalue", 5*time.Minute)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	val, found, err := backend.Get(ctx, "mykey")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if !found {
		t.Error("Key should be found")
	}
	if val != "myvalue" {
		t.Errorf("Expected myvalue, got %v", val)
	}
}

func TestDefaultRedisClusterConfig(t *testing.T) {
	config := DefaultRedisClusterConfig()

	if len(config.Addrs) != 3 {
		t.Errorf("Expected 3 default addresses, got %d", len(config.Addrs))
	}
	if config.DefaultTTL != 30*time.Minute {
		t.Errorf("Expected default TTL 30m, got %v", config.DefaultTTL)
	}
	if config.MaxTTL != 24*time.Hour {
		t.Errorf("Expected max TTL 24h, got %v", config.MaxTTL)
	}
	if config.PoolSize != 10 {
		t.Errorf("Expected pool size 10, got %d", config.PoolSize)
	}
}
