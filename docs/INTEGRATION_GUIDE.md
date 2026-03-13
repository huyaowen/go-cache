# Go-Cache Framework - 集成指南

## 1. 快速开始（5 分钟上手）

### 第一步：安装

```bash
go get github.com/coderiser/go-cache
go install github.com/coderiser/go-cache/cmd/generator@latest
```

### 第二步：定义服务

```go
package service

import "github.com/yourorg/go-cache"

type UserService struct{}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id string) (*User, error) {
    // 模拟数据库查询
    return db.FindUser(id)
}

// 全局变量（必需）
var UserService = &UserService{}

// init() 注册装饰器（必需）
func init() {
    cache.AutoDecorate(&UserService)
}
```

### 第三步：生成元数据

```bash
# 在项目根目录执行
go generate ./...
```

### 第四步：使用

```go
package main

import (
    "fmt"
    "your-project/service"
)

func main() {
    // 直接调用，缓存自动生效
    user, err := service.UserService.GetUser("123")
    if err != nil {
        panic(err)
    }
    fmt.Println(user.Name)
}
```

**完成！** 🎉 现在你的方法已经具备缓存能力。

## 2. 完整集成步骤

### 2.1 项目初始化

```bash
# 创建项目目录
mkdir my-project && cd my-project

# 初始化 Go 模块
go mod init my-project

# 安装 go-cache
go get github.com/coderiser/go-cache

# 安装代码生成器
go install github.com/coderiser/go-cache/cmd/generator@latest
```

### 2.2 配置文件

创建 `cache.yaml` 配置文件：

```yaml
# cache.yaml
default_cache: memory

caches:
  users:
    backend: memory
    default_ttl: 30m
    max_size: 10000
    
  products:
    backend: redis
    addr: localhost:6379
    password: ""
    db: 0
    prefix: "prod:"
    default_ttl: 1h
    max_ttl: 24h
    pool_size: 10
    min_idle_conns: 5
    dial_timeout: 5s
    read_timeout: 3s
    write_timeout: 3s
    
  sessions:
    backend: redis
    addr: localhost:6379
    db: 1
    prefix: "sessions:"
    default_ttl: 24h
    pool_size: 20
```

### Redis 配置参数说明

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `addr` | string | `localhost:6379` | Redis 服务器地址 |
| `password` | string | `""` | Redis 密码（可选） |
| `db` | int | `0` | Redis 数据库编号 |
| `prefix` | string | `""` | Key 前缀，用于命名空间隔离 |
| `default_ttl` | duration | `30m` | 默认过期时间 |
| `max_ttl` | duration | `24h` | 最大允许 TTL |
| `pool_size` | int | `10` | 连接池大小 |
| `min_idle_conns` | int | `5` | 最小空闲连接数 |
| `dial_timeout` | duration | `5s` | 连接超时 |
| `read_timeout` | duration | `3s` | 读取超时 |
| `write_timeout` | duration | `3s` | 写入超时 |

### 2.3 初始化缓存管理器

```go
package main

import (
    "log"
    "time"
    
    "github.com/coderiser/go-cache/pkg/backend"
    "github.com/coderiser/go-cache/pkg/core"
)

func initCache() core.CacheManager {
    // 创建管理器
    manager := core.NewCacheManager()
    
    // 注册内存后端
    manager.RegisterCache("users", backend.NewMemoryBackend(&backend.MemoryConfig{
        MaxSize:    10000,
        DefaultTTL: 30 * time.Minute,
    }))
    
    // 注册 Redis 后端
    redisBackend, err := backend.NewRedisBackend(&backend.RedisConfig{
        Addr:         "localhost:6379",
        Password:     "",
        DB:           0,
        Prefix:       "prod:",
        DefaultTTL:   1 * time.Hour,
        MaxTTL:       24 * time.Hour,
        PoolSize:     10,
        MinIdleConns: 5,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
    })
    if err != nil {
        log.Fatalf("Failed to create Redis backend: %v", err)
    }
    manager.RegisterCache("products", redisBackend)
    
    // 注册另一个 Redis 后端（不同数据库）
    sessionsBackend, err := backend.NewRedisBackend(&backend.RedisConfig{
        Addr:         "localhost:6379",
        DB:           1,
        Prefix:       "sessions:",
        DefaultTTL:   24 * time.Hour,
        PoolSize:     20,
        MinIdleConns: 10,
    })
    if err != nil {
        log.Fatalf("Failed to create sessions Redis backend: %v", err)
    }
    manager.RegisterCache("sessions", sessionsBackend)
    
    return manager
            DB:       getInt(cfg, "redis_db", 0),
            Prefix:   getString(cfg, "redis_prefix", ""),
            DefaultTTL: getDuration(cfg, "default_ttl", 1*time.Hour),
        }), nil
    })
    
    // 创建缓存实例
    for name, cacheCfg := range cfg.Caches {
        factory, err := manager.GetBackendFactory(cacheCfg.Backend)
        if err != nil {
            log.Fatalf("Unknown backend %q for cache %q", cacheCfg.Backend, name)
        }
        
        backend, err := factory(cacheCfg.ToMap())
        if err != nil {
            log.Fatalf("Failed to create backend for cache %q: %v", name, err)
        }
        
        manager.RegisterCache(name, backend)
    }
    
    return manager
}

func main() {
    manager := initCache()
    
    // 将管理器注入到框架
    cache.SetManager(manager)
    
    // 启动应用
    // ...
}
```

