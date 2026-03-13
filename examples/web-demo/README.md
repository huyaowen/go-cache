# Go-Cache Framework Web Demo

简单的 Web 演示应用，展示 Go-Cache Framework 的缓存功能。

## 快速启动

```bash
cd examples/web-demo
go run .
# 或
go build -o web-demo . && ./web-demo
```

访问：http://localhost:8081

## API 端点

| 端点 | 说明 |
|------|------|
| `GET /` | Web 界面 |
| `GET /api/user/:id` | 获取用户（带缓存） |
| `GET /api/stats` | 缓存统计 |
| `GET /api/benchmark/:id` | 性能测试（10 次请求） |

## 测试示例

```bash
# 第一次请求（cache miss）
curl http://localhost:8081/api/user/1

# 第二次请求（cache hit）
curl http://localhost:8081/api/user/1

# 查看统计
curl http://localhost:8081/api/stats

# 基准测试
curl http://localhost:8081/api/benchmark/1
```

## 预期输出

```
=== Test 1: Get User 1 ===
{"success":true,"data":{"id":1,"name":"Alice","email":"alice@example.com"}}

=== Test 2: Get User 1 again ===
{"success":true,"data":{"id":1,"name":"Alice","email":"alice@example.com"}}

=== Stats ===
{"success":true,"data":{"hit_rate":"50.00%","hits":1,"misses":1,"sets":1}}
```

## 缓存日志

服务器日志会显示缓存命中/未命中：
```
🔍 CACHE MISS: user:1
📦 CACHE HIT: user:1
```
