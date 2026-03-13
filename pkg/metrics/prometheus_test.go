package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestPrometheusExporter(t *testing.T) {
	reg := prometheus.NewRegistry()
	exporter := NewPrometheusExporterWithRegistry(reg)
	if exporter == nil {
		t.Fatal("Failed to create PrometheusExporter")
	}

	// 测试记录命中
	exporter.RecordHit("users", "memory")
	exporter.RecordHit("users", "memory")

	// 测试记录未命中
	exporter.RecordMiss("users", "memory")

	// 测试记录延迟
	exporter.RecordLatency("users", "memory", "get", 100*time.Millisecond)

	// 测试记录驱逐
	exporter.RecordEviction("users", "memory")

	// 测试记录设置
	exporter.RecordSet("users", "memory")

	// 测试记录删除
	exporter.RecordDelete("users", "memory")

	// 验证指标已注册
	if exporter.hits == nil || exporter.misses == nil || exporter.latency == nil {
		t.Error("Metrics not properly initialized")
	}
}

func TestMetricsCacheBackend(t *testing.T) {
	reg := prometheus.NewRegistry()
	exporter := NewPrometheusExporterWithRegistry(reg)
	memBackend, err := backend.NewMemoryBackend(backend.DefaultCacheConfig("test"))
	if err != nil {
		t.Fatalf("Failed to create memory backend: %v", err)
	}
	metricsBackend := NewMetricsCacheBackend(memBackend, exporter, "test", "memory")

	ctx := context.Background()

	// 测试 Get（未命中）
	_, found, err := metricsBackend.Get(ctx, "nonexistent")
	if found {
		t.Error("Expected miss for nonexistent key")
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// 测试 Set
	err = metricsBackend.Set(ctx, "key1", "value1", time.Minute)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	// 测试 Get（命中）
	value, found, err := metricsBackend.Get(ctx, "key1")
	if !found {
		t.Error("Expected hit for existing key")
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}

	// 测试 Delete
	err = metricsBackend.Delete(ctx, "key1")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	// 验证指标
	hitsCount := testutil.ToFloat64(exporter.hits.WithLabelValues("test", "memory"))
	if hitsCount < 1 {
		t.Errorf("Expected at least 1 hit, got %f", hitsCount)
	}
}

func TestMetricsWrapper(t *testing.T) {
	reg := prometheus.NewRegistry()
	exporter := NewPrometheusExporterWithRegistry(reg)
	config := backend.DefaultCacheConfig("users")
	memBackend, err := backend.NewMemoryBackend(config)
	if err != nil {
		t.Fatalf("Failed to create memory backend: %v", err)
	}
	wrapped := NewMetricsCacheBackend(memBackend, exporter, "users", "memory")

	ctx := context.Background()

	// 多次操作
	for i := 0; i < 10; i++ {
		key := "key" + string(rune('0'+i))
		wrapped.Set(ctx, key, i, time.Minute)
		wrapped.Get(ctx, key)
	}

	// 验证指标增长
	hitsCount := testutil.ToFloat64(exporter.hits.WithLabelValues("users", "memory"))
	setsCount := testutil.ToFloat64(exporter.sets.WithLabelValues("users", "memory"))

	if setsCount != 10 {
		t.Errorf("Expected 10 sets, got %f", setsCount)
	}
	if hitsCount != 10 {
		t.Errorf("Expected 10 hits, got %f", hitsCount)
	}
}
