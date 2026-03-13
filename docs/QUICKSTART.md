# 快速开始 - Go-Cache Framework

## 3 步使用缓存

Go-Cache Framework 采用**代码生成方案（方案 A）**，遵循"用户友好，业务代码侵入少"的原则。

### 步骤 1: 添加注解

在你的服务方法上添加缓存注解：

```go
package service

// UserServiceInterface 用户服务接口
type UserServiceInterface interface {
    GetUser(id int64) (*model.User, error)
    CreateUser(user *model.User) (*model.User, error)
    UpdateUser(id int64, user *model.User) (*model.User, error)
    DeleteUser(id int64) error
}

// userService 用户服务实现
type userService struct {
    // 业务依赖
}

// GetUser 获取用户 - 带 @cacheable 注解
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *userService) GetUser(id int64) (*model.User, error) {
    // 纯业务代码，无需关心缓存
    // ...
}

// CreateUser 创建用户 - 带 @cacheput 注解
// @cacheput(cache="users", key="#user.ID", ttl="30m")
func (s *userService) CreateUser(user *model.User) (*model.User, error) {
    // 纯业务代码，无需关心缓存
    // ...
}
```

### 步骤 2: 添加 generate 指令

在 service 包的文件顶部添加代码生成指令：

```go
//go:generate go run ../../../cmd/generator/main.go ./...

package service
```

### 步骤 3: 运行代码生成

```bash
# 生成注解元数据和包装器代码
go generate ./...

# 编译项目
go build
```

完成！业务代码无需其他改动。

---

## 初始化服务

在 `init.go` 或 `main.go` 中初始化服务：

```go
package service

import (
    "github.com/coderiser/go-cache/pkg/core"
    "github.com/coderiser/go-cache/pkg/proxy"
)

var (
    cacheManager core.CacheManager
    UserService  UserServiceInterface  // 使用接口类型
)

func InitCache() {
    // 1. 创建缓存管理器
    cacheManager = core.NewCacheManager()
    
    // 2. 配置缓存后端（Memory/Redis/Hybrid）
    // ...
    
    // 3. 创建原始服务并使用泛型装饰
    rawService := NewUserService()
    decorated := proxy.SimpleDecorateWithManager(rawService, cacheManager)
    
    // 4. 使用生成的包装器实现接口（方案 A: 代码生成）
    UserService = NewDecoratedUserService(decorated)
}
```

---

## 使用服务

在 handler 或其他业务代码中直接使用：

```go
package handler

type UserHandler struct {
    userService service.UserServiceInterface  // ✅ 依赖接口
}

func (h *UserHandler) GetUser(c *gin.Context) {
    id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
    
    // 直接使用接口方法（类型安全，自动缓存）
    user, err := h.userService.GetUser(id)
    if err != nil {
        c.JSON(404, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, user)
}
```

---

## 核心注解

| 注解 | 说明 | 示例 |
|------|------|------|
| `@cacheable` | 缓存读取（未命中时执行方法并缓存结果） | `@cacheable(cache="users", key="#id", ttl="30m")` |
| `@cacheput` | 强制更新缓存（总是执行方法并更新缓存） | `@cacheput(cache="users", key="#user.ID", ttl="30m")` |
| `@cacheevict` | 删除缓存 | `@cacheevict(cache="users", key="#id")` |

---

## SpEL 表达式

```go
// 引用参数
@cacheable(cache="orders", key="#userId + '_' + #status")

// 引用返回值（unless）
@cacheable(cache="data", key="#id", unless="#result == nil")

// 复杂表达式
@cacheable(cache="products", key="category:#catId:page:#page")
```

---

## 示例项目

查看完整示例：

- [gin-web](../examples/gin-web) - Web API 示例
- [grpc-service](../examples/grpc-service) - gRPC 服务示例
- [cron-job](../examples/cron-job) - 定时任务示例

---

*基于代码生成方案 A，业务代码零侵入，类型安全，IDE 友好*
