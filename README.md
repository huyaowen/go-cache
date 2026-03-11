# Go-Cache Framework

[![Go Reference](https://pkg.go.dev/badge/github.com/your-org/go-cache.svg)](https://pkg.go.dev/github.com/your-org/go-cache)
[![Go Report Card](https://goreportcard.com/badge/github.com/your-org/go-cache)](https://goreportcard.com/report/github.com/your-org/go-cache)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> **Go 语言注解式缓存框架** —— 类似 Spring Cache 的优雅缓存解决方案

## 🚀 特性

- ✨ **注解式缓存** - 通过简单注解实现缓存逻辑，业务代码零侵入
- 🔧 **自动装饰** - 运行时自动代理，无需手动包装
- 🎯 **SpEL 表达式** - 强大的动态缓存 Key 生成（基于 `expr` 引擎）
- 💾 **多后端支持** - Memory / Redis 可插拔切换
- 📊 **缓存统计** - 内置命中率、延迟等指标
- 🛡️ **异常保护** - 穿透/击穿/雪崩保护机制
- 📝 **代码生成** - `go-cache-gen` CLI 自动生成元数据

## 📦 快速开始

### 1. 安装

```bash
go get github.com/your-org/go-cache
```

### 2. 定义服务

```go
package service

import "github.com/your-org/go-cache"

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

### 3. 生成元数据

```bash
go-cache-gen ./...
```

### 4. 使用（完全透明）

```go
user, err := UserService.GetUser("123")  // 自动缓存！
```

## 📖 核心注解

| 注解 | 说明 | 示例 |
|------|------|------|
| `@cacheable` | 缓存读取 | `@cacheable(cache="users", key="#id", ttl="30m")` |
| `@cacheput` | 强制更新缓存 | `@cacheput(cache="users", key="#user.Id")` |
| `@cacheevict` | 删除缓存 | `@cacheevict(cache="users", key="#id", before=true)` |

## 🎯 SpEL 表达式

```go
// 引用参数
@cacheable(cache="orders", key="#userId + '_' + #status")

// 引用返回值（unless）
@cacheable(cache="data", key="#id", unless="#result == nil")

// 复杂表达式
@cacheable(cache="products", key="category:#catId:page:#page")
```

## 📚 文档

- [架构设计](docs/ARCHITECTURE.md)
- [接口定义](docs/INTERFACE_SPEC.md)
- [集成指南](docs/INTEGRATION_GUIDE.md)

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

| 场景 | 延迟 |
|------|------|
| Memory 命中 | < 1ms |
| Redis 命中 | < 5ms |
| SpEL 求值 | < 50μs |

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

---

**Made with ❤️ by Go-Cache Team**
