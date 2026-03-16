# Redis Cluster Cache Example

Redis Cluster 分布式缓存示例

## 架构说明

```
┌─────────────┐
│  Application│
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────┐
│      Redis Cluster (6 nodes)    │
│  ┌─────┐ ┌─────┐ ┌─────┐       │
│  │Master│ │Master│ │Master│      │
│  │ :7000│ │ :7001│ │ :7002│      │
│  └──┬───┘ └──┬───┘ └──┬───┘       │
│     │        │        │            │
│  ┌──┴───┐ ┌──┴───┐ ┌──┴───┐       │
│  │Replica│ │Replica│ │Replica│    │
│  │ :7003│ │ :7004│ │ :7005│      │
│  └─────┘ └─────┘ └─────┘       │
└─────────────────────────────────┘
```

## 特性

- ✅ **分布式缓存**: 数据自动分片到多个节点
- ✅ **高可用**: 主从复制，自动故障转移
- ✅ **水平扩展**: 支持动态添加节点
- ✅ **大规模**: 适合 TB 级数据缓存
- ✅ **自动重连**: 连接池管理和自动重试

## 快速开始

### 1. 使用 Docker Compose 启动 Redis Cluster

```bash
# 创建 docker-compose.yml
version: '3.8'
services:
  redis-node-1:
    image: redis:latest
    ports:
      - "7000:6379"
    command: redis-server --cluster-enabled yes --cluster-config-file nodes.conf --cluster-node-timeout 5000 --appendonly yes

  redis-node-2:
    image: redis:latest
    ports:
      - "7001:6379"
    command: redis-server --cluster-enabled yes --cluster-config-file nodes.conf --cluster-node-timeout 5000 --appendonly yes

  redis-node-3:
    image: redis:latest
    ports:
      - "7002:6379"
    command: redis-server --cluster-enabled yes --cluster-config-file nodes.conf --cluster-node-timeout 5000 --appendonly yes

  redis-node-4:
    image: redis:latest
    ports:
      - "7003:6379"
    command: redis-server --cluster-enabled yes --cluster-config-file nodes.conf --cluster-node-timeout 5000 --appendonly yes

  redis-node-5:
    image: redis:latest
    ports:
      - "7004:6379"
    command: redis-server --cluster-enabled yes --cluster-config-file nodes.conf --cluster-node-timeout 5000 --appendonly yes

  redis-node-6:
    image: redis:latest
    ports:
      - "7005:6379"
    command: redis-server --cluster-enabled yes --cluster-config-file nodes.conf --cluster-node-timeout 5000 --appendonly yes

  redis-cluster-creator:
    image: redis:latest
    depends_on:
      - redis-node-1
      - redis-node-2
      - redis-node-3
      - redis-node-4
      - redis-node-5
      - redis-node-6
    command: >
      redis-cli --cluster create
      redis-node-1:6379 redis-node-2:6379 redis-node-3:6379
      redis-node-4:6379 redis-node-5:6379 redis-node-6:6379
      --cluster-replicas 1 --cluster-yes
```

```bash
# 启动集群
docker-compose up -d
```

### 2. 运行示例

```bash
cd examples/redis-cluster
go run main.go
```

## 配置说明

### 基础配置

```go
config := backend.DefaultRedisClusterConfig()
config.Addrs = []string{
    "localhost:7000",
    "localhost:7001",
    "localhost:7002",
    "localhost:7003",
    "localhost:7004",
    "localhost:7005",
}
config.DefaultTTL = 30 * time.Minute
config.MaxTTL = 2 * time.Hour
```

### 生产环境配置

```go
config := backend.DefaultRedisClusterConfig()

// 集群节点地址（至少 3 个 master）
config.Addrs = []string{
    "redis-master-1:6379",
    "redis-master-2:6379",
    "redis-master-3:6379",
    "redis-replica-1:6379",
    "redis-replica-2:6379",
    "redis-replica-3:6379",
}

// 认证
config.Password = "your-password"

// 连接池配置
config.PoolSize = 50          // 最大连接数
config.MinIdleConns = 10      // 最小空闲连接

// 超时配置
config.DialTimeout = 5 * time.Second
config.ReadTimeout = 3 * time.Second
config.WriteTimeout = 3 * time.Second

// 重试配置
config.MaxRetries = 3
config.MinRetryBackoff = 8 * time.Millisecond
config.MaxRetryBackoff = 512 * time.Millisecond

// TTL 配置
config.DefaultTTL = 30 * time.Minute
config.MaxTTL = 24 * time.Hour
```

## 使用场景

### 1. 大规模用户缓存

```go
// 适用于百万级用户数据
config.Addrs = []string{ /* 6 个节点 */ }
config.PoolSize = 100
config.DefaultTTL = 1 * time.Hour
```

### 2. 会话存储

```go
// 分布式会话存储
config.Addrs = []string{ /* 6 个节点 */ }
config.DefaultTTL = 24 * time.Hour
config.MaxTTL = 7 * 24 * time.Hour // 最长 7 天
```

### 3. 热点数据缓存

```go
// 电商商品缓存
config.Addrs = []string{ /* 6 个节点 */ }
config.DefaultTTL = 15 * time.Minute
config.PoolSize = 80 // 高并发
```

## 性能指标

| 指标 | 值 | 说明 |
|------|-----|------|
| 单节点 QPS | ~100,000 | 取决于硬件 |
| 集群 QPS | ~300,000 | 3 个 master |
| 平均延迟 | < 1ms | 内网访问 |
| 可用性 | 99.99% | 带副本 |

## 监控指标

通过 `clusterBackend.Stats()` 获取：

- `Hits`: 命中次数
- `Misses`: 未命中次数
- `Sets`: 设置次数
- `Deletes`: 删除次数
- `HitRate`: 命中率
- `Size`: 缓存大小

## 故障排查

### 集群连接失败

```
❌ Failed to create Redis Cluster backend: cluster has no slots
```

**解决**: 确保集群已正确初始化
```bash
docker-compose up redis-cluster-creator
```

### 节点不可用

```
❌ Get failed: CLUSTERDOWN The cluster is down
```

**解决**: 检查所有节点状态
```bash
docker-compose ps
```

## 最佳实践

### 1. Key 命名规范

```go
// ✅ 推荐：使用冒号分隔
key := "user:12345:profile"
key := "product:67890:details"

// ❌ 避免：无规律命名
key := "user12345profile"
```

### 2. 批量操作

```go
// ✅ 推荐：批量设置
for i := 0; i < 100; i++ {
    key := fmt.Sprintf("user:%d", i)
    clusterBackend.Set(ctx, key, data, ttl)
}

// ❌ 避免：单次大对象
largeData := make([]byte, 10<<20) // 10MB
clusterBackend.Set(ctx, "large", largeData, ttl)
```

### 3. 连接池管理

```go
// ✅ 推荐：复用 backend 实例
var globalBackend *backend.RedisClusterBackend

func init() {
    globalBackend, _ = backend.NewRedisClusterBackend(config)
}

// ❌ 避免：频繁创建关闭
func handler() {
    b, _ := backend.NewRedisClusterBackend(config)
    defer b.Close()
}
```

## 下一步

- 查看 [Hybrid 缓存示例](../hybrid-cache/) 了解 L1+L2 架构
- 查看 [PubSub 缓存失效](../pubsub-invalidation/) 了解缓存一致性
- 查看 [Redis 官方文档](https://redis.io/topics/cluster-tutorial) 学习集群管理
