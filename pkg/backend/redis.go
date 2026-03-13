package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"github.com/coderiser/go-cache/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// RedisConfig Redis 后端配置
type RedisConfig struct {
	Addr         string        // Redis 地址，如 "localhost:6379"
	Password     string        // Redis 密码
	DB           int           // Redis 数据库编号
	Prefix       string        // Key 前缀
	DefaultTTL   time.Duration // 默认 TTL
	MaxTTL       time.Duration // 最大 TTL
	PoolSize     int           // 连接池大小
	MinIdleConns int           // 最小空闲连接
	MaxRetries   int           // 最大重试次数
	DialTimeout  time.Duration // 连接超时
	ReadTimeout  time.Duration // 读取超时
	WriteTimeout time.Duration // 写入超时
}

// DefaultRedisConfig 默认 Redis 配置
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		Prefix:       "",
		DefaultTTL:   30 * time.Minute,
		MaxTTL:       24 * time.Hour,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// RedisBackend Redis 缓存后端实现
type RedisBackend struct {
	client    *redis.Client
	config    *RedisConfig
	stats     *RedisStats
	ttlMgr    *TTLManager
	keyBuilder *DefaultKeyBuilder
	closed    int32
}

// RedisStats Redis 缓存统计
type RedisStats struct {
	hits      int64
	misses    int64
	sets      int64
	deletes   int64
	errors    int64
	size      int64 // 通过 DBSIZE 估算
	lastCheck int64
}

// NewRedisBackend 创建 Redis 后端实例
func NewRedisBackend(config *RedisConfig) (*RedisBackend, error) {
	if config == nil {
		config = DefaultRedisConfig()
	}

	if config.Addr == "" {
		return nil, errors.New("Redis address is required")
	}

	logger.Info("Redis backend: Connecting to Redis at %s", config.Addr)

	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		logger.Error("Redis backend: Failed to connect to Redis at %s, error=%v", config.Addr, err)
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Redis backend: Successfully connected to Redis at %s (poolSize=%d, minIdleConns=%d)", 
		config.Addr, config.PoolSize, config.MinIdleConns)

	return &RedisBackend{
		client:     client,
		config:     config,
		stats:      &RedisStats{},
		ttlMgr:     NewTTLManager(config.DefaultTTL, config.MaxTTL),
		keyBuilder: NewDefaultKeyBuilder(":", config.Prefix),
	}, nil
}

// buildKey 构建完整的 Redis key
func (r *RedisBackend) buildKey(key string) string {
	if r.config.Prefix != "" {
		return r.config.Prefix + ":" + key
	}
	return key
}

// Get 从 Redis 获取缓存值
func (r *RedisBackend) Get(ctx context.Context, key string) (interface{}, bool, error) {
	if atomic.LoadInt32(&r.closed) == 1 {
		logger.Debug("Redis backend: Get called on closed backend, key=%s", key)
		return nil, false, errors.New("RedisBackend is closed")
	}

	fullKey := r.buildKey(key)
	logger.Debug("Redis backend: Getting cache key=%s, fullKey=%s", key, fullKey)

	val, err := r.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			logger.Debug("Redis backend: Cache miss, key=%s", key)
			atomic.AddInt64(&r.stats.misses, 1)
			return nil, false, nil
		}
		logger.Error("Redis backend: Get failed, key=%s, error=%v", key, err)
		atomic.AddInt64(&r.stats.errors, 1)
		return nil, false, err
	}

	// 检查是否为空值标记（缓存穿透保护）
	if string(val) == NilMarker {
		logger.Debug("Redis backend: Cache miss (nil marker), key=%s", key)
		atomic.AddInt64(&r.stats.misses, 1)
		return nil, false, nil
	}

	// 反序列化
	var result interface{}
	if err := json.Unmarshal(val, &result); err != nil {
		// 反序列化失败时，记录日志并返回原始字符串
		logger.Warn("Redis backend: failed to unmarshal value for key %s: %v, returning as string", key, err)
		result = string(val)
	}

	logger.Debug("Redis backend: Cache hit, key=%s", key)
	atomic.AddInt64(&r.stats.hits, 1)
	return result, true, nil
}

