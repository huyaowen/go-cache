package core

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/go-cache-framework/pkg/backend"
	"github.com/go-cache-framework/pkg/spel"
)

// CacheBackend 别名
type CacheBackend = backend.CacheBackend

// CacheStats 别名
type CacheStats = backend.CacheStats

// CacheConfig 别名
type CacheConfig = backend.CacheConfig

// BackendFactory 别名
type BackendFactory = backend.BackendFactory

// DefaultCacheConfig 别名
func DefaultCacheConfig(name string) *CacheConfig {
	return backend.DefaultCacheConfig(name)
}

// StatsCounter 统计计数器
type StatsCounter = backend.StatsCounter

// NewStatsCounter 创建统计计数器
func NewStatsCounter(maxSize int64) *StatsCounter {
	return backend.NewStatsCounter(maxSize)
}

// CacheItem 缓存项
type CacheItem = backend.CacheItem

// MethodMeta 方法元数据
type MethodMeta struct {
	CacheName, KeyExpr, TTLExpr, Condition, Unless, CacheType string
	Sync, Before bool
}

// CacheManager 缓存管理器接口
type CacheManager interface {
	GetCache(name string) (CacheBackend, error)
	RegisterBackend(name string, factory BackendFactory) error
	Execute(ctx context.Context, meta *MethodMeta, args []reflect.Value) (interface{}, error)
	Close() error
}

// cacheManagerImpl 实现
type cacheManagerImpl struct {
	mu               sync.RWMutex
	caches           map[string]CacheBackend
	configs          map[string]*CacheConfig
	backendFactories map[string]BackendFactory
	evaluator        *spel.SpELEvaluator
	defaultConfig    *CacheConfig
}

// NewCacheManager 创建缓存管理器
func NewCacheManager() CacheManager {
	m := &cacheManagerImpl{
		caches:           make(map[string]CacheBackend),
		configs:          make(map[string]*CacheConfig),
		backendFactories: make(map[string]BackendFactory),
		evaluator:        spel.NewSpELEvaluator(),
		defaultConfig:    DefaultCacheConfig("default"),
	}
	for n, f := range backend.BackendRegistry {
		m.backendFactories[n] = f
	}
	return m
}

// GetCache 获取缓存
func (m *cacheManagerImpl) GetCache(name string) (CacheBackend, error) {
	m.mu.RLock()
	if c, ok := m.caches[name]; ok {
		m.mu.RUnlock()
		return c, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.caches[name]; ok {
		return c, nil
	}

	cfg := m.configs[name]
	if cfg == nil {
		cfg = DefaultCacheConfig(name)
		m.configs[name] = cfg
	}

	factory := m.backendFactories["memory"]
	if factory == nil {
		return nil, fmt.Errorf("no memory backend")
	}

	c, err := factory(cfg)
	if err != nil {
		return nil, err
	}
	m.caches[name] = c
	return c, nil
}

// RegisterBackend 注册后端
func (m *cacheManagerImpl) RegisterBackend(name string, factory BackendFactory) error {
	if name == "" || factory == nil {
		return fmt.Errorf("invalid")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.backendFactories[name] = factory
	return nil
}

// RegisterCacheConfig 注册配置
func (m *cacheManagerImpl) RegisterCacheConfig(name string, cfg *CacheConfig) error {
	if name == "" || cfg == nil {
		return fmt.Errorf("invalid")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[name] = cfg
	return nil
}

// Execute 执行缓存操作
func (m *cacheManagerImpl) Execute(ctx context.Context, meta *MethodMeta, args []reflect.Value) (interface{}, error) {
	switch meta.CacheType {
	case "cacheable":
		return m.execCacheable(ctx, meta, args)
	case "cacheput":
		return m.execCachePut(ctx, meta, args)
	case "cacheevict":
		return m.execCacheEvict(ctx, meta, args)
	}
	return nil, fmt.Errorf("unknown type: %s", meta.CacheType)
}

func (m *cacheManagerImpl) execCacheable(ctx context.Context, meta *MethodMeta, args []reflect.Value) (interface{}, error) {
	cache, err := m.GetCache(meta.CacheName)
	if err != nil {
		return nil, err
	}
	evalCtx := m.buildCtx(meta, args, nil)
	key, err := m.evaluator.EvaluateToString(meta.KeyExpr, evalCtx)
	if err != nil {
		return nil, err
	}
	v, found, _ := cache.Get(ctx, key)
	if found {
		return v, nil
	}
	return nil, nil
}

func (m *cacheManagerImpl) execCachePut(ctx context.Context, meta *MethodMeta, args []reflect.Value) (interface{}, error) {
	return nil, nil
}

func (m *cacheManagerImpl) execCacheEvict(ctx context.Context, meta *MethodMeta, args []reflect.Value) (interface{}, error) {
	cache, err := m.GetCache(meta.CacheName)
	if err != nil {
		return nil, err
	}
	evalCtx := m.buildCtx(meta, args, nil)
	key, _ := m.evaluator.EvaluateToString(meta.KeyExpr, evalCtx)
	cache.Delete(ctx, key)
	return nil, nil
}

func (m *cacheManagerImpl) buildCtx(meta *MethodMeta, args []reflect.Value, result interface{}) *spel.EvaluationContext {
	ctx := spel.NewEvaluationContext()
	for i, a := range args {
		v := a.Interface()
		ctx.SetArgByIndex(i, v)
		ctx.SetArg(fmt.Sprintf("p%d", i), v)
	}
	if result != nil {
		ctx.SetResult(result)
	}
	ctx.CacheName = meta.CacheName
	return ctx
}

// GetEvaluator 获取求值器
func (m *cacheManagerImpl) GetEvaluator() *spel.SpELEvaluator {
	return m.evaluator
}

// Close 关闭
func (m *cacheManagerImpl) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for n, c := range m.caches {
		c.Close()
		delete(m.caches, n)
	}
	m.evaluator.ClearCache()
	return nil
}

var _ CacheManager = (*cacheManagerImpl)(nil)
