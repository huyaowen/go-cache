package backend

import (
	"context"
	"sync"
	"time"
)

// HybridBackend 混合缓存后端（L1+L2 两级缓存）
// L1: 本地 Memory 缓存（快速访问）
// L2: Redis 缓存（分布式共享）
type HybridBackend struct {
	mu          sync.RWMutex
	l1          *MemoryBackend  // 本地缓存
	l2          *RedisBackend   // Redis 缓存
	config      *CacheConfig
	stats       *HybridStats
	ttlMgr      *TTLManager
	keyBuilder  *DefaultKeyBuilder
	l1WriteBack time.Duration // L1 回写 TTL
	closed      bool
}

// HybridStats 混合缓存统计
type HybridStats struct {
	l1Hits, l1Misses           int64
	l2Hits, l2Misses           int64
	l2Fallbacks                int64 // L1 miss 后 L2 命中的次数
	l1Backfills                int64 // 从 L2 回写 L1 的次数
	sets, deletes, errors      int64
}

// HybridConfig 混合缓存配置
type HybridConfig struct {
	L1Config      *CacheConfig  // L1 配置
	L2Config      *RedisConfig  // L2 配置
	L1WriteBackTTL time.Duration // L1 回写 TTL（默认 5 分钟）
}

// DefaultHybridConfig 默认混合缓存配置
func DefaultHybridConfig() *HybridConfig {
	return &HybridConfig{
		L1Config: &CacheConfig{
			Name:           "hybrid-l1",
			MaxSize:        1000, // L1 较小，只存热点
			DefaultTTL:     5 * time.Minute,
			MaxTTL:         30 * time.Minute,
			EvictionPolicy: "lru",
		},
		L2Config: DefaultRedisConfig(),
		L1WriteBackTTL: 5 * time.Minute,
	}
}

// NewHybridBackend 创建混合缓存后端
func NewHybridBackend(config *HybridConfig) (*HybridBackend, error) {
	if config == nil {
		config = DefaultHybridConfig()
	}

	// 创建 L1 缓存
	l1, err := NewMemoryBackend(config.L1Config)
	if err != nil {
		return nil, err
	}

	// 创建 L2 缓存
	l2, err := NewRedisBackend(config.L2Config)
	if err != nil {
		l1.Close()
		return nil, err
	}

	return &HybridBackend{
		l1:          l1,
		l2:          l2,
		config:      config.L1Config,
		stats:       &HybridStats{},
		ttlMgr:      NewTTLManager(config.L1Config.DefaultTTL, config.L1Config.MaxTTL),
		keyBuilder:  NewDefaultKeyBuilder(":", config.L1Config.Name),
		l1WriteBack: config.L1WriteBackTTL,
	}, nil
}

// Get 获取缓存值（L1 → L2 级联查询）
func (h *HybridBackend) Get(ctx context.Context, key string) (interface{}, bool, error) {
	h.mu.RLock()
	if h.closed {
		h.mu.RUnlock()
		return nil, false, nil
	}
	h.mu.RUnlock()

	// 1. 先查 L1（本地缓存）
	if val, found, _ := h.l1.Get(ctx, key); found {
		h.stats.recordL1Hit()
		return val, true, nil
	}
	h.stats.recordL1Miss()

	// 2. L1 未命中，查 L2（Redis 缓存）
	if val, found, _ := h.l2.Get(ctx, key); found {
		h.stats.recordL2Hit()
		h.stats.recordL2Fallback()
		
		// 3. 回写 L1（提升后续访问速度）
		writeBackTTL := h.l1WriteBack
		if writeBackTTL <= 0 {
			writeBackTTL = 5 * time.Minute
		}
		_ = h.l1.Set(ctx, key, val, writeBackTTL)
		h.stats.recordL1Backfill()
		
		return val, true, nil
	}
	h.stats.recordL2Miss()

	return nil, false, nil
}

// Set 设置缓存值（同时写入 L1 和 L2）
func (h *HybridBackend) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	h.mu.RLock()
	if h.closed {
		h.mu.RUnlock()
		return nil
	}
	h.mu.RUnlock()

	// 同时写入 L1 和 L2
	err1 := h.l1.Set(ctx, key, value, ttl)
	err2 := h.l2.Set(ctx, key, value, ttl)

	h.stats.recordSet()

	if err1 != nil {
		return err1
	}
	return err2
}

