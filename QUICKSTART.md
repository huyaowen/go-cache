# Go-Cache 快速开始

**版本:** v2.0  
**更新日期:** 2026-03-16  
**特性:** gocache 代码生成，零配置使用

---

## 🚀 5 分钟快速开始

### 步骤 1: 安装框架

```bash
# 安装框架
go get github.com/coderiser/go-cache@latest

# 安装 CLI 工具
go install github.com/coderiser/go-cache/cmd/gocache@latest

# 验证安装
gocache --help
```

### 步骤 2: 定义 Service 和注解

```go
// service/user.go
package service

import "database/sql"

//go:generate gocache scan .

// UserServiceInterface 定义服务接口
type UserServiceInterface interface {
    GetUser(id int64) (*User, error)
    CreateUser(name, email string) (*User, error)
    DeleteUser(id int64) error
}

// userService 服务实现
type userService struct {
    db *sql.DB
}

// NewUserServiceRaw 创建原始服务（不带缓存）
func NewUserServiceRaw(db *sql.DB) *userService {
    return &userService{db: db}
}

// GetUser 获取用户 - 带缓存
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *userService) GetUser(id int64) (*User, error) {
    // 业务逻辑 - 从数据库查询
    var user User
    err := s.db.QueryRow("SELECT * FROM users WHERE id = ?", id).Scan(&user.ID, &user.Name, &user.Email)
    return &user, err
}

// CreateUser 创建用户 - 带缓存更新
// @cacheput(cache="users", key="#result.ID", ttl="30m")
func (s *userService) CreateUser(name, email string) (*User, error) {
    // 业务逻辑 - 插入数据
    result, err := s.db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", name, email)
    if err != nil {
        return nil, err
    }
    id, _ := result.LastInsertId()
    return &User{ID: id, Name: name, Email: email}, nil
}

// DeleteUser 删除用户 - 带缓存清除
// @cacheevict(cache="users", key="#id")
func (s *userService) DeleteUser(id int64) error {
    // 业务逻辑 - 删除数据
    _, err := s.db.Exec("DELETE FROM users WHERE id = ?", id)
    return err
}
```

### 步骤 3: 生成缓存包装器

```bash
# 扫描并生成代码
gocache scan ./service

# 生成文件：service/user_cached.go
# ✅ 无需手动导入，自动生成
```

### 步骤 4: 配置缓存管理器

```go
// main.go
package main

import (
    "log"
    "time"
    
    "github.com/coderiser/go-cache/pkg/cache"
    "github.com/coderiser/go-cache/pkg/core"
    "github.com/coderiser/go-cache/pkg/backend"
    "your-project/service"
)

func main() {
    // 1. 创建缓存管理器
    manager := core.NewCacheManager()
    
    // 2. 配置 Redis 后端（生产环境推荐）
    redisBackend, err := backend.NewRedisBackend(&backend.RedisConfig{
        Addr:       "localhost:6379",
        Password:   "",
        DB:         0,
        Prefix:     "myapp",
        DefaultTTL: 30 * time.Minute,
        PoolSize:   10,
    })
    if err != nil {
        log.Printf("Redis 连接失败，降级到内存后端：%v", err)
        
        // 降级到 Memory 后端（开发环境可用）
        redisBackend = backend.NewMemoryBackend(&backend.CacheConfig{
            Name:           "users",
            MaxSize:        10000,
            DefaultTTL:     30 * time.Minute,
            EvictionPolicy: "lru",
        })
    }
    
    // 3. 注册缓存
    manager.RegisterCache("users", redisBackend)
    
    // 4. 设置为全局管理器
    cache.SetGlobalManager(manager)
    defer cache.CloseGlobalManager()
    
    // 5. 初始化数据库
    db := initDB()
    
    // 服务会在使用时自动创建（通过 NewUserService）
    
    // 运行业务逻辑
    runServer()
}
```

### 步骤 5: 使用服务

```go
// handler/user_handler.go
package handler

import "your-project/service"

// 方式 1：使用生成的 NewUserService（推荐）
func GetUserHandler(id int64) {
    userService := service.NewUserService(db)  // 自动带缓存
    user, err := userService.GetUser(id)
    if err != nil {
        log.Printf("获取用户失败：%v", err)
        return
    }
    // 使用 user...
}

// 方式 2：使用 go:generate（更简洁）
// 在 service/user.go 顶部添加：
// //go:generate gocache scan .
// 然后运行：go generate ./...
```

---

## 📝 注解语法

### @cacheable (缓存读取)

```go
// 基本用法
// @cacheable(cache="users", key="#id", ttl="30m")
func GetUser(id int64) (*User, error)

// 支持 SpEL 表达式
// @cacheable(cache="users", key="#user.Id", ttl="1h")
func GetUser(user *UserRequest) (*User, error)

// 条件缓存
// @cacheable(cache="users", key="#id", condition="#id > 0")
func GetUser(id int64) (*User, error)

// 排除条件
// @cacheable(cache="users", key="#id", unless="#result == nil")
func GetUser(id int64) (*User, error)
```

### @cacheput (缓存更新)

```go
// 基本用法
// @cacheput(cache="users", key="#id", ttl="30m")
func UpdateUser(id int64, name string) (*User, error)

// 使用返回值
// @cacheput(cache="users", key="#result.ID", ttl="30m")
func CreateUser(name string) (*User, error)
```

### @cacheevict (缓存清除)

```go
// 方法执行后清除（默认）
// @cacheevict(cache="users", key="#id")
func DeleteUser(id int64) error

// 方法执行前清除
// @cacheevict(cache="users", key="#id", before=true)
func DeleteUser(id int64) error
```


