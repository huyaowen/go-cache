package backend

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// CacheItem 缓存项
type CacheItem struct {
	Value      interface{}
	ExpiresAt  time.Time
	CreatedAt  time.Time
	LastAccess time.Time
}

func (i *CacheItem) IsExpired() bool {
	return !i.ExpiresAt.IsZero() && time.Now().After(i.ExpiresAt)
}

// StatsCounter 统计计数器
type StatsCounter struct {
	hits, misses, sets, deletes, evictions, size, maxSize, lastAccess int64
}

func NewStatsCounter(maxSize int64) *StatsCounter {
	return &StatsCounter{maxSize: maxSize}
}

func (c *StatsCounter) RecordHit()       { atomicAddInt64(&c.hits, 1); atomicStoreInt64(&c.lastAccess, time.Now().UnixNano()) }
func (c *StatsCounter) RecordMiss()      { atomicAddInt64(&c.misses, 1); atomicStoreInt64(&c.lastAccess, time.Now().UnixNano()) }
func (c *StatsCounter) RecordSet()       { atomicAddInt64(&c.sets, 1); atomicStoreInt64(&c.lastAccess, time.Now().UnixNano()) }
func (c *StatsCounter) RecordDelete()    { atomicAddInt64(&c.deletes, 1) }
func (c *StatsCounter) RecordEviction()  { atomicAddInt64(&c.evictions, 1) }
func (c *StatsCounter) SetSize(s int64)  { atomicStoreInt64(&c.size, s) }
func (c *StatsCounter) IncSize()         { atomicAddInt64(&c.size, 1) }
func (c *StatsCounter) DecSize()         { atomicAddInt64(&c.size, -1) }

func (c *StatsCounter) Snapshot() *CacheStats {
	hits, misses := atomicLoadInt64(&c.hits), atomicLoadInt64(&c.misses)
	total := hits + misses
	rate := 0.0
	if total > 0 { rate = float64(hits) / float64(total) }
	return &CacheStats{
		Hits: hits, Misses: misses, Sets: atomicLoadInt64(&c.sets),
		Deletes: atomicLoadInt64(&c.deletes), Evictions: atomicLoadInt64(&c.evictions),
		Size: atomicLoadInt64(&c.size), MaxSize: c.maxSize, HitRate: rate,
	}
}

// 原子操作包装
func atomicAddInt64(addr *int64, delta int64) { atomic.AddInt64(addr, delta) }
func atomicLoadInt64(addr *int64) int64       { return atomic.LoadInt64(addr) }
func atomicStoreInt64(addr *int64, val int64) { atomic.StoreInt64(addr, val) }

// MemoryBackend 内存缓存后端
type MemoryBackend struct {
	mu          sync.RWMutex
	data        map[string]*CacheItem
	config      *CacheConfig
	stats       *StatsCounter
	ttlMgr      *TTLManager
	keyBuilder  *DefaultKeyBuilder
	accessOrder []string
	accessIndex map[string]int
	stopCleanup chan struct{}
	cleanupDone chan struct{}
	closed      bool
}

func NewMemoryBackend(config *CacheConfig) (*MemoryBackend, error) {
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	b := &MemoryBackend{
		data:        make(map[string]*CacheItem, config.MaxSize/10+1),
		config:      config,
		stats:       NewStatsCounter(config.MaxSize),
		ttlMgr:      NewTTLManager(config.DefaultTTL, config.MaxTTL),
		keyBuilder:  NewDefaultKeyBuilder(":", config.Name),
		accessOrder: make([]string, 0, config.MaxSize),
		accessIndex: make(map[string]int),
		stopCleanup: make(chan struct{}),
		cleanupDone: make(chan struct{}),
	}
	go b.startCleanup()
	return b, nil
}

func (m *MemoryBackend) startCleanup() {
	defer close(m.cleanupDone)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.cleanupExpired()
		case <-m.stopCleanup:
			return
		}
	}
}

func (m *MemoryBackend) cleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, item := range m.data {
		if item.IsExpired() {
			delete(m.data, key)
			m.removeAccessOrder(key)
			m.stats.DecSize()
			m.stats.RecordEviction()
		}
	}
}

