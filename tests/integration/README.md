# Go-Cache Integration Tests

本目录包含 Go-Cache Framework 的集成测试和端到端测试。

## 测试类型

### 1. Redis 后端集成测试 (`redis_integration_test.go`)
测试 Redis 后端的完整功能：
- Set/Get 操作
- Delete 操作
- TTL 过期
- 并发访问
- 统计信息

### 2. 端到端测试 (`e2e_test.go`)
测试完整的缓存管理器流程：
- 缓存管理器初始化
- 多缓存实例
- 内存后端 LRU 驱逐
- 并发场景

## 运行测试

### 前置条件

1. **本地 Redis 实例** (可选，用于 Redis 测试)
   ```bash
   docker run -d -p 6379:6379 --name redis-test redis:7-alpine
   ```

2. **或使用 Docker Compose**
   ```bash
   cd tests/integration
   docker-compose up --build
   ```

### 运行方式

#### 快速测试 (跳过集成测试)
```bash
go test ./... -short
```

#### 完整测试 (需要 Redis)
```bash
# 启动 Redis
docker run -d -p 6379:6379 --name redis-test redis:7-alpine

# 运行所有测试
go test ./... -v

# 清理
docker stop redis-test && docker rm redis-test
```

#### 仅运行集成测试
```bash
go test ./tests/integration/... -v
```

#### 运行基准测试
```bash
go test ./pkg/backend/... -bench=. -benchmem -benchtime=1s
```

## 测试覆盖率

生成覆盖率报告：
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 使用 Docker Compose

完整的测试环境：
```bash
cd tests/integration
docker-compose up --build
```

这会：
1. 启动 Redis 容器
2. 等待 Redis 就绪
3. 运行所有集成测试
4. 显示测试结果

## 故障排查

### Redis 连接失败
```bash
# 检查 Redis 是否运行
docker ps | grep redis

# 测试连接
redis-cli ping

# 查看日志
docker logs redis-test
```

### 测试超时
增加超时时间：
```bash
go test ./tests/integration/... -v -timeout 5m
```

## 性能基准

对比优化前后的性能：
```bash
# 优化前基准
git stash
go test ./pkg/backend -bench=BenchmarkMemoryBackend -benchmem

# 应用优化
git stash pop

# 优化后基准
go test ./pkg/backend -bench=BenchmarkMemoryBackend -benchmem
```

## 持续集成

在 CI 环境中运行测试：
```yaml
# GitHub Actions 示例
- name: Run Integration Tests
  services:
    redis:
      image: redis:7-alpine
      ports:
        - 6379:6379
  run: go test ./tests/integration/... -v
```
