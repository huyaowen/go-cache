# Go-Cache Framework

[![Go Reference](https://pkg.go.dev/badge/github.com/coderiser/go-cache.svg)](https://pkg.go.dev/github.com/coderiser/go-cache)
[![Go Report Card](https://goreportcard.com/badge/github.com/coderiser/go-cache)](https://goreportcard.com/report/github.com/coderiser/go-cache)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> **Go 语言注解式缓存框架** —— 让缓存变得简单优雅

---

## 🚀 特性

- ✨ **注解式缓存** - 一行注解，缓存自动生效
- 🚀 **零配置启动** - `InitXxxService()` 或 `NewCachedXxxService()` 直接使用
- 🎯 **SpEL 表达式** - 动态缓存 Key，灵活强大
- 💾 **多后端支持** - Memory / Redis / Hybrid 无缝切换
- 📊 **内置监控** - Prometheus 指标 + OpenTelemetry 追踪
- 🛡️ **异常保护** - 穿透/击穿/雪崩全保护
- 🛠️ **代码生成** - `gocache scan` 自动生成类型安全包装器

---

## 📦 快速开始

### 0️⃣ 安装

```bash
# 安装框架
go get github.com/coderiser/go-cache@latest

# 安装 CLI 工具
go install github.com/coderiser/go-cache/cmd/gocache@latest

# 验证安装
gocache --help
```

### 1️⃣ 编写业务代码

```go
// service/user.go
package service

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
}

// CreateUser 创建用户 - 带缓存更新
// @cacheput(cache="users", key="#result.ID", ttl="30m")
func (s *userService) CreateUser(name, email string) (*User, error) {
    // 业务逻辑 - 插入数据
}

// DeleteUser 删除用户 - 带缓存清除
// @cacheevict(cache="users", key="#id")
func (s *userService) DeleteUser(id int64) error {
    // 业务逻辑 - 删除数据
}
```

### 2️⃣ 生成缓存包装器

```bash
# 扫描并生成代码
gocache scan ./service

# 生成文件：service/user_cached.go
```

### 3️⃣ 配置缓存管理器

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
    
    // 2. 配置 Redis 后端
    redisBackend, err := backend.NewRedisBackend(&backend.RedisConfig{
        Addr:       "localhost:6379",
        DefaultTTL: 30 * time.Minute,
    })
    if err != nil {
        log.Fatal(err)
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

### 4️⃣ 使用服务

```go
// handler/user_handler.go
package handler

import "your-project/service"

// 方式 1：使用生成的 NewUserService（推荐）
func GetUserHandler(id int64) {
    userService := service.NewUserService(db)  // 自动带缓存
    user, err := userService.GetUser(id)
    if err != nil {
        // 处理错误
    }
}

// 方式 2：使用 go:generate（更简洁）
// 在 service/user.go 顶部添加：
// //go:generate gocache scan .
// 然后运行：go generate ./...
```

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

// 条件缓存
@cacheable(cache="data", key="#id", condition="#id > 0")

// 排除条件
@cacheable(cache="data", key="#id", unless="#result == nil")
```

**支持变量:** `#id`, `#user` (参数名) | `#p0`, `#0` (参数索引) | `result` (返回值)


## 🛠️ 代码生成

```bash
# 扫描注解并生成缓存包装器
gocache scan ./service

# 扫描多个目录
gocache scan ./service ./repo

# 查看帮助
gocache scan -h
```

## 🛠️ 构建命令

```bash
# 快速构建
./build.sh

# 或使用 Makefile
make build

# 运行测试
make test

# 查看帮助
make help
```

---

## 📚 文档

| 文档 | 说明 |
|------|------|
| [快速开始](QUICKSTART.md) | 5 分钟上手指南 |
| [用户指南](docs/user-guide.md) | 完整使用文档 |
| [API 参考](docs/api-reference.md) | pkg/cache 包文档 |
| [迁移指南](docs/migration-guide.md) | 从旧版本迁移 |
| [开发指南](CONTRIBUTING.md) | 贡献者指南 |

---

## 🧪 测试

```bash
# 单元测试
go test ./... -v

# 覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 性能基准
go test -bench=. -benchmem ./...
```

---

## 📊 性能指标

| 场景 | 延迟 | 说明 |
|------|------|------|
| Memory 命中 | < 1ms | 纯内存操作 |
| Redis 命中 | < 5ms | 网络 + 序列化 |
| SpEL 求值 | < 50μs | expr 引擎 |

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

**开发流程:**
```bash
# 1. Fork 项目
# 2. 创建特性分支
git checkout -b feature/AmazingFeature

# 3. 提交更改
git commit -m 'Add some AmazingFeature'

# 4. 推送到分支
git push origin feature/AmazingFeature

# 5. 创建 Pull Request
```

---

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

---

## 🙏 致谢

感谢所有贡献者和用户！

---

**Made with ❤️ by Go-Cache Team**

| | |
|---|---|
| **版本** | v1.0 |
| **最后更新** | 2026-03-14 |
| **状态** | ✅ 生产可用 |
| **GitHub** | [coderiser/go-cache](https://github.com/coderiser/go-cache) |
