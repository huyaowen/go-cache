package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// CacheConfig 单个缓存配置
type CacheConfig struct {
	Backend    string        `yaml:"backend"`              // 后端类型：memory, redis
	Addr       string        `yaml:"addr"`                 // Redis 地址
	Password   string        `yaml:"password"`             // Redis 密码
	DB         int           `yaml:"db"`                   // Redis 数据库
	MaxSize    int64         `yaml:"max_size"`             // 内存后端最大条目数
	DefaultTTL time.Duration `yaml:"default_ttl"`          // 默认 TTL
	MaxTTL     time.Duration `yaml:"max_ttl"`              // 最大 TTL
	Prefix     string        `yaml:"prefix"`               // Key 前缀
}

// Config 根配置
type Config struct {
	Caches map[string]*CacheConfig `yaml:"caches"` // 缓存配置映射
}

// Load 从 YAML 文件加载配置
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// 应用默认值
	for name, cache := range cfg.Caches {
		if cache == nil {
			cache = &CacheConfig{}
			cfg.Caches[name] = cache
		}
		if cache.Backend == "" {
			cache.Backend = "memory"
		}
		if cache.DefaultTTL == 0 {
			cache.DefaultTTL = 30 * time.Minute
		}
		if cache.MaxTTL == 0 {
			cache.MaxTTL = 24 * time.Hour
		}
		if cache.Backend == "memory" && cache.MaxSize == 0 {
			cache.MaxSize = 10000
		}
		_ = name // 可用于日志
	}

	return &cfg, nil
}

// LoadFromString 从 YAML 字符串加载配置（用于测试）
func LoadFromString(data string) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal([]byte(data), &cfg); err != nil {
		return nil, err
	}

	// 应用默认值
	for name, cache := range cfg.Caches {
		if cache == nil {
			cache = &CacheConfig{}
			cfg.Caches[name] = cache
		}
		if cache.Backend == "" {
			cache.Backend = "memory"
		}
		if cache.DefaultTTL == 0 {
			cache.DefaultTTL = 30 * time.Minute
		}
		if cache.MaxTTL == 0 {
			cache.MaxTTL = 24 * time.Hour
		}
		if cache.Backend == "memory" && cache.MaxSize == 0 {
			cache.MaxSize = 10000
		}
		_ = name
	}

	return &cfg, nil
}