### 2.4 添加 go:generate 指令

在每个包含缓存注解的包中，添加生成指令：

```go
//go:generate go-cache-gen ./...
```

或者在项目根目录创建 `generate.go`：

```go
package main

//go:generate go-cache-gen ./...
```

### 2.5 构建流程

```bash
# 开发时
go generate ./...
go build

# 或者使用 Makefile
make build

# Makefile 示例
.PHONY: generate build

generate:
    go generate ./...

build: generate
    go build -o bin/app ./cmd/app

test: generate
    go test ./...
```

## 3. 注解语法详解

### 3.1 @cacheable - 缓存读取

最常用的注解，用于标记可缓存的方法。

```go
// 基础用法
// @cacheable(cache="users", key="#id", ttl="30m")
func GetUser(id string) (*User, error)

// 带条件检查
// @cacheable(cache="users", key="#id", condition="#id != \"\"")
func GetUser(id string) (*User, error)

// 带结果检查
// @cacheable(cache="users", key="#id", unless="#result == nil")
func GetUser(id string) (*User, error)

// 防缓存击穿（同步刷新）
// @cacheable(cache="hot_data", key="#id", sync=true)
func GetHotData(id string) (*Data, error)

// 复杂 key 表达式
// @cacheable(cache="users", key="#orgId + \":user:\" + #userId", ttl="1h")
func GetUser(orgId, userId string) (*User, error)
```

**执行流程：**

```
1. 检查 condition 表达式（如果为 false，跳过缓存直接执行方法）
2. 生成缓存 key
3. 尝试从缓存获取
4. 如果命中，返回缓存值
5. 如果未命中，执行原始方法
6. 检查 unless 表达式（如果为 true，不缓存结果）
7. 将结果写入缓存
8. 返回结果
```

### 3.2 @cacheput - 缓存更新

总是执行方法，并将结果写入缓存。

```go
// 更新用户信息
// @cacheput(cache="users", key="#user.Id", ttl="30m")
func UpdateUser(user *User) error

// 创建并缓存
// @cacheput(cache="users", key="#result.Id", ttl="30m")
func CreateUser(name string) (*User, error)
```

**适用场景：**
- 数据更新操作
- 需要确保缓存与数据源一致的场景

### 3.3 @cacheevict - 缓存失效

使缓存项失效。

```go
// 删除单个缓存项
// @cacheevict(cache="users", key="#id")
func DeleteUser(id string) error

// 清空整个缓存
// @cacheevict(cache="users", allEntries=true)
func ClearUserCache() error

// 方法执行前失效（适用于删除操作）
// @cacheevict(cache="users", key="#id", beforeInvocation=true)
func DeleteUser(id string) error
```

**适用场景：**
- 数据删除操作
- 批量更新后清空缓存
- 管理后台的缓存刷新功能

## 4. 最佳实践

### 4.1 缓存键设计

**✅ 推荐：**

