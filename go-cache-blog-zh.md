# 知乎/掘金/SegmentFault 格式

---

**标题**：我用 Go 实现了类似 Spring Cache 的注解式缓存框架

**标签**：Go # 缓存 # 开源项目 # 后端开发 # 架构设计

---

## 背景

在 Java 生态中，Spring Cache 几乎是缓存的标准解决方案。只需要在方法上添加 `@Cacheable` 注解，就能自动实现缓存逻辑，业务代码完全无侵入。

但在 Go 语言中，由于缺乏注解系统和运行时 AOP 机制，实现类似的功能一直是个挑战。现有的 Go 缓存库大多需要手动编写缓存逻辑，或者侵入业务代码。

作为一个追求极致开发体验的开发者，我决定挑战这个难题 —— **在 Go 中实现一个真正优雅的注解式缓存框架**。

**项目地址**：https://github.com/coderiser/go-cache

---

## 目标

我想要实现的框架需要满足：

1. **零侵入** - 业务代码不需要任何缓存逻辑
2. **注解驱动** - 通过简单注解声明缓存行为
3. **自动代理** - 运行时自动拦截方法调用
4. **多后端支持** - Memory / Redis 可插拔
5. **高性能** - 缓存命中延迟 < 1ms

---

## 技术挑战与解决方案

### 挑战 1：Go 没有注解

Java 的 `@Cacheable` 是语言级别的注解，但 Go 没有这个特性。

**解决方案**：使用**注释 + AST 解析**

```go
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id string) (*User, error) {
    // 业务代码
}
```

通过 `go/packages` 和 `go/ast` 在编译时扫描注释，提取缓存配置。

---

### 挑战 2：Go 没有运行时 AOP

Spring 可以在运行时动态代理方法，但 Go 不支持这种机制。

**解决方案**：**反射 + 动态代理**

```go
type Proxy interface {
    Call(methodName string, args []reflect.Value) []reflect.Value
}

func (p *proxyImpl) Call(methodName string, args []reflect.Value) []reflect.Value {
    // 1. 检查缓存
    // 2. 命中则返回
    // 3. 未命中则调用原方法
    // 4. 写入缓存
}
```

---

### 挑战 3：如何最小化用户集成成本？

最初的方案需要用户手动调用 `cache.Decorate()`，但这不够优雅。

**最终方案**：**全局变量 + init() 自动装饰**

```go
// 1. 定义全局变量
var UserService = &UserService{}

// 2. 添加注解
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id string) (*User, error) {
    return db.FindUser(id)
}

// 3. init() 自动装饰
func init() {
    cache.AutoDecorate(&UserService)
}

// 使用时（完全透明）
user, _ := UserService.GetUser("123")  // 自动缓存！
```

---

## 核心功能

### 1. @Cacheable 缓存读取

```go
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id string) (*User, error) {
    var u User
    s.db.First(&u, id)
    return &u, nil
}
```

- 首次调用执行原方法，结果写入缓存
- 后续调用直接返回缓存结果
- 支持 SpEL 表达式动态生成 Key

---

### 2. @CachePut 强制更新

```go
// @cacheput(cache="users", key="#user.Id")
func (s *UserService) UpdateUser(user *User) error {
    return s.db.Save(user).Error
}
```

- 始终执行原方法
- 方法执行后强制更新缓存

---

### 3. @CacheEvict 删除缓存

```go
// @cacheevict(cache="users", key="#userId", before=true)
func (s *UserService) DeleteUser(userId string) error {
    return s.db.Delete(&User{}, userId).Error
}
```

- 支持 `before=true` 在方法执行前删除
- 支持 `allEntries=true` 清空整个缓存

---

## SpEL 表达式

框架集成了 `expr` 引擎，支持强大的表达式语法：

```go
// 引用参数
@cacheable(cache="orders", key="#userId + '_' + #status")

// 引用返回值（条件过滤）
@cacheable(cache="data", key="#id", unless="#result == nil")

// 复杂表达式
@cacheable(cache="products", key="category:#catId:page:#page")

// 三元表达式
@cacheable(cache="config", key="#env == 'prod' ? 'p_' + #id : 't_' + #id")
```

---

## 缓存异常保护

### 缓存穿透（查询不存在的数据）

```go
// 自动空值缓存
if result == nil {
    cache.Set(key, nilMarker, 5*time.Minute)
}
```

### 缓存击穿（热点 Key 过期）

```go
// Singleflight 单飞模式
group.Do(key, func() (interface{}, error) {
    // 只让一个请求查数据库
})
```

### 缓存雪崩（大量 Key 同时过期）

```go
// TTL 随机偏移
ttl := baseTTL + time.Duration(rand.Int63n(jitter))
```

---

## 性能表现

| 场景 | 延迟 |
|------|------|
| Memory 命中 | < 1ms |
| Redis 命中 | < 5ms |
| SpEL 求值 | < 50μs |
| 代码生成 | < 1s (100 方法) |

测试覆盖率：**83%+**

---

## 快速开始

### 安装

```bash
go get github.com/coderiser/go-cache
```

### 使用示例

```go
package service

import "github.com/coderiser/go-cache"

var UserService = &UserService{}

type UserService struct {
    db *gorm.DB
}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id string) (*User, error) {
    var u User
    err := s.db.Where("id = ?", id).First(&u).Error
    return &u, err
}

func init() {
    cache.AutoDecorate(&UserService)
}
```

### 生成元数据

```bash
go-cache-gen ./...
```

---

## 技术选型

| 组件 | 技术 | 理由 |
|------|------|------|
| SpEL 引擎 | `expr` | 纯 Go 实现，性能优秀 |
| Redis 客户端 | `go-redis/v9` | 社区活跃，功能完整 |
| AST 解析 | `go/packages` | 官方标准库 |
| 动态代理 | `reflect.MakeFunc` | 运行时反射 |

---

## 项目地址

**GitHub**: https://github.com/coderiser/go-cache

**文档**: 
- [架构设计](https://github.com/coderiser/go-cache/blob/main/docs/ARCHITECTURE.md)
- [接口定义](https://github.com/coderiser/go-cache/blob/main/docs/INTERFACE_SPEC.md)
- [集成指南](https://github.com/coderiser/go-cache/blob/main/docs/INTEGRATION_GUIDE.md)

---

## 总结

在 Go 中实现注解式缓存框架确实有挑战，但通过合理的技术选型和架构设计，我们成功实现了：

✅ 零侵入业务代码  
✅ 注解驱动缓存逻辑  
✅ 运行时自动代理  
✅ 多后端支持  
✅ 高性能表现  

虽然 Go 没有 Java 那样的注解和 AOP，但我们找到了 Go 语言下的最优解 —— **一行 init() + go generate**，既保持了 Go 的简洁性，又获得了接近 Spring Cache 的开发体验。

---

*欢迎 Star、Fork、提 Issue！*

---

**（完）**
