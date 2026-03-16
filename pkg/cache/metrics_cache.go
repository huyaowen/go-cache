package cache

import (
	"context"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// cacheHits 缓存命中次数
	cacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_name"},
	)

	// cacheMisses 缓存未命中次数
	cacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_name"},
	)

	// cacheSets 缓存设置次数
	cacheSets = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_cache_sets_total",
			Help: "Total number of cache sets",
		},
		[]string{"cache_name"},
	)

	// cacheDeletes 缓存删除次数
	cacheDeletes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_cache_deletes_total",
			Help: "Total number of cache deletes",
		},
		[]string{"cache_name"},
	)

	// cacheErrors 缓存错误次数
	cacheErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_cache_errors_total",
			Help: "Total number of cache errors",
		},
		[]string{"cache_name", "operation"},
	)

	// cacheOperationDuration 缓存操作耗时
	cacheOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "go_cache_operation_duration_seconds",
			Help:    "Duration of cache operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"cache_name", "operation"},
	)

	// cacheSize 缓存大小
	cacheSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "go_cache_size",
			Help: "Current size of the cache",
		},
		[]string{"cache_name"},
	)
)

// init 注册 Prometheus 指标
func init() {
	prometheus.MustRegister(cacheHits)
	prometheus.MustRegister(cacheMisses)
	prometheus.MustRegister(cacheSets)
	prometheus.MustRegister(cacheDeletes)
	prometheus.MustRegister(cacheErrors)
	prometheus.MustRegister(cacheOperationDuration)
	prometheus.MustRegister(cacheSize)
}

// MetricsCache 带 Prometheus 指标的缓存包装器
type MetricsCache struct {
	backend   backend.CacheBackend
	cacheName string
}

// NewMetricsCache 创建带指标的缓存包装器
func NewMetricsCache(backend backend.CacheBackend, cacheName string) *MetricsCache {
	return &MetricsCache{
		backend:   backend,
		cacheName: cacheName,
	}
}

// Get 获取缓存值并记录指标
func (m *MetricsCache) Get(ctx context.Context, key string) (interface{}, bool, error) {
	start := time.Now()
	value, found, err := m.backend.Get(ctx, key)
	duration := time.Since(start).Seconds()

	// 记录操作耗时
	cacheOperationDuration.WithLabelValues(m.cacheName, "get").Observe(duration)

	if err != nil {
		cacheErrors.WithLabelValues(m.cacheName, "get").Inc()
	} else {
		if found {
			cacheHits.WithLabelValues(m.cacheName).Inc()
		} else {
			cacheMisses.WithLabelValues(m.cacheName).Inc()
		}
	}

	return value, found, err
}

// Set 设置缓存值并记录指标
func (m *MetricsCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	start := time.Now()
	err := m.backend.Set(ctx, key, value, ttl)
	duration := time.Since(start).Seconds()

	cacheOperationDuration.WithLabelValues(m.cacheName, "set").Observe(duration)

	if err != nil {
		cacheErrors.WithLabelValues(m.cacheName, "set").Inc()
	} else {
		cacheSets.WithLabelValues(m.cacheName).Inc()
	}

	return err
}

// Delete 删除缓存值并记录指标
func (m *MetricsCache) Delete(ctx context.Context, key string) error {
	start := time.Now()
	err := m.backend.Delete(ctx, key)
	duration := time.Since(start).Seconds()

	cacheOperationDuration.WithLabelValues(m.cacheName, "delete").Observe(duration)

	if err != nil {
		cacheErrors.WithLabelValues(m.cacheName, "delete").Inc()
	} else {
		cacheDeletes.WithLabelValues(m.cacheName).Inc()
	}

	return err
}

// Close 关闭缓存
func (m *MetricsCache) Close() error {
	return m.backend.Close()
}

// Stats 获取统计信息并更新指标
func (m *MetricsCache) Stats() *backend.CacheStats {
	stats := m.backend.Stats()
	
	// 更新缓存大小指标
	cacheSize.WithLabelValues(m.cacheName).Set(float64(stats.Size))
	
	return stats
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	// 是否启用指标
	Enabled bool
	// 是否记录详细指标（包括操作耗时）
	EnableHistograms bool
	// 缓存名称前缀
	NamePrefix string
}

// DefaultMetricsConfig 默认指标配置
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled:          true,
		EnableHistograms: true,
		NamePrefix:       "",
	}
}

