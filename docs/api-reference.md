# Go-Cache Framework API 参考

**版本:** v1.0  
**最后更新:** 2026-03-14

---

## 目录

1. [pkg/cache 包](#1-pkgcache-包)
2. [pkg/core 包](#2-pkgcore-包)
3. [pkg/backend 包](#3-pkgbackend-包)
4. [pkg/proxy 包](#4-pkgproxy-包)
5. [pkg/spel 包](#5-pkgspel-包)
6. [代码生成器](#6-代码生成器)
7. [配置选项](#7-配置选项)

---

## 1. pkg/cache 包

方案 G 的核心包，提供全局管理器注册和便捷函数。

### 1.1 全局管理器

#### GetGlobalManager

```go
func GetGlobalManager() core.CacheManager
```

获取全局缓存管理器（懒加载）。

**说明:**
- 首次调用时自动创建默认 Manager
- 线程安全（使用 sync.Once）
- 返回单例实例

**示例:**
```go
manager := cache.GetGlobalManager()
cache, _ := manager.GetCache("users")
```

#### SetGlobalManager

```go
func SetGlobalManager(manager core.CacheManager)
```

设置全局缓存管理器。

**说明:**
- 在应用启动时调用
- 覆盖默认的懒加载 Manager
- 高级用户使用

**示例:**
```go
// main.go
manager := core.NewCacheManager()
// 配置 Redis 后端...
cache.SetGlobalManager(manager)
```

#### CloseGlobalManager

```go
func CloseGlobalManager()
```

关闭全局缓存管理器。

**说明:**
- 在应用退出时调用
- 释放资源（连接池等）
- 建议使用 defer

**示例:**
```go
func main() {
    defer cache.CloseGlobalManager()
    
    // 业务逻辑...
}
```

### 1.2 注解注册

#### RegisterGlobalAnnotation

```go
func RegisterGlobalAnnotation(typeName, methodName string, annotation *proxy.CacheAnnotation)
```

注册全局注解（供生成代码调用）。

**参数:**
- `typeName` - 类型名称（如 "ProductService"）
- `methodName` - 方法名称（如 "GetProduct"）
- `annotation` - 注解对象

**说明:**
- 由生成代码在 init() 中调用
- 用户无需手动调用

#### GetGlobalAnnotation

```go
func GetGlobalAnnotation(typeName, methodName string) *proxy.CacheAnnotation
```

获取已注册的全局注解。

**参数:**
- `typeName` - 类型名称
- `methodName` - 方法名称

**返回:**
- 注解对象，如果未注册返回 nil

#### GetAllAnnotations

```go
func GetAllAnnotations(typeName string) map[string]*proxy.CacheAnnotation
```

获取某类型的所有注解。

**参数:**
- `typeName` - 类型名称

**返回:**
- 方法名到注解的映射

---

## 2. pkg/core 包

核心包，提供缓存管理器、保护机制等。

### 2.1 CacheManager

#### NewCacheManager

```go
func NewCacheManager() CacheManager
```

创建新的缓存管理器。

**示例:**
```go
manager := core.NewCacheManager()
```

#### RegisterCache

```go
func (m *CacheManager) RegisterCache(name string, backend backend.CacheBackend)
```

注册缓存后端。

**参数:**
- `name` - 缓存名称
- `backend` - 缓存后端实例

**示例:**
```go
manager.RegisterCache("users", memoryBackend)
manager.RegisterCache("products", redisBackend)
```

#### GetCache

```go
func (m *CacheManager) GetCache(name string) (backend.CacheBackend, error)
```

获取已注册的缓存。

**参数:**
- `name` - 缓存名称

**返回:**
- 缓存后端实例
- 错误（如果未找到）

#### EnableMetrics

```go
func (m *CacheManager) EnableMetrics()
```

启用 Prometheus 指标。

**说明:**
- 在 /metrics 端点暴露
- 包含命中率、延迟等指标

### 2.2 CacheProtection

#### NewCacheProtection

```go
func NewCacheProtection(config *ProtectionConfig) *CacheProtection
```

创建缓存保护机制。

**参数:**
- `config` - 保护配置

#### DefaultProtectionConfig

```go
func DefaultProtectionConfig() *ProtectionConfig
```

获取默认保护配置。

**返回:**
```go
&ProtectionConfig{
    EnablePenetrationProtection: true,
    EmptyValueTTL:               5 * time.Minute,
    EnableBreakdownProtection:   true,
    EnableAvalancheProtection:   true,
    TTLJitterFactor:             0.1,
}
```

#### WrapForStorage

```go
func (p *CacheProtection) WrapForStorage(value interface{}) interface{}
```

包装值用于存储（处理空值）。

#### UnwrapFromStorage

```go
func (p *CacheProtection) UnwrapFromStorage(wrapped interface{}) interface{}
```

从存储中解包值。

#### ApplyBreakdownProtection

```go
func (p *CacheProtection) ApplyBreakdownProtection(
    ctx context.Context,
    key string,
    fn func() (interface{}, error),
) (interface{}, error, bool)
```

应用击穿保护（Singleflight）。

**返回:**
- 结果
- 错误
- shared - 是否复用了其他请求的结果

#### CalculateTTLWithJitter

```go
func (p *CacheProtection) CalculateTTLWithJitter(baseTTL time.Duration, jitterFactor float64) time.Duration
```

计算带抖动的 TTL。

**参数:**
- `baseTTL` - 基础 TTL
- `jitterFactor` - 抖动因子（0.0-0.5）

**返回:**
- 实际 TTL（baseTTL ± jitterFactor）

### 2.3 配置结构

#### ProtectionConfig

```go
type ProtectionConfig struct {
    EnablePenetrationProtection bool          // 穿透保护
    EmptyValueTTL               time.Duration // 空值缓存 TTL
    EnableBreakdownProtection   bool          // 击穿保护
    EnableAvalancheProtection   bool          // 雪崩保护
    TTLJitterFactor             float64       // TTL 抖动因子
}
```

#### CacheStats

```go
type CacheStats struct {
    Hits     int64   // 命中次数
    Misses   int64   // 未命中次数
    HitRate  float64 // 命中率（0.0-1.0）
    Size     int64   // 当前大小
    Errors   int64   // 错误次数
}
```

---

## 3. pkg/backend 包

后端包，提供不同缓存后端实现。

### 3.1 MemoryBackend

#### NewMemoryBackend

```go
func NewMemoryBackend(config *MemoryConfig) *MemoryBackend
```

创建内存缓存后端。

**参数:**
```go
&MemoryConfig{
    MaxSize:     10000,        // 最大条目数
    DefaultTTL:  30 * time.Minute,
    CleanupInterval: 1 * time.Minute, // 清理间隔
}
```

### 3.2 RedisBackend

#### NewRedisBackend

```go
func NewRedisBackend(config *RedisConfig) (*RedisBackend, error)
```

创建 Redis 缓存后端。

**参数:**
```go
&RedisConfig{
    Addr:         "localhost:6379",
    Password:     "",
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
}
```

#### Ping

```go
func (b *RedisBackend) Ping(ctx context.Context) error
```

测试 Redis 连接。

### 3.3 HybridBackend

#### NewHybridBackend

```go
func NewHybridBackend(l1, l2 backend.CacheBackend) *HybridBackend
```

创建混合缓存后端（L1 Memory + L2 Redis）。

**说明:**
- L1 用于热点数据（快速访问）
- L2 用于持久化（大容量）
- 自动回写 L1

### 3.4 CacheBackend 接口

```go
type CacheBackend interface {
    Get(ctx context.Context, key string) (interface{}, bool, error)
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    Stats() *CacheStats
    Close() error
}
```

---

## 4. pkg/proxy 包

代理包，提供服务装饰和拦截。

### 4.1 装饰函数

#### SimpleDecorate

```go
func SimpleDecorate(service interface{}) interface{}
```

简单装饰服务（使用全局 Manager）。

**说明:**
- 自动创建默认 Manager
- 失败时返回原始服务（降级）
- 适合快速原型

**示例:**
```go
var UserService = proxy.SimpleDecorate(&UserService{})
```

#### SimpleDecorateWithError

```go
func SimpleDecorateWithError(service interface{}) (interface{}, error)
```

带错误处理的装饰。

**说明:**
- 返回装饰后的服务和错误
- 适合生产环境
- 可显式处理错误

**示例:**
```go
decorated, err := proxy.SimpleDecorateWithError(&UserService{})
if err != nil {
    log.Printf("Cache decoration failed: %v", err)
    UserService = &UserService{}
    return
}
UserService = decorated.(*UserService)
```

#### SimpleDecorateWithManager

```go
func SimpleDecorateWithManager(service interface{}, manager core.CacheManager) interface{}
```

使用指定 Manager 装饰服务。

**参数:**
- `service` - 原始服务
- `manager` - 缓存管理器

**示例:**
```go
manager := core.NewCacheManager()
decorated := proxy.SimpleDecorateWithManager(&UserService{}, manager)
```

### 4.2 CacheAnnotation

```go
type CacheAnnotation struct {
    Type       string // cacheable, cacheput, cacheevict
    CacheName  string // 缓存名称
    Key        string // Key 表达式
    TTL        string // 过期时间
    Condition  string // 条件表达式
    Unless     string // 排除条件
    Before     bool   // 是否在方法前执行（仅 cacheevict）
    AllEntries bool   // 是否清除所有（仅 cacheevict）
}
```

---

## 5. pkg/spel 包

SpEL 表达式包，提供表达式求值。

### 5.1 SpELEvaluator

#### NewSpELEvaluator

```go
func NewSpELEvaluator() *SpELEvaluator
```

创建 SpEL 表达式求值器。

#### EvaluateToString

```go
func (e *SpELEvaluator) EvaluateToString(expr string, ctx *EvaluationContext) (string, error)
```

求值表达式并返回字符串。

**参数:**
- `expr` - 表达式字符串
- `ctx` - 求值上下文

**返回:**
- 求值结果（字符串）
- 错误

#### Evaluate

```go
func (e *SpELEvaluator) Evaluate(expr string, ctx *EvaluationContext) (interface{}, error)
```

求值表达式并返回任意类型。

### 5.2 EvaluationContext

#### NewEvaluationContext

```go
func NewEvaluationContext() *EvaluationContext
```

创建求值上下文。

#### SetArgByIndex

```go
func (ctx *EvaluationContext) SetArgByIndex(index int, value interface{})
```

按索引设置参数。

#### SetArgByName

```go
func (ctx *EvaluationContext) SetArgByName(name string, value interface{})
```

按名称设置参数。

#### SetResult

```go
func (ctx *EvaluationContext) SetResult(value interface{})
```

设置返回值（用于 unless 表达式）。

---

## 6. 代码生成器

### 6.1 安装

```bash
go install github.com/coderiser/go-cache/cmd/generator@latest
```

### 6.2 使用

#### 命令行方式

```bash
go-cache-gen ./...
```

#### go:generate 方式

在代码中添加：
```go
//go:generate go run ../../../cmd/generator/main.go .

package service
```

然后运行：
```bash
go generate ./...
```

### 6.3 生成文件

| 文件 | 说明 |
|------|------|
| `.cache-gen/auto_register.go` | 注解自动注册到全局表 |
| `.cache-gen/<type>_cached.go` | 带缓存的实现类 |

### 6.4 生成代码示例

```go
// .cache-gen/product_cached.go
package cache

import (
    "context"
    "time"
    "github.com/coderiser/go-cache/pkg/core"
    "github.com/coderiser/go-cache/pkg/cache"
    "your-module/service"
)

// cachedProductService 带缓存的包装实现
type cachedProductService struct {
    decorated service.ProductServiceInterface
    manager   core.CacheManager
}

// NewProductService 创建带缓存的服务实例（使用全局 Manager）
func NewProductService() service.ProductServiceInterface {
    return NewProductServiceWithManager(cache.GetGlobalManager())
}

// NewProductServiceWithManager 创建带缓存的服务实例（使用指定 Manager）
func NewProductServiceWithManager(manager core.CacheManager) service.ProductServiceInterface {
    raw := service.NewProductService()
    return &cachedProductService{
        decorated: raw,
        manager:   manager,
    }
}

// GetProduct 带缓存的实现
func (c *cachedProductService) GetProduct(id int64) (*model.Product, error) {
    // 1. 获取缓存
    cache, err := c.manager.GetCache("products")
    if err != nil {
        return c.decorated.GetProduct(id)
    }
    
    // 2. 生成缓存 Key
    key := fmt.Sprintf("products:%v", id)
    
    // 3. 查询缓存
    if val, found, _ := cache.Get(context.Background(), key); found {
        return val.(*model.Product), nil
    }
    
    // 4. 执行原始方法
    result, err := c.decorated.GetProduct(id)
    if err != nil {
        return nil, err
    }
    
    // 5. 写入缓存
    ttl, _ := time.ParseDuration("1h")
    _ = cache.Set(context.Background(), key, result, ttl)
    
    return result, nil
}
```

---

## 7. 配置选项

### 7.1 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `REDIS_ADDR` | Redis 地址 | localhost:6379 |
| `REDIS_PASSWORD` | Redis 密码 | - |
| `REDIS_DB` | Redis 数据库 | 0 |
| `CACHE_MAX_SIZE` | 最大缓存条目 | 10000 |
| `CACHE_DEFAULT_TTL` | 默认 TTL | 30m |
| `CACHE_METRICS_ENABLED` | 启用指标 | false |

### 7.2 YAML 配置

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
    pool_size: 50
    min_idle_conns: 20
    
  products:
    backend: memory
    max_size: 5000
    default_ttl: 1h
    
  sessions:
    backend: hybrid
    l1_max_size: 1000
    l1_ttl: 10m
    l2_addr: localhost:6379
    l2_ttl: 2h

protection:
  enable_penetration: true
  empty_value_ttl: 5m
  enable_breakdown: true
  enable_avalanche: true
  ttl_jitter_factor: 0.1
```

### 7.3 注解参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `cache` | string | 是 | 缓存名称 |
| `key` | string | 是 | Key 表达式（SpEL） |
| `ttl` | string | 否 | 过期时间（如 "30m"） |
| `condition` | string | 否 | 执行条件（SpEL） |
| `unless` | string | 否 | 排除条件（SpEL） |
| `before` | bool | 否 | 方法前执行（cacheevict） |
| `allEntries` | bool | 否 | 清除所有（cacheevict） |

---

## 附录：完整示例

### 完整项目结构

```
myapp/
├── go.mod
├── main.go
├── service/
│   ├── user.go          # 服务定义 + 注解
│   ├── init.go          # 初始化
│   └── .cache-gen/      # 生成代码
│       ├── auto_register.go
│       └── user_cached.go
└── model/
    └── user.go          # 数据模型
```

### main.go

```go
package main

import (
    "log"
    cached "myapp/service/.cache-gen"
)

func main() {
    defer cache.CloseGlobalManager()
    
    // 零配置！
    userService := cached.NewUserService()
    
    // 使用
    user, err := userService.GetUser(123)
    if err != nil {
        log.Printf("Get user failed: %v", err)
        return
    }
    
    log.Printf("Got user: %+v", user)
}
```

### service/user.go

```go
//go:generate go run ../../../cmd/generator/main.go .

package service

import (
    "gorm.io/gorm"
    "myapp/model"
)

type UserServiceInterface interface {
    GetUser(id int64) (*model.User, error)
    CreateUser(user *model.User) (*model.User, error)
    UpdateUser(id int64, name string) (*model.User, error)
    DeleteUser(id int64) error
}

type userService struct {
    db *gorm.DB
}

func NewUserService() UserServiceInterface {
    return &userService{db: getDB()}
}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *userService) GetUser(id int64) (*model.User, error) {
    var user model.User
    err := s.db.First(&user, id).Error
    return &user, err
}

// @cacheput(cache="users", key="#user.ID", ttl="30m")
func (s *userService) CreateUser(user *model.User) (*model.User, error) {
    err := s.db.Create(user).Error
    return user, err
}

// @cacheput(cache="users", key="#id", ttl="30m")
func (s *userService) UpdateUser(id int64, name string) (*model.User, error) {
    var user model.User
    err := s.db.Model(&user).Where("id = ?", id).Update("name", name).Error
    return &user, err
}

// @cacheevict(cache="users", key="#id")
func (s *userService) DeleteUser(id int64) error {
    return s.db.Delete(&model.User{}, id).Error
}
```

---

**更多示例:** 查看 [examples](../examples/) 目录。

**问题反馈:** [GitHub Issues](https://github.com/coderiser/go-cache/issues)
