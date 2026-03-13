package core

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
	"github.com/coderiser/go-cache/pkg/spel"
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
	GetProtection() *CacheProtection
	SetProtectionConfig(config *ProtectionConfig) error
	// Invalidate 使缓存失效
	Invalidate(ctx context.Context, cache string, key string) error
}

// cacheManagerImpl 实现
type cacheManagerImpl struct {
	mu               sync.RWMutex
	caches           map[string]CacheBackend
	configs          map[string]*CacheConfig
	backendFactories map[string]BackendFactory
	evaluator        *spel.SpELEvaluator
	defaultConfig    *CacheConfig
	protection       *CacheProtection
	protectionConfig *ProtectionConfig
}

// NewCacheManager 创建缓存管理器
func NewCacheManager() CacheManager {
	m := &cacheManagerImpl{
		caches:           make(map[string]CacheBackend),
		configs:          make(map[string]*CacheConfig),
		backendFactories: make(map[string]BackendFactory),
		evaluator:        spel.NewSpELEvaluator(),
		defaultConfig:    DefaultCacheConfig("default"),
		protectionConfig: DefaultProtectionConfig(),
	}
	m.protection = NewCacheProtection(m.protectionConfig)
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

	// 使用保护机制获取缓存
	protection := m.GetProtection()
	if protection == nil {
		// 无保护机制，直接获取
		v, found, _ := cache.Get(ctx, key)
		if found {
			return v, nil
		}
		return nil, nil
	}

	// 使用受保护的获取操作
	cacheGet := func() (interface{}, bool, error) {
		return cache.Get(ctx, key)
	}

	cacheSet := func(value interface{}, ttl time.Duration) error {
		// 应用雪崩保护的 TTL
		protectedTTL := protection.ApplyAvalancheProtection(ttl)
		wrappedValue := protection.WrapForStorage(value)
		return cache.Set(ctx, key, wrappedValue, protectedTTL)
	}

	// 注意：这里需要传入实际的执行函数，暂时返回 nil
	// 实际使用时，应该由调用方提供 cacheMissFn
	cacheMissFn := func() (interface{}, error) {
		// 这里应该执行原始方法，但在拦截器中需要特殊处理
		// 暂时返回 nil，表示需要调用原始方法
		return nil, nil
	}

	result, err := protection.ProtectedGet(ctx, key, cacheGet, cacheMissFn, cacheSet)
	return result, err
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

// GetProtection 获取缓存保护器
func (m *cacheManagerImpl) GetProtection() *CacheProtection {
	return m.protection
}

// SetProtectionConfig 设置缓存保护配置
func (m *cacheManagerImpl) SetProtectionConfig(config *ProtectionConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.protectionConfig = config
	m.protection = NewCacheProtection(config)
	return nil
}

// GetProtectionConfig 获取当前保护配置
func (m *cacheManagerImpl) GetProtectionConfig() *ProtectionConfig {
	return m.protectionConfig
}

// Invalidate 使缓存失效
func (m *cacheManagerImpl) Invalidate(ctx context.Context, cache string, key string) error {
	cacheBackend, err := m.GetCache(cache)
	if err != nil {
		return err
	}
	return cacheBackend.Delete(ctx, key)
}

var _ CacheManager = (*cacheManagerImpl)(nil)
