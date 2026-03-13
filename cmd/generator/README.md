# Go-Cache Generator

代码生成器 for Go-Cache Framework - 自动生成缓存代理包装器和注册表代码

## 概述

本工具通过解析 Go 源代码中的 `@cacheable` 注解注释，自动生成缓存代理包装器和服务注册表代码。基于 `go/ast` 和 `text/template` 实现，零外部依赖（仅 Go 标准库）。

## 特性

- ✅ **零依赖** - 仅使用 Go 标准库
- ✅ **类型安全** - 基于 AST 的完整类型信息
- ✅ **高性能** - 标准库优化，编译速度快
- ✅ **易扩展** - 支持自定义模板
- ✅ **简洁设计** - 清晰的代码结构

## 安装

```bash
# 从源码安装
cd cmd/generator
go build -o generator .
```

## 快速开始

### 1. 定义服务

```go
// service/user_service.go
package service

type UserService struct{}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id string) (*User, error) {
    // 业务逻辑
}
```

### 2. 运行生成器

```bash
# 生成代码
generator --output ./generated ./service/*.go
```

### 3. 使用生成的代码

生成的代码包含：
- `wrapper.go` - 服务包装器
- `registry.go` - 服务注册表

```go
import "your/module/generated"

func main() {
    registry := generated.NewRegistry()
    registry.RegisterService(&UserService{})
    
    // 使用装饰后的服务
    userService := registry.UserServiceDecorator
}
```

## 命令行选项

```
Usage: generator [options] <source-files...>

Options:
  -output string
        Output directory for generated code (default "./generated")
  -verbose
        Enable verbose logging

Example:
  generator --output ./pkg/cache/wrapper ./services/*.go
```

## 注解格式

### @cacheable
缓存方法返回值

```go
// @cacheable(cache="users", key="#id", ttl="30m")
func GetUser(id string) (*User, error)
```

**参数**:
- `cache` - 缓存名称（必填）
- `key` - 缓存键，支持 SpEL 表达式（必填）
- `ttl` - 过期时间，如 "30m", "1h"（可选）

## 项目结构

```
cmd/generator/
├── main.go              # CLI 入口
├── go.mod               # 模块定义
├── README.md            # 本文档
│
├── parser/
│   └── parser.go        # AST 解析和注解提取
│
├── extractor/
│   └── extractor.go     # 类型定义和注解解析
│
├── generator/
│   ├── generator.go     # 代码生成器
│   └── templates.go     # 模板定义
│
├── proxy/
│   └── proxy.go         # 代理服务（生成的代码依赖）
│
└── testdata/
    └── user_service.go  # 测试示例
```

## 架构设计

### 核心流程

```
源文件 → parser.ParseFile() → AST → 
  parser.ExtractAnnotations() → 
  parser.ExtractServices() → 
  generator.GenerateWrapper() → 生成文件
```

### 技术栈

- `go/ast` - AST 解析和遍历
- `go/parser` - 源码解析
- `go/token` - token 管理
- `text/template` - 模板引擎

## 开发指南

### 构建

```bash
cd cmd/generator
go build -o generator .
```

### 测试

```bash
# 使用测试数据生成代码
./generator --verbose --output ./output ./testdata/user_service.go

# 验证生成的代码可编译
cd output
go build ./...
```

### 添加新的注解类型

1. 在 `extractor/extractor.go` 中添加新的注解类型定义
2. 在 `extractor.ExtractAnnotation()` 中添加解析逻辑
3. 在 `generator/templates.go` 中添加对应的模板

## 示例输出

生成的 `wrapper.go`:

```go
package cache

import "generator/proxy"

type UserServiceDecorator struct {
    decorated *proxy.DecoratedService[*UserService]
}

func NewUserServiceDecorator(decorated *proxy.DecoratedService[*UserService]) *UserServiceDecorator {
    return &UserServiceDecorator{decorated: decorated}
}

func (d *UserServiceDecorator) GetUser(id string) (*User, error) {
    results, err := d.decorated.Invoke("GetUser", id)
    if err != nil {
        var zero0 *User
        return zero0, err
    }
    result0, _ := results[0].(*User)
    result1, _ := results[1].(error)
    return result0, result1
}
```

## 参考框架

- [mockery](https://github.com/vektra/mockery) - Mock 代码生成
- [stringer](https://pkg.go.dev/golang.org/x/tools/cmd/stringer) - String() 方法生成
- [protoc-gen-go](https://github.com/protocolbuffers/protobuf-go) - Protocol Buffers 代码生成

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

---

**Go-Cache Generator** - 让缓存代码生成更简单
