# Go-Cache Framework 迁移指南

**版本:** v1.0  
**最后更新:** 2026-03-14

---

## 目录

1. [概述](#1-概述)
2. [从方案 D 迁移到方案 G](#2-从方案-d-迁移到方案-g)
3. [从手动缓存迁移](#3-从手动缓存迁移)
4. [从其他框架迁移](#4-从其他框架迁移)
5. [代码对比示例](#5-代码对比示例)
6. [常见问题解答](#6-常见问题解答)

---

## 1. 概述

本指南帮助你从旧版本或其他缓存方案迁移到 Go-Cache Framework 方案 G（Beego 融合版）。

### 迁移收益

- ✅ **代码更简洁** - `cache.NewProductService()` 一行搞定
- ✅ **零配置** - 默认懒加载 Manager，无需手动初始化
- ✅ **向后兼容** - 保留旧 API，平滑迁移
- ✅ **类型安全** - 编译时检查，IDE 友好

---

## 2. 从方案 D 迁移到方案 G

### 2.1 方案 D（旧方式）

```go
// main.go (方案 D)
import (
    "github.com/coderiser/go-cache/pkg/core"
    "github.com/coderiser/go-cache/pkg/proxy"
)

func main() {
    // 1. 创建 Manager
    manager := core.NewCacheManager()
    
    // 2. 配置后端
    // ...
    
    // 3. 创建原始服务
    rawService := &ProductService{}
    
    // 4. 装饰服务（需要传 manager）
    decorated := proxy.SimpleDecorateWithManager(rawService, manager)
    
    // 5. 使用
    product, err := decorated.GetProduct(1)
}
```

### 2.2 方案 G（新方式）

```go
// main.go (方案 G)
import cached "your-module/service/.cache-gen"

func main() {
    // ✅ 一行搞定！缓存自动生效
    svc := cached.NewProductService()
    
    product, err := svc.GetProduct(1)
}
```

### 2.3 迁移步骤

#### 步骤 1: 更新代码生成器

```bash
# 安装最新版生成器
go install github.com/coderiser/go-cache/cmd/generator@latest
```

#### 步骤 2: 修改 Service 定义

**旧代码:**
```go
// service/product.go
type ProductService struct {
    db *gorm.DB
}

// @cacheable(cache="products", key="#id", ttl="1h")
func (s *ProductService) GetProduct(id int64) (*model.Product, error) {
    // ...
}
```

**新代码:**
```go
// service/product.go
//go:generate go run ../../../cmd/generator/main.go .

type ProductServiceInterface interface {
    GetProduct(id int64) (*model.Product, error)
}

type productService struct {
    db *gorm.DB
}

// @cacheable(cache="products", key="#id", ttl="1h")
func (s *productService) GetProduct(id int64) (*model.Product, error) {
    // ...
}
```

#### 步骤 3: 重新生成代码

```bash
go generate ./...
```

#### 步骤 4: 更新调用代码

**旧代码:**
```go
// main.go
manager := core.NewCacheManager()
rawService := &ProductService{}
decorated := proxy.SimpleDecorateWithManager(rawService, manager)

product, err := decorated.GetProduct(1)
```

**新代码:**
```go
// main.go
import cached "your-module/service/.cache-gen"

svc := cached.NewProductService()
product, err := svc.GetProduct(1)
```

### 2.4 保留自定义 Manager（可选）

如果需要使用自定义 Manager：

```go
// main.go
import (
    "github.com/coderiser/go-cache/pkg/core"
    cached "your-module/service/.cache-gen"
)

func main() {
    // 创建自定义 Manager
    manager := core.NewCacheManager()
    // 配置 Redis...
    
    // 设置为全局 Manager
    cache.SetGlobalManager(manager)
    
    // 使用（内部会自动使用全局 Manager）
    svc := cached.NewProductService()
    
    // 或者显式指定 Manager
    svc := cached.NewProductServiceWithManager(manager)
}
```

---

## 3. 从手动缓存迁移

### 3.1 手动缓存（旧方式）

```go
// service/product.go
type ProductService struct {
    cache *redis.Client
    db    *gorm.DB
}

func (s *ProductService) GetProduct(id int64) (*model.Product, error) {
    // 1. 检查缓存
    key := fmt.Sprintf("product:%d", id)
    cached, err := s.cache.Get(ctx, key).Result()
    if err == nil {
        var product model.Product
        json.Unmarshal([]byte(cached), &product)
        return &product, nil
    }
    
    // 2. 缓存未命中，查询数据库
    var product model.Product
    err = s.db.First(&product, id).Error
    if err != nil {
        return nil, err
    }
    
    // 3. 写入缓存
    data, _ := json.Marshal(product)
    s.cache.Set(ctx, key, data, 1*time.Hour)
    
    return &product, nil
}

func (s *ProductService) UpdatePrice(id int64, price float64) (*model.Product, error) {
    // 更新数据库
    // ...
    
    // 删除缓存
    key := fmt.Sprintf("product:%d", id)
    s.cache.Del(ctx, key)
    
    return product, nil
}
```

### 3.2 使用 Go-Cache（新方式）

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
    // 纯业务逻辑，无需关心缓存
    var product model.Product
    err := s.db.First(&product, id).Error
    return &product, err
}

// @cacheput(cache="products", key="#id", ttl="1h")
func (s *productService) UpdatePrice(id int64, price float64) (*model.Product, error) {
    // 纯业务逻辑
    var product model.Product
    err := s.db.Model(&product).Where("id = ?", id).Update("price", price).Error
    return &product, err
}
```

### 3.3 迁移收益

| 维度 | 手动缓存 | Go-Cache |
|------|----------|----------|
| 代码行数 | ~30 行 | ~5 行 |
| 缓存逻辑 | 分散在各方法 | 集中管理 |
| 错误处理 | 手动处理 | 自动降级 |
| 可测试性 | 需 mock 缓存 | 接口隔离 |
| 可维护性 | 低 | 高 |

---

## 4. 从其他框架迁移

### 4.1 从 go-cache/go-cache 迁移

**旧代码:**
```go
import "github.com/patrickmn/go-cache"

var cache = cache.New(5*time.Minute, 10*time.Minute)

func GetUser(id string) (*User, error) {
    if cached, found := cache.Get("user:" + id); found {
        return cached.(*User), nil
    }
    
    user, err := db.GetUser(id)
    if err != nil {
        return nil, err
    }
    
    cache.Set("user:" + id, user, cache.DefaultExpiration)
    return user, nil
}
```

**新代码:**
```go
//go:generate go run ../../../cmd/generator/main.go .

type UserServiceInterface interface {
    GetUser(id string) (*User, error)
}

// @cacheable(cache="users", key="#id", ttl="5m")
func (s *userService) GetUser(id string) (*User, error) {
    return db.GetUser(id)
}
```

### 4.2 从 singleflight 迁移

**旧代码:**
```go
import "golang.org/x/sync/singleflight"

var group singleflight.Group

func GetData(key string) (*Data, error) {
    v, err, _ := group.Do(key, func() (interface{}, error) {
        return db.GetData(key)
    })
    return v.(*Data), err
}
```

**新代码:**
```go
// 框架内置 Singleflight 保护
// @cacheable(cache="data", key="#key")
func (s *dataService) GetData(key string) (*Data, error) {
    return db.GetData(key)
}

// 配置保护
config := &core.ProtectionConfig{
    EnableBreakdownProtection: true,
}
```

---

## 5. 代码对比示例

### 5.1 用户服务完整对比

#### 旧方式（手动缓存）

```go
// service/user.go
type UserService struct {
    cache *redis.Client
    db    *gorm.DB
    mu    sync.Mutex
}

func (s *UserService) GetUser(id int64) (*User, error) {
    ctx := context.Background()
    key := fmt.Sprintf("user:%d", id)
    
    // 检查缓存
    cached, err := s.cache.Get(ctx, key).Result()
    if err == nil {
        var user User
        if err := json.Unmarshal([]byte(cached), &user); err == nil {
            return &user, nil
        }
    }
    
    // 防止缓存击穿
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // 双重检查
    cached, err = s.cache.Get(ctx, key).Result()
    if err == nil {
        var user User
        if err := json.Unmarshal([]byte(cached), &user); err == nil {
            return &user, nil
        }
    }
    
    // 查询数据库
    var user User
    if err := s.db.First(&user, id).Error; err != nil {
        return nil, err
    }
    
    // 写入缓存
    data, _ := json.Marshal(user)
    s.cache.Set(ctx, key, data, 30*time.Minute)
    
    return &user, nil
}

func (s *UserService) UpdateUser(id int64, name string) (*User, error) {
    // 更新数据库
    if err := s.db.Model(&User{}).Where("id = ?", id).Update("name", name).Error; err != nil {
        return nil, err
    }
    
    // 删除缓存
    key := fmt.Sprintf("user:%d", id)
    s.cache.Del(context.Background(), key)
    
    return &User{ID: id, Name: name}, nil
}

func (s *UserService) DeleteUser(id int64) error {
    // 删除数据库
    if err := s.db.Delete(&User{}, id).Error; err != nil {
        return err
    }
    
    // 删除缓存
    key := fmt.Sprintf("user:%d", id)
    s.cache.Del(context.Background(), key)
    
    return nil
}
```

#### 新方式（Go-Cache 方案 G）

```go
// service/user.go
//go:generate go run ../../../cmd/generator/main.go .

type UserServiceInterface interface {
    GetUser(id int64) (*User, error)
    UpdateUser(id int64, name string) (*User, error)
    DeleteUser(id int64) error
}

type userService struct {
    db *gorm.DB
}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *userService) GetUser(id int64) (*User, error) {
    var user User
    err := s.db.First(&user, id).Error
    return &user, err
}

// @cacheput(cache="users", key="#id", ttl="30m")
func (s *userService) UpdateUser(id int64, name string) (*User, error) {
    if err := s.db.Model(&User{}).Where("id = ?", id).Update("name", name).Error; err != nil {
        return nil, err
    }
    return &User{ID: id, Name: name}, nil
}

// @cacheevict(cache="users", key="#id")
func (s *userService) DeleteUser(id int64) error {
    return s.db.Delete(&User{}, id).Error
}
```

### 5.2 调用代码对比

#### 旧方式

```go
// main.go
func main() {
    // 初始化 Redis
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    // 初始化服务
    userService := &UserService{
        cache: redisClient,
        db:    db,
    }
    
    // 使用
    user, err := userService.GetUser(123)
}
```

#### 新方式

```go
// main.go
import cached "your-module/service/.cache-gen"

func main() {
    // 零配置！
    userService := cached.NewUserService()
    
    // 使用
    user, err := userService.GetUser(123)
}
```

---

## 6. 常见问题解答

### Q1: 迁移需要多长时间？

**A:** 对于中等规模项目（10-20 个 Service），通常需要 1-2 天：
- 半天：理解方案和修改 Service 定义
- 半天：重新生成代码和更新调用
- 半天：测试和调试

### Q2: 迁移过程中会影响线上服务吗？

**A:** 不会。方案 G 向后兼容，可以：
1. 先在开发环境测试
2. 逐步迁移 Service（一个接一个）
3. 验证无误后上线

### Q3: 旧的缓存数据怎么办？

**A:** 如果 Key 格式不变，旧缓存数据仍然可用。如果 Key 格式变化，建议：
1. 上线前清空缓存
2. 或者设置较短的 TTL，让旧数据自然过期

### Q4: 性能会受影响吗？

**A:** 不会。方案 G 只是简化了调用方式，底层缓存逻辑相同。性能测试显示：
- 缓存命中延迟：< 1ms（Memory）/ < 5ms（Redis）
- SpEL 求值：< 50μs
- 代码生成：零运行时开销

### Q5: 如何回滚？

**A:** 方案 G 保留了旧 API，可以随时回滚：
```go
// 如果遇到问题，可以使用旧方式
manager := core.NewCacheManager()
rawService := &ProductService{}
decorated := proxy.SimpleDecorateWithManager(rawService, manager)
```

### Q6: 支持 Gradual Migration（渐进式迁移）吗？

**A:** 支持。可以：
1. 先迁移不关键的 Service
2. 验证稳定性和性能
3. 逐步迁移核心 Service

### Q7: 测试需要重写吗？

**A:** 不需要。由于使用接口，测试代码基本不变：
```go
// 测试代码（迁移前后相同）
func TestUserService(t *testing.T) {
    service := NewUserService()  // 或 cached.NewUserService()
    
    user, err := service.GetUser(123)
    assert.NoError(t, err)
    assert.NotNil(t, user)
}
```

### Q8: 遇到兼容性问题怎么办？

**A:** 提交 Issue 到 [GitHub](https://github.com/coderiser/go-cache/issues)，我们会提供：
- 详细迁移指导
- 兼容性修复
- 必要时提供迁移工具

---

## 附录：迁移检查清单

### 准备阶段

- [ ] 备份现有代码
- [ ] 安装最新版生成器
- [ ] 阅读本迁移指南
- [ ] 在开发环境测试

### 迁移阶段

- [ ] 修改 Service 定义（添加接口）
- [ ] 添加 `//go:generate` 指令
- [ ] 运行 `go generate ./...`
- [ ] 更新调用代码
- [ ] 编译验证

### 测试阶段

- [ ] 单元测试通过
- [ ] 集成测试通过
- [ ] 性能测试通过
- [ ] 缓存命中验证

### 上线阶段

- [ ] 灰度发布
- [ ] 监控缓存指标
- [ ] 观察错误日志
- [ ] 全量发布

---

**需要帮助？** 欢迎联系 [Go-Cache Team](mailto:support@go-cache.dev) 或提交 [GitHub Issue](https://github.com/coderiser/go-cache/issues)。
