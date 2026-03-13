# Go-Cache 方案 G 快速开始

**版本:** v1.0  
**更新日期:** 2026-03-14  
**特性:** 零配置，注解后直接使用

---

## 🚀 5 分钟快速开始

### 步骤 1: 定义 Service 和注解

```go
// service/product.go
//go:generate go run ../../../cmd/generator/main.go .

type ProductServiceInterface interface {
    GetProduct(id int64) (*model.Product, error)
}

type productService struct {
    manager core.CacheManager
}

func NewProductService(manager core.CacheManager) *productService {
    return &productService{manager: manager}
}

// @cacheable(cache="products", key="#id", ttl="1h")
func (s *productService) GetProduct(id int64) (*model.Product, error) {
    // 业务逻辑 - 从数据库查询
    return s.getProductFromDB(id)
}

// @cacheput(cache="products", key="#id", ttl="1h")
func (s *productService) UpdatePrice(id int64, price float64) (*model.Product, error) {
    // 更新价格
}
```

### 步骤 2: 执行代码生成

```bash
go generate ./...
```

生成文件:
- `.cache-gen/auto_register.go` - 注解自动注册
- `.cache-gen/product_cached.go` - 带缓存的实现

### 步骤 3: 使用服务 (零配置!)

```go
// main.go
import cached "your-module/service/.cache-gen"

func main() {
    // ✅ 一行搞定！缓存自动生效
    svc := cached.NewProductService()
    
    product, err := svc.GetProduct(1)
    
    // 第一次调用：查询数据库 + 写入缓存
    // 第二次调用：直接返回缓存
}
```

---

## 📝 注解语法

### @cacheable (缓存读取)

```go
// @cacheable(cache="缓存名", key="Key 表达式", ttl="过期时间")
@cacheable(cache="users", key="#id", ttl="30m")
func GetUser(id int64) (*User, error)

// 支持 SpEL 表达式
@cacheable(cache="users", key="#user.Id", ttl="1h")
func GetUser(user *UserRequest) (*User, error)

// 条件缓存
@cacheable(cache="users", key="#id", condition="#id > 0")
func GetUser(id int64) (*User, error)

// 排除条件
@cacheable(cache="users", key="#id", unless="#result == nil")
func GetUser(id int64) (*User, error)
```

### @cacheput (缓存更新)

```go
// @cacheput(cache="缓存名", key="Key 表达式", ttl="过期时间")
@cacheput(cache="users", key="#id", ttl="30m")
func UpdateUser(id int64, name string) (*User, error)
```

### @cacheevict (缓存清除)

```go
// 方法执行后清除
@cacheevict(cache="users", key="#id")
func DeleteUser(id int64) error

// 方法执行前清除
@cacheevict(cache="users", key="#id", before=true)
func DeleteUser(id int64) error
```

---

## 🎯 SpEL 表达式语法

### 支持的变量

| 变量 | 说明 | 示例 |
|------|------|------|
| `#id`, `#user` | 参数名 | `key="#id"` |
| `#p0`, `#p1` | 参数索引 | `key="#p0"` |
| `#0`, `#1` | 参数索引 (简写) | `key="#0"` |
| `result` | 返回值 (仅 unless) | `unless="#result == nil"` |

### 表达式示例

```go
// 访问参数属性
@cacheable(cache="users", key="#user.Id")

// 访问嵌套属性
@cacheable(cache="orders", key="#order.Customer.Id")

// 静态方法调用
@cacheable(cache="data", key="T(md5).Sum(#id)")

// 条件表达式
@cacheable(cache="data", key="#id", condition="#id > 0 && #id < 1000")
```

---

## ⚙️ 高级配置

### 自定义缓存管理器

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
}
```

### 优雅关闭

```go
func main() {
    defer cache.CloseGlobalManager()
    
    svc := cached.NewProductService()
    // 业务逻辑...
}
```

---

## 📊 缓存统计

```go
func printStats(manager core.CacheManager) {
    cache, _ := manager.GetCache("products")
    stats := cache.Stats()
    
    fmt.Printf("Hits: %d\n", stats.Hits)
    fmt.Printf("Misses: %d\n", stats.Misses)
    fmt.Printf("Hit Rate: %.1f%%\n", stats.HitRate * 100)
}
```

---

## 🔍 常见问题

### Q: 生成的代码在哪里？

A: 在执行 `go generate` 的目录下的 `.cache-gen/` 文件夹中。

### Q: 如何修改缓存配置？

A: 修改注解参数即可：
```go
// @cacheable(cache="products", key="#id", ttl="2h")  // 改为 2 小时
```

### Q: 如何禁用缓存？

A: 删除注解或注释掉 `go generate`。

### Q: 支持哪些后端？

A: 支持内存、Redis、混合后端。通过 `SetGlobalManager()` 配置。

---

## 📚 更多资源

- [实施方案](./docs/cache-implementation-plan.md)
- [设计文档](./docs/cache-integration-beego-proposal.md)
- [示例代码](./examples/cron-job/)

---

**最后更新:** 2026-03-14  
**状态:** ✅ 生产可用
