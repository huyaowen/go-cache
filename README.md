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
- 🚀 **零配置启动** - 方案 G：`cache.NewProductService()` 一行搞定

## 📦 快速开始

### 方案 G：Beego 风格（推荐）

**零配置，注解后直接使用！**

#### 1. 定义服务和注解

```go
// service/product.go
//go:generate go run ../../../cmd/generator/main.go .

type ProductServiceInterface interface {
    GetProduct(id int64) (*model.Product, error)
}

type productService struct {
    // 业务依赖
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

#### 2. 执行代码生成

```bash
go generate ./...
```

生成文件:
- `.cache-gen/auto_register.go` - 注解自动注册
- `.cache-gen/product_cached.go` - 带缓存的实现

#### 3. 使用服务（零配置！）

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

### 方案 D：显式初始化（高级用户）

适合需要自定义缓存管理器的场景：

```go
// main.go
import (
    "github.com/coderiser/go-cache/pkg/core"
    "github.com/coderiser/go-cache/pkg/proxy"
)

func main() {
    // 创建自定义 Manager
    manager := core.NewCacheManager()
    // 配置 Redis 后端...
    
    // 装饰服务
    rawService := &ProductService{}
    decorated := proxy.SimpleDecorateWithManager(rawService, manager)
    
    // 使用
    product, err := decorated.GetProduct(1)
}
```

## 🎯 核心注解

| 注解 | 说明 | 示例 |
|------|------|------|
| `@cacheable` | 缓存读取 | `@cacheable(cache="users", key="#id", ttl="30m")` |
| `@cacheput` | 缓存更新 | `@cacheput(cache="users", key="#user.ID", ttl="30m")` |
| `@cacheevict` | 缓存清除 | `@cacheevict(cache="users", key="#id", before=true)` |

## 📖 注解语法

### @cacheable (缓存读取)

```go
// 基本用法
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

## 🎯 SpEL 表达式参考

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

## 🛡️ 缓存异常保护

框架内置三种保护机制：

1. **穿透保护** - 空值缓存（Nil Marker）
2. **击穿保护** - Singleflight（请求合并）
3. **雪崩保护** - TTL 随机偏移（Jitter）

```go
// 配置保护
config := &core.ProtectionConfig{
    EnablePenetrationProtection: true,
    EmptyValueTTL:               5 * time.Minute,
    EnableBreakdownProtection:   true,
    EnableAvalancheProtection:   true,
    TTLJitterFactor:             0.1,
}
```

## 📚 文档

- [快速开始](QUICKSTART.md) - 5 分钟上手
- [用户指南](docs/user-guide.md) - 详细使用指南
- [迁移指南](docs/migration-guide.md) - 从旧方案迁移
- [API 参考](docs/api-reference.md) - pkg/cache 包文档
- [架构设计](docs/ARCHITECTURE.md) - 技术架构说明
- [集成指南](docs/INTEGRATION_GUIDE.md) - 集成到现有项目

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

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

---

**Made with ❤️ by Go-Cache Team**

**版本:** v1.0  
**最后更新:** 2026-03-14  
**状态:** ✅ 生产可用
