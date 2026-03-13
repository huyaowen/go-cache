package tracing

import (
	"context"
	"testing"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// setupTestTracer 设置测试用 Tracer
func setupTestTracer(t *testing.T) (func(), error) {
	// 创建简单的 exporter 用于测试
	exporter, err := stdouttrace.New()
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)

	return func() {
		tp.Shutdown(context.Background())
		exporter.Shutdown(context.Background())
	}, nil
}

func TestCacheTracer(t *testing.T) {
	tracer := NewCacheTracer()
	if tracer == nil {
		t.Fatal("Failed to create CacheTracer")
	}

	ctx := context.Background()

	// 测试 StartGetSpan
	ctx, span := tracer.StartGetSpan(ctx, "users", "user:1")
	if span == nil {
		t.Error("Expected non-nil span")
	}
	tracer.RecordCacheHit(span)
	tracer.EndSpan(span)

	// 测试 StartSetSpan
	ctx, span = tracer.StartSetSpan(ctx, "users", "user:1", 300)
	if span == nil {
		t.Error("Expected non-nil span")
	}
	tracer.EndSpan(span)

	// 测试 StartDeleteSpan
	ctx, span = tracer.StartDeleteSpan(ctx, "users", "user:1")
	if span == nil {
		t.Error("Expected non-nil span")
	}
	tracer.EndSpan(span)
}

func TestTracedCacheBackend(t *testing.T) {
	// 设置测试 tracer
	cleanup, err := setupTestTracer(t)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer cleanup()

	tracer := NewCacheTracer()
	memBackend, err := backend.NewMemoryBackend(backend.DefaultCacheConfig("test"))
	if err != nil {
		t.Fatalf("Failed to create memory backend: %v", err)
	}
	tracedBackend := NewTracedCacheBackend(memBackend, tracer, "test")

	ctx := context.Background()

	// 测试 Get（未命中）
	_, found, err := tracedBackend.Get(ctx, "nonexistent")
	if found {
		t.Error("Expected miss")
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// 测试 Set
	err = tracedBackend.Set(ctx, "key1", "value1", time.Minute)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	// 测试 Get（命中）
	value, found, err := tracedBackend.Get(ctx, "key1")
	if !found {
		t.Error("Expected hit")
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}

	// 测试 Delete
	err = tracedBackend.Delete(ctx, "key1")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}
}

func TestTracerAttributes(t *testing.T) {
	tracer := NewCacheTracer()
	ctx := context.Background()

	// 测试 Execute Span
	ctx, span := tracer.StartExecuteSpan(ctx, "users", "GetUser", "#id")
	if span == nil {
		t.Error("Expected non-nil span")
	}
	tracer.EndSpan(span)

	// 测试错误记录
	ctx, span = tracer.StartGetSpan(ctx, "users", "key1")
	testErr := &testError{msg: "test error"}
	tracer.RecordError(span, testErr)
	tracer.EndSpan(span)
}

func TestCacheSpanContext(t *testing.T) {
	tracer := NewCacheTracer()
	ctx := context.Background()

	// 测试上下文传递
	ctx = tracer.WithCacheSpan(ctx, "users")
	cacheName, ok := tracer.GetCacheNameFromContext(ctx)
	if !ok {
		t.Error("Expected to retrieve cache name from context")
	}
	if cacheName != "users" {
		t.Errorf("Expected 'users', got '%s'", cacheName)
	}
}

// testError 测试用错误
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
