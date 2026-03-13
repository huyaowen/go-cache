# Go-Cache Framework 开发指南

## 🚀 快速开始

### 构建项目

**方式 1: 使用构建脚本 (推荐)**
```bash
./build.sh
```

**方式 2: 使用 Makefile**
```bash
make build
```

**方式 3: 手动执行**
```bash
go generate ./...
go build .
```

### 验证构建

```bash
# 运行示例
cd examples/cron-job
./cron-job --help
```

### 运行测试

```bash
# 使用 Makefile
make test

# 或使用 go test
go test ./... -v
```

### 清理构建

```bash
# 清理生成的代码和测试报告
make clean
```

---

## 开发工作流

### 1. 修改代码后

```bash
# 重新生成代码并构建
make build

# 或完整流程 (生成 + 构建 + 测试)
make all
```

### 2. 添加新注解

```bash
# 1. 在 service 文件中添加注解
// @cacheable(cache="users", key="#id", ttl="30m")
func GetUser(id int64) (*User, error) { ... }

# 2. 重新生成代码
go generate ./...

# 3. 构建并测试
make build
```

### 3. IDE 集成

#### VS Code

创建 `.vscode/settings.json`:
```json
{
    "go.generateOnSave": true,
    "go.buildOnSave": true,
    "go.testOnSave": false
}
```

#### GoLand

1. File → Settings → Go → Go Modules
2. 启用 "Run 'go generate' before build"

---

## 常用命令

| 命令 | 说明 |
|------|------|
| `./build.sh` | 快速构建 |
| `./build.sh --test` | 构建 + 测试 |
| `make build` | 构建项目 |
| `make test` | 运行测试 |
| `make clean` | 清理 |
| `make generate` | 仅代码生成 |
| `make lint` | 代码检查 |
| `make fmt` | 格式化代码 |
| `make all` | 完整构建 |

---

## 目录结构

```
go-cache-framework/
├── build.sh              # 构建脚本
├── Makefile              # Make 配置
├── cmd/
│   └── generator/        # 代码生成器
├── pkg/
│   ├── cache/           # 缓存核心
│   ├── core/            # 核心接口
│   ├── backend/         # 后端存储
│   └── proxy/           # 代理拦截
├── examples/            # 示例代码
└── docs/                # 文档
```

---

## 故障排查

### 问题 1: 编译失败 "no required module"

**解决:**
```bash
go mod tidy
```

### 问题 2: 生成代码找不到

**检查:**
```bash
# 确认已执行 go generate
go generate ./...

# 检查生成目录
find . -name ".cache-gen" -type d
```

### 问题 3: 测试失败

**调试:**
```bash
# 运行详细测试
go test ./... -v

# 运行特定测试
go test ./pkg/cache -run TestGetGlobalManager -v
```

---

## 性能优化

### 基准测试

```bash
# 运行基准测试
go test ./pkg/cache -bench=. -benchmem

# 生成性能报告
go test ./pkg/cache -bench=. -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

### 代码优化建议

1. **避免频繁调用 `go generate`** - 仅在注解变更时执行
2. **使用增量构建** - `go build` 会自动缓存
3. **并行测试** - `go test ./... -parallel 4`

---

## 贡献指南

### 提交代码前

```bash
# 1. 格式化代码
make fmt

# 2. 运行检查
make lint

# 3. 运行测试
make test

# 4. 完整构建
make all
```

### 代码规范

- 遵循 Go 官方代码规范
- 添加单元测试
- 更新文档

---

**最后更新:** 2026-03-14  
**状态:** ✅ 生产可用