```go
// 使用有意义的命名空间
// @cacheable(cache="users", key="\"user:\" + #id")

// 包含所有影响结果的参数
// @cacheable(cache="products", key="#categoryId + \":page:\" + #page")

// 使用参数属性
// @cacheable(cache="users", key="#user.Email")
```

**❌ 避免：**

```go
// 过于简单的 key（可能冲突）
// @cacheable(cache="data", key="#id")

// 包含时间戳（永远不会命中）
// @cacheable(cache="data", key="#id + \":\" + time.Now())

// 包含随机值
// @cacheable(cache="data", key="#id + \":\" + uuid.New())
```

### 4.2 TTL 设置

| 数据类型 | 推荐 TTL | 说明 |
|----------|----------|------|
| 用户信息 | 30m - 1h | 频繁访问，变更不频繁 |
| 商品信息 | 5m - 30m | 价格/库存可能变化 |
| 配置数据 | 1h - 24h | 几乎不变 |
| 会话数据 | 30m - 24h | 根据安全要求 |
| 统计报表 | 5m - 1h | 计算成本高 |
| 热点数据 | 1m - 5m | 高并发，防击穿 |

### 4.3 条件使用

```go
// 只缓存非空结果
// @cacheable(cache="users", key="#id", unless="#result == nil")

// 只缓存特定条件的数据
// @cacheable(cache="premium", key="#id", condition="#user.IsPremium")

// 避免缓存错误
// @cacheable(cache="data", key="#id", unless="#err != nil")
```

### 4.4 防缓存击穿

```go
// 对热点数据使用同步刷新
// @cacheable(cache="hot_products", key="#id", sync=true, ttl="5m")

// 或者使用互斥锁
func GetHotProduct(id string) (*Product, error) {
    key := "hot_products:" + id
    if val, found := cache.Get(key); found {
        return val.(*Product), nil
    }
    
    // 获取分布式锁
    mu := lock.Get(key)
    mu.Lock()
    defer mu.Unlock()
    
    // 双重检查
    if val, found := cache.Get(key); found {
        return val.(*Product), nil
    }
    
    product, err := db.FindProduct(id)
    if err != nil {
        return nil, err
    }
    
    cache.Set(key, product, 5*time.Minute)
    return product, nil
}
```

### 4.5 错误处理

```go
// 缓存失败不应影响主流程
func GetUserWithFallback(id string) (*User, error) {
    // 尝试缓存
    if val, found, _ := cache.Get("user:" + id); found {
        return val.(*User), nil
    }
    
    // 缓存未命中或失败，查询数据库
    user, err := db.FindUser(id)
    if err != nil {
        return nil, err
    }
    
    // 异步写入缓存（不阻塞）
    go func() {
        _ = cache.Set("user:"+id, user, 30*time.Minute)
    }()
    
    return user, nil
}
```

### 4.6 监控和指标

```go
// 暴露缓存指标
func RegisterMetrics(manager core.CacheManager) {
    prometheus.MustRegister(prometheus.NewGaugeFunc(
        prometheus.GaugeOpts{
            Name: "cache_hits_total",
            Help: "Total cache hits",
        },
        func() float64 {
            stats := manager.Stats()
            return float64(stats.Hits)
        },
    ))
    
    prometheus.MustRegister(prometheus.NewGaugeFunc(
        prometheus.GaugeOpts{
            Name: "cache_hit_rate",
            Help: "Cache hit rate",
        },
        func() float64 {
            stats := manager.Stats()
            return stats.HitRate
        },
    ))
}
```

## 5. 常见问题 FAQ

### Q1: 为什么我的缓存没有生效？

**检查清单：**

1. ✅ 是否运行了 `go generate ./...`？
2. ✅ `init()` 中是否调用了 `cache.AutoDecorate()`？
3. ✅ 缓存名称是否在配置中定义？
4. ✅ 方法签名是否正确（必须有返回值）？
5. ✅ 是否使用了全局变量（而非局部变量）？

**调试方法：**

```go
// 启用调试日志
cache.SetLogLevel("debug")

// 检查生成的元数据
meta := cache.GetMethodMeta("UserService", "GetUser")
fmt.Printf("Meta: %+v\n", meta)
```

### Q2: 如何处理带多个返回值的方法？