func (m *MemoryBackend) Get(ctx context.Context, key string) (interface{}, bool, error) {
	m.mu.RLock()
	item, exists := m.data[key]
	m.mu.RUnlock()

	if !exists {
		m.stats.RecordMiss()
		return nil, false, nil
	}
	if item.IsExpired() {
		go m.Delete(ctx, key)
		m.stats.RecordMiss()
		return nil, false, nil
	}

	m.mu.Lock()
	item.LastAccess = time.Now()
	m.updateAccessOrder(key)
	m.mu.Unlock()

	m.stats.RecordHit()
	return item.Value, true, nil
}

func (m *MemoryBackend) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	normalizedTTL := m.ttlMgr.Normalize(ttl)

	m.mu.Lock()
	defer m.mu.Unlock()

	if int64(len(m.data)) >= m.config.MaxSize {
		m.evictIfNeeded()
	}

	now := time.Now()
	var expiresAt time.Time
	if normalizedTTL > 0 {
		expiresAt = now.Add(normalizedTTL)
	}

	item := &CacheItem{Value: value, ExpiresAt: expiresAt, CreatedAt: now, LastAccess: now}
	_, exists := m.data[key]
	m.data[key] = item

	if !exists {
		m.addToAccessOrder(key)
		m.stats.IncSize()
	}
	m.stats.RecordSet()
	return nil
}

func (m *MemoryBackend) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.data[key]; exists {
		delete(m.data, key)
		m.removeAccessOrder(key)
		m.stats.DecSize()
		m.stats.RecordDelete()
	}
	return nil
}

func (m *MemoryBackend) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	m.mu.Unlock()

	close(m.stopCleanup)
	<-m.cleanupDone

	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = nil
	m.accessOrder = nil
	m.accessIndex = nil
	return nil
}

func (m *MemoryBackend) Stats() *CacheStats {
	m.stats.SetSize(int64(len(m.data)))
	return m.stats.Snapshot()
}

func (m *MemoryBackend) evictIfNeeded() {
	if len(m.accessOrder) == 0 { return }
	switch m.config.EvictionPolicy {
	case "lru", "":
		m.evictLRU()
	case "fifo":
		m.evictFIFO()
	default:
		m.evictLRU()
	}
}

func (m *MemoryBackend) evictLRU() {
	if len(m.accessOrder) == 0 { return }
	key := m.accessOrder[0]
	m.accessOrder = m.accessOrder[1:]
	delete(m.accessIndex, key)
	if item, exists := m.data[key]; exists {
		delete(m.data, key)
		m.stats.RecordEviction()
		_ = item
	}
}

func (m *MemoryBackend) evictFIFO() { m.evictLRU() }

func (m *MemoryBackend) addToAccessOrder(key string) {
	m.accessOrder = append(m.accessOrder, key)
	m.accessIndex[key] = len(m.accessOrder) - 1
}

func (m *MemoryBackend) updateAccessOrder(key string) {
	idx, exists := m.accessIndex[key]
	if !exists || idx == len(m.accessOrder)-1 { return }
	m.accessOrder = append(m.accessOrder[:idx], m.accessOrder[idx+1:]...)
	m.accessOrder = append(m.accessOrder, key)
	m.accessIndex[key] = len(m.accessOrder) - 1
	for i := idx; i < len(m.accessOrder)-1; i++ {
		m.accessIndex[m.accessOrder[i]] = i
	}
}

func (m *MemoryBackend) removeAccessOrder(key string) {
	idx, exists := m.accessIndex[key]
	if !exists { return }
	delete(m.accessIndex, key)
	m.accessOrder = append(m.accessOrder[:idx], m.accessOrder[idx+1:]...)
	for i := idx; i < len(m.accessOrder); i++ {
		m.accessIndex[m.accessOrder[i]] = i
	}
}

var _ CacheBackend = (*MemoryBackend)(nil)

func init() {
	Register("memory", func(config *CacheConfig) (CacheBackend, error) {
		return NewMemoryBackend(config)
	})
}
