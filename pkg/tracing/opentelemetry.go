package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	// TracerName OpenTelemetry Tracer 名称
	TracerName = "go-cache"
)

// CacheTracer 缓存追踪器
type CacheTracer struct {
	tracer trace.Tracer
}

// NewCacheTracer 创建缓存追踪器
func NewCacheTracer() *CacheTracer {
	return &CacheTracer{
		tracer: otel.Tracer(TracerName),
	}
}

// StartSpan 开始追踪 Span
func (ct *CacheTracer) StartSpan(ctx context.Context, operation, cacheName string) (context.Context, trace.Span) {
	ctx, span := ct.tracer.Start(ctx, fmt.Sprintf("cache.%s", operation))
	span.SetAttributes(
		attribute.String("cache.name", cacheName),
		attribute.String("cache.operation", operation),
	)
	return ctx, span
}

// StartGetSpan 开始 Get 操作追踪
func (ct *CacheTracer) StartGetSpan(ctx context.Context, cacheName, key string) (context.Context, trace.Span) {
	ctx, span := ct.tracer.Start(ctx, "cache.get")
	span.SetAttributes(
		attribute.String("cache.name", cacheName),
		attribute.String("cache.key", key),
		attribute.String("cache.operation", "get"),
	)
	return ctx, span
}

// StartSetSpan 开始 Set 操作追踪
func (ct *CacheTracer) StartSetSpan(ctx context.Context, cacheName, key string, ttlSeconds float64) (context.Context, trace.Span) {
	ctx, span := ct.tracer.Start(ctx, "cache.set")
	span.SetAttributes(
		attribute.String("cache.name", cacheName),
		attribute.String("cache.key", key),
		attribute.String("cache.operation", "set"),
		attribute.Float64("cache.ttl_seconds", ttlSeconds),
	)
	return ctx, span
}

// StartDeleteSpan 开始 Delete 操作追踪
func (ct *CacheTracer) StartDeleteSpan(ctx context.Context, cacheName, key string) (context.Context, trace.Span) {
	ctx, span := ct.tracer.Start(ctx, "cache.delete")
	span.SetAttributes(
		attribute.String("cache.name", cacheName),
		attribute.String("cache.key", key),
		attribute.String("cache.operation", "delete"),
	)
	return ctx, span
}

// StartExecuteSpan 开始 Execute 操作追踪（用于方法级缓存）
func (ct *CacheTracer) StartExecuteSpan(ctx context.Context, cacheName, methodName, keyExpr string) (context.Context, trace.Span) {
	ctx, span := ct.tracer.Start(ctx, "cache.execute")
	span.SetAttributes(
		attribute.String("cache.name", cacheName),
		attribute.String("cache.method", methodName),
		attribute.String("cache.key_expr", keyExpr),
	)
	return ctx, span
}

// RecordCacheHit 记录缓存命中
func (ct *CacheTracer) RecordCacheHit(span trace.Span) {
	if span != nil && span.IsRecording() {
		span.SetAttributes(attribute.Bool("cache.hit", true))
	}
}

// RecordCacheMiss 记录缓存未命中
func (ct *CacheTracer) RecordCacheMiss(span trace.Span) {
	if span != nil && span.IsRecording() {
		span.SetAttributes(attribute.Bool("cache.hit", false))
	}
}

// RecordError 记录错误
func (ct *CacheTracer) RecordError(span trace.Span, err error) {
	if span != nil && span.IsRecording() && err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("cache.error", true))
	}
}

// EndSpan 结束 Span
func (ct *CacheTracer) EndSpan(span trace.Span) {
	if span != nil {
		span.End()
	}
}

// WithCacheSpan 在上下文中添加缓存 Span 信息
func (ct *CacheTracer) WithCacheSpan(ctx context.Context, cacheName string) context.Context {
	return context.WithValue(ctx, cacheSpanContextKey{}, cacheName)
}

// GetCacheNameFromContext 从上下文获取缓存名称
func (ct *CacheTracer) GetCacheNameFromContext(ctx context.Context) (string, bool) {
	cacheName, ok := ctx.Value(cacheSpanContextKey{}).(string)
	return cacheName, ok
}

type cacheSpanContextKey struct{}