```go
// ✅ 正确：最后一个返回值是 error
// @cacheable(cache="users", key="#id")
func GetUser(id string) (*User, error)

// ✅ 正确：可以忽略 error 进行缓存
// @cacheable(cache="users", key="#id", unless="#err != nil")
func GetUser(id string) (*User, error)

// ❌ 错误：多个非 error 返回值不支持
// @cacheable(cache="data", key="#id")
func GetData(id string) (*Data, int, error)  // 不支持！
```

**解决方案：** 封装成结构体

```go
type DataResult struct {
    Data *Data
    Count int
}

// @cacheable(cache="data", key="#id")
func GetData(id string) (*DataResult, error) {
    data, count, err := db.FindData(id)
    return &DataResult{Data: data, Count: count}, err
}
```

### Q3: 如何缓存泛型方法？

Go 的泛型在反射中有限制，建议：

```go
// ❌ 不推荐：泛型方法
func Get[T any](id string) (*T, error)

// ✅ 推荐：具体类型方法
func GetUser(id string) (*User, error)
func GetProduct(id string) (*Product, error)

// 或者使用接口
// @cacheable(cache="entities", key="#id")
func GetEntity(entityType, id string) (interface{}, error)
```

### Q4: 缓存穿透怎么办？

**方案 1: 缓存空值**

```go
// @cacheable(cache="users", key="#id", ttl="5m")
func GetUser(id string) (*User, error) {
    user, err := db.FindUser(id)
    if err == ErrNotFound {
        // 缓存空标记，防止反复查询
        return nil, ErrNotFound  // 配合 unless 使用
    }
    return user, err
}
```

**方案 2: 布隆过滤器**

```go
func GetUser(id string) (*User, error) {
    // 先检查布隆过滤器
    if !bloomFilter.Contains(id) {
        return nil, ErrNotFound
    }
    
    user, err := db.FindUser(id)
    if err == ErrNotFound {
        bloomFilter.Add(id)  // 记录不存在
    }
    return user, err
}
```

### Q5: 如何测试缓存逻辑？

```go
func TestUserService_GetUser(t *testing.T) {
    // 设置测试缓存
    testCache := memory.New(&memory.Config{MaxSize: 100})
    manager := core.NewCacheManager()
    manager.RegisterCache("users", testCache)
    cache.SetManager(manager)
    
    service := &UserService{}
    
    // 第一次调用（缓存未命中）
    user1, err := service.GetUser("123")
    assert.NoError(t, err)
    assert.Equal(t, 0, testCache.Stats().Hits)
    
    // 第二次调用（缓存命中）
    user2, err := service.GetUser("123")
    assert.NoError(t, err)
    assert.Equal(t, 1, testCache.Stats().Hits)
    assert.Equal(t, user1, user2)
}
```

### Q6: 如何在集群中保持缓存一致？

**方案 1: 使用 Redis 作为共享缓存**

```yaml
caches:
  users:
    backend: redis
    redis_addr: redis-cluster:6379
```

**方案 2: 缓存失效广播**

```go
// 删除时广播失效消息
// @cacheevict(cache="users", key="#id")
func DeleteUser(id string) error {
    err := db.DeleteUser(id)
    if err != nil {
        return err
    }
    
    // 广播到其他节点
    pubsub.Publish("cache_evict", CacheEvictMsg{
        Cache: "users",
        Key:   id,
    })
    
    return nil
}

// 订阅失效消息
func SubscribeEvictions() {
    sub := pubsub.Subscribe("cache_evict")
    for msg := range sub {
        evict := msg.(CacheEvictMsg)
        localCache.Delete(evict.Cache + ":" + evict.Key)
    }
}
```

### Q7: 性能如何？有基准测试吗？

```bash
# 运行基准测试
go test -bench=. -benchmem ./...

# 典型结果（内存后端）
BenchmarkCache_Get_Hit-8        50000000    25.3 ns/op    0 B/op    0 allocs/op
BenchmarkCache_Get_Miss-8       10000000    120 ns/op     32 B/op   1 allocs/op
BenchmarkCache_Set-8            20000000    55.6 ns/op    16 B/op   1 allocs/op

# Redis 后端（本地）
BenchmarkRedis_Get_Hit-8         5000000    250 ns/op     64 B/op   2 allocs/op
BenchmarkRedis_Set-8             3000000    380 ns/op     96 B/op   3 allocs/op
```

