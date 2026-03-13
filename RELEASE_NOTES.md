# Go-Cache Framework v1.0.0 - 方案 G 正式发布

## 🎉 重大更新

**方案 G (Beego 融合版)** 正式发布！实现"注解后直接使用"的零配置体验。

---

## ✨ 核心特性

### 零配置启动
```go
// main.go
import cached "your-module/service/.cache-gen"

func main() {
    // ✅ 一行搞定！缓存自动生效
    svc := cached.NewProductService()
    product, err := svc.GetProduct(1)
}
```

### 注解式缓存
```go
// @cacheable(cache="products", key="#id", ttl="1h")
func GetProduct(id int64) (*model.Product, error) { ... }

// @cacheput(cache="products", key="#id", ttl="1h")
func UpdatePrice(id int64, price float64) (*model.Product, error) { ... }

// @cacheevict(cache="products", key="#id")
func DeleteProduct(id int64) error { ... }
```

### 全局管理器懒加载
- `GetGlobalManager()` - 自动创建单例
- `SetGlobalManager()` - 可选自定义配置
- `CloseGlobalManager()` - 优雅关闭

---

## 🚀 快速开始

### 方式 1: 构建脚本
```bash
./build.sh
```

### 方式 2: Makefile
```bash
make build
```

### 方式 3: 手动
```bash
go generate ./...
go build .
```

---

## 📦 安装

```bash
go get github.com/coderiser/go-cache@latest
```

---

## 📝 文档

- [快速开始](QUICKSTART.md)
- [用户指南](docs/user-guide.md)
- [API 参考](docs/api-reference.md)
- [迁移指南](docs/migration-guide.md)
- [开发指南](CONTRIBUTING.md)

---

## 🔧 技术栈

- **表达式引擎**: [expr](https://github.com/antonmedv/expr)
- **后端支持**: Memory / Redis / Hybrid
- **指标导出**: Prometheus
- **链路追踪**: OpenTelemetry

---

## 📊 测试报告

```
✅ 单元测试：全部通过
✅ 集成测试：全部通过 (Redis 跳过)
✅ 示例程序：验证通过
```

---

## 🎯 性能对比

| 场景 | 旧方案 | 方案 G |
|------|--------|--------|
| 初始化代码 | 3 行 | 1 行 |
| 配置复杂度 | 需传 Manager | 零配置 |
| 学习成本 | 理解装饰器 | 直接使用 |

---

## ⚠️ BREAKING CHANGES

### 生成代码包名变更

**旧:** `package cache`  
**新:** `package cached`

**迁移:**
```go
// 旧代码
import "your-module/service/.cache-gen"
svc := cache.NewProductService()

// 新代码
import cached "your-module/service/.cache-gen"
svc := cached.NewProductService()
```

---

## 🐛 Bug Fixes

- 修复包名冲突问题
- 修复生成代码导入路径
- 修复注解解析逻辑

---

## 📚 完整 Changelog

### New
- `pkg/cache/global.go` - 全局 Manager 懒加载
- `pkg/cache/registry.go` - 全局注解注册表
- `pkg/cache/interceptor.go` - 全局拦截器
- `build.sh` - 构建脚本
- `Makefile` - Make 配置
- `docs/user-guide.md` - 完整用户指南
- `docs/migration-guide.md` - 迁移指南
- `docs/api-reference.md` - API 参考

### Changed
- `cmd/generator/main.go` - 支持方案 G 代码生成
- `README.md` - 更新快速开始

### Removed
- 删除过时文档 (ARCHITECTURE.md, INTEGRATION_GUIDE.md 等)
- 删除示例 README (减少冗余)

---

## 🙏 致谢

感谢所有贡献者和用户！

---

**Full Changelog:** https://github.com/coderiser/go-cache/compare/v0.9.0...v1.0.0
