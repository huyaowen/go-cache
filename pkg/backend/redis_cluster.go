package backend

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"github.com/coderiser/go-cache/pkg/serializer"
	"github.com/redis/go-redis/v9"
)

// RedisClusterConfig Redis Cluster 配置
type RedisClusterConfig struct {
	Addrs         []string      // Cluster 节点地址列表
	Password      string        // Redis 密码
	DialTimeout   time.Duration // 连接超时
	ReadTimeout   time.Duration // 读取超时
	WriteTimeout  time.Duration // 写入超时
	PoolSize      int           // 连接池大小
	MinIdleConns  int           // 最小空闲连接
	MaxRetries    int           // 最大重试次数
	Prefix        string        // Key 前缀
	DefaultTTL    time.Duration // 默认 TTL
	MaxTTL        time.Duration // 最大 TTL
	RouteByPrefix bool          // 是否按前缀路由（高级特性）
	Serializer    string        // 序列化器类型：json, gob, msgpack
}

// DefaultRedisClusterConfig 默认 Cluster 配置
func DefaultRedisClusterConfig() *RedisClusterConfig {
	return &RedisClusterConfig{
		Addrs:         []string{"localhost:7000", "localhost:7001", "localhost:7002"},
		Password:      "",
		DialTimeout:   5 * time.Second,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		PoolSize:      10,
		MinIdleConns:  5,
		MaxRetries:    3,
		Prefix:        "",
		DefaultTTL:    30 * time.Minute,
		MaxTTL:        24 * time.Hour,
		RouteByPrefix: false,
		Serializer:    "json",
	}
}

// RedisClusterBackend Redis Cluster 缓存后端实现
type RedisClusterBackend struct {
	client     *redis.ClusterClient
	config     *RedisClusterConfig
	stats      *RedisClusterStats
	ttlMgr     *TTLManager
	keyBuilder *DefaultKeyBuilder
	serializer serializer.Serializer
	closed     int32
}

// RedisClusterStats Cluster 统计信息
type RedisClusterStats struct {
	hits      int64
	misses    int64
	sets      int64
	deletes   int64
	errors    int64
	size      int64
	lastCheck int64
}

// NewRedisClusterBackend 创建 Redis Cluster 后端实例
func NewRedisClusterBackend(config *RedisClusterConfig) (*RedisClusterBackend, error) {
	if config == nil {
		config = DefaultRedisClusterConfig()
	}

	if len(config.Addrs) == 0 {
		return nil, errors.New("at least one cluster node address is required")
	}

	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        config.Addrs,
		Password:     config.Password,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries:   config.MaxRetries,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to Redis Cluster: %w", err)
	}

	// 检查集群状态
	clusterInfo, err := client.ClusterSlots(ctx).Result()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to get cluster info: %w", err)
	}

	_ = clusterInfo // 可用于日志

	// 初始化序列化器
	ser, err := serializer.Get(config.Serializer)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to get serializer: %w", err)
	}

	return &RedisClusterBackend{
		client:     client,
		config:     config,
		stats:      &RedisClusterStats{},
		ttlMgr:     NewTTLManager(config.DefaultTTL, config.MaxTTL),
		keyBuilder: NewDefaultKeyBuilder(":", config.Prefix),
		serializer: ser,
	}, nil
}

// buildKey 构建完整的 Redis key
func (r *RedisClusterBackend) buildKey(key string) string {
	if r.config.Prefix != "" {
		return r.config.Prefix + ":" + key
	}
	return key
}

// Get 从 Redis Cluster 获取缓存值
func (r *RedisClusterBackend) Get(ctx context.Context, key string) (interface{}, bool, error) {
	if atomic.LoadInt32(&r.closed) == 1 {
		return nil, false, errors.New("RedisClusterBackend is closed")
	}

	fullKey := r.buildKey(key)

	val, err := r.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			atomic.AddInt64(&r.stats.misses, 1)
			return nil, false, nil
		}
		atomic.AddInt64(&r.stats.errors, 1)
		return nil, false, err
	}

	// 检查是否为空值标记
	if string(val) == NilMarker {
		atomic.AddInt64(&r.stats.misses, 1)
		return nil, false, nil
	}

	// 反序列化
	var result interface{}
	if err := r.serializer.Unmarshal(val, &result); err != nil {
		result = string(val)
	}

	atomic.AddInt64(&r.stats.hits, 1)
	return result, true, nil
}