### Q8: 支持哪些 Go 版本？

- **最低要求**: Go 1.21+
- **推荐版本**: Go 1.22+
- **测试版本**: Go 1.21, 1.22, 1.23

## 6. 故障排查

### 6.1 常见问题速查

| 问题 | 可能原因 | 解决方案 |
|------|----------|----------|
| 缓存不生效 | 未运行 go generate | 执行 `go generate ./...` |
| 编译错误 | 生成器未安装 | `go install go-cache-gen@latest` |
| 空指针 | 未调用 AutoDecorate | 在 init() 中注册 |
| 缓存键冲突 | key 表达式过于简单 | 添加命名空间前缀 |
| 内存泄漏 | TTL 设置过长或无限 | 设置合理的 ttl |
| Redis 连接失败 | 配置错误 | 检查 redis_addr 和密码 |

### 6.2 调试模式

```go
// 启用详细日志
cache.SetLogLevel("debug")

// 打印缓存操作
cache.EnableTracing(true)

// 获取缓存状态
stats := manager.Stats()
fmt.Printf("Hits: %d, Misses: %d, HitRate: %.2f%%\n", 
    stats.Hits, stats.Misses, stats.HitRate*100)
```

### 6.3 性能分析

```bash
# CPU 分析
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# 内存分析
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof

# 执行跟踪
go test -trace=trace.out -bench=.
go tool trace trace.out
```

## 7. Redis 后端集成

### 7.1 安装依赖

```bash
go get github.com/redis/go-redis/v9
go get golang.org/x/sync
```

### 7.2 创建 Redis 后端

```go
package main

import (
    "log"
    "time"
    
    "github.com/coderiser/go-cache/pkg/backend"
    "github.com/coderiser/go-cache/pkg/core"
)

func initRedis() backend.CacheBackend {
    config := &backend.RedisConfig{
        Addr:         "localhost:6379",
        Password:     "", // 如有密码请设置
        DB:           0,
        Prefix:       "myapp", // Key 前缀
        DefaultTTL:   30 * time.Minute,
        MaxTTL:       24 * time.Hour,
        PoolSize:     10,
        MinIdleConns: 5,
        MaxRetries:   3,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
    }
    
    redisBackend, err := backend.NewRedisBackend(config)
    if err != nil {
        log.Fatalf("Failed to create Redis backend: %v", err)
    }
    
    return redisBackend
}

func main() {
    manager := core.NewCacheManager()
    manager.RegisterCache("users", initRedis())
    
    // 使用缓存...
}
```

### 7.3 连接池配置建议

| 场景 | PoolSize | MinIdleConns | 说明 |
|------|----------|--------------|------|
| 开发环境 | 5 | 2 | 节省资源 |
| 生产环境（低并发） | 10 | 5 | 平衡性能和资源 |
| 生产环境（高并发） | 50-100 | 20-30 | 根据并发量调整 |
| 集群环境 | 100+ | 50+ | 配合 Redis Cluster 使用 |

### 7.4 Docker Compose 示例

```yaml
version: '3.8'

services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes
    
  app:
    build: .
    depends_on:
      - redis
    environment:
      - REDIS_ADDR=redis:6379

volumes:
  redis-data:
```

## 8. 缓存异常保护机制

### 8.1 三种缓存问题

| 问题 | 描述 | 解决方案 |
|------|------|----------|
| **缓存穿透** | 查询不存在的数据，缓存无法命中 | 空值缓存（Nil Marker） |
| **缓存击穿** | 热点 key 过期，并发请求直达数据库 | Singleflight 请求合并 |
| **缓存雪崩** | 大量缓存同时过期 | TTL 随机偏移（Jitter） |

### 8.2 启用保护机制

```go
package service

import (
    "context"
    "time"
    
    "github.com/coderiser/go-cache/pkg/core"
    "github.com/coderiser/go-cache/pkg/backend"
)

type UserService struct {
    cache      backend.CacheBackend
    protection *core.CacheProtection
}

func NewUserService(cache backend.CacheBackend) *UserService {
    // 使用默认配置（三种保护全部启用）
    protection := core.NewCacheProtection(core.DefaultProtectionConfig())
    
    return &UserService{
        cache:      cache,
        protection: protection,
    }
}
```

