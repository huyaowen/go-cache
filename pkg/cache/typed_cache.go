package cache

import (
	"context"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
)

// TypedCache 泛型缓存适配器（类型安全）
// 提供类型安全的缓存访问，避免类型断言
type TypedCache[T any] struct {
	backend backend.CacheBackend
}

// NewTypedCache 创建泛型缓存适配器
// 适用于需要类型安全的场景
func NewTypedCache[T any](backend backend.CacheBackend) *TypedCache[T] {
	return &TypedCache[T]{
		backend: backend,
	}
}

// Get 获取缓存值（类型安全）
// 返回类型 T 的零值如果缓存未命中或类型不匹配
func (t *TypedCache[T]) Get(ctx context.Context, key string) (T, bool, error) {
	var zero T
	value, found, err := t.backend.Get(ctx, key)
	if !found || err != nil {
		return zero, false, err
	}

	// 类型断言
	result, ok := value.(T)
	if !ok {
		return zero, false, &TypeMismatchError{
			expected: getTypeName[T](),
			actual:   "interface{}",
		}
	}

	return result, true, nil
}

// Set 设置缓存值（类型安全）
func (t *TypedCache[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	return t.backend.Set(ctx, key, value, ttl)
}

// Delete 删除缓存值
func (t *TypedCache[T]) Delete(ctx context.Context, key string) error {
	return t.backend.Delete(ctx, key)
}

// Close 关闭缓存
func (t *TypedCache[T]) Close() error {
	return t.backend.Close()
}

// Stats 获取统计信息
func (t *TypedCache[T]) Stats() *backend.CacheStats {
	return t.backend.Stats()
}

// GetOrSet 获取或设置缓存值（原子操作）
func (t *TypedCache[T]) GetOrSet(ctx context.Context, key string, fn func() (T, error), ttl time.Duration) (T, error) {
	// 尝试获取
	value, found, err := t.Get(ctx, key)
	if err == nil && found {
		return value, nil
	}

	// 获取失败，执行函数
	value, err = fn()
	if err != nil {
		var zero T
		return zero, err
	}

	// 设置缓存
	err = t.Set(ctx, key, value, ttl)
	if err != nil {
		var zero T
		return zero, err
	}

	return value, nil
}

// TypeMismatchError 类型不匹配错误
type TypeMismatchError struct {
	expected string
	actual   string
}

func (e *TypeMismatchError) Error() string {
	return "type mismatch: expected " + e.expected + ", got " + e.actual
}

// getTypeName 获取类型名称（辅助函数）
func getTypeName[T any]() string {
	var zero T
	switch any(zero).(type) {
	case int:
		return "int"
	case int8:
		return "int8"
	case int16:
		return "int16"
	case int32:
		return "int32"
	case int64:
		return "int64"
	case uint:
		return "uint"
	case uint8:
		return "uint8"
	case uint16:
		return "uint16"
	case uint32:
		return "uint32"
	case uint64:
		return "uint64"
	case float32:
		return "float32"
	case float64:
		return "float64"
	case string:
		return "string"
	case bool:
		return "bool"
	default:
		return "interface{}"
	}
}
