# Go-Cache Framework

[![Go Reference](https://pkg.go.dev/badge/github.com/coderiser/go-cache.svg)](https://pkg.go.dev/github.com/coderiser/go-cache)
[![Go Report Card](https://goreportcard.com/badge/github.com/coderiser/go-cache)](https://goreportcard.com/report/github.com/coderiser/go-cache)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> **Go 语言注解式缓存框架** —— 让缓存变得简单优雅

---

## 🚀 特性

- ✨ **注解式缓存** - 一行注解，缓存自动生效
- 🚀 **零配置启动** - `cached.NewXxxService()` 直接使用
- 🎯 **SpEL 表达式** - 动态缓存 Key，灵活强大
- 💾 **多后端支持** - Memory / Redis / Hybrid 无缝切换
- 📊 **内置监控** - Prometheus 指标 + OpenTelemetry 追踪
- 🛡️ **异常保护** - 穿透/击穿/雪崩全保护
- 📝 **代码生成** - `go generate` 自动生成类型安全代码

---

## 📦 快速开始

### 0️⃣ 安装

```bash
go get github.com/coderiser/go-cache@latest
```

### 1️⃣ 添加注解

```go
// service/product.go
//go:generate go run ../../../cmd/generator/main.go .

// @cacheable(cache="products", key="#id", ttl="1h")
func GetProduct(id int64) (*model.Product, error) {
    // 业务逻辑 - 从数据库查询
}

// @cacheput(cache="products", key="#id", ttl="1h")
func UpdatePrice(id int64, price float64) (*model.Product, error) {
    // 更新价格
}
```

### 2️⃣ 生成代码

```bash
go generate ./...
```

### 3️⃣ 直接使用

```go
// main.go
import cached "your-module/service/.cache-gen"

func main() {
    // ✅ 零配置！缓存自动生效
    svc := cached.NewProductService()
    product, _ := svc.GetProduct(1)
}
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

---

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