### 8.3 自定义保护配置

```go
config := &core.ProtectionConfig{
    // 穿透保护
    EnablePenetrationProtection: true,
    EmptyValueTTL:               5 * time.Minute, // 空值缓存 5 分钟
    
    // 击穿保护
    EnableBreakdownProtection:   true,
    
    // 雪崩保护
    EnableAvalancheProtection:   true,
    TTLJitterFactor:             0.1, // 10% 随机偏移
}

protection := core.NewCacheProtection(config)
```

### 8.4 使用受保护的缓存操作

```go
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    key := "user:" + id
    
    result, err := s.protection.ProtectedGet(
        ctx,
        key,
        // 1. 缓存获取
        func() (interface{}, bool, error) {
            return s.cache.Get(ctx, key)
        },
        // 2. 缓存未命中时的数据获取（自动应用击穿保护）
        func() (interface{}, error) {
            return s.db.FindUser(id)
        },
        // 3. 缓存设置（自动应用穿透和雪崩保护）
        func(value interface{}, ttl time.Duration) error {
            return s.cache.Set(ctx, key, value, ttl)
        },
    )
    
    if err != nil {
        return nil, err
    }
    
    return result.(*User), nil
}
```

### 8.5 直接使用 Singleflight

```go
// 合并并发请求
result, err, shared := protection.ApplyBreakdownProtection(ctx, "hot-key", func() (interface{}, error) {
    // 这个函数在并发下只执行一次
    return db.GetHotData("hot-key")
})

if shared {
    log.Println("Result shared from concurrent request")
}
```

### 8.6 TTL 抖动示例

```go
baseTTL := 30 * time.Minute

// 应用默认抖动（10%）
actualTTL := protection.ApplyAvalancheProtection(baseTTL)
// 结果：27-33 分钟之间

// 自定义抖动因子（20%）
actualTTL = protection.CalculateTTLWithJitter(baseTTL, 0.2)
// 结果：24-36 分钟之间
```

### 8.7 生产环境最佳实践

```go
// 完整的生产环境配置示例
func NewProductionCache() (*UserService, error) {
    // 1. 创建 Redis 后端
    redisConfig := &backend.RedisConfig{
        Addr:         "redis-cluster:6379",
        Password:     os.Getenv("REDIS_PASSWORD"),
        PoolSize:     50,
        MinIdleConns: 20,
        MaxRetries:   3,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
    }
    
    redisBackend, err := backend.NewRedisBackend(redisConfig)
    if err != nil {
        return nil, err
    }
    
    // 2. 创建保护机制
    protectionConfig := &core.ProtectionConfig{
        EnablePenetrationProtection: true,
        EmptyValueTTL:               5 * time.Minute,
        EnableBreakdownProtection:   true,
        EnableAvalancheProtection:   true,
        TTLJitterFactor:             0.1,
    }
    protection := core.NewCacheProtection(protectionConfig)
    
    // 3. 创建服务
    return &UserService{
        cache:      redisBackend,
        protection: protection,
        db:         NewDatabase(),
    }, nil
}
```

## 9. P2 高级功能集成

### 9.1 代码生成器集成

**方式 1: 手动运行**
```bash
go-cache-gen ./...
```

**方式 2: go generate 集成**
```go
// 在包级别添加
//go:generate go-cache-gen ./...

package service
```

然后运行:
```bash
go generate ./...
```

**方式 3: Makefile 集成**
```makefile
.PHONY: generate
generate:
	go generate ./...
	go-cache-gen ./...

build: generate
	go build -o app .
```

### 9.2 一行初始化（推荐）

```go
package service

import (
	"github.com/coderiser/go-cache/pkg/proxy"
)

type UserService struct {
	db *gorm.DB
}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id string) (*User, error) {
	var u User
	err := s.db.Where("id = ?", id).First(&u).Error
	return &u, err
}

// 一行代码完成初始化！
var UserService = proxy.SimpleDecorate(&UserService{})
```

### 9.3 多级缓存集成

