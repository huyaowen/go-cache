# Go-Cache Framework - 接口规范文档

## 1. 概述

本文档定义 Go-Cache Framework 的所有核心接口、数据结构和使用规范。

## 2. 核心接口定义

### 2.1 CacheBackend

缓存后端接口，定义缓存存储的基本操作。

```go
package core

import (
    "context"
    "time"
)

// CacheStats 缓存统计信息
type CacheStats struct {
    Hits      int64   `json:"hits"`       // 命中次数
    Misses    int64   `json:"misses"`     // 未命中次数
    Sets      int64   `json:"sets"`       // 设置次数
    Deletes   int64   `json:"deletes"`    // 删除次数
    Evictions int64   `json:"evictions"`  // 驱逐次数
    Size      int64   `json:"size"`       // 当前缓存项数量
    HitRate   float64 `json:"hit_rate"`   // 命中率
}

// CacheBackend 缓存后端接口
type CacheBackend interface {
    // Get 获取缓存项
    // 返回：value(缓存值), found(是否找到), err(错误)
    Get(ctx context.Context, key string) (interface{}, bool, error)
    
    // Set 设置缓存项
    // ttl: 过期时间，0 表示永不过期
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    
    // Delete 删除缓存项
    // 返回：deleted(是否成功删除), err(错误)
    Delete(ctx context.Context, key string) error
    
    // Close 关闭缓存连接，释放资源
    Close() error
    
    // Stats 获取缓存统计信息
    Stats() *CacheStats
    
    // Clear 清空所有缓存项
    Clear() error
}
```

### 2.2 CacheManager

缓存管理器，管理多个缓存实例和缓存执行逻辑。

```go
package core

import (
    "context"
    "reflect"
)

// BackendFactory 后端工厂函数
type BackendFactory func(config map[string]interface{}) (CacheBackend, error)

// CacheManager 缓存管理器接口
type CacheManager interface {
    // GetCache 获取命名缓存实例
    // 如果缓存不存在则返回错误
    GetCache(name string) (CacheBackend, error)
    
    // RegisterBackend 注册缓存后端工厂
    // name: 后端名称 (如 "memory", "redis")
    // factory: 工厂函数
    RegisterBackend(name string, factory BackendFactory) error
    
    // Execute 执行缓存逻辑
    // meta: 方法元数据（包含注解信息）
    // args: 方法参数
    // 返回：结果，错误
    Execute(ctx context.Context, meta *MethodMeta, args []reflect.Value) (interface{}, error)
    
    // Evict 使缓存失效
    // cache: 缓存名称
    // key: 缓存键（支持通配符）
    Evict(cache string, key string) error
    
    // EvictAll 使整个缓存失效
    EvictAll(cache string) error
}
```

### 2.3 MethodMeta

方法元数据，由编译时代码生成器生成。

```go
package core

// CacheAnnotation 缓存注解配置
type CacheAnnotation struct {
    Type       string            `json:"type"`        // 注解类型：cacheable, cacheput, cacheevict
    Cache      string            `json:"cache"`       // 缓存名称
    Key        string            `json:"key"`         // 缓存键表达式
    Ttl        string            `json:"ttl"`         // 过期时间 (如 "30m", "1h")
    Condition  string            `json:"condition"`   // 条件表达式 (执行前检查)
    Unless     string            `json:"unless"`      // 除非表达式 (执行后检查)
    Sync       bool              `json:"sync"`        // 是否同步刷新 (防击穿)
}

// MethodMeta 方法元数据
type MethodMeta struct {
    TypeName   string           `json:"type_name"`   // 类型名称
    MethodName string           `json:"method_name"` // 方法名称
    Annotation *CacheAnnotation `json:"annotation"`  // 缓存注解
    ParamNames []string         `json:"param_names"` // 参数名称列表
    ResultType reflect.Type     `json:"result_type"` // 返回类型
}
```

### 2.4 ExpressionEvaluator

表达式求值器接口。

```go
package expression

// EvalContext 表达式求值上下文
type EvalContext struct {
    Params   map[string]interface{} `json:"params"`    // 方法参数
    Result   interface{}            `json:"result"`    // 方法返回值 (仅 unless 使用)
    Error    error                  `json:"error"`     // 方法错误
}

// ExpressionEvaluator 表达式求值器接口
type ExpressionEvaluator interface {
    // Eval 求值表达式
    // expr: 表达式字符串 (如 "#id", "#result != nil")
    // ctx: 求值上下文
    // 返回：结果值，错误
    Eval(expr string, ctx *EvalContext) (interface{}, error)
    
    // EvalBool 求值布尔表达式
    EvalBool(expr string, ctx *EvalContext) (bool, error)
    
    // EvalString 求值字符串表达式 (用于 key 生成)
    EvalString(expr string, ctx *EvalContext) (string, error)
}
```

## 3. 数据结构定义

### 3.1 缓存配置

