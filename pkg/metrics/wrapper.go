package metrics

import (
	"context"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
)

// MetricsCacheBackend 带指标统计的缓存包装器
type MetricsCacheBackend struct {
	backend     backend.CacheBackend
	exporter    *PrometheusExporter
	cacheName   string
	backendName string
}

// NewMetricsCacheBackend 创建带指标的缓存包装器
func NewMetricsCacheBackend(backend backend.CacheBackend, exporter *PrometheusExporter, cacheName, backendName string) *MetricsCacheBackend {
	return &MetricsCacheBackend{
		backend:     backend,
		exporter:    exporter,
		cacheName:   cacheName,
		backendName: backendName,
	}
}

// Get 获取缓存值并记录指标
func (m *MetricsCacheBackend) Get(ctx context.Context, key string) (interface{}, bool, error) {
	start := time.Now()
	value, found, err := m.backend.Get(ctx, key)
	duration := time.Since(start)

	m.exporter.RecordLatency(m.cacheName, m.backendName, "get", duration)
	if found {
		m.exporter.RecordHit(m.cacheName, m.backendName)
	} else {
		m.exporter.RecordMiss(m.cacheName, m.backendName)
	}

	return value, found, err
}

// Set 设置缓存值并记录指标
func (m *MetricsCacheBackend) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	start := time.Now()
	err := m.backend.Set(ctx, key, value, ttl)
	duration := time.Since(start)

	m.exporter.RecordLatency(m.cacheName, m.backendName, "set", duration)
	m.exporter.RecordSet(m.cacheName, m.backendName)

	return err
}

// Delete 删除缓存值并记录指标
func (m *MetricsCacheBackend) Delete(ctx context.Context, key string) error {
	start := time.Now()
	err := m.backend.Delete(ctx, key)
	duration := time.Since(start)

	m.exporter.RecordLatency(m.cacheName, m.backendName, "delete", duration)
	m.exporter.RecordDelete(m.cacheName, m.backendName)

	return err
}

// Close 关闭缓存
func (m *MetricsCacheBackend) Close() error {
	return m.backend.Close()
}

// Stats 获取统计信息
func (m *MetricsCacheBackend) Stats() *backend.CacheStats {
	return m.backend.Stats()
}
