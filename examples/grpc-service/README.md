# gRPC Service Example - Go-Cache Framework

本示例展示如何在 gRPC 微服务中使用 Go-Cache 框架的注解式缓存功能。

## 项目结构

```
grpc-service/
├── main.go              # 服务入口
├── go.mod               # Go 模块定义
├── proto/
│   ├── user.proto       # Protobuf 定义
│   ├── user.pb.go       # 生成的 Protobuf Go 代码
│   └── user_grpc.pb.go  # 生成的 gRPC Go 代码
├── server/
│   └── user.go          # gRPC 服务实现
├── service/
│   └── user.go          # 业务服务（带缓存注解）
├── model/
│   └── user.go          # 数据模型
├── .cache-gen/
│   └── auto_register.go # 自动生成的注解注册（由 go-cache-gen 生成）
└── README.md            # 使用说明
```

## 功能特性

- ✅ **注解式缓存**：使用 `@cacheable`、`@cacheput`、`@cacheevict` 注解
- ✅ **Memory 后端**：高性能内存缓存
- ✅ **缓存统计**：命中率、缓存大小等指标
- ✅ **gRPC 服务**：完整的 CRUD 操作
- ✅ **优雅关闭**：自动打印缓存统计

## 缓存注解说明

### @cacheable
用于查询操作，先查缓存，未命中则执行方法并缓存结果。

```go
// GetUser 获取用户信息
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id int32) (*model.User, error)
```

### @cacheput
用于创建/更新操作，执行方法后将结果缓存。

```go
// CreateUser 创建用户
// @cacheput(cache="users", key="#result.ID", ttl="30m")
func (s *UserService) CreateUser(name, email string) (*model.User, error)

// UpdateUser 更新用户信息
// @cacheput(cache="users", key="#user.ID", ttl="30m")
func (s *UserService) UpdateUser(id int32, name, email string) (*model.User, error)
```

### @cacheevict
用于删除操作，清除缓存中的对应条目。

```go
// DeleteUser 删除用户
// @cacheevict(cache="users", key="#id", before=false)
func (s *UserService) DeleteUser(id int32) error
```

## 快速开始

### 1. 安装依赖

```bash
cd examples/grpc-service
go mod tidy
```

### 2. 生成 Protobuf 代码

```bash
# 安装 protoc 插件（如果未安装）
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 生成 Go 代码
protoc --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative proto/user.proto
```

### 3. 启动服务

```bash
go run main.go
```

预期输出：
```
[INFO] Cache initialized with memory backend
[INFO] UserService initialized with sample data
[INFO] Cache decorators initialized for UserService
[INFO] UserService registered with gRPC server
[INFO] gRPC server starting on :50051
```

### 4. 测试服务

#### 使用 grpcurl 测试

```bash
# 获取用户（缓存未命中）
grpcurl -plaintext -d '{"id": 1}' localhost:50051 user.UserService/GetUser

# 再次获取（缓存命中）
grpcurl -plaintext -d '{"id": 1}' localhost:50051 user.UserService/GetUser

# 创建用户
grpcurl -plaintext -d '{"name": "David", "email": "david@example.com"}' localhost:50051 user.UserService/CreateUser

# 更新用户
grpcurl -plaintext -d '{"id": 1, "name": "Alice Updated", "email": "alice.updated@example.com"}' localhost:50051 user.UserService/UpdateUser

# 删除用户
grpcurl -plaintext -d '{"id": 1}' localhost:50051 user.UserService/DeleteUser
```

#### 使用 Go 客户端测试

创建 `client/main.go`：

```go
package main

import (
    "context"
    "log"
    "time"

    pb "github.com/coderiser/go-cache/examples/grpc-service/proto"
    "google.golang.org/grpc"
)

func main() {
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewUserServiceClient(conn)
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()

    // 获取用户
    resp, err := client.GetUser(ctx, &pb.GetUserRequest{Id: 1})
    if err != nil {
        log.Fatalf("GetUser failed: %v", err)
    }
    log.Printf("User: %+v", resp)
}
```

## 缓存统计

服务关闭时（Ctrl+C）会自动打印缓存统计：

```
=== Cache Statistics ===
Total Requests: 10
Cache Hits: 7
Cache Misses: 3
Hit Rate: 70.00%
Cache Size: 3 / 1000
========================
```

## 预期日志输出

```
[INFO] gRPC server starting on :50051
[gRPC] GetUser called with id: 1
[INFO] Cache miss: users:1
[INFO] Querying database...
[gRPC] GetUser success: &{ID:1 Name:Alice Email:alice@example.com}

[gRPC] GetUser called with id: 1
[INFO] Cache hit: users:1
[gRPC] GetUser success: &{ID:1 Name:Alice Email:alice@example.com}
```

## 测试命令

```bash
# 运行服务
go run main.go

# 在另一个终端测试
# 安装 grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# 测试获取用户
grpcurl -plaintext -d '{"id": 1}' localhost:50051 user.UserService/GetUser
```

## 代码生成（可选）

实际项目中，可以使用 `go-cache-gen` 工具自动生成注解注册代码：

```bash
# 安装代码生成工具
go install github.com/coderiser/go-cache/cmd/go-cache-gen@latest

# 生成注解注册代码
go generate ./...

# 或手动运行
go-cache-gen -input ./service -output ./.cache-gen
```

## 常见问题

### Q: 如何调整缓存大小？
A: 修改 `main.go` 中的 `MaxSize` 配置。

### Q: 如何更改缓存 TTL？
A: 修改注解中的 `ttl` 参数，如 `ttl="1h"`。

### Q: 如何集成 Redis？
A: 修改 `main.go`，使用 `backend.NewRedisBackend()` 创建 Redis 后端。

### Q: 注解不生效？
A: 确保：
1. 服务方法有正确的注解注释
2. 使用 `proxy.NewProxyFactory()` 创建代理
3. 调用 `RegisterAnnotation()` 注册注解
4. 通过代理调用方法（而不是直接调用）

## 许可证

MIT License
