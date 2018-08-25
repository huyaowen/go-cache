package store

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

const (
	NoExpiration           time.Duration = -1
	DefaultExpiration      time.Duration = 0
	DefaultCleanupInterval time.Duration = 5 * time.Minute
)

type Item struct {
	Object     interface{}
	Expiration int64
}

// Returns true if the item has expired.
func (item Item) Expired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

type memory_store struct {
	defaultExpiration time.Duration
	items             map[interface{}]Item
	mu                sync.RWMutex
	onEvicted         func(interface{}, interface{})
	janitor           *janitor
}

func (m *memory_store) Get(key interface{}) (result interface{}, err error) {
	m.mu.RLock()
	item, found := m.items[key]
	if !found {
		m.mu.RUnlock()
		return nil, nil
	}
	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			m.mu.RUnlock()
			return nil, nil
		}
	}
	m.mu.RUnlock()
	return item.Object, nil
}

func (m *memory_store) Set(key, value interface{}, duration time.Duration) (err error) {
	var e int64

	if duration == 0 {
		duration = DefaultExpiration
	}

	if duration == DefaultExpiration {
		duration = m.defaultExpiration
	}
	if duration > 0 {
		e = time.Now().Add(duration).UnixNano()
	}
	m.mu.Lock()
	m.items[key] = Item{
		Object:     value,
		Expiration: e,
	}
	m.mu.Unlock()

	return nil
}

//set if not exist
func (m *memory_store) SetE(key, value interface{}, duration time.Duration) (err error) {
	m.mu.Lock()
	r, _ := m.Get(key)
	if r != nil {
		m.mu.Unlock()
		return fmt.Errorf("Item %s already exists", key)
	}
	m.Set(key, value, duration)
	m.mu.Unlock()
	return nil
}

func (m *memory_store) Delete(key interface{}) (v interface{}, err error) {
	m.mu.Lock()
	var evicted bool
	var value interface{}
	if m.onEvicted != nil {
		if v, found := m.items[key]; found {
			evicted = true
			value = v.Object
		}
	}
	delete(m.items, key)
	m.mu.Unlock()
	if evicted {
		m.onEvicted(key, value)
	}

	return value, nil
}

type keyAndValue struct {
	key   interface{}
	value interface{}
}

func (m *memory_store) DeleteExpired() (err error) {
	var evictedItems []keyAndValue
	now := time.Now().UnixNano()
	m.mu.Lock()
	for k, v := range m.items {
		// "Inlining" of expired
		if v.Expiration > 0 && now > v.Expiration {
			ov, _ := m.Delete(k)
			if ov != nil {
				evictedItems = append(evictedItems, keyAndValue{k, ov})
			}
		}
	}
	m.mu.Unlock()
	for _, v := range evictedItems {
		m.onEvicted(v.key, v.value)
	}
	return nil
}

func (m *memory_store) Flush() (err error) {
	m.mu.Lock()
	m.items = map[interface{}]Item{}
	m.mu.Unlock()
	return nil
}

func (m *memory_store) Keys() (keys []interface{}, err error) {
	m.mu.Lock()
	for k := range m.items {
		keys = append(keys, k)
	}
	m.mu.Unlock()
	return
}

// -----------------------------------GC AND STOP----------------
type janitor struct {
	Interval time.Duration
	stop     chan bool
}

func (j *janitor) Run(c *memory_store) {
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			c.DeleteExpired()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}

func stopJanitor(c *memory_store) {
	c.janitor.stop <- true
}

func runJanitor(c *memory_store, ci time.Duration) {
	j := &janitor{
		Interval: ci,
		stop:     make(chan bool),
	}
	c.janitor = j
	go j.Run(c)
}

func newMemoryStore(de time.Duration, m map[interface{}]Item) *memory_store {
	if de == 0 {
		de = -1
	}
	c := &memory_store{
		defaultExpiration: de,
		items:             m,
	}
	return c
}

func newCacheWithJanitor(de time.Duration, ci time.Duration, m map[interface{}]Item) *memory_store {
	// This trick ensures that the janitor goroutine (which--granted it
	// was enabled--is running DeleteExpired on c forever) does not keep
	// the returned C object from being garbage collected. When it is
	// garbage collected, the finalizer stops the janitor goroutine, after
	// which c can be collected.
	C := newMemoryStore(de, m)
	if ci > 0 {
		runJanitor(C, ci)
		runtime.SetFinalizer(C, stopJanitor)
	}
	return C
}

// Return a new cache with a given default expiration duration and cleanup
// interval. If the expiration duration is less than one (or NoExpiration),
// the items in the cache never expire (by default), and must be deleted
// manually. If the cleanup interval is less than one, expired items are not
// deleted from the cache before calling c.DeleteExpired().
func NewMemoryStore(defaultExpiration, cleanupInterval time.Duration) *memory_store {
	items := make(map[interface{}]Item)
	return newCacheWithJanitor(defaultExpiration, cleanupInterval, items)
}
