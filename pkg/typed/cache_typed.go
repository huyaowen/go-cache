package typed

import (
	"context"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
)

// TypedCacheBackend 泛型缓存后端接口
type TypedCacheBackend[T any] interface {
	Get(ctx context.Context, key string) (T, bool, error)
	Set(ctx context.Context, key string, value T, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Close() error
	Stats() *backend.CacheStats
}

// TypedCache 泛型缓存适配器
type TypedCache[T any] struct {
	backend backend.CacheBackend
}

// NewTypedCache 创建泛型缓存适配器
func NewTypedCache[T any](backend backend.CacheBackend) *TypedCache[T] {
	return &TypedCache[T]{
		backend: backend,
	}
}

// Get 获取缓存值（类型安全）
func (t *TypedCache[T]) Get(ctx context.Context, key string) (T, bool, error) {
	var zero T
	value, found, err := t.backend.Get(ctx, key)
	if !found {
		return zero, false, err
	}
	if err != nil {
		return zero, false, err
	}

	// 类型断言
	result, ok := value.(T)
	if !ok {
		return zero, false, ErrTypeMismatch
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

// ErrTypeMismatch 类型不匹配错误
var ErrTypeMismatch = &TypeMismatchError{
	message: "cached value type does not match expected type",
}

// TypeMismatchError 类型不匹配错误
type TypeMismatchError struct {
	message string
}

func (e *TypeMismatchError) Error() string {
	return e.message
}