// Set 设置缓存值
func (r *RedisBackend) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if atomic.LoadInt32(&r.closed) == 1 {
		logger.Debug("Redis backend: Set called on closed backend, key=%s", key)
		return errors.New("RedisBackend is closed")
	}

	logger.Debug("Redis backend: Setting cache key=%s, ttl=%v", key, ttl)

	// 处理空值标记（缓存穿透保护）
	if value == nil {
		value = NilMarker
	}

	// 序列化
	data, err := json.Marshal(value)
	if err != nil {
		logger.Error("Redis backend: Failed to marshal value for key=%s, error=%v", key, err)
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// 标准化 TTL
	normalizedTTL := r.ttlMgr.Normalize(ttl)

	fullKey := r.buildKey(key)
	if err := r.client.Set(ctx, fullKey, data, normalizedTTL).Err(); err != nil {
		logger.Error("Redis backend: Set failed, key=%s, error=%v", key, err)
		atomic.AddInt64(&r.stats.errors, 1)
		return err
	}

	logger.Debug("Redis backend: Cache set successful, key=%s", key)
	atomic.AddInt64(&r.stats.sets, 1)
	return nil
}

// SetWithJitter 设置带随机 TTL 偏移的缓存值（防雪崩）
func (r *RedisBackend) SetWithJitter(ctx context.Context, key string, value interface{}, baseTTL time.Duration, jitterFactor float64) error {
	if jitterFactor <= 0 {
		jitterFactor = 0.1 // 默认 10% 抖动
	}
	if jitterFactor > 0.5 {
		jitterFactor = 0.5 // 最大 50% 抖动
	}

	// 计算随机 TTL
	normalizedTTL := r.ttlMgr.Normalize(baseTTL)
	jitter := time.Duration(float64(normalizedTTL) * jitterFactor)
	
	// 使用安全的随机数生成器生成随机偏移
	randomOffset := time.Duration(rand.Int64N(int64(jitter*2))) - jitter
	
	actualTTL := normalizedTTL + randomOffset
	if actualTTL < time.Second {
		actualTTL = time.Second
	}

	return r.Set(ctx, key, value, actualTTL)
}

// Delete 删除缓存值
func (r *RedisBackend) Delete(ctx context.Context, key string) error {
	if atomic.LoadInt32(&r.closed) == 1 {
		logger.Debug("Redis backend: Delete called on closed backend, key=%s", key)
		return errors.New("RedisBackend is closed")
	}

	logger.Debug("Redis backend: Deleting cache key=%s", key)

	fullKey := r.buildKey(key)
	if err := r.client.Del(ctx, fullKey).Err(); err != nil {
		logger.Error("Redis backend: Delete failed, key=%s, error=%v", key, err)
		atomic.AddInt64(&r.stats.errors, 1)
		return err
	}

	logger.Debug("Redis backend: Cache delete successful, key=%s", key)
	atomic.AddInt64(&r.stats.deletes, 1)
	return nil
}

// Close 关闭 Redis 连接
func (r *RedisBackend) Close() error {
	if !atomic.CompareAndSwapInt32(&r.closed, 0, 1) {
		logger.Debug("Redis backend: Close called on already closed backend")
		return nil // 已经关闭
	}
	logger.Info("Redis backend: Closing Redis connection")
	return r.client.Close()
}

// Stats 获取缓存统计信息
func (r *RedisBackend) Stats() *CacheStats {
	// 异步更新大小信息
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		
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
		Evictions: 0, // Redis 自动淘汰，不单独统计
		Size:      atomic.LoadInt64(&r.stats.size),
		MaxSize:   0, // Redis 取决于内存配置
		HitRate:   hitRate,
	}
}

// Client 获取底层 Redis 客户端（用于高级操作）
func (r *RedisBackend) Client() *redis.Client {
	return r.client
}

// Ping 测试 Redis 连接
func (r *RedisBackend) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Clear 清空所有缓存（危险操作）
func (r *RedisBackend) Clear(ctx context.Context) error {
	if r.config.Prefix != "" {
		// 有前缀时只删除带前缀的 key
		pattern := r.config.Prefix + ":*"
		iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()
		for iter.Next(ctx) {
			if err := r.client.Del(ctx, iter.Val()).Err(); err != nil {
				return err
			}
		}
		return iter.Err()
	}
	
	// 无前缀时清空整个数据库（谨慎使用）
	return r.client.FlushDB(ctx).Err()
}

// 确保实现 CacheBackend 接口
var _ CacheBackend = (*RedisBackend)(nil)

// init 注册 Redis 后端
func init() {
	Register("redis", func(config *CacheConfig) (CacheBackend, error) {
		redisConfig := DefaultRedisConfig()
		redisConfig.DefaultTTL = config.DefaultTTL
		redisConfig.MaxTTL = config.MaxTTL
		return NewRedisBackend(redisConfig)
	})
}
