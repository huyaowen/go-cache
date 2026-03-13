package backend

import (
	"context"
	"time"
)

// NilMarker 空值标记，用于缓存穿透保护（导出供其他包使用）
const NilMarker = "__GO_CACHE_NIL__"

// CacheBackend 缓存后端接口（本地定义避免循环导入）
type CacheBackend interface {
	Get(ctx context.Context, key string) (interface{}, bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Close() error
	Stats() *CacheStats
}

// CacheStats 缓存统计
type CacheStats struct {
	Hits, Misses, Sets, Deletes, Evictions, Size, MaxSize int64
	HitRate                                               float64
}

// BackendFactory 后端工厂函数类型
type BackendFactory func(config *CacheConfig) (CacheBackend, error)

// CacheConfig 缓存配置
type CacheConfig struct {
	Name           string
	MaxSize        int64
	DefaultTTL     time.Duration
	MaxTTL         time.Duration
	EvictionPolicy string
}

// BackendRegistry 后端注册表
var BackendRegistry = make(map[string]BackendFactory)

// Register 注册缓存后端实现
func Register(name string, factory BackendFactory) {
	BackendRegistry[name] = factory
}

// GetFactory 获取后端工厂
func GetFactory(name string) (BackendFactory, bool) {
	factory, ok := BackendRegistry[name]
	return factory, ok
}

// ValidateConfig 验证配置
func ValidateConfig(config *CacheConfig) error {
	if config.Name == "" {
		return ErrEmptyName
	}
	if config.MaxSize <= 0 {
		return ErrInvalidMaxSize
	}
	return nil
}

// DefaultCacheConfig 默认配置
func DefaultCacheConfig(name string) *CacheConfig {
	return &CacheConfig{
		Name:           name,
		MaxSize:        10000,
		DefaultTTL:     30 * time.Minute,
		MaxTTL:         24 * time.Hour,
		EvictionPolicy: "lru",
	}
}

// 错误定义
type BackendError struct {
	Code    string
	Message string
}

func (e *BackendError) Error() string {
	return e.Message
}

var (
	ErrEmptyName      = &BackendError{Code: "EMPTY_NAME", Message: "缓存名称不能为空"}
	ErrInvalidMaxSize = &BackendError{Code: "INVALID_MAX_SIZE", Message: "最大容量必须大于 0"}
)

// KeyBuilder 键构建器
type KeyBuilder interface {
	Build(parts ...string) string
}

// DefaultKeyBuilder 默认键构建器
type DefaultKeyBuilder struct {
	Separator string
	Prefix    string
}

func NewDefaultKeyBuilder(separator, prefix string) *DefaultKeyBuilder {
	if separator == "" {
		separator = ":"
	}
	return &DefaultKeyBuilder{Separator: separator, Prefix: prefix}
}

func (kb *DefaultKeyBuilder) Build(parts ...string) string {
	result := kb.Prefix
	for i, part := range parts {
		if i > 0 || kb.Prefix != "" {
			result += kb.Separator
		}
		result += part
	}
	return result
}

// TTLManager TTL 管理器
type TTLManager struct {
	DefaultTTL, MaxTTL time.Duration
}

func NewTTLManager(defaultTTL, maxTTL time.Duration) *TTLManager {
	return &TTLManager{DefaultTTL: defaultTTL, MaxTTL: maxTTL}
}

func (tm *TTLManager) Normalize(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return tm.DefaultTTL
	}
	if tm.MaxTTL > 0 && ttl > tm.MaxTTL {
		return tm.MaxTTL
	}
	return ttl
}
