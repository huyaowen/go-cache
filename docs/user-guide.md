# Go-Cache Framework 用户指南

**版本:** v1.0  
**最后更新:** 2026-03-14

---

## 目录

1. [快速开始](#1-快速开始)
2. [注解语法](#2-注解语法)
3. [SpEL 表达式参考](#3-spel-表达式参考)
4. [缓存后端配置](#4-缓存后端配置)
5. [高级配置](#5-高级配置)
6. [缓存统计与监控](#6-缓存统计与监控)
7. [最佳实践](#7-最佳实践)
8. [常见问题](#8-常见问题)

---

## 1. 快速开始

### 1.1 安装

```bash
go get github.com/coderiser/go-cache
```

### 1.2 5 分钟上手（方案 G）

#### 步骤 1: 定义 Service 和注解

```go
// service/product.go
//go:generate go run ../../../cmd/generator/main.go .

type ProductServiceInterface interface {
    GetProduct(id int64) (*model.Product, error)
    UpdatePrice(id int64, price float64) (*model.Product, error)
}

type productService struct {
    db *gorm.DB
}

// @cacheable(cache="products", key="#id", ttl="1h")
func (s *productService) GetProduct(id int64) (*model.Product, error) {
    var product model.Product
    err := s.db.First(&product, id).Error
    return &product, err
}

// @cacheput(cache="products", key="#id", ttl="1h")
func (s *productService) UpdatePrice(id int64, price float64) (*model.Product, error) {
    var product model.Product
    err := s.db.Model(&product).Where("id = ?", id).Update("price", price).Error
    return &product, err
}
```

#### 步骤 2: 执行代码生成

```bash
go generate ./...
```

生成文件:
- `.cache-gen/auto_register.go` - 注解自动注册到全局表
- `.cache-gen/product_cached.go` - 带缓存的实现

#### 步骤 3: 使用服务（零配置！）

```go
// main.go
import cached "your-module/service/.cache-gen"

func main() {
    // ✅ 一行搞定！缓存自动生效
    svc := cached.NewProductService()
    
    // 第一次调用：查询数据库 + 写入缓存
    product1, _ := svc.GetProduct(1)
    
    // 第二次调用：直接返回缓存
    product2, _ := svc.GetProduct(1)
}
```

### 1.3 自定义缓存管理器（可选）

```go
// main.go
import (
    "github.com/coderiser/go-cache/pkg/core"
    cached "your-module/service/.cache-gen"
)

func main() {
    // 创建自定义 Manager (Redis 后端)
    manager := core.NewCacheManager()
    // 配置 Redis...
    
    // 设置为全局 Manager
    cache.SetGlobalManager(manager)
    
    // 使用自定义 Manager 创建服务
    svc := cached.NewProductService()
    
    // 优雅关闭
    defer cache.CloseGlobalManager()
}
```

---

## 2. 注解语法

### 2.1 @cacheable (缓存读取)

在方法执行前检查缓存，如果命中则直接返回，否则执行方法并将结果写入缓存。

**语法:**
```go
// @cacheable(cache="缓存名", key="Key 表达式", ttl="过期时间", condition="条件", unless="排除条件")
```

**参数说明:**

| 参数 | 必填 | 说明 | 默认值 |
|------|------|------|--------|
| `cache` | 是 | 缓存名称 | - |
| `key` | 是 | 缓存 Key 表达式（支持 SpEL） | - |
| `ttl` | 否 | 过期时间（如 "30m", "1h"） | 30m |
| `condition` | 否 | 执行条件（SpEL 表达式） | - |
| `unless` | 否 | 排除条件（SpEL 表达式） | - |

**示例:**

```go
// 基本用法
// @cacheable(cache="users", key="#id", ttl="30m")
func GetUser(id int64) (*User, error)

// 使用 SpEL 访问参数属性
// @cacheable(cache="users", key="#user.Id", ttl="1h")
func GetUserByRequest(user *UserRequest) (*User, error)

// 条件缓存：只缓存 id > 0 的情况
// @cacheable(cache="users", key="#id", condition="#id > 0")
func GetUser(id int64) (*User, error)

// 排除条件：不缓存空结果
// @cacheable(cache="users", key="#id", unless="#result == nil")
func GetUser(id int64) (*User, error)

// 组合使用
// @cacheable(cache="users", key="#id", ttl="1h", condition="#id > 0", unless="#result == nil")
func GetUser(id int64) (*User, error)
```

### 2.2 @cacheput (缓存更新)

总是执行方法，并将结果写入缓存。适用于更新操作。

**语法:**
```go
// @cacheput(cache="缓存名", key="Key 表达式", ttl="过期时间")
```

**示例:**

```go
// 更新用户信息
// @cacheput(cache="users", key="#id", ttl="30m")
func UpdateUser(id int64, name string) (*User, error)

// 更新价格（使用返回值属性作为 Key）
// @cacheput(cache="products", key="#result.Id", ttl="1h")
func CreateProduct(product *Product) (*Product, error)
```

### 2.3 @cacheevict (缓存清除)

删除缓存中的指定 Key。

**语法:**
```go
// @cacheevict(cache="缓存名", key="Key 表达式", before=false, allEntries=false)
```

**参数说明:**

| 参数 | 必填 | 说明 | 默认值 |
|------|------|------|--------|
| `cache` | 是 | 缓存名称 | - |
| `key` | 否 | 缓存 Key 表达式 | - |
| `before` | 否 | 是否在方法执行前清除 | false |
| `allEntries` | 否 | 是否清除所有条目 | false |

**示例:**

```go
// 方法执行后清除
// @cacheevict(cache="users", key="#id")
func DeleteUser(id int64) error

// 方法执行前清除（避免脏读）
// @cacheevict(cache="users", key="#id", before=true)
func UpdateUser(id int64, name string) (*User, error)

// 清除所有条目（慎用）
// @cacheevict(cache="users", allEntries=true)
func ClearAllUsers() error
```

---

## 3. SpEL 表达式参考

Go-Cache 使用 [expr](https://github.com/antonmedve/expr) 引擎支持 SpEL 表达式。

### 3.1 可用变量

| 变量 | 说明 | 示例 |
|------|------|------|
| `#参数名` | 方法参数 | `key="#id"`, `key="#user.Id"` |
| `#p0`, `#p1` | 参数索引（从 0 开始） | `key="#p0"` |
| `#0`, `#1` | 参数索引（简写） | `key="#0"` |
| `result` | 返回值（仅 `unless` 可用） | `unless="#result == nil"` |

### 3.2 表达式语法

#### 访问参数

```go
// 简单参数
// @cacheable(cache="users", key="#id")
func GetUser(id int64) (*User, error)

// 访问参数属性
// @cacheable(cache="users", key="#user.Id")
func GetUser(user *UserRequest) (*User, error)

// 访问嵌套属性
// @cacheable(cache="orders", key="#order.Customer.Id")
func GetOrder(order *OrderRequest) (*Order, error)
```

#### 字符串拼接

```go
// 拼接多个参数
// @cacheable(cache="users", key="#prefix + '_' + #id")
func GetUser(prefix string, id int64) (*User, error)

// 使用格式化
// @cacheable(cache="users", key="fmt.Sprintf('user:%d', #id)")
func GetUser(id int64) (*User, error)
```

#### 条件表达式

```go
// 比较运算
// @cacheable(cache="data", key="#id", condition="#id > 0")
// @cacheable(cache="data", key="#id", condition="#id <= 1000")
// @cacheable(cache="data", key="#id", condition="#id == #defaultId")

// 逻辑运算
// @cacheable(cache="data", key="#id", condition="#id > 0 && #id < 1000")
// @cacheable(cache="data", key="#id", condition="#id > 0 || #id == -1")

// 字符串匹配
// @cacheable(cache="data", key="#key", condition="#key.startsWith('user_')")
```

#### 静态方法调用

```go
// 调用标准库函数
// @cacheable(cache="data", key="md5.Sum(#id)")
// @cacheable(cache="data", key="fmt.Sprintf('user:%d', #id)")

// 调用自定义函数（需注册）
// @cacheable(cache="data", key="ToKey(#id)")
```

#### 返回值引用（unless）

```go
// 不缓存空结果
// @cacheable(cache="users", key="#id", unless="#result == nil")
func GetUser(id int64) (*User, error)

// 不缓存错误结果
// @cacheable(cache="data", key="#id", unless="#err != nil")
func GetData(id int64) (interface{}, error)
```

### 3.3 复杂表达式示例

```go
// 组合条件
// @cacheable(
//     cache="products",
//     key="#category.Id + ':' + #page",
//     condition="#category.Id > 0 && #page >= 0",
//     unless="#result == nil || len(#result.Items) == 0"
// )
func GetProducts(category *Category, page int) (*ProductList, error)

// 使用三元表达式
// @cacheable(cache="data", key="#id > 0 ? #id : #defaultId")
func GetData(id int64, defaultId int64) (*Data, error)

// 访问数组/切片
// @cacheable(cache="data", key="#ids[0]")
func GetDataByIds(ids []int64) (*Data, error)

// 访问 Map
// @cacheable(cache="data", key="#params['id']")
func GetData(params map[string]interface{}) (*Data, error)
```

---

## 4. 缓存后端配置

### 4.1 Memory 后端（默认）

```go
import "github.com/coderiser/go-cache/pkg/backend"

// 创建 Memory 后端
memoryBackend := backend.NewMemoryBackend(&backend.MemoryConfig{
    MaxSize:     10000,        // 最大条目数
    DefaultTTL:  30 * time.Minute,
})
```

### 4.2 Redis 后端

```go
import "github.com/coderiser/go-cache/pkg/backend"

// 创建 Redis 后端
redisBackend, err := backend.NewRedisBackend(&backend.RedisConfig{
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
})
if err != nil {
    // 降级到 Memory 后端
    memoryBackend := backend.NewMemoryBackend(&backend.MemoryConfig{
        MaxSize:     10000,
        DefaultTTL:  30 * time.Minute,
    })
}
```

### 4.3 Hybrid 后端（L1 + L2）

```go
import "github.com/coderiser/go-cache/pkg/backend"

// 创建 L1（Memory）和 L2（Redis）
l1Backend := backend.NewMemoryBackend(&backend.MemoryConfig{
    MaxSize:     5000,
    DefaultTTL:  10 * time.Minute,
})

l2Backend, _ := backend.NewRedisBackend(&backend.RedisConfig{
    Addr:       "localhost:6379",
    DefaultTTL: 1 * time.Hour,
})

// 创建 Hybrid 后端
hybridBackend := backend.NewHybridBackend(l1Backend, l2Backend)
```

### 4.4 注册到管理器

```go
import "github.com/coderiser/go-cache/pkg/core"

manager := core.NewCacheManager()
manager.RegisterCache("users", memoryBackend)
manager.RegisterCache("products", redisBackend)
manager.RegisterCache("sessions", hybridBackend)
```

---

## 5. 高级配置

### 5.1 缓存保护机制

```go
import "github.com/coderiser/go-cache/pkg/core"

// 配置保护机制
protectionConfig := &core.ProtectionConfig{
    EnablePenetrationProtection: true,  // 穿透保护（空值缓存）
    EmptyValueTTL:               5 * time.Minute,
    
    EnableBreakdownProtection:   true,  // 击穿保护（Singleflight）
    EnableAvalancheProtection:   true,  // 雪崩保护（TTL 抖动）
    TTLJitterFactor:             0.1,   // 10% 抖动
}

protection := core.NewCacheProtection(protectionConfig)
```

### 5.2 自定义 Key 生成器

```go
import "github.com/coderiser/go-cache/pkg/core"

// 实现 KeyGenerator 接口
type CustomKeyGenerator struct{}

func (g *CustomKeyGenerator) Generate(methodName string, args []interface{}) string {
    // 自定义 Key 生成逻辑
    return fmt.Sprintf("custom:%s:%v", methodName, args[0])
}

// 注册自定义 Key 生成器
manager.SetKeyGenerator(&CustomKeyGenerator{})
```

### 5.3 缓存监听器

```go
import "github.com/coderiser/go-cache/pkg/core"

// 实现 CacheListener 接口
type MyListener struct{}

func (l *MyListener) OnCacheHit(cacheName, key string) {
    log.Printf("Cache hit: %s:%s", cacheName, key)
}

func (l *MyListener) OnCacheMiss(cacheName, key string) {
    log.Printf("Cache miss: %s:%s", cacheName, key)
}

// 注册监听器
manager.AddListener(&MyListener{})
```

### 5.4 优雅关闭

```go
func main() {
    defer cache.CloseGlobalManager()
    
    // 业务逻辑...
}
```

---

## 6. 缓存统计与监控

### 6.1 获取缓存统计

```go
import "github.com/coderiser/go-cache/pkg/core"

func printStats(manager core.CacheManager) {
    cache, _ := manager.GetCache("products")
    stats := cache.Stats()
    
    fmt.Printf("Hits: %d\n", stats.Hits)
    fmt.Printf("Misses: %d\n", stats.Misses)
    fmt.Printf("Hit Rate: %.1f%%\n", stats.HitRate * 100)
    fmt.Printf("Size: %d\n", stats.Size)
}
```

### 6.2 Prometheus 指标

```go
// 启用 Prometheus 指标
manager.EnableMetrics()

// 在 /metrics 端点暴露
// http.Handle("/metrics", promhttp.Handler())
```

### 6.3 监控缓存健康

```go
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        cache, _ := manager.GetCache("users")
        stats := cache.Stats()
        
        if stats.HitRate < 0.5 {
            log.Printf("⚠️  Low cache hit rate: %.2f", stats.HitRate)
        }
        
        if stats.Size > 9000 {
            log.Printf("⚠️  Cache size near limit: %d", stats.Size)
        }
    }
}()
```

---

## 7. 最佳实践

### 7.1 缓存 Key 设计

✅ **推荐:**
```go
// 使用有意义的命名空间
@cacheable(cache="users", key="#id")
@cacheable(cache="products", key="#category.Id + ':' + #page")

// 使用参数属性
@cacheable(cache="orders", key="#order.Customer.Id + ':' + #status")
```

❌ **避免:**
```go
// 避免使用可变数据
@cacheable(cache="data", key="time.Now().String()")  // ❌

// 避免过于复杂的表达式
@cacheable(cache="data", key="#a + #b + #c + #d + #e")  // ❌ 难以维护
```

### 7.2 TTL 设置

| 数据类型 | 推荐 TTL | 说明 |
|----------|----------|------|
| 用户信息 | 30m - 1h | 频繁访问，变化较少 |
| 商品信息 | 1h - 24h | 变化较少 |
| 配置数据 | 24h - 7d | 很少变化 |
| 会话数据 | 2h - 24h | 根据业务需求 |
| 临时数据 | 5m - 30m | 短期有效 |

### 7.3 缓存更新策略

```go
// 策略 1: 更新时同步更新缓存
@cacheput(cache="users", key="#id", ttl="30m")
func UpdateUser(id int64, name string) (*User, error)

// 策略 2: 更新时清除缓存（下次读取时重建）
@cacheevict(cache="users", key="#id")
func UpdateUser(id int64, name string) error

// 策略 3: 批量更新时清除所有
@cacheevict(cache="users", allEntries=true)
func BatchUpdateUsers(users []*User) error
```

### 7.4 错误处理

```go
// 推荐：带错误处理的初始化
func init() {
    decorated, err := proxy.SimpleDecorateWithError(&UserService{})
    if err != nil {
        log.Printf("⚠️  Cache decoration failed: %v, using fallback", err)
        UserService = &UserService{}
        return
    }
    UserService = decorated.(*UserService)
    log.Println("✓ Cache initialized successfully")
}
```

---

## 8. 常见问题

### Q1: 生成的代码在哪里？

**A:** 在执行 `go generate` 的目录下的 `.cache-gen/` 文件夹中。

### Q2: 如何修改缓存配置？

**A:** 直接修改注解参数：
```go
// @cacheable(cache="products", key="#id", ttl="2h")  // 改为 2 小时
```

### Q3: 如何禁用缓存？

**A:** 删除注解或注释掉 `go generate`。

### Q4: 支持哪些后端？

**A:** 支持 Memory、Redis、Hybrid（L1+L2）后端。通过 `SetGlobalManager()` 配置。

### Q5: 如何处理缓存穿透？

**A:** 框架内置空值缓存保护：
```go
config := &core.ProtectionConfig{
    EnablePenetrationProtection: true,
    EmptyValueTTL:               5 * time.Minute,
}
```

### Q6: 如何监控缓存命中率？

**A:** 使用 `cache.Stats()` 或启用 Prometheus 指标：
```go
manager.EnableMetrics()
```

### Q7: SpEL 表达式性能如何？

**A:** SpEL 求值 < 50μs，对性能影响极小。复杂表达式可考虑预编译。

### Q8: 支持并发访问吗？

**A:** 是的，所有缓存后端都是线程安全的。

---

**更多问题？** 欢迎提交 Issue 或查看 [GitHub Discussions](https://github.com/coderiser/go-cache/discussions)。
