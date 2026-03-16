package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TracedCache 带 OpenTelemetry 追踪的缓存包装器
// 自动记录缓存操作的追踪信息
type TracedCache struct {
	backend   backend.CacheBackend
	tracer    trace.Tracer
	cacheName string
}

// NewTracedCache 创建带追踪的缓存包装器
// 使用全局 OpenTelemetry TracerProvider
func NewTracedCache(backend backend.CacheBackend, cacheName string) *TracedCache {
	return &TracedCache{
		backend:   backend,
		tracer:    otel.GetTracerProvider().Tracer("go-cache"),
		cacheName: cacheName,
	}
}

// NewTracedCacheWithTracer 使用自定义 Tracer 创建追踪缓存
func NewTracedCacheWithTracer(backend backend.CacheBackend, tracer trace.Tracer, cacheName string) *TracedCache {
	return &TracedCache{
		backend:   backend,
		tracer:    tracer,
		cacheName: cacheName,
	}
}

// Get 获取缓存值并记录追踪
func (t *TracedCache) Get(ctx context.Context, key string) (interface{}, bool, error) {
	ctx, span := t.tracer.Start(ctx, "cache.get",
		trace.WithAttributes(
			attribute.String("cache.name", t.cacheName),
			attribute.String("cache.key", key),
		),
	)
	defer span.End()

	value, found, err := t.backend.Get(ctx, key)

	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("cache.error", true))
	} else {
		if found {
			span.SetAttributes(attribute.Bool("cache.hit", true))
		} else {
			span.SetAttributes(attribute.Bool("cache.hit", false))
		}
	}

	return value, found, err
}

// Set 设置缓存值并记录追踪
func (t *TracedCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	ctx, span := t.tracer.Start(ctx, "cache.set",
		trace.WithAttributes(
			attribute.String("cache.name", t.cacheName),
			attribute.String("cache.key", key),
			attribute.Float64("cache.ttl_seconds", ttl.Seconds()),
		),
	)
	defer span.End()

	err := t.backend.Set(ctx, key, value, ttl)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("cache.error", true))
	}

	return err
}

// Delete 删除缓存值并记录追踪
func (t *TracedCache) Delete(ctx context.Context, key string) error {
	ctx, span := t.tracer.Start(ctx, "cache.delete",
		trace.WithAttributes(
			attribute.String("cache.name", t.cacheName),
			attribute.String("cache.key", key),
		),
	)
	defer span.End()

	err := t.backend.Delete(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("cache.error", true))
	}

	return err
}

// Close 关闭缓存
func (t *TracedCache) Close() error {
	return t.backend.Close()
}

// Stats 获取统计信息
func (t *TracedCache) Stats() *backend.CacheStats {
	return t.backend.Stats()
}

// GetTracer 获取 Tracer（用于自定义追踪）
func (t *TracedCache) GetTracer() trace.Tracer {
	return t.tracer
}

// CacheOperation 缓存操作类型
type CacheOperation string

const (
	OperationGet    CacheOperation = "get"
	OperationSet    CacheOperation = "set"
	OperationDelete CacheOperation = "delete"
)

// TraceConfig 追踪配置
type TraceConfig struct {
	// 是否启用追踪
	Enabled bool
	// 是否记录详细属性（包括缓存值）
	Verbose bool
	// 是否仅记录错误
	ErrorsOnly bool
}

// DefaultTraceConfig 默认追踪配置
func DefaultTraceConfig() *TraceConfig {
	return &TraceConfig{
		Enabled:   true,
		Verbose:   false,
		ErrorsOnly: false,
	}
}

// TracedCacheWithConfig 带配置的可追踪缓存
type TracedCacheWithConfig struct {
	backend backend.CacheBackend
	tracer  trace.Tracer
	config  *TraceConfig
	name    string
}

// NewTracedCacheWithConfig 创建带配置的追踪缓存
func NewTracedCacheWithConfig(backend backend.CacheBackend, name string, config *TraceConfig) *TracedCacheWithConfig {
	if config == nil {
		config = DefaultTraceConfig()
	}
	return &TracedCacheWithConfig{
		backend: backend,
		tracer:  otel.GetTracerProvider().Tracer("go-cache"),
		config:  config,
		name:    name,
	}
}

// Get 获取缓存值
func (t *TracedCacheWithConfig) Get(ctx context.Context, key string) (interface{}, bool, error) {
	if !t.config.Enabled {
		return t.backend.Get(ctx, key)
	}

	if t.config.ErrorsOnly {
		// 仅记录错误模式
		value, found, err := t.backend.Get(ctx, key)
		if err != nil {
			_, span := t.tracer.Start(ctx, "cache.get.error")
			span.RecordError(err)
			span.End()
		}
		return value, found, err
	}

	// 完整追踪模式
	ctx, span := t.tracer.Start(ctx, "cache.get",
		trace.WithAttributes(
			attribute.String("cache.name", t.name),
			attribute.String("cache.key", key),
		),
	)
	defer span.End()

	value, found, err := t.backend.Get(ctx, key)

	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("cache.error", true))
	} else {
		span.SetAttributes(attribute.Bool("cache.hit", found))
		if t.config.Verbose && found {
			span.SetAttributes(attribute.String("cache.value", fmt.Sprintf("%v", value)))
		}
	}

	return value, found, err
}

// Set 设置缓存值
func (t *TracedCacheWithConfig) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !t.config.Enabled {
		return t.backend.Set(ctx, key, value, ttl)
	}

	if t.config.ErrorsOnly {
		err := t.backend.Set(ctx, key, value, ttl)
		if err != nil {
			_, span := t.tracer.Start(ctx, "cache.set.error")
			span.RecordError(err)
			span.End()
		}
		return err
	}

	ctx, span := t.tracer.Start(ctx, "cache.set",
		trace.WithAttributes(
			attribute.String("cache.name", t.name),
			attribute.String("cache.key", key),
			attribute.Float64("cache.ttl_seconds", ttl.Seconds()),
		),
	)
	defer span.End()

	err := t.backend.Set(ctx, key, value, ttl)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("cache.error", true))
	}

	if t.config.Verbose {
		span.SetAttributes(attribute.String("cache.value", fmt.Sprintf("%v", value)))
	}

	return err
}

// Delete 删除缓存值
func (t *TracedCacheWithConfig) Delete(ctx context.Context, key string) error {
	if !t.config.Enabled {
		return t.backend.Delete(ctx, key)
	}

	if t.config.ErrorsOnly {
		err := t.backend.Delete(ctx, key)
		if err != nil {
			_, span := t.tracer.Start(ctx, "cache.delete.error")
			span.RecordError(err)
			span.End()
		}
		return err
	}

	ctx, span := t.tracer.Start(ctx, "cache.delete",
		trace.WithAttributes(
			attribute.String("cache.name", t.name),
			attribute.String("cache.key", key),
		),
	)
	defer span.End()

	err := t.backend.Delete(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("cache.error", true))
	}

	return err
}

// Close 关闭缓存
func (t *TracedCacheWithConfig) Close() error {
	return t.backend.Close()
}

// Stats 获取统计信息
func (t *TracedCacheWithConfig) Stats() *backend.CacheStats {
	return t.backend.Stats()
}
