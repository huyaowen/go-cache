package core

import (
	"context"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
	"golang.org/x/sync/singleflight"
)

// rng 全局随机数生成器（线程安全）
// Using package-level functions from math/rand/v2 for simplicity

// IsNilMarker 检查值是否为空值标记
func IsNilMarker(v interface{}) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok {
		return s == backend.NilMarker
	}
	return false
}

// WrapNilMarker 将 nil 值包装为空值标记
func WrapNilMarker(v interface{}) interface{} {
	if v == nil {
		return backend.NilMarker
	}
	return v
}

// UnwrapNilMarker 将空值标记解包为 nil
func UnwrapNilMarker(v interface{}) interface{} {
	if IsNilMarker(v) {
		return nil
	}
	return v
}

// ProtectionConfig 缓存保护配置
type ProtectionConfig struct {
	// 穿透保护配置
	EnablePenetrationProtection bool          // 是否启用穿透保护
	EmptyValueTTL               time.Duration // 空值缓存 TTL（默认 5 分钟）

	// 击穿保护配置
	EnableBreakdownProtection bool // 是否启用击穿保护（singleflight）

	// 雪崩保护配置
	EnableAvalancheProtection bool          // 是否启用雪崩保护
	TTLJitterFactor           float64       // TTL 随机偏移因子（0.0-0.5，默认 0.1 即 10%）
}

// DefaultProtectionConfig 默认保护配置
func DefaultProtectionConfig() *ProtectionConfig {
	return &ProtectionConfig{
		EnablePenetrationProtection: true,
		EmptyValueTTL:               5 * time.Minute,
		EnableBreakdownProtection:   true,
		EnableAvalancheProtection:   true,
		TTLJitterFactor:             0.1,
	}
}

// CacheProtection 缓存异常保护器
type CacheProtection struct {
	config      *ProtectionConfig
	singleFlyer *singleflight.Group
	mu          sync.RWMutex
}

// NewCacheProtection 创建缓存保护器
func NewCacheProtection(config *ProtectionConfig) *CacheProtection {
	if config == nil {
		config = DefaultProtectionConfig()
	}
	return &CacheProtection{
		config:      config,
		singleFlyer: &singleflight.Group{},
	}
}

// GetProtectionConfig 获取保护配置
func (p *CacheProtection) GetProtectionConfig() *ProtectionConfig {
	return p.config
}

// ApplyPenetrationProtection 应用穿透保护
// 如果值是否为空值标记，返回 true 表示应该视为缓存未命中
func (p *CacheProtection) ApplyPenetrationProtection(value interface{}) (interface{}, bool) {
	if !p.config.EnablePenetrationProtection {
		return value, false
	}

	if IsNilMarker(value) {
		return nil, true // 空值标记，视为未命中
	}

	return value, false
}

// WrapForStorage 包装值用于存储（处理 nil 值）
func (p *CacheProtection) WrapForStorage(value interface{}) interface{} {
	if !p.config.EnablePenetrationProtection {
		return value
	}
	return WrapNilMarker(value)
}

// UnwrapFromStorage 从存储解包值
func (p *CacheProtection) UnwrapFromStorage(value interface{}) interface{} {
	if !p.config.EnablePenetrationProtection {
		return value
	}
	return UnwrapNilMarker(value)
}

// GetEmptyValueTTL 获取空值缓存的 TTL
func (p *CacheProtection) GetEmptyValueTTL() time.Duration {
	return p.config.EmptyValueTTL
}

// ApplyBreakdownProtection 应用击穿保护（singleflight）
// key: 缓存键
// fn: 实际的数据获取函数
// 返回：(结果，错误，是否从 singleflight 获取)
func (p *CacheProtection) ApplyBreakdownProtection(ctx context.Context, key string, fn func() (interface{}, error)) (interface{}, error, bool) {
	if !p.config.EnableBreakdownProtection {
		// 未启用击穿保护，直接执行
		result, err := fn()
		return result, err, false
	}

	// 使用 singleflight 合并并发请求
	v, err, shared := p.singleFlyer.Do(key, func() (interface{}, error) {
		return fn()
	})

	return v, err, shared
}