```go
package main

import (
	"time"
	"github.com/coderiser/go-cache/pkg/backend"
	"github.com/coderiser/go-cache/pkg/core"
	"github.com/coderiser/go-cache/pkg/proxy"
)

func init() {
	// L1: Memory (快速)
	memoryBackend := backend.NewMemoryBackend(&backend.MemoryConfig{
		MaxSize:    1000,
		DefaultTTL: 5 * time.Minute,
	})
	
	// L2: Redis (持久)
	redisBackend, _ := backend.NewRedisBackend(&backend.RedisConfig{
		Addr:       "localhost:6379",
		DefaultTTL: 30 * time.Minute,
	})
	
	// 创建混合后端
	hybridBackend := backend.NewHybridBackend(
		"users",
		memoryBackend,
		redisBackend,
		&backend.HybridConfig{
			AsyncL2Write: true,
		},
	)
	
	// 注册到管理器
	manager := core.NewCacheManager()
	manager.RegisterCache("users", hybridBackend)
	
	// 装饰服务
	UserService = proxy.SimpleDecorateWithManager(&UserService{}, manager).(*UserService)
}
```

### 9.4 监控集成

**Prometheus 指标**:
```go
package main

import (
	"net/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/coderiser/go-cache/pkg/core"
)

func main() {
	manager := core.NewCacheManager()
	manager.EnableMetrics()
	
	// 暴露 /metrics 端点
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)
}
```

**Grafana 仪表盘**:
```json
{
  "dashboard": {
    "title": "Go-Cache Metrics",
    "panels": [
      {
        "title": "Cache Hit Rate",
        "targets": [
          {
            "expr": "rate(go_cache_hits_total[5m]) / (rate(go_cache_hits_total[5m]) + rate(go_cache_misses_total[5m]))"
          }
        ]
      }
    ]
  }
}
```

### 9.5 分布式追踪集成

```go
package main

import (
	"go.opentelemetry.io/otel"
	"github.com/coderiser/go-cache/pkg/core"
	"github.com/coderiser/go-cache/pkg/tracing"
)

func init() {
	manager := core.NewCacheManager()
	
	// 启用 OpenTelemetry 追踪
	tracer := otel.Tracer("go-cache")
	wrapper := tracing.NewOpenTelemetryWrapper(tracer)
	manager.SetTracingWrapper(wrapper)
}
```

**Jaeger 查询示例**:
```
# 查找所有缓存未命中
{service.name="my-app"} | cache.operation="get" | cache.hit=false

# 查看慢查询
{service.name="my-app"} | cache.duration > 10ms
```

### 9.6 完整生产配置

```go
func NewProductionService() (*UserService, error) {
	// 1. 创建 Redis 后端
	redisBackend, err := backend.NewRedisBackend(&backend.RedisConfig{
		Addr:         os.Getenv("REDIS_ADDR"),
		Password:     os.Getenv("REDIS_PASSWORD"),
		PoolSize:     50,
		MinIdleConns: 20,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	
	// 2. 创建管理器
	manager := core.NewCacheManager()
	manager.RegisterCache("users", redisBackend)
	
	// 3. 启用监控
	manager.EnableMetrics()
	manager.EnableTracing(otel.Tracer("go-cache"))
	
	// 4. 配置保护机制
	protection := core.NewCacheProtection(&core.ProtectionConfig{
		EnablePenetrationProtection: true,
		EmptyValueTTL:               5 * time.Minute,
		EnableBreakdownProtection:   true,
		EnableAvalancheProtection:   true,
		TTLJitterFactor:             0.1,
	})
	manager.SetProtection(protection)
	
	// 5. 装饰服务
	decorated := proxy.SimpleDecorateWithManager(&UserService{}, manager)
	return decorated.(*UserService), nil
}
```

## 10. 下一步

- 📖 阅读 [ARCHITECTURE.md](ARCHITECTURE.md) 了解架构设计
- 📐 阅读 [INTERFACE_SPEC.md](INTERFACE_SPEC.md) 查看完整接口定义
- 🚀 阅读 [P2_FEATURES.md](P2_FEATURES.md) 了解 P2 新增功能
- 🔧 查看示例项目：`github.com/coderiser/go-cache/examples`
- 💬 加入社区：`github.com/coderiser/go-cache/discussions`

---

**文档版本**: 2.0 (P2)  
**最后更新**: 2026-03-13
