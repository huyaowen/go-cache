package backend

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheInvalidator 缓存失效广播器
type CacheInvalidator struct {
	client   *redis.Client
	pubsub   *redis.PubSub
	channel  string
	closed   int32
	mu       sync.RWMutex
	handlers []func(key string)
}

// CacheInvalidatorConfig 缓存失效广播器配置
type CacheInvalidatorConfig struct {
	Addr        string        // Redis 地址
	Password    string        // Redis 密码
	DB          int           // Redis 数据库
	Channel     string        // Pub/Sub 频道名称
	DialTimeout time.Duration // 连接超时
	ReadTimeout time.Duration // 读取超时
}

// DefaultCacheInvalidatorConfig 默认配置
func DefaultCacheInvalidatorConfig() *CacheInvalidatorConfig {
	return &CacheInvalidatorConfig{
		Addr:        "localhost:6379",
		Password:    "",
		DB:          0,
		Channel:     "go-cache:invalidation",
		DialTimeout: 5 * time.Second,
		ReadTimeout: 3 * time.Second,
	}
}

// NewCacheInvalidator 创建缓存失效广播器
func NewCacheInvalidator(config *CacheInvalidatorConfig) (*CacheInvalidator, error) {
	if config == nil {
		config = DefaultCacheInvalidatorConfig()
	}

	if config.Addr == "" {
		return nil, fmt.Errorf("redis address is required")
	}

	if config.Channel == "" {
		config.Channel = "go-cache:invalidation"
	}

	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	pubsub := client.Subscribe(ctx, config.Channel)

	// 等待订阅确认
	subCtx, subCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer subCancel()

	_, err := pubsub.Receive(subCtx)
	if err != nil {
		pubsub.Close()
		client.Close()
		return nil, fmt.Errorf("failed to subscribe to channel: %w", err)
	}

	return &CacheInvalidator{
		client:   client,
		pubsub:   pubsub,
		channel:  config.Channel,
		handlers: make([]func(key string), 0),
	}, nil
}

// Broadcast 广播缓存失效消息
func (ci *CacheInvalidator) Broadcast(key string) error {
	if atomic.LoadInt32(&ci.closed) == 1 {
		return fmt.Errorf("CacheInvalidator is closed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := &InvalidationMessage{
		Key:       key,
		Timestamp: time.Now().UnixNano(),
	}

	data, err := message.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return ci.client.Publish(ctx, ci.channel, string(data)).Err()
}

// BroadcastWithCache 广播带缓存名称的失效消息
func (ci *CacheInvalidator) BroadcastWithCache(cacheName, key string) error {
	if atomic.LoadInt32(&ci.closed) == 1 {
		return fmt.Errorf("CacheInvalidator is closed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := &InvalidationMessage{
		CacheName: cacheName,
		Key:       key,
		Timestamp: time.Now().UnixNano(),
	}

	data, err := message.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return ci.client.Publish(ctx, ci.channel, string(data)).Err()
}

// Subscribe 订阅缓存失效消息（阻塞式）
func (ci *CacheInvalidator) Subscribe(callback func(key string)) {
	ci.mu.Lock()
	ci.handlers = append(ci.handlers, callback)
	ci.mu.Unlock()

	ch := ci.pubsub.Channel()
	for msg := range ch {
		if atomic.LoadInt32(&ci.closed) == 1 {
			return
		}

		message, err := UnmarshalInvalidationMessage([]byte(msg.Payload))
		if err != nil {
			// 解析失败，尝试直接使用 payload 作为 key
			callback(msg.Payload)
			continue
		}

		callback(message.Key)
	}
}

// SubscribeWithContext 带上下文的订阅（可取消）
func (ci *CacheInvalidator) SubscribeWithContext(ctx context.Context, callback func(key string)) error {
	ci.mu.Lock()
	ci.handlers = append(ci.handlers, callback)
	ci.mu.Unlock()

	ch := ci.pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return fmt.Errorf("subscription channel closed")
			}
			if atomic.LoadInt32(&ci.closed) == 1 {
				return fmt.Errorf("CacheInvalidator is closed")
			}

			message, err := UnmarshalInvalidationMessage([]byte(msg.Payload))
			if err != nil {
				callback(msg.Payload)
				continue
			}

			callback(message.Key)
		}
	}
}

// OnInvalidation 注册失效回调（非阻塞）
func (ci *CacheInvalidator) OnInvalidation(handler func(key string)) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	ci.handlers = append(ci.handlers, handler)
}

// Close 关闭广播器
func (ci *CacheInvalidator) Close() error {
	if !atomic.CompareAndSwapInt32(&ci.closed, 0, 1) {
		return nil
	}
	return ci.pubsub.Close()
}

// IsClosed 检查是否已关闭
func (ci *CacheInvalidator) IsClosed() bool {
	return atomic.LoadInt32(&ci.closed) == 1
}

// Channel 获取频道名称
func (ci *CacheInvalidator) Channel() string {
	return ci.channel
}

// InvalidationMessage 失效消息
type InvalidationMessage struct {
	CacheName string `json:"cache_name"`
	Key       string `json:"key"`
	Timestamp int64  `json:"timestamp"`
}

// Marshal 序列化消息
func (m *InvalidationMessage) Marshal() ([]byte, error) {
	// 简单序列化，避免循环依赖
	if m.CacheName != "" {
		return []byte(fmt.Sprintf("%s:%s", m.CacheName, m.Key)), nil
	}
	return []byte(m.Key), nil
}

// UnmarshalInvalidationMessage 反序列化消息
func UnmarshalInvalidationMessage(data []byte) (*InvalidationMessage, error) {
	payload := string(data)

	// 尝试解析 cache:key 格式（从后往前找第一个冒号）
	for i := len(payload) - 1; i >= 0; i-- {
		if payload[i] == ':' && i > 0 {
			// 检查冒号后面是否是纯数字（时间戳）
			isTimestamp := true
			for j := i + 1; j < len(payload); j++ {
				if payload[j] < '0' || payload[j] > '9' {
					isTimestamp = false
					break
				}
			}
			if isTimestamp && len(payload[i+1:]) > 0 {
				// 格式：cache:key:timestamp，继续往前找 cache:key 分隔符
				for k := i - 1; k >= 0; k-- {
					if payload[k] == ':' {
						return &InvalidationMessage{
							CacheName: payload[:k],
							Key:       payload[k+1:i],
							Timestamp: parseInt64(payload[i+1:]),
						}, nil
					}
				}
				// 只有 key:timestamp 格式
				return &InvalidationMessage{
					Key:       payload[:i],
					Timestamp: parseInt64(payload[i+1:]),
				}, nil
			} else {
				// 格式：cache:key（没有时间戳）
				return &InvalidationMessage{
					CacheName: payload[:i],
					Key:       payload[i+1:],
					Timestamp: time.Now().UnixNano(),
				}, nil
			}
		}
	}

	// 简单格式：直接返回 key
	return &InvalidationMessage{
		Key:       payload,
		Timestamp: time.Now().UnixNano(),
	}, nil
}

func parseInt64(s string) int64 {
	var result int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		result = result*10 + int64(c-'0')
	}
	return result
}
