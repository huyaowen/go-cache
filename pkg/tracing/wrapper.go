package tracing

import (
	"context"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
	"go.opentelemetry.io/otel/trace"
)

// TracedCacheBackend 带追踪的缓存包装器
type TracedCacheBackend struct {
	backend   backend.CacheBackend
	tracer    *CacheTracer
	cacheName string
}

// NewTracedCacheBackend 创建带追踪的缓存包装器
func NewTracedCacheBackend(backend backend.CacheBackend, tracer *CacheTracer, cacheName string) *TracedCacheBackend {
	return &TracedCacheBackend{
		backend:   backend,
		tracer:    tracer,
		cacheName: cacheName,
	}
}

// Get 获取缓存值并记录追踪
func (t *TracedCacheBackend) Get(ctx context.Context, key string) (interface{}, bool, error) {
	ctx, span := t.tracer.StartGetSpan(ctx, t.cacheName, key)
	defer t.tracer.EndSpan(span)

	value, found, err := t.backend.Get(ctx, key)

	if found {
		t.tracer.RecordCacheHit(span)
	} else {
		t.tracer.RecordCacheMiss(span)
	}

	t.tracer.RecordError(span, err)

	return value, found, err
}

// Set 设置缓存值并记录追踪
func (t *TracedCacheBackend) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	ctx, span := t.tracer.StartSetSpan(ctx, t.cacheName, key, ttl.Seconds())
	defer t.tracer.EndSpan(span)

	err := t.backend.Set(ctx, key, value, ttl)
	t.tracer.RecordError(span, err)

	return err
}

// Delete 删除缓存值并记录追踪
func (t *TracedCacheBackend) Delete(ctx context.Context, key string) error {
	ctx, span := t.tracer.StartDeleteSpan(ctx, t.cacheName, key)
	defer t.tracer.EndSpan(span)

	err := t.backend.Delete(ctx, key)
	t.tracer.RecordError(span, err)

	return err
}

// Close 关闭缓存
func (t *TracedCacheBackend) Close() error {
	return t.backend.Close()
}

// Stats 获取统计信息
func (t *TracedCacheBackend) Stats() *backend.CacheStats {
	return t.backend.Stats()
}

// WithTracingSpan 在上下文中添加追踪 Span
func (t *TracedCacheBackend) WithTracingSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	return t.tracer.StartSpan(ctx, operation, t.cacheName)
}