```go
func GetUser(userId int64) (*User, error)

func CreateOrder(userId int64, order *Order) error
```

---

## 🎯 SpEL 表达式语法

### 支持的变量

| 变量 | 说明 | 示例 |
|------|------|------|
| `#id`, `#user` | 参数名 | `key="#id"` |
| `#p0`, `#p1` | 参数索引 | `key="#p0"` |
| `#0`, `#1` | 参数索引 (简写) | `key="#0"` |
| `#result` | 返回值 (仅 `@cacheput`) | `key="#result.ID"` |

### 表达式示例

```go
// 访问参数属性
// @cacheable(cache="users", key="#user.Id")

// 访问嵌套属性
// @cacheable(cache="orders", key="#order.Customer.Id")

// 字符串拼接
// @cacheable(cache="data", key="#prefix + ':' + #id")

// 条件表达式
// @cacheable(cache="data", key="#id", condition="#id > 0 && #id < 1000")
```

---

## ⚙️ 高级配置

### 自定义缓存管理器

```go
// main.go
import (
    "github.com/coderiser/go-cache/pkg/core"
    "github.com/coderiser/go-cache/pkg/cache"
    "github.com/coderiser/go-cache/pkg/backend"
)

func main() {
    // 创建自定义 Manager (Redis 后端)
    manager := core.NewCacheManager()
    
    redisBackend, _ := backend.NewRedisBackend(&backend.RedisConfig{
        Addr:       "localhost:6379",
        DefaultTTL: 30 * time.Minute,
    })
    
    manager.RegisterCache("users", redisBackend)
    cache.SetGlobalManager(manager)
    defer cache.CloseGlobalManager()
    
    // 使用服务（直接调用 NewUserService）
    // userService := service.NewUserService(db)
}
```

### 混合后端 (L1 + L2)

```go
// 使用 HybridBackend 实现 L1 (Memory) + L2 (Redis)
hybridBackend, _ := backend.NewHybridBackend(&backend.HybridConfig{
    L1Config: &backend.CacheConfig{
        Name:      "local",
        MaxSize:   1000,
        DefaultTTL: 5 * time.Minute,
    },
    L2Config: &backend.RedisConfig{
        Addr:       "localhost:6379",
        DefaultTTL: 30 * time.Minute,
    },
    L1WriteBackTTL: 5 * time.Minute,
})

manager.RegisterCache("products", hybridBackend)
```

---

## 📊 缓存统计

```go
func printStats() {
    manager := cache.GetGlobalManager()
    userCache, _ := manager.GetCache("users")
    stats := userCache.Stats()
    
    log.Printf("Hits: %d", stats.Hits)
    log.Printf("Misses: %d", stats.Misses)
    log.Printf("Hit Rate: %.1f%%", stats.HitRate * 100)
    log.Printf("Sets: %d", stats.Sets)
    log.Printf("Deletes: %d", stats.Deletes)
}
```

---

## 🛠️ 代码生成命令

```bash
# 基本用法
gocache scan [directories]

# 示例
gocache scan ./...          # 扫描所有包
gocache scan ./service      # 扫描指定目录
gocache scan ./service ./repo  # 多目录扫描
gocache scan -h             # 查看帮助
```

**生成文件**: 在扫描目录中生成 `xxx_cached.go` 文件（如 `service/user_cached.go`）

---

## 🔍 常见问题

### Q: 生成的代码在哪里？

**A**: 在扫描目录中，与源文件同目录，文件名格式为 `xxx_cached.go`。

例如：`service/user.go` → `service/user_cached.go`

### Q: 如何修改缓存配置？

**A**: 修改注解参数即可：
```go
// @cacheable(cache="users", key="#id", ttl="2h")  // 改为 2 小时
```

### Q: 注解不生效怎么办？

**A**: 检查以下步骤：
1. 注解格式是否正确：`// @cacheable(cache="users", key="#id", ttl="30m")`
2. 是否定义了 `NewXxxServiceRaw()` 函数（如 `NewUserServiceRaw`）
3. 是否执行了 `gocache scan ./service` 或 `go generate ./...`
4. 生成的 `*_cached.go` 文件是否存在
5. 是否使用了 `service.NewUserService(db)`（不是 `NewUserServiceRaw`）

### Q: 如何禁用缓存？

**A**: 删除注解或不调用 `InitXxxService()`，直接使用原始服务：
```go
userService := service.NewUserService(db)  // 原始服务，无缓存
```

### Q: 支持哪些后端？

**A**: 支持以下后端：
- **Memory**: 内存缓存，适合开发和单机场景
- **Redis**: 分布式缓存，适合生产环境
- **Hybrid**: L1 (Memory) + L2 (Redis) 两级缓存

### Q: SpEL 表达式支持哪些操作？

**A**: 支持以下操作：
- 属性访问：`#user.Id`, `#order.Customer.Name`
- 字符串拼接：`#prefix + ':' + #id`
- 数值比较：`#id > 0`, `#count < 100`
- 逻辑运算：`#id > 0 && #status == 1`
- 三元运算：`#id > 0 ? #id : 0`

---

## 📚 更多资源

- [完整集成指南](./INTEGRATION_PLAN.md) - 详细集成步骤
- [用户指南](./docs/user-guide.md) - 完整使用文档
- [API 参考](./docs/api-reference.md) - pkg/cache 包文档
- [示例代码](./examples/) - Gin Web 和 gRPC 示例

---

**最后更新:** 2026-03-16  
**状态:** ✅ 生产可用