// ApplyAvalancheProtection 应用雪崩保护（计算带抖动的 TTL）
func (p *CacheProtection) ApplyAvalancheProtection(baseTTL time.Duration) time.Duration {
	if !p.config.EnableAvalancheProtection {
		return baseTTL
	}

	if p.config.TTLJitterFactor <= 0 {
		return baseTTL
	}

	jitterFactor := p.config.TTLJitterFactor
	if jitterFactor > 0.5 {
		jitterFactor = 0.5
	}

	// 计算随机抖动
	jitter := time.Duration(float64(baseTTL) * jitterFactor)
	
	// 使用安全的随机数生成器生成随机偏移
	randomOffset := time.Duration(rand.Int64N(int64(jitter*2))) - jitter
	
	actualTTL := baseTTL + randomOffset
	if actualTTL < time.Second {
		actualTTL = time.Second
	}

	return actualTTL
}

// CalculateTTLWithJitter 计算带抖动的 TTL（便捷方法）
func (p *CacheProtection) CalculateTTLWithJitter(baseTTL time.Duration, jitterFactor float64) time.Duration {
	if jitterFactor <= 0 {
		jitterFactor = p.config.TTLJitterFactor
	}

	jitter := time.Duration(float64(baseTTL) * jitterFactor)
	randomOffset := time.Duration(rand.Int64N(int64(jitter*2))) - jitter
	
	actualTTL := baseTTL + randomOffset
	if actualTTL < time.Second {
		actualTTL = time.Second
	}

	return actualTTL
}

// SingleFlightDo 直接使用 singleflight（高级用法）
func (p *CacheProtection) SingleFlightDo(key string, fn func() (interface{}, error)) (interface{}, error, bool) {
	return p.singleFlyer.Do(key, fn)
}

// CancelSingleFlight 取消正在进行的 singleflight 请求（通过删除 key）
// 注意：这不会取消正在执行的函数，只会移除 singleflight 的缓存
func (p *CacheProtection) CancelSingleFlight(key string) {
	p.singleFlyer.Forget(key)
}

// ProtectedGet 受保护的缓存获取操作
// 整合穿透保护和击穿保护
func (p *CacheProtection) ProtectedGet(
	ctx context.Context,
	key string,
	cacheGet func() (interface{}, bool, error),
	cacheMissFn func() (interface{}, error),
	cacheSet func(interface{}, time.Duration) error,
) (interface{}, error) {
	// 1. 尝试从缓存获取
	value, found, err := cacheGet()
	if err != nil {
		// 缓存获取失败，直接执行原始函数
		result, execErr := cacheMissFn()
		return result, execErr
	}

	if found {
		// 2. 应用穿透保护检查
		unwrapped, isEmpty := p.ApplyPenetrationProtection(value)
		if isEmpty {
			// 空值标记，视为命中但值为 nil（穿透保护已生效）
			return nil, nil
		}
		return unwrapped, nil
	}

	// 3. 缓存未命中，应用击穿保护
	// 3. 缓存未命中，应用击穿保护
	var result interface{}
	var execErr error

	result, execErr, _ = p.ApplyBreakdownProtection(ctx, key, func() (interface{}, error) {
		// 执行原始函数
		missResult, missErr := cacheMissFn()
		if missErr != nil {
			return missResult, missErr
		}

		// 4. 写入缓存（应用穿透和雪崩保护）
		var ttl time.Duration
		if missResult == nil {
			// 空值使用较短的 TTL
			ttl = p.GetEmptyValueTTL()
		} else {
			// 正常值使用带抖动的 TTL
			ttl = p.ApplyAvalancheProtection(30 * time.Minute) // 默认 30 分钟，可配置
		}

		wrappedValue := p.WrapForStorage(missResult)
		_ = cacheSet(wrappedValue, ttl)

		return missResult, nil
	})

	return result, execErr
}

// ProtectionStats 保护机制统计
type ProtectionStats struct {
	PenetrationBlocked int64 // 穿透拦截次数
	BreakdownMerged    int64 // 击穿合并次数
	AvalancheJittered  int64 // 雪崩抖动次数
}

// GetStats 获取保护统计（当前实现返回空统计，可扩展）
func (p *CacheProtection) GetStats() *ProtectionStats {
	return &ProtectionStats{}
}