// MetricsCacheWithConfig 带配置的指标缓存
type MetricsCacheWithConfig struct {
	backend backend.CacheBackend
	config  *MetricsConfig
	name    string
}

// NewMetricsCacheWithConfig 创建带配置的指标缓存
func NewMetricsCacheWithConfig(backend backend.CacheBackend, name string, config *MetricsConfig) *MetricsCacheWithConfig {
	if config == nil {
		config = DefaultMetricsConfig()
	}
	return &MetricsCacheWithConfig{
		backend: backend,
		config:  config,
		name:    config.NamePrefix + name,
	}
}

// Get 获取缓存值
func (m *MetricsCacheWithConfig) Get(ctx context.Context, key string) (interface{}, bool, error) {
	if !m.config.Enabled {
		return m.backend.Get(ctx, key)
	}

	var value interface{}
	var found bool
	var err error

	if m.config.EnableHistograms {
		start := time.Now()
		value, found, err = m.backend.Get(ctx, key)
		duration := time.Since(start).Seconds()
		cacheOperationDuration.WithLabelValues(m.name, "get").Observe(duration)
	} else {
		value, found, err = m.backend.Get(ctx, key)
	}

	if err != nil {
		cacheErrors.WithLabelValues(m.name, "get").Inc()
	} else {
		if found {
			cacheHits.WithLabelValues(m.name).Inc()
		} else {
			cacheMisses.WithLabelValues(m.name).Inc()
		}
	}

	return value, found, err
}

// Set 设置缓存值
func (m *MetricsCacheWithConfig) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !m.config.Enabled {
		return m.backend.Set(ctx, key, value, ttl)
	}

	var err error
	if m.config.EnableHistograms {
		start := time.Now()
		err = m.backend.Set(ctx, key, value, ttl)
		duration := time.Since(start).Seconds()
		cacheOperationDuration.WithLabelValues(m.name, "set").Observe(duration)
	} else {
		err = m.backend.Set(ctx, key, value, ttl)
	}

	if err != nil {
		cacheErrors.WithLabelValues(m.name, "set").Inc()
	} else {
		cacheSets.WithLabelValues(m.name).Inc()
	}

	return err
}

// Delete 删除缓存值
func (m *MetricsCacheWithConfig) Delete(ctx context.Context, key string) error {
	if !m.config.Enabled {
		return m.backend.Delete(ctx, key)
	}

	var err error
	if m.config.EnableHistograms {
		start := time.Now()
		err = m.backend.Delete(ctx, key)
		duration := time.Since(start).Seconds()
		cacheOperationDuration.WithLabelValues(m.name, "delete").Observe(duration)
	} else {
		err = m.backend.Delete(ctx, key)
	}

	if err != nil {
		cacheErrors.WithLabelValues(m.name, "delete").Inc()
	} else {
		cacheDeletes.WithLabelValues(m.name).Inc()
	}

	return err
}

// Close 关闭缓存
func (m *MetricsCacheWithConfig) Close() error {
	return m.backend.Close()
}

// Stats 获取统计信息
func (m *MetricsCacheWithConfig) Stats() *backend.CacheStats {
	stats := m.backend.Stats()
	cacheSize.WithLabelValues(m.name).Set(float64(stats.Size))
	return stats
}

// UnregisterMetrics 注销指定缓存的指标（用于清理）
func UnregisterMetrics(cacheName string) {
	cacheHits.DeleteLabelValues(cacheName)
	cacheMisses.DeleteLabelValues(cacheName)
	cacheSets.DeleteLabelValues(cacheName)
	cacheDeletes.DeleteLabelValues(cacheName)
	cacheErrors.DeleteLabelValues(cacheName, "get")
	cacheErrors.DeleteLabelValues(cacheName, "set")
	cacheErrors.DeleteLabelValues(cacheName, "delete")
	cacheSize.DeleteLabelValues(cacheName)
}

// GetAllMetrics 获取所有指标的当前值（用于调试）
func GetAllMetrics(cacheName string) map[string]interface{} {
	return map[string]interface{}{
		"hits":     cacheHits.WithLabelValues(cacheName),
		"misses":   cacheMisses.WithLabelValues(cacheName),
		"sets":     cacheSets.WithLabelValues(cacheName),
		"deletes":  cacheDeletes.WithLabelValues(cacheName),
		"errors":   cacheErrors.WithLabelValues(cacheName, "get"),
		"size":     cacheSize.WithLabelValues(cacheName),
	}
}
