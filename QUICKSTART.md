# Go-Cache 快速开始

**版本:** v2.0 (运行时扫描)  
**更新日期:** 2026-03-14  
**特性:** 真正零配置，无需代码生成

---

## 🚀 5 分钟快速开始

### 步骤 1: 定义 Service 和注解

```go
// service/product.go
type ProductService struct {
    // 业务逻辑
}

// @cacheable(cache="products", key="#id", ttl="1h")
func (s *ProductService) GetProduct(id int64) (*model.Product, error) {
    // 业务逻辑 - 从数据库查询
    return s.getProductFromDB(id)
}

// @cacheput(cache="products", key="#id", ttl="1h")
func (s *ProductService) UpdatePrice(id int64, price float64) (*model.Product, error) {
    // 更新价格
}
```

### 步骤 2: 初始化服务

```go
// service/init.go
import (
    "github.com/coderiser/go-cache/pkg/proxy"
    _ "github.com/coderiser/go-cache/pkg/cache"  // 导入触发自动扫描
)

// 零配置！自动应用缓存
var ProductService = proxy.SimpleDecorate(&ProductService{})
```

### 步骤 3: 直接使用

```go
// main.go
import (
    "your-module/service"
)

func main() {
    // ✅ 真正零配置！无需代码生成
    product, err := service.ProductService.GetProduct(1)
    
    // 第一次调用：查询数据库 + 写入缓存
    // 第二次调用：直接返回缓存
}
```

**就这么简单！无需运行 `go generate`！**

---

## 🎯 核心注解

| 注解 | 说明 | 示例 |
|------|------|------|
| `@cacheable` | 缓存读取 | `@cacheable(cache="users", key="#id", ttl="30m")` |
| `@cacheput` | 缓存更新 | `@cacheput(cache="users", key="#id", ttl="30m")` |
| `@cacheevict` | 缓存清除 | `@cacheevict(cache="users", key="#id")` |

---

## 🎨 SpEL 表达式

```go
// 参数引用
@cacheable(cache="users", key="#id")

// 嵌套属性
@cacheable(cache="orders", key="#order.Customer.Id")

// 返回值字段
@cacheput(cache="users", key="#result.ID")

// 条件缓存
@cacheable(cache="data", key="#id", condition="#id > 0")

// 排除条件
@cacheable(cache="data", key="#id", unless="#result == nil")
```

**支持变量:** `#id`, `#user` (参数名) | `#p0`, `#0` (参数索引) | `result` (返回值)

---

## ⚙️ 高级配置

### 自定义缓存管理器

```go
// main.go
import (
    "github.com/coderiser/go-cache/pkg/cache"
    "github.com/coderiser/go-cache/pkg/core"
    "github.com/coderiser/go-cache/pkg/proxy"
)

func main() {
    // 创建自定义 Manager (Redis 后端)
    manager := core.NewCacheManager()
    // 配置 Redis...
    
    // 使用自定义 Manager 装饰服务
    var ProductService = proxy.SimpleDecorateWithManager(
        &ProductService{}, 
        manager,
    )
}
```

### 优雅关闭

```go
func main() {
    defer cache.CloseGlobalManager()
    
    // 应用逻辑...
}
```

---

## 🔍 常见问题

### Q: 需要运行代码生成器吗？

A: **不需要！** cache 包会在 init() 中自动扫描源代码中的注解。

### Q: 如何修改缓存配置？

A: 修改注解参数即可：
```go
// @cacheable(cache="products", key="#id", ttl="2h")  // 改为 2 小时
```

### Q: 支持哪些后端？

A: 支持内存、Redis、混合后端。通过 `proxy.SimpleDecorateWithManager()` 配置。

### Q: 自动扫描会影响性能吗？

A: 扫描只在程序启动时执行一次，运行时零开销。

---

## 📚 更多资源

- [用户指南](docs/user-guide.md) - 完整文档
- [API 参考](docs/api-reference.md) - pkg/cache 包文档
- [注解流程](ANNOTATION_FLOW.md) - 注解注册与使用流程
- [示例代码](examples/) - 完整示例

---

**最后更新:** 2026-03-14  
**状态:** ✅ 生产可用
