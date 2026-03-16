# Hybrid Cache Example

Hybrid 缓存示例 - L1 + L2 两级缓存架构

## 架构说明

```
┌─────────────┐
│  Application│
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│   L1: Memory    │ ← 快速访问（< 1ms）
│   (本地缓存)     │
└──────┬──────────┘
       │ Miss
       ▼
┌─────────────────┐
│   L2: Redis     │ ← 分布式共享（< 5ms）
│   (远程缓存)     │
└──────┬──────────┘
       │ Miss
       ▼
┌─────────────────┐
│    Database     │
└─────────────────┘
```

## 特性

- ✅ **L1 本地缓存**: 内存缓存，快速访问
- ✅ **L2 远程缓存**: Redis 缓存，分布式共享
- ✅ **自动回写**: L1 miss 后从 L2 回写数据
- ✅ **优雅降级**: Redis 不可用时自动降级为纯内存缓存
- ✅ **统计监控**: L1/L2 命中率、回写次数等指标

## 运行示例

### 前提条件

```bash
# 可选：启动 Redis（如果 Redis 不可用，示例会自动降级为内存缓存）
docker run -d -p 6379:6379 redis:latest
```

### 运行

```bash
cd examples/hybrid-cache
go run main.go
```

### 预期输出

```
=== Hybrid Cache Example ===

1️⃣  Setting cache...
   ✅ Set success

2️⃣  Getting cache (L1 hit)...
   ✅ Get success: map[id:1 name:iPhone 15 Pro price:7999]

3️⃣  Getting cache (L1 miss, L2 fallback)...
   ✅ Get success from L2: map[id:1 name:iPhone 15 Pro price:7999]

4️⃣  Cache Statistics:
   L1 Hits: 1
   L1 Misses: 1
   L2 Hits: 1
   L2 Misses: 0
   L2 Fallbacks: 1 (L1 miss → L2 hit)
   L1 Backfills: 1 (L2 → L1 backfill)
   Hit Rate: 66.67%

=== Example Complete ===
```

## 配置说明

### Hybrid 缓存配置

```go
hybridConfig := backend.DefaultHybridConfig()

// L1 配置（本地内存缓存）
hybridConfig.L1Config.Name = "products"
hybridConfig.L1Config.MaxSize = 1000           // 最大容量
hybridConfig.L1Config.DefaultTTL = 5 * time.Minute
hybridConfig.L1Config.MaxTTL = 30 * time.Minute
hybridConfig.L1Config.EvictionPolicy = "lru"   // LRU 淘汰策略

// L2 配置（Redis 缓存）
hybridConfig.L2Config.Addr = "localhost:6379"
hybridConfig.L2Config.DefaultTTL = 30 * time.Minute
hybridConfig.L2Config.MaxTTL = 2 * time.Hour

// L1 回写 TTL（从 L2 读取后写入 L1 的 TTL）
hybridConfig.L1WriteBackTTL = 5 * time.Minute
```

### 降级使用纯内存缓存

如果 Redis 不可用，示例会自动降级：

```go
config := backend.DefaultCacheConfig("products")
config.MaxSize = 1000
config.DefaultTTL = 30 * time.Minute

memoryBackend, _ := backend.NewMemoryBackend(config)
```

## 使用场景

### 1. 热点数据缓存

```go
// 商品详情（高频访问）
hybridConfig.L1Config.Name = "products"
hybridConfig.L1Config.DefaultTTL = 5 * time.Minute  // L1 短 TTL
hybridConfig.L2Config.DefaultTTL = 30 * time.Minute // L2 长 TTL
```

### 2. 用户会话缓存

```go
// 用户会话（需要分布式共享）
hybridConfig.L1Config.Name = "sessions"
hybridConfig.L1Config.DefaultTTL = 15 * time.Minute
hybridConfig.L2Config.DefaultTTL = 2 * time.Hour
```

### 3. 配置数据缓存

```go
// 配置数据（低频更新）
hybridConfig.L1Config.Name = "configs"
hybridConfig.L1Config.DefaultTTL = 10 * time.Minute
hybridConfig.L2Config.DefaultTTL = 24 * time.Hour
```

## 性能指标

| 场景 | 延迟 | 说明 |
|------|------|------|
| L1 命中 | < 1ms | 纯内存操作 |
| L1 Miss + L2 命中 | < 5ms | 网络 + 序列化 |
| L1 Miss + L2 Miss | > 10ms | 数据库查询 |
| L1 回写 | < 1ms | 异步操作 |

## 监控指标

通过 `hybridBackend.Stats()` 获取：

- `L1Hits`: L1 命中次数
- `L1Misses`: L1 未命中次数
- `L2Hits`: L2 命中次数
- `L2Misses`: L2 未命中次数
- `L2Fallbacks`: L1 miss 后 L2 命中的次数
- `L1Backfills`: 从 L2 回写 L1 的次数
- `HitRate`: 综合命中率

## 故障排查

### Redis 连接失败

```
❌ Failed to create hybrid backend: failed to connect to Redis
💡 Note: This example requires Redis running on localhost:6379
```

**解决**: 
1. 启动 Redis: `docker run -d -p 6379:6379 redis:latest`
2. 或检查 Redis 地址配置

### L1 命中率低

**可能原因**:
- L1 TTL 过短
- L1 容量过小导致频繁淘汰
- 数据访问模式分散

**优化**:
- 增加 L1 TTL
- 增加 L1 MaxSize
- 调整 L1WriteBackTTL

## 下一步

- 查看 [Redis Cluster 示例](../redis-cluster/) 了解分布式缓存
- 查看 [PubSub 缓存失效](../pubsub-invalidation/) 了解缓存一致性
- 查看 [用户指南](../../docs/user-guide.md) 了解更多配置选项
