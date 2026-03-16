# PubSub Cache Invalidation Example

多实例缓存一致性解决方案 - 基于 Redis PubSub

## 架构说明

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Instance 1  │     │  Instance 2  │     │  Instance 3  │
│  ┌────────┐  │     │  ┌────────┐  │     │  ┌────────┐  │
│  │ Cache  │  │     │  │ Cache  │  │     │  │ Cache  │  │
│  └───┬────┘  │     │  └───┬────┘  │     │  └───┬────┘  │
└──────┼───────┘     └──────┼───────┘     └──────┼───────┘
       │                    │                    │
       └────────────────────┼────────────────────┘
                            │
                            ▼
                 ┌───────────────────┐
                 │  Redis PubSub     │
                 │  (Channel)        │
                 └───────────────────┘
                            │
       ┌────────────────────┼────────────────────┐
       │                    │                    │
       ▼                    ▼                    ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ Invalidate   │     │ Invalidate   │     │ Invalidate   │
│ Cache Key    │     │ Cache Key    │     │ Cache Key    │
└──────────────┘     └──────────────┘     └──────────────┘
```

## 特性

- ✅ **多实例缓存一致性**: 一个实例更新，所有实例失效
- ✅ **实时广播**: 基于 Redis PubSub，毫秒级传播
- ✅ **自动重连**: 连接断开自动恢复
- ✅ **避免循环**: 实例 ID 过滤，避免处理自己的消息
- ✅ **优雅关闭**: 资源清理和订阅取消

## 快速开始

### 1. 启动 Redis

```bash
docker run -d -p 6379:6379 redis:latest
```

### 2. 运行示例

```bash
cd examples/pubsub-invalidation
go run main.go
```

### 预期输出

```
=== PubSub Cache Invalidation Example ===

1️⃣  Instance 1: Updating data...
   ✅ Database updated: map[id:1 name:John Doe Updated email:john.updated@example.com]
   ✅ Cache updated
   ✅ Invalidation message broadcasted

2️⃣  Instance 2: Received invalidation message
   ✅ Local cache cleared for key: user:1

3️⃣  Instance 3: Reloading from database...
   ✅ Cache reloaded: map[id:1 name:John Doe Updated email:john.updated@example.com]

=== Example Complete ===

💡 Key Points:
   • PubSub 实现多实例缓存一致性
   • 更新时广播失效消息
   • 接收方清除本地缓存并重新加载
```

## 配置说明

### 基础配置

```go
config := backend.DefaultPubSubInvalidatorConfig()
config.Channel = "cache-invalidation"      // PubSub 频道名
config.RedisAddr = "localhost:6379"        // Redis 地址
config.InstanceID = "instance-1"           // 实例 ID（避免处理自己的消息）
```

### 生产环境配置

```go
config := backend.DefaultPubSubInvalidatorConfig()

// Redis 配置
config.RedisAddr = "redis-master:6379"
config.Password = "your-password"
config.DB = 0

// PubSub 配置
config.Channel = "cache-invalidation-prod"
config.InstanceID = "instance-1"  // 每个实例唯一 ID

// 连接配置
config.PoolSize = 20
config.DialTimeout = 5 * time.Second
config.ReadTimeout = 3 * time.Second

// 重试配置
config.MaxRetries = 3
config.MinRetryBackoff = 100 * time.Millisecond
config.MaxRetryBackoff = 1 * time.Second
```

## 使用场景

### 1. 多实例 Web 应用

```go
// 3 个 Web 实例共享缓存
instances := []string{"web-1", "web-2", "web-3"}

for _, id := range instances {
    invalidator, _ := createPubSubInvalidator(id)
    defer invalidator.Close()
}
```

### 2. 微服务架构

```go
// 不同服务监听不同的频道
userServiceChannel := "user-cache-invalidation"
orderServiceChannel := "order-cache-invalidation"
productServiceChannel := "product-cache-invalidation"
```

### 3. 区域部署

```go
// 不同区域使用不同的频道
config.Channel = "cache-invalidation-cn"    // 中国区
config.Channel = "cache-invalidation-us"    // 美国区
config.Channel = "cache-invalidation-eu"    // 欧洲区
```

## 完整示例

### 更新数据并广播失效

```go
func UpdateUser(ctx context.Context, userID int64, data *User) error {
    // 1. 更新数据库
    err := db.UpdateUser(userID, data)
    if err != nil {
        return err
    }

    // 2. 更新本地缓存
    cache.Set(ctx, fmt.Sprintf("user:%d", userID), data, 30*time.Minute)

    // 3. 广播失效消息（通知其他实例）
    invalidator.Invalidate(ctx, fmt.Sprintf("user:%d", userID))

    return nil
}
```

### 接收失效消息并清除缓存

```go
func StartInvalidationListener(ctx context.Context, invalidator *backend.PubSubInvalidator) {
    invalidator.OnInvalidate(func(key string) {
        fmt.Printf("Received invalidation for key: %s\n", key)
        
        // 清除本地缓存
        cache.Delete(ctx, key)
        
        // 可选：从数据库重新加载
        // reloadData(key)
    })
}
```

## 最佳实践

### 1. 实例 ID 命名

```go
// ✅ 推荐：使用唯一标识
config.InstanceID = fmt.Sprintf("%s-%s-%d", service, hostname, pid)

// ❌ 避免：硬编码
config.InstanceID = "instance-1"
```

### 2. 频道命名

```go
// ✅ 推荐：包含环境和业务信息
config.Channel = fmt.Sprintf("cache-invalidation-%s-%s", env, service)

// ❌ 避免：过于简单
config.Channel = "cache"
```

### 3. 错误处理

```go
invalidator, err := backend.NewPubSubInvalidator(config)
if err != nil {
    log.Printf("Failed to create invalidator: %v", err)
    // 降级：不使用 PubSub，仅本地缓存
    return
}
defer invalidator.Close()
```

## 监控指标

通过 `invalidator.Stats()` 获取：

- `MessagesSent`: 发送的失效消息数
- `MessagesReceived`: 接收的失效消息数
- `MessagesProcessed`: 处理的失效消息数
- `Errors`: 错误次数
- `Reconnects`: 重连次数

## 故障排查

### 连接失败

```
❌ Failed to create PubSub invalidator: dial tcp [::1]:6379: connect: connection refused
```

**解决**: 启动 Redis
```bash
docker run -d -p 6379:6379 redis:latest
```

### 消息未收到

**可能原因**:
- 频道名称不匹配
- 实例 ID 过滤了自己的消息
- 网络连接问题

**检查**:
```bash
# 手动测试 PubSub
redis-cli SUBSCRIBE cache-invalidation
```

## 性能考虑

| 指标 | 值 | 说明 |
|------|-----|------|
| 消息延迟 | < 10ms | 内网传播 |
| 吞吐量 | ~50,000 msg/s | 单频道 |
| 连接开销 | ~100KB | 每实例 |

## 下一步

- 查看 [Hybrid 缓存示例](../hybrid-cache/) 了解 L1+L2 架构
- 查看 [Redis Cluster 示例](../redis-cluster/) 了解分布式缓存
- 查看 [用户指南](../../docs/user-guide.md) 了解更多配置选项
