# Go-Cache Framework

[![Go Reference](https://pkg.go.dev/badge/github.com/coderiser/go-cache.svg)](https://pkg.go.dev/github.com/coderiser/go-cache)
[![Go Report Card](https://goreportcard.com/badge/github.com/coderiser/go-cache)](https://goreportcard.com/report/github.com/coderiser/go-cache)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> **Go 语言注解式缓存框架** —— 类似 Spring Cache 的优雅缓存解决方案

## 🚀 特性

- ✨ **注解式缓存** - 通过简单注解实现缓存逻辑，业务代码零侵入
- 🔧 **自动装饰** - 运行时自动代理，无需手动包装
- 🎯 **SpEL 表达式** - 强大的动态缓存 Key 生成（基于 `expr` 引擎）
- 💾 **多后端支持** - Memory / Redis / Hybrid 可插拔切换
- 📊 **缓存统计** - 内置命中率、延迟等指标（Prometheus）
- 🔍 **分布式追踪** - OpenTelemetry 集成
- 🛡️ **异常保护** - 穿透/击穿/雪崩保护机制（空值缓存、Singleflight、TTL 抖动）
- 📝 **代码生成** - `go-cache-gen` CLI 自动生成元数据
- 🚀 **一行初始化** - `proxy.SimpleDecorate(&Service{})` 完成所有配置

## 📦 快速开始

### 1. 安装

```bash
go get github.com/coderiser/go-cache
```

### 2. 定义服务接口和实现

```go
package service

// 1. 定义接口（接口模式的关键）
type UserServiceInterface interface {
    GetUser(id string) (*User, error)
    CreateUser(user *User) (*User, error)
}

// 2. 实现接口
type UserService struct {
    db *gorm.DB
}

// 3. 添加缓存注解
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id string) (*User, error) {
    var u User
    err := s.db.Where("id = ?", id).First(&u).Error
    return &u, err
}

// @cacheput(cache="users", key="#user.ID", ttl="30m")
func (s *UserService) CreateUser(user *User) (*User, error) {
    err := s.db.Create(user).Error
    return user, err
}
```

### 3. 初始化（一行代码）

```go
package service

import (
    "github.com/coderiser/go-cache/pkg/proxy"
)

// 全局变量 - 一行代码完成初始化！
// 简单模式：自动降级，适合快速原型开发
var UserService = proxy.SimpleDecorate(&UserService{})
```

**或者使用 init 函数（更灵活，推荐用于生产环境）**:

```go
package service

import (
    "log"
    "github.com/coderiser/go-cache/pkg/core"
    "github.com/coderiser/go-cache/pkg/proxy"
)

var UserService UserServiceInterface

func init() {
    // 方式 1: 简单模式（推荐，自动降级）
    // var UserService = proxy.SimpleDecorate(&UserService{})
    
    // 方式 2: 带错误处理（推荐用于生产环境）
    decorated, err := proxy.SimpleDecorateWithError(&UserService{})
    if err != nil {
        log.Printf("⚠️  Cache decoration failed: %v, using fallback", err)
        UserService = &UserService{}
        return
    }
    UserService = decorated.(*UserService)
    log.Println("✓ Cache initialized successfully")
    
    // 方式 3: 自定义缓存管理器（Redis 后端）
    // manager := core.NewCacheManager()
    // 
    // // 配置 Redis 后端
    // redisBackend, err := backend.NewRedisBackend(&backend.RedisConfig{
    //     Addr:         "localhost:6379",
    //     DefaultTTL:   30 * time.Minute,
    //     MaxRetries:   3,
    //     DialTimeout:  5 * time.Second,
    // })
    // if err != nil {
    //     log.Printf("⚠️  Redis connection failed: %v, using memory backend", err)
    //     // 降级到 Memory 后端
    //     memoryBackend := backend.NewMemoryBackend(backend.DefaultCacheConfig("users"))
    //     manager.RegisterCache("users", memoryBackend)
    // } else {
    //     manager.RegisterCache("users", redisBackend)
    // }
    // 
    // // 装饰服务
    // decorated := proxy.SimpleDecorateWithManager(&UserService{}, manager)
    // UserService = decorated.(*UserService)
}
```

### 4. 生成元数据

```bash
# 安装代码生成器
go install github.com/coderiser/go-cache/cmd/generator@latest

# 生成注解元数据（扫描整个项目）
go-cache-gen ./...

# 或在代码中添加 go:generate 指令
//go:generate go-cache-gen ./...
# 然后运行
go generate ./...
```

生成器会自动扫描所有 `// @cacheable`, `// @cacheput`, `// @cacheevict` 注解，
并在 `.cache-gen/auto_register.go` 中生成注册代码。

### 5. 使用（完全透明）

```go
// 通过接口调用，自动应用缓存
user, err := UserService.GetUser("123")  // 自动缓存！

// 调用方无需关心缓存逻辑
```

## 🔧 Redis 后端配置

### 基本配置（带错误处理和降级）

```go
package main

import (
    "context"
    "log"
    "os"
    "time"
    "github.com/coderiser/go-cache/pkg/backend"
    "github.com/coderiser/go-cache/pkg/core"
)

func main() {
    manager := core.NewCacheManager()
    
    // 尝试连接 Redis
    redisBackend, err := backend.NewRedisBackend(&backend.RedisConfig{
        Addr:         getEnv("REDIS_ADDR", "localhost:6379"),
        Password:     getEnv("REDIS_PASSWORD", ""),
        DB:           0,
        Prefix:       "myapp",
        DefaultTTL:   30 * time.Minute,
        MaxTTL:       24 * time.Hour,
        PoolSize:     50,
        MinIdleConns: 20,
        MaxRetries:   3,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
    })
    if err != nil {
        log.Printf("⚠️  Redis connection failed: %v", err)
        log.Println("✓ Falling back to memory backend")
        
        // 降级到 Memory 后端
        memoryBackend := backend.NewMemoryBackend(&backend.MemoryConfig{
            MaxSize:     10000,
            DefaultTTL:  30 * time.Minute,
        })
        manager.RegisterCache("users", memoryBackend)
    } else {
        log.Println("✓ Redis backend initialized")
        manager.RegisterCache("users", redisBackend)
        
        // 测试 Redis 连接
        if err := redisBackend.Ping(context.Background()); err != nil {
            log.Printf("⚠️  Redis ping failed: %v", err)
        }
    }
    
    // 启用监控
    manager.EnableMetrics()
    
    // 装饰服务
    UserService = proxy.SimpleDecorateWithManager(&UserService{}, manager).(*UserService)
}

// 辅助函数：获取环境变量
func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

### YAML 配置示例

```yaml
# cache.yaml
caches:
  users:
    backend: redis
    addr: localhost:6379
    password: ""
    db: 0
    prefix: "users"
    default_ttl: 30m
    max_ttl: 24h
    pool_size: 10
    min_idle_conns: 5
    
  sessions:
    backend: redis
    addr: localhost:6379
    db: 1
    prefix: "sessions"
    default_ttl: 2h
    
  products:
    backend: memory
    max_size: 10000
    default_ttl: 1h
```

### Redis 连接池调优

```go
// 高并发场景配置
&backend.RedisConfig{
    Addr:         "redis-cluster:6379",
    PoolSize:     100,      // 增大连接池
    MinIdleConns: 20,       // 保持更多空闲连接
    MaxRetries:   5,        // 增加重试次数
    DialTimeout:  3 * time.Second,
    ReadTimeout:  2 * time.Second,
    WriteTimeout: 2 * time.Second,
}
```

## 🎯 Go 语言接口模式说明

由于 Go 语言没有运行时注解，本框架采用**接口模式**：

### 核心思路

1. **定义接口**：为服务定义清晰的接口
2. **注解标注**：在实现的方法上添加 `// @cacheable(...)` 注释
3. **代码生成**：使用 `go-cache-gen` 生成注解元数据
4. **代理装饰**：通过 `DecorateAndReturn` 创建代理对象
5. **接口调用**：通过接口变量调用，自动应用缓存

### 为什么需要接口？

Go 的反射系统无法直接修改方法调用，但可以通过：
- 创建代理对象实现相同接口
- 拦截接口方法调用
- 在调用前后执行缓存逻辑

### 完整示例

**项目结构**:
```
myapp/
├── go.mod
├── main.go
└── service/
    ├── user.go      # 服务定义
    └── init.go      # 初始化
```

**1. 定义数据模型** (`service/user.go`):
```go
package service

// User 用户模型
type User struct {
    ID    int
    Name  string
    Email string
}
```

**2. 定义服务和缓存注解** (`service/user.go`):
```go
package service

// UserServiceInterface 服务接口
type UserServiceInterface interface {
    GetUser(id int) *User
    CreateUser(user *User) *User
}

// UserService 服务实现
type UserService struct {
    // 可以添加数据库连接等依赖
    // db *gorm.DB
}

// GetUser 获取用户（带缓存）
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id int) *User {
    // 实际业务逻辑：从数据库查询
    // 这里简化示例，直接返回模拟数据
    return &User{
        ID:    id,
        Name:  "Alice",
        Email: "alice@example.com",
    }
}

// CreateUser 创建用户（带缓存更新）
// @cacheput(cache="users", key="#user.ID", ttl="30m")
func (s *UserService) CreateUser(user *User) *User {
    // 实际业务逻辑：保存到数据库
    // s.db.Create(user)
    return user
}
```

**3. 初始化服务** (`service/init.go`):
```go
package service

import (
    "log"
    "github.com/coderiser/go-cache/pkg/proxy"
)

// 全局服务实例（一行代码完成初始化）
// 简单模式：适合快速原型开发，自动降级
var UserService = proxy.SimpleDecorate(&UserService{})

// 生产环境推荐：带错误处理的初始化
// var UserService UserServiceInterface
// 
// func init() {
//     decorated, err := proxy.SimpleDecorateWithError(&UserService{})
//     if err != nil {
//         log.Printf("⚠️  Cache decoration failed: %v, using fallback", err)
//         UserService = &UserService{}
//         return
//     }
//     UserService = decorated.(*UserService)
//     log.Println("✓ Cache initialized successfully")
// }
```

**4. 使用服务** (`main.go`):
```go
package main

import (
    "fmt"
    "log"
    "myapp/service"
)

func main() {
    // 第一次调用：缓存未命中，执行实际查询
    user1, err := service.UserService.GetUser(123)
    if err != nil {
        log.Printf("Get user failed: %v", err)
        return
    }
    fmt.Printf("Got user: %+v\n", user1)
    
    // 第二次调用：缓存命中，直接返回
    user2, err := service.UserService.GetUser(123)
    if err != nil {
        log.Printf("Get user failed: %v", err)
        return
    }
    fmt.Printf("Got cached user: %+v\n", user2)
    
    // 创建用户（自动更新缓存）
    newUser := &User{ID: 456, Name: "Bob"}
    savedUser, err := service.UserService.CreateUser(newUser)
    if err != nil {
        log.Printf("Create user failed: %v", err)
        return
    }
    fmt.Printf("Created user: %+v\n", savedUser)
}
```

**5. 生成元数据并运行**:
```bash
# 生成注解元数据
go-cache-gen ./...

# 运行程序
go run .
```

**输出**:
```
Got user: &{ID:123 Name:Alice Email:alice@example.com}
Got cached user: &{ID:123 Name:Alice Email:alice@example.com}
```

## 📖 核心注解

| 注解 | 说明 | 示例 |
|------|------|------|
| `@cacheable` | 缓存读取 | `@cacheable(cache="users", key="#id", ttl="30m")` |
| `@cacheput` | 强制更新缓存 | `@cacheput(cache="users", key="#user.Id")` |
| `@cacheevict` | 删除缓存 | `@cacheevict(cache="users", key="#id", before=true)` |

## 🎯 SpEL 表达式

```go
// 引用参数
@cacheable(cache="orders", key="#userId + '_' + #status")

// 引用返回值（unless）
@cacheable(cache="data", key="#id", unless="#result == nil")

// 复杂表达式
@cacheable(cache="products", key="category:#catId:page:#page")
```

## 🔌 Redis 后端配置

### 基础配置（带错误处理）

```go
package main

import (
    "context"
    "log"
    "os"
    "time"
    "github.com/coderiser/go-cache/pkg/backend"
    "github.com/coderiser/go-cache/pkg/core"
)

func init() {
    // 创建 Redis 后端（带环境变量配置）
    redisConfig := &backend.RedisConfig{
        Addr:         getEnv("REDIS_ADDR", "localhost:6379"),
        Password:     getEnv("REDIS_PASSWORD", ""),
        DB:           0,
        Prefix:       "myapp",
        DefaultTTL:   30 * time.Minute,
        MaxTTL:       24 * time.Hour,
        PoolSize:     10,
        MinIdleConns: 5,
        MaxRetries:   3,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
    }
    
    redisBackend, err := backend.NewRedisBackend(redisConfig)
    if err != nil {
        log.Printf("⚠️  Failed to create Redis backend: %v", err)
        log.Println("✓ Using memory backend as fallback")
        // 降级到 Memory 后端
        memoryBackend := backend.NewMemoryBackend(&backend.MemoryConfig{
            MaxSize:     10000,
            DefaultTTL:  30 * time.Minute,
        })
        manager := core.NewCacheManager()
        manager.RegisterCache("users", memoryBackend)
        return
    }
    
    // 测试 Redis 连接
    if err := redisBackend.Ping(context.Background()); err != nil {
        log.Printf("⚠️  Redis ping failed: %v", err)
    }
    
    // 注册到缓存管理器
    manager := core.NewCacheManager()
    manager.RegisterCache("users", redisBackend)
    log.Println("✓ Redis backend initialized successfully")
}

// 辅助函数：获取环境变量
func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

### YAML 配置示例

```yaml
# cache.yaml
caches:
  users:
    backend: redis
    addr: localhost:6379
    password: ""
    db: 0
    prefix: "users:"
    default_ttl: 30m
    max_ttl: 24h
    pool_size: 10
    min_idle_conns: 5
    
  sessions:
    backend: redis
    addr: localhost:6379
    db: 1
    prefix: "sessions:"
    default_ttl: 2h
```

### 连接池最佳实践

```go
// 生产环境推荐配置
redisConfig := &backend.RedisConfig{
    Addr:         getEnv("REDIS_ADDR", "redis-cluster:6379"),
    Password:     getEnv("REDIS_PASSWORD", ""),
    PoolSize:     50,              // 根据并发量调整
    MinIdleConns: 20,              // 保持最小空闲连接
    MaxRetries:   3,               // 失败重试次数
    DialTimeout:  5 * time.Second, // 连接超时
    ReadTimeout:  3 * time.Second, // 读取超时
    WriteTimeout: 3 * time.Second, // 写入超时
}

redisBackend, err := backend.NewRedisBackend(redisConfig)
if err != nil {
    log.Printf("⚠️  Redis connection failed: %v", err)
    // 降级到 Memory 后端
    memoryBackend := backend.NewMemoryBackend(&backend.MemoryConfig{
        MaxSize:     10000,
        DefaultTTL:  30 * time.Minute,
    })
    manager.RegisterCache("users", memoryBackend)
} else {
    log.Println("✓ Redis backend initialized")
    manager.RegisterCache("users", redisBackend)
}
```

## 🛡️ 缓存异常保护机制

Go-Cache Framework 内置三种缓存异常保护机制，确保高并发场景下的稳定性：

### 1. 缓存穿透保护（Penetration Protection）

**问题**：查询不存在的数据，缓存无法命中，请求直达数据库。

**解决方案**：空值缓存（Nil Marker）

```go
import "github.com/coderiser/go-cache/pkg/core"

protection := core.NewCacheProtection(core.DefaultProtectionConfig())

// 空值会被包装为特殊标记
wrapped := protection.WrapForStorage(nil)
// 存储到缓存：__GO_CACHE_NIL__

// 获取时自动解包
value := protection.UnwrapFromStorage(wrapped)
// 返回：nil
```

**配置**：
```go
config := &core.ProtectionConfig{
    EnablePenetrationProtection: true,
    EmptyValueTTL:               5 * time.Minute, // 空值缓存 5 分钟
}
```

### 2. 缓存击穿保护（Breakdown Protection）

**问题**：热点 key 过期瞬间，大量并发请求直达数据库。

**解决方案**：Singleflight（请求合并）

```go
import "github.com/coderiser/go-cache/pkg/core"

protection := core.NewCacheProtection(core.DefaultProtectionConfig())

ctx := context.Background()
result, err, shared := protection.ApplyBreakdownProtection(ctx, "hot-key", func() (interface{}, error) {
    // 这个函数在并发请求下只会执行一次
    return db.GetHotData("hot-key")
})

// shared=true 表示本次请求复用了其他请求的结果
```

**效果**：100 个并发请求 → 只执行 1 次数据库查询 → 99 个请求共享结果

### 3. 缓存雪崩保护（Avalanche Protection）

**问题**：大量缓存同时过期，导致数据库瞬时压力激增。

**解决方案**：TTL 随机偏移（Jitter）

```go
import "github.com/coderiser/go-cache/pkg/core"

protection := core.NewCacheProtection(core.DefaultProtectionConfig())

baseTTL := 30 * time.Minute
actualTTL := protection.ApplyAvalancheProtection(baseTTL)
// 实际 TTL = 30 分钟 ± 10% 随机偏移

// 自定义抖动因子（0.0-0.5）
actualTTL = protection.CalculateTTLWithJitter(baseTTL, 0.2)
// 实际 TTL = 30 分钟 ± 20% 随机偏移
```

**效果**：30 分钟 TTL → 实际分布在 27-33 分钟之间 → 避免同时过期

### 完整保护示例

```go
package service

import (
    "context"
    "github.com/coderiser/go-cache/pkg/core"
    "github.com/coderiser/go-cache/pkg/backend"
)

type UserService struct {
    cache      backend.CacheBackend
    protection *core.CacheProtection
    db         *Database
}

func NewUserService(cache backend.CacheBackend) *UserService {
    return &UserService{
        cache:      cache,
        protection: core.NewCacheProtection(core.DefaultProtectionConfig()),
        db:         NewDatabase(),
    }
}

func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    key := "user:" + id
    
    // 使用受保护的获取方法
    result, err := s.protection.ProtectedGet(
        ctx,
        key,
        // 缓存获取
        func() (interface{}, bool, error) {
            return s.cache.Get(ctx, key)
        },
        // 缓存未命中时的数据获取
        func() (interface{}, error) {
            return s.db.FindUser(id)
        },
        // 缓存设置
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

### 保护配置推荐

```go
// 生产环境推荐配置
protectionConfig := &core.ProtectionConfig{
    EnablePenetrationProtection: true,  // 启用穿透保护
    EmptyValueTTL:               5 * time.Minute,  // 空值缓存 5 分钟
    
    EnableBreakdownProtection:   true,  // 启用击穿保护
    EnableAvalancheProtection:   true,  // 启用雪崩保护
    TTLJitterFactor:             0.1,   // 10% TTL 抖动
}
```

## 🛡️ 错误处理最佳实践

### 1. 初始化错误处理

```go
// 推荐：使用带错误处理的初始化
func init() {
    decorated, err := proxy.SimpleDecorateWithError(&UserService{})
    if err != nil {
        log.Printf("Cache decoration failed: %v, using fallback", err)
        UserService = &UserService{}
        return
    }
    UserService = decorated.(*UserService)
}
```

### 2. Redis 降级策略

```go
// 配置 Redis 后端，失败时自动降级到 Memory
redisBackend, err := backend.NewRedisBackend(redisConfig)
if err != nil {
    // 降级到 Memory 后端
    memoryBackend := backend.NewMemoryBackend(memoryConfig)
    manager.RegisterCache("users", memoryBackend)
} else {
    manager.RegisterCache("users", redisBackend)
}
```

### 3. 缓存操作错误处理

```go
// 获取缓存（自动处理错误）
user, err := UserService.GetUser(id)
if err != nil {
    // 处理业务错误
    log.Printf("Get user failed: %v", err)
    return nil, err
}

// 缓存失败不会影响到业务逻辑
// 框架会自动降级到直接调用原始方法
```

### 4. 监控和告警

```go
// 启用 Prometheus 指标
manager.EnableMetrics()

// 定期检查缓存健康状态
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        stats := cache.Stats()
        if stats.HitRate < 0.5 {
            log.Printf("⚠️  Low cache hit rate: %.2f", stats.HitRate)
        }
    }
}()
```

### 5. 环境变量配置

```go
// 使用环境变量管理配置
func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

// 在配置中使用
redisConfig := &backend.RedisConfig{
    Addr:         getEnv("REDIS_ADDR", "localhost:6379"),
    Password:     getEnv("REDIS_PASSWORD", ""),
    DefaultTTL:   30 * time.Minute,
    MaxRetries:   getEnvAsInt("REDIS_MAX_RETRIES", 3),
    DialTimeout:  5 * time.Second,
}

// 辅助函数：获取整数类型环境变量
func getEnvAsInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.Atoi(value); err == nil {
            return intValue
        }
    }
    return defaultValue
}
```

## 📚 文档

- [架构设计](docs/ARCHITECTURE.md)
- [接口定义](docs/INTERFACE_SPEC.md)
- [集成指南](docs/INTEGRATION_GUIDE.md)
- [P2 功能总览](docs/P2_FEATURES.md) - 多级缓存、指标、追踪、保护机制
- [变更日志](CHANGELOG.md)

## 🧪 测试

```bash
# 运行测试
go test ./...

# 覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 性能基准
go test -bench=. -benchmem ./...
```

## 📊 性能

| 场景 | 延迟 | 说明 |
|------|------|------|
| Memory 命中 | < 1ms | 纯内存操作 |
| Redis 命中 | < 5ms | 网络 + 序列化 |
| SpEL 求值 | < 50μs | 基于 expr 引擎 |
| L1 → L2 穿透 | < 10ms | 自动回写 L1 |

**优化技巧**:
- 使用 HybridBackend 可获得最佳性价比
- 热点数据建议启用 Singleflight 保护
- 大规模部署建议开启指标监控

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

---

**Made with ❤️ by Go-Cache Team**