// Set 设置缓存值
func (r *RedisClusterBackend) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if atomic.LoadInt32(&r.closed) == 1 {
		return errors.New("RedisClusterBackend is closed")
	}

	if value == nil {
		value = NilMarker
	}

	data, err := r.serializer.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	normalizedTTL := r.ttlMgr.Normalize(ttl)

	fullKey := r.buildKey(key)
	if err := r.client.Set(ctx, fullKey, data, normalizedTTL).Err(); err != nil {
		atomic.AddInt64(&r.stats.errors, 1)
		return err
	}

	atomic.AddInt64(&r.stats.sets, 1)
	return nil
}

// SetWithJitter 设置带随机 TTL 偏移的缓存值（防雪崩）
func (r *RedisClusterBackend) SetWithJitter(ctx context.Context, key string, value interface{}, baseTTL time.Duration, jitterFactor float64) error {
	if jitterFactor <= 0 {
		jitterFactor = 0.1
	}
	if jitterFactor > 0.5 {
		jitterFactor = 0.5
	}

	normalizedTTL := r.ttlMgr.Normalize(baseTTL)
	jitter := time.Duration(float64(normalizedTTL) * jitterFactor)
	randomOffset := time.Duration(rand.Int64N(int64(jitter*2))) - jitter

	actualTTL := normalizedTTL + randomOffset
	if actualTTL < time.Second {
		actualTTL = time.Second
	}

	return r.Set(ctx, key, value, actualTTL)
}

// Delete 删除缓存值
func (r *RedisClusterBackend) Delete(ctx context.Context, key string) error {
	if atomic.LoadInt32(&r.closed) == 1 {
		return errors.New("RedisClusterBackend is closed")
	}

	fullKey := r.buildKey(key)
	if err := r.client.Del(ctx, fullKey).Err(); err != nil {
		atomic.AddInt64(&r.stats.errors, 1)
		return err
	}

	atomic.AddInt64(&r.stats.deletes, 1)
	return nil
}

// Close 关闭连接
func (r *RedisClusterBackend) Close() error {
	if !atomic.CompareAndSwapInt32(&r.closed, 0, 1) {
		return nil
	}
	return r.client.Close()
}

// Stats 获取统计信息
func (r *RedisClusterBackend) Stats() *CacheStats {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Cluster 模式下获取所有节点的大小
		size, err := r.client.DBSize(ctx).Result()
		if err == nil {
			atomic.StoreInt64(&r.stats.size, size)
			atomic.StoreInt64(&r.stats.lastCheck, time.Now().UnixNano())
		}
	}()

	hits := atomic.LoadInt64(&r.stats.hits)
	misses := atomic.LoadInt64(&r.stats.misses)
	total := hits + misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return &CacheStats{
		Hits:      hits,
		Misses:    misses,
		Sets:      atomic.LoadInt64(&r.stats.sets),
		Deletes:   atomic.LoadInt64(&r.stats.deletes),
		Evictions: 0,
		Size:      atomic.LoadInt64(&r.stats.size),
		MaxSize:   0,
		HitRate:   hitRate,
	}
}

// Client 获取底层 Cluster 客户端
func (r *RedisClusterBackend) Client() *redis.ClusterClient {
	return r.client
}

// Ping 测试连接
func (r *RedisClusterBackend) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// ClusterSlots 获取集群槽位信息
func (r *RedisClusterBackend) ClusterSlots(ctx context.Context) ([]redis.ClusterSlot, error) {
	return r.client.ClusterSlots(ctx).Result()
}

// ClusterNodes 获取集群节点信息
func (r *RedisClusterBackend) ClusterNodes(ctx context.Context) (string, error) {
	return r.client.ClusterNodes(ctx).Result()
}

// Clear 清空缓存（危险操作）
func (r *RedisClusterBackend) Clear(ctx context.Context) error {
	if r.config.Prefix != "" {
		pattern := r.config.Prefix + ":*"
		iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()
		for iter.Next(ctx) {
			if err := r.client.Del(ctx, iter.Val()).Err(); err != nil {
				return err
			}
		}
		return iter.Err()
	}

	// Cluster 模式下不支持 FLUSHDB，需要遍历所有槽位
	return errors.New("Clear without prefix is not supported in cluster mode")
}

// 确保实现 CacheBackend 接口
var _ CacheBackend = (*RedisClusterBackend)(nil)

// init 注册 Redis Cluster 后端
func init() {
	Register("redis-cluster", func(config *CacheConfig) (CacheBackend, error) {
		clusterConfig := DefaultRedisClusterConfig()
		clusterConfig.DefaultTTL = config.DefaultTTL
		clusterConfig.MaxTTL = config.MaxTTL
		return NewRedisClusterBackend(clusterConfig)
	})
}