```go
package config

// CacheConfig 缓存配置
type CacheConfig struct {
    // 基础配置
    Name         string            `yaml:"name"`          // 缓存名称
    Backend      string            `yaml:"backend"`       // 后端类型
    DefaultTTL   time.Duration     `yaml:"default_ttl"`   // 默认过期时间
    
    // 内存后端配置
    MaxSize      int               `yaml:"max_size"`      // 最大缓存项数
    CleanupInterval time.Duration  `yaml:"cleanup_interval"` // 清理间隔
    
    // Redis 后端配置
    RedisAddr    string            `yaml:"redis_addr"`    // Redis 地址
    RedisPassword string           `yaml:"redis_password"`// Redis 密码
    RedisDB      int               `yaml:"redis_db"`      // Redis 数据库
    RedisPrefix  string            `yaml:"redis_prefix"`  // 键前缀
    
    // 高级配置
    StatsEnabled bool              `yaml:"stats_enabled"` // 启用统计
    MetricsPort  int               `yaml:"metrics_port"`  // 指标端口
}

// GlobalConfig 全局配置
type GlobalConfig struct {
    Caches       map[string]*CacheConfig `yaml:"caches"`
    DefaultCache string                  `yaml:"default_cache"`
}
```

### 3.2 缓存项

```go
package core

// CacheItem 缓存项（内部使用）
type CacheItem struct {
    Value      interface{} `json:"value"`       // 缓存值
    ExpireAt   time.Time   `json:"expire_at"`   // 过期时间
    CreatedAt  time.Time   `json:"created_at"`  // 创建时间
    AccessCount int64      `json:"access_count"`// 访问次数
}

// IsExpired 检查是否过期
func (item *CacheItem) IsExpired() bool {
    if item.ExpireAt.IsZero() {
        return false
    }
    return time.Now().After(item.ExpireAt)
}
```

## 4. 错误处理规范

### 4.1 错误类型定义

```go
package errors

import "errors"

var (
    // 缓存未命中
    ErrCacheMiss = errors.New("cache: key not found")
    
    // 缓存未注册
    ErrCacheNotRegistered = errors.New("cache: not registered")
    
    // 后端未注册
    ErrBackendNotRegistered = errors.New("cache: backend not registered")
    
    // 注解解析错误
    ErrAnnotationParse = errors.New("cache: failed to parse annotation")
    
    // 表达式求值错误
    ErrExpressionEval = errors.New("cache: failed to evaluate expression")
    
    // 序列化错误
    ErrSerialization = errors.New("cache: serialization failed")
    
    // 配置错误
    ErrInvalidConfig = errors.New("cache: invalid configuration")
)

// CacheError 缓存错误（带上下文）
type CacheError struct {
    Op      string `json:"op"`       // 操作类型
    Key     string `json:"key"`      // 缓存键
    Cache   string `json:"cache"`    // 缓存名称
    Err     error  `json:"err"`      // 原始错误
}

func (e *CacheError) Error() string {
    return fmt.Sprintf("cache: %s failed for key=%q in cache=%q: %v", 
        e.Op, e.Key, e.Cache, e.Err)
}

func (e *CacheError) Unwrap() error {
    return e.Err
}
```

### 4.2 错误处理策略

```go
package example

import (
    "context"
    "github.com/yourorg/go-cache/core"
    "github.com/yourorg/go-cache/errors"
)

// 示例：安全的缓存操作
func safeGet(ctx context.Context, manager core.CacheManager, key string) (interface{}, error) {
    cache, err := manager.GetCache("users")
    if err != nil {
        // 缓存未配置，降级处理
        return nil, errors.ErrCacheNotRegistered
    }
    
    value, found, err := cache.Get(ctx, key)
    if err != nil {
        if errors.Is(err, errors.ErrCacheMiss) {
            // 缓存未命中是正常情况
            return nil, nil
        }
        // 其他错误需要记录日志
        log.Printf("cache get error: %v", err)
        return nil, err
    }
    
    if !found {
        return nil, nil
    }
    
    return value, nil
}
```

## 5. 使用示例

### 5.1 基础使用

```go
package main

import (
    "context"
    "time"
    
    "github.com/yourorg/go-cache/core"
    "github.com/yourorg/go-cache/backend/memory"
)

func main() {
    // 创建内存后端
    backend := memory.New(&memory.Config{
        MaxSize: 10000,
        DefaultTTL: 30 * time.Minute,
    })
    
    // 创建缓存管理器
    manager := core.NewCacheManager()
    manager.RegisterBackend("memory", func(cfg map[string]interface{}) (core.CacheBackend, error) {
        return backend, nil
    })
    
    // 使用缓存
    ctx := context.Background()
    
    // Set
    backend.Set(ctx, "user:123", &User{Name: "Alice"}, 30*time.Minute)
    
    // Get
    value, found, _ := backend.Get(ctx, "user:123")
    if found {
        user := value.(*User)
        println(user.Name)
    }
    
    // Delete
    backend.Delete(ctx, "user:123")
    
    // Stats
    stats := backend.Stats()
    println("Hit rate:", stats.HitRate)
}
```