// Delete 删除缓存值（同时删除 L1 和 L2）
func (h *HybridBackend) Delete(ctx context.Context, key string) error {
	h.mu.RLock()
	if h.closed {
		h.mu.RUnlock()
		return nil
	}
	h.mu.RUnlock()

	err1 := h.l1.Delete(ctx, key)
	err2 := h.l2.Delete(ctx, key)

	h.stats.recordDelete()

	if err1 != nil {
		return err1
	}
	return err2
}

// Close 关闭缓存后端
func (h *HybridBackend) Close() error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil
	}
	h.closed = true
	h.mu.Unlock()

	// 先关闭 L2，再关闭 L1
	err1 := h.l2.Close()
	err2 := h.l1.Close()

	if err1 != nil {
		return err1
	}
	return err2
}

// Stats 获取缓存统计信息
func (h *HybridBackend) Stats() *CacheStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	l1Stats := h.l1.Stats()
	l2Stats := h.l2.Stats()

	// 合并统计信息
	totalHits := h.stats.getL1Hits() + h.stats.getL2Hits()
	totalMisses := h.stats.getL1Misses() + h.stats.getL2Misses()
	total := totalHits + totalMisses

	hitRate := 0.0
	if total > 0 {
		hitRate = float64(totalHits) / float64(total)
	}

	return &CacheStats{
		Hits:      totalHits,
		Misses:    totalMisses,
		Sets:      h.stats.getSets(),
		Deletes:   h.stats.getDeletes(),
		Evictions: l1Stats.Evictions, // L1 的淘汰数
		Size:      l1Stats.Size + l2Stats.Size,
		MaxSize:   l1Stats.MaxSize, // L1 的最大容量
		HitRate:   hitRate,
	}
}

// HybridStats 原子操作
func (s *HybridStats) recordL1Hit()     { atomicAddInt64(&s.l1Hits, 1) }
func (s *HybridStats) recordL1Miss()    { atomicAddInt64(&s.l1Misses, 1) }
func (s *HybridStats) recordL2Hit()     { atomicAddInt64(&s.l2Hits, 1) }
func (s *HybridStats) recordL2Miss()    { atomicAddInt64(&s.l2Misses, 1) }
func (s *HybridStats) recordL2Fallback() { atomicAddInt64(&s.l2Fallbacks, 1) }
func (s *HybridStats) recordL1Backfill() { atomicAddInt64(&s.l1Backfills, 1) }
func (s *HybridStats) recordSet()       { atomicAddInt64(&s.sets, 1) }
func (s *HybridStats) recordDelete()    { atomicAddInt64(&s.deletes, 1) }

func (s *HybridStats) getL1Hits() int64     { return atomicLoadInt64(&s.l1Hits) }
func (s *HybridStats) getL1Misses() int64   { return atomicLoadInt64(&s.l1Misses) }
func (s *HybridStats) getL2Hits() int64     { return atomicLoadInt64(&s.l2Hits) }
func (s *HybridStats) getL2Misses() int64   { return atomicLoadInt64(&s.l2Misses) }
func (s *HybridStats) getL2Fallbacks() int64 { return atomicLoadInt64(&s.l2Fallbacks) }
func (s *HybridStats) getL1Backfills() int64 { return atomicLoadInt64(&s.l1Backfills) }
func (s *HybridStats) getSets() int64       { return atomicLoadInt64(&s.sets) }
func (s *HybridStats) getDeletes() int64    { return atomicLoadInt64(&s.deletes) }

// GetL1 获取 L1 缓存（用于高级操作）
func (h *HybridBackend) GetL1() *MemoryBackend {
	return h.l1
}

// GetL2 获取 L2 缓存（用于高级操作）
func (h *HybridBackend) GetL2() *RedisBackend {
	return h.l2
}

// GetHybridStats 获取混合缓存详细统计
func (h *HybridBackend) GetHybridStats() *HybridStats {
	return h.stats
}

// 确保实现 CacheBackend 接口
var _ CacheBackend = (*HybridBackend)(nil)

// init 注册混合缓存后端
func init() {
	Register("hybrid", func(config *CacheConfig) (CacheBackend, error) {
		hybridConfig := DefaultHybridConfig()
		hybridConfig.L1Config = config
		hybridConfig.L2Config.Addr = "localhost:6379" // 默认 Redis 地址
		return NewHybridBackend(hybridConfig)
	})
}
