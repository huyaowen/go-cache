# Gin Web API 示例 - Go-Cache Framework

本示例展示如何在 Gin Web 应用中使用 Go-Cache Framework 实现注解式缓存。

## 项目结构

```
examples/gin-web/
├── main.go              # 应用入口
├── go.mod               # Go 模块定义
├── handler/
│   └── user.go          # HTTP 处理器层
├── service/
│   └── user.go          # 业务服务层（带缓存注解）
├── model/
│   └── user.go          # 数据模型
└── README.md            # 使用说明
```

## 快速开始

### 1. 安装依赖

```bash
cd examples/gin-web
go mod tidy
```

### 2. 运行应用

```bash
go run main.go
```

### 3. 测试 API

#### 创建用户

```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'
```

**响应:**
```json
{
  "id": 12345,
  "name": "Alice",
  "email": "alice@example.com",
  "created_at": "2024-01-15T10:30:00Z"
}
```

#### 获取用户（首次 - 缓存未命中）

```bash
curl http://localhost:8080/api/users/12345
```

**控制台输出:**
```
[INFO] Cache miss: users:12345
[INFO] Querying database for user 12345...
[INFO] Found user: Alice
[INFO] Cache set: users:12345 (TTL: 30m)
```

#### 获取用户（第二次 - 缓存命中）

```bash
curl http://localhost:8080/api/users/12345
```

**控制台输出:**
```
[INFO] Cache hit: users:12345
[INFO] Returning cached user: Alice
```

#### 更新用户

```bash
curl -X PUT http://localhost:8080/api/users/12345 \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice Updated","email":"alice.new@example.com"}'
```

**控制台输出:**
```
[INFO] Updating user 12345: Alice -> Alice Updated
[INFO] Cache put: users:12345 (TTL: 30m)
```

#### 删除用户

```bash
curl -X DELETE http://localhost:8080/api/users/12345
```

**控制台输出:**
```
[INFO] Deleting user 12345...
[INFO] Cache evict: users:12345
```

## API 端点

| 方法   | 路径              | 描述     |
|--------|-------------------|----------|
| GET    | /api/users/:id    | 获取用户 |
| POST   | /api/users        | 创建用户 |
| PUT    | /api/users/:id    | 更新用户 |
| DELETE | /api/users/:id    | 删除用户 |
| GET    | /health           | 健康检查 |

## 缓存注解说明

### @cacheable

用于查询操作，自动处理缓存读取：

```go
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id int64) (*model.User, error)
```

- **cache**: 缓存命名空间
- **key**: SpEL 表达式，`#id` 表示方法参数 id
- **ttl**: 缓存过期时间

### @cacheput

用于创建/更新操作，自动更新缓存：

```go
// @cacheput(cache="users", key="#user.ID", ttl="30m")
func (s *UserService) CreateUser(user *model.User) (*model.User, error)
```

### @cacheevict

用于删除操作，自动清除缓存：

```go
// @cacheevict(cache="users", key="#id")
func (s *UserService) DeleteUser(id int64) error
```

## 缓存配置

- **后端**: Memory（内存缓存）
- **TTL**: 30 分钟
- **缓存键格式**: `users:{id}`

## 预期日志输出

### 缓存未命中
```
[INFO] Cache miss: users:1
[INFO] Querying database for user 1...
[INFO] Found user: Alice
[INFO] Cache set: users:1 (TTL: 30m)
```

### 缓存命中
```
[INFO] Cache hit: users:1
[INFO] Returning cached user: Alice
```

### 缓存更新
```
[INFO] Updating user 1: Alice -> Alice Updated
[INFO] Cache put: users:1 (TTL: 30m)
```

### 缓存清除
```
[INFO] Deleting user 1...
[INFO] Cache evict: users:1
```

## 完整测试脚本

```bash
#!/bin/bash

BASE_URL="http://localhost:8080"

echo "=== 1. 创建用户 ==="
curl -X POST "$BASE_URL/api/users" \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'
echo -e "\n"

echo "=== 2. 获取用户（缓存未命中）==="
curl "$BASE_URL/api/users/1"
echo -e "\n"

echo "=== 3. 获取用户（缓存命中）==="
curl "$BASE_URL/api/users/1"
echo -e "\n"

echo "=== 4. 更新用户 ==="
curl -X PUT "$BASE_URL/api/users/1" \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice Updated","email":"alice.new@example.com"}'
echo -e "\n"

echo "=== 5. 删除用户 ==="
curl -X DELETE "$BASE_URL/api/users/1"
echo -e "\n"

echo "=== 6. 健康检查 ==="
curl "$BASE_URL/health"
echo
```

## 注意事项

1. 本示例使用内存存储模拟数据库，重启后数据会丢失
2. 缓存注解需要配合 `go generate` 生成元数据（生产环境）
3. 生产环境建议使用 Redis 后端替代 Memory 后端

## 扩展阅读

- [Go-Cache Framework 核心文档](../../README.md)
- [SpEL 表达式语法](../../docs/spel.md)
- [Redis 后端配置](../redis-example/)