### 5.2 自定义后端

```go
package redis

import (
    "context"
    "encoding/json"
    "time"
    
    "github.com/redis/go-redis/v9"
    "github.com/yourorg/go-cache/core"
)

// RedisBackend Redis 缓存后端实现
type RedisBackend struct {
    client *redis.Client
    prefix string
    stats  core.CacheStats
}

// NewRedisBackend 创建 Redis 后端
func NewRedisBackend(addr, password string, db int) *RedisBackend {
    return &RedisBackend{
        client: redis.NewClient(&redis.Options{
            Addr:     addr,
            Password: password,
            DB:       db,
        }),
        prefix: "cache:",
    }
}

func (b *RedisBackend) Get(ctx context.Context, key string) (interface{}, bool, error) {
    data, err := b.client.Get(ctx, b.prefix+key).Bytes()
    if err == redis.Nil {
        b.stats.Misses++
        return nil, false, nil
    }
    if err != nil {
        return nil, false, err
    }
    
    b.stats.Hits++
    
    var value interface{}
    err = json.Unmarshal(data, &value)
    return value, true, err
}

func (b *RedisBackend) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    
    b.stats.Sets++
    return b.client.Set(ctx, b.prefix+key, data, ttl).Err()
}

func (b *RedisBackend) Delete(ctx context.Context, key string) error {
    b.stats.Deletes++
    return b.client.Del(ctx, b.prefix+key).Err()
}

func (b *RedisBackend) Close() error {
    return b.client.Close()
}

func (b *RedisBackend) Stats() *core.CacheStats {
    stats := b.stats
    total := stats.Hits + stats.Misses
    if total > 0 {
        stats.HitRate = float64(stats.Hits) / float64(total)
    }
    return &stats
}

func (b *RedisBackend) Clear() error {
    return b.client.FlushDB(context.Background()).Err()
}
```

### 5.3 注解使用

```go
package service

import (
    "github.com/yourorg/go-cache"
)

type UserService struct{}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id string) (*User, error) {
    return db.FindUser(id)
}

// @cacheable(cache="users", key="#username", ttl="1h", condition="#username != \"\"")
func (s *UserService) GetUserByName(username string) (*User, error) {
    return db.FindUserByName(username)
}

// @cacheput(cache="users", key="#user.Id", ttl="30m")
func (s *UserService) UpdateUser(user *User) error {
    return db.UpdateUser(user)
}

// @cacheevict(cache="users", key="#id")
func (s *UserService) DeleteUser(id string) error {
    return db.DeleteUser(id)
}

// @cacheevict(cache="users", allEntries=true)
func (s *UserService) ClearUserCache() error {
    return nil
}

func init() {
    cache.AutoDecorate(&UserService)
}
```

## 6. 注解语法详解

### 6.1 @cacheable

用于标记可缓存的方法。

```go
// @cacheable(cache="cache_name", key="#param", ttl="30m")
// @cacheable(cache="users", key="#id", ttl="1h", condition="#id != \"\"", unless="#result == nil")
// @cacheable(cache="data", key="#type + \":\" + #id", sync=true)
```

**参数说明：**

| 参数 | 必填 | 说明 | 示例 |
|------|------|------|------|
| cache | 是 | 缓存名称 | `cache="users"` |
| key | 是 | 缓存键表达式 | `key="#id"` |
| ttl | 否 | 过期时间 | `ttl="30m"`, `ttl="1h"` |
| condition | 否 | 执行前条件 | `condition="#id > 0"` |
| unless | 否 | 执行后条件 | `unless="#result == nil"` |
| sync | 否 | 同步刷新 | `sync=true` |

### 6.2 @cacheput

用于更新缓存（总是执行方法并更新缓存）。

```go
// @cacheput(cache="users", key="#user.Id", ttl="30m")
```

### 6.3 @cacheevict

用于使缓存失效。

```go
// @cacheevict(cache="users", key="#id")
// @cacheevict(cache="users", allEntries=true)
// @cacheevict(cache="users", key="#id", beforeInvocation=true)
```

**参数说明：**

| 参数 | 必填 | 说明 | 示例 |
|------|------|------|------|
| cache | 是 | 缓存名称 | `cache="users"` |
| key | 否 | 缓存键 | `key="#id"` |
| allEntries | 否 | 清空整个缓存 | `allEntries=true` |
| beforeInvocation | 否 | 方法执行前失效 | `beforeInvocation=true` |

## 7. 表达式语法

### 7.1 参数引用

```
#id              // 引用名为 id 的参数
#user.Name       // 引用参数的属性
#ctx.Value("key") // 引用上下文值
```

### 7.2 字符串拼接

```
#type + ":" + #id
"user:" + #id
```

### 7.3 条件表达式

```
#id != ""
#result != nil
#user.Age > 18
#result == nil || #result.Error != nil
```

### 7.4 内置函数

```
#root          // 根对象（返回值）
#this          // 当前服务对象
T(time).Now()  // 调用静态方法
```
