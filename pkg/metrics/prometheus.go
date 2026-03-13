package metrics

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// globalRegistry 全局注册表（用于单例模式）
var (
	globalRegistry prometheus.Registerer = prometheus.DefaultRegisterer
	globalGatherer prometheus.Gatherer   = prometheus.DefaultGatherer
)

// PrometheusExporter Prometheus 指标导出器
type PrometheusExporter struct {
	mu        sync.RWMutex
	hits      *prometheus.CounterVec
	misses    *prometheus.CounterVec
	latency   *prometheus.HistogramVec
	evictions *prometheus.CounterVec
	sets      *prometheus.CounterVec
	deletes   *prometheus.CounterVec
}

// NewPrometheusExporter 创建 Prometheus 导出器
func NewPrometheusExporter() *PrometheusExporter {
	return NewPrometheusExporterWithRegistry(prometheus.DefaultRegisterer)
}

// NewPrometheusExporterWithRegistry 使用自定义注册表创建导出器（用于测试）
func NewPrometheusExporterWithRegistry(reg prometheus.Registerer) *PrometheusExporter {
	e := &PrometheusExporter{
		hits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:      "go_cache_hits_total",
				Help:      "Total number of cache hits",
				Namespace: "go_cache",
			},
			[]string{"cache", "backend"},
		),
		misses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:      "go_cache_misses_total",
				Help:      "Total number of cache misses",
				Namespace: "go_cache",
			},
			[]string{"cache", "backend"},
		),
		latency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:      "go_cache_latency_seconds",
				Help:      "Cache operation latency in seconds",
				Namespace: "go_cache",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"cache", "backend", "operation"},
		),
		evictions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:      "go_cache_evictions_total",
				Help:      "Total number of cache evictions",
				Namespace: "go_cache",
			},
			[]string{"cache", "backend"},
		),
		sets: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:      "go_cache_sets_total",
				Help:      "Total number of cache sets",
				Namespace: "go_cache",
			},
			[]string{"cache", "backend"},
		),
		deletes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:      "go_cache_deletes_total",
				Help:      "Total number of cache deletes",
				Namespace: "go_cache",
			},
			[]string{"cache", "backend"},
		),
	}

	// 注册所有指标
	reg.MustRegister(e.hits)
	reg.MustRegister(e.misses)
	reg.MustRegister(e.latency)
	reg.MustRegister(e.evictions)
	reg.MustRegister(e.sets)
	reg.MustRegister(e.deletes)

	return e
}

// RecordHit 记录命中
func (e *PrometheusExporter) RecordHit(cacheName, backend string) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	e.hits.WithLabelValues(cacheName, backend).Inc()
}

// RecordMiss 记录未命中
func (e *PrometheusExporter) RecordMiss(cacheName, backend string) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	e.misses.WithLabelValues(cacheName, backend).Inc()
}

// RecordLatency 记录延迟
func (e *PrometheusExporter) RecordLatency(cacheName, backend, operation string, duration time.Duration) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	e.latency.WithLabelValues(cacheName, backend, operation).Observe(duration.Seconds())
}

// RecordEviction 记录驱逐
func (e *PrometheusExporter) RecordEviction(cacheName, backend string) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	e.evictions.WithLabelValues(cacheName, backend).Inc()
}

// RecordSet 记录设置操作
func (e *PrometheusExporter) RecordSet(cacheName, backend string) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	e.sets.WithLabelValues(cacheName, backend).Inc()
}

// RecordDelete 记录删除操作
func (e *PrometheusExporter) RecordDelete(cacheName, backend string) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	e.deletes.WithLabelValues(cacheName, backend).Inc()
}

// ServeHTTP HTTP 处理函数，暴露 /metrics 端点
func (e *PrometheusExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}

// StartMetricsServer 启动指标 HTTP 服务器
func (e *PrometheusExporter) StartMetricsServer(addr string) error {
	return http.ListenAndServe(addr, e)
}

// GetCollector 获取收集器（用于测试）
func (e *PrometheusExporter) GetCollector(name string) prometheus.Collector {
	switch name {
	case "hits":
		return e.hits
	case "misses":
		return e.misses
	case "latency":
		return e.latency
	case "evictions":
		return e.evictions
	case "sets":
		return e.sets
	case "deletes":
		return e.deletes
	default:
		return nil
	}
}
