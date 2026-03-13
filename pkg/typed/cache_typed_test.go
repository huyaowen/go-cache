package typed

import (
	"context"
	"testing"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
)

// User 测试用结构体
type User struct {
	ID   int
	Name string
}

func TestTypedCache_String(t *testing.T) {
	memBackend, err := backend.NewMemoryBackend(backend.DefaultCacheConfig("strings"))
	if err != nil {
		t.Fatalf("Failed to create memory backend: %v", err)
	}
	typedCache := NewTypedCache[string](memBackend)

	ctx := context.Background()

	// 测试 Set
	err = typedCache.Set(ctx, "key1", "value1", time.Minute)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	// 测试 Get（命中）
	value, found, err := typedCache.Get(ctx, "key1")
	if !found {
		t.Error("Expected hit")
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}

	// 测试 Get（未命中）
	_, found, err = typedCache.Get(ctx, "nonexistent")
	if found {
		t.Error("Expected miss")
	}
}

func TestTypedCache_Int(t *testing.T) {
	memBackend, err := backend.NewMemoryBackend(backend.DefaultCacheConfig("ints"))
	if err != nil {
		t.Fatalf("Failed to create memory backend: %v", err)
	}
	typedCache := NewTypedCache[int](memBackend)

	ctx := context.Background()

	// 测试 Set
	err = typedCache.Set(ctx, "num1", 42, time.Minute)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	// 测试 Get
	value, found, err := typedCache.Get(ctx, "num1")
	if !found {
		t.Error("Expected hit")
	}
	if value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}
}

func TestTypedCache_Struct(t *testing.T) {
	memBackend, err := backend.NewMemoryBackend(backend.DefaultCacheConfig("users"))
	if err != nil {
		t.Fatalf("Failed to create memory backend: %v", err)
	}
	typedCache := NewTypedCache[User](memBackend)

	ctx := context.Background()

	user := User{ID: 1, Name: "Alice"}

	// 测试 Set
	err = typedCache.Set(ctx, "user:1", user, time.Minute)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	// 测试 Get
	retrieved, found, err := typedCache.Get(ctx, "user:1")
	if !found {
		t.Error("Expected hit")
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if retrieved.ID != user.ID || retrieved.Name != user.Name {
		t.Errorf("Expected %+v, got %+v", user, retrieved)
	}
}

func TestTypedCache_Slice(t *testing.T) {
	memBackend, err := backend.NewMemoryBackend(backend.DefaultCacheConfig("lists"))
	if err != nil {
		t.Fatalf("Failed to create memory backend: %v", err)
	}
	typedCache := NewTypedCache[[]int](memBackend)

	ctx := context.Background()

	data := []int{1, 2, 3, 4, 5}

	// 测试 Set
	err = typedCache.Set(ctx, "list1", data, time.Minute)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	// 测试 Get
	retrieved, found, err := typedCache.Get(ctx, "list1")
	if !found {
		t.Error("Expected hit")
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(retrieved) != len(data) {
		t.Errorf("Expected length %d, got %d", len(data), len(retrieved))
	}
}

func TestTypedCache_Delete(t *testing.T) {
	memBackend, err := backend.NewMemoryBackend(backend.DefaultCacheConfig("test"))
	if err != nil {
		t.Fatalf("Failed to create memory backend: %v", err)
	}
	typedCache := NewTypedCache[string](memBackend)

	ctx := context.Background()

	// Set + Delete
	typedCache.Set(ctx, "key1", "value1", time.Minute)
	err = typedCache.Delete(ctx, "key1")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	// Verify deletion
	_, found, _ := typedCache.Get(ctx, "key1")
	if found {
		t.Error("Expected miss after delete")
	}
}

func TestTypedCache_Stats(t *testing.T) {
	memBackend, err := backend.NewMemoryBackend(backend.DefaultCacheConfig("test"))
	if err != nil {
		t.Fatalf("Failed to create memory backend: %v", err)
	}
	typedCache := NewTypedCache[string](memBackend)

	ctx := context.Background()

	// 一些操作
	typedCache.Set(ctx, "key1", "value1", time.Minute)
	typedCache.Get(ctx, "key1")
	typedCache.Get(ctx, "nonexistent")

	stats := typedCache.Stats()
	if stats == nil {
		t.Error("Expected non-nil stats")
	}
	if stats.Sets != 1 {
		t.Errorf("Expected 1 set, got %d", stats.Sets)
	}
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
}
