# Go-Cache Framework: Cron Job Example

本示例展示如何在后台定时任务中使用 Go-Cache 框架，包括**缓存预热**、**定时刷新**等真实场景。

## 📁 项目结构

```
examples/cron-job/
├── main.go              # 应用入口
├── go.mod               # Go 模块定义
├── job/
│   ├── cache_warmup.go  # 缓存预热任务
│   └── cache_refresh.go # 缓存刷新任务
├── service/
│   └── product.go       # 商品服务（带缓存注解）
├── model/
│   └── product.go       # 数据模型
└── README.md            # 使用说明
```

## 🚀 快速开始

### 1. 安装依赖

```bash
cd examples/cron-job
go mod tidy
```

### 2. 运行示例

```bash
go run main.go
```

### 3. 自定义配置

```bash
# 修改刷新间隔为 1 分钟
go run main.go -interval=1m

# 禁用启动时预热
go run main.go -warmup=false

# 同时使用多个参数
go run main.go -interval=2m -warmup=true
```

## 📋 功能特性

### ✅ 缓存预热（Cache Warmup）

应用启动时自动预热热点数据，避免缓存击穿：

- **热点商品列表**: 预热前 5 个热门商品
- **指定商品**: 预热 ID 为 1-5 的商品
- **并发预热**: 支持配置并发数，提高预热效率
- **预热统计**: 输出预热成功/失败/跳过数量

**示例输出**:
```
[INFO] Starting cache warmup...
[INFO] Cache warmup completed in 125ms
[INFO] Warmed up 6/6 entries (Success: 6, Failed: 0, Skipped: 0)
```

### ✅ 定时刷新（Cache Refresh）

后台定时任务定期刷新缓存，确保数据新鲜度：

- **可配置间隔**: 默认 5 分钟，可通过 `-interval` 参数调整
- **自动刷新**: 热点商品、配置缓存
- **统计输出**: 每 30 分钟输出一次刷新统计
- **优雅关闭**: 支持 SIGINT/SIGTERM 信号处理

**示例输出**:
```
[INFO] Scheduled cache refresh every 5m0s
[INFO] Running cache refresh #1...
[INFO] Cache refresh completed in 15ms
```

### ✅ 缓存注解

商品服务使用框架的缓存注解：

```go
// @cacheable(cache="products", key="#id", ttl="1h")
func (s *ProductService) GetProduct(id int64) (*model.Product, error)

// @cacheable(cache="hot_products", key="list", ttl="5m")
func (s *ProductService) GetHotProducts() ([]*model.Product, error)

// @cacheput(cache="products", key="#id", ttl="1h")
func (s *ProductService) UpdatePrice(id int64, price float64) (*model.Product, error)
```

### ✅ 缓存统计

每 10 秒输出一次缓存命中率统计：

```
[STATS] products: Hits=15, Misses=3, Sets=6, HitRate=83.3%
[STATS] hot_products: Hits=42, Misses=1, Sets=7, HitRate=97.7%
[STATS] config: Hits=8, Misses=1, Sets=6, HitRate=88.9%
```

## 🔧 配置说明

### 命令行参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-interval` | duration | `5m` | 缓存刷新间隔 |
| `-warmup` | bool | `true` | 是否启用启动时预热 |
| `-config` | string | `""` | 配置文件路径（预留） |

### 预热配置

在 `job/cache_warmup.go` 中可配置：

```go
type WarmupConfig struct {
    Concurrency  int           // 并发数（默认 4）
    Timeout      time.Duration // 单个条目超时（默认 10 秒）
    SkipExisting bool          // 跳过已存在的缓存（默认 false）
    ProductIDs   []int64       // 预热商品 ID 列表
}
```

### 刷新配置

在 `job/cache_refresh.go` 中可配置：

```go
type RefreshConfig struct {
    Interval          time.Duration // 刷新间隔（默认 5 分钟）
    WarmupHotProducts bool          // 是否预热热点商品
    RefreshConfig     bool          // 是否刷新配置缓存
    LogStats          bool          // 是否记录统计
    StatsInterval     int64         // 统计输出间隔（每 N 次）
}
```

## 📊 运行示例

### 完整运行日志

```bash
$ go run main.go -interval=1m

=== Go-Cache Framework: Cron Job Example ===

[INFO] Initialized 10 products
[INFO] Starting cache warmup...
[INFO] Cache warmup completed in 125ms
[INFO] Warmed up 6/6 entries (Success: 6, Failed: 0, Skipped: 0)

[INFO] Scheduled cache refresh every 1m0s
[INFO] Application running. Press Ctrl+C to exit...

[STATS] products: Hits=12, Misses=0, Sets=6, HitRate=100.0%
[STATS] hot_products: Hits=24, Misses=0, Sets=1, HitRate=100.0%

[INFO] Running cache refresh #1...
[INFO] Refreshed 5 hot products
[INFO] Refreshed config cache
[INFO] Cache refresh completed in 15ms

[STATS] products: Hits=25, Misses=0, Sets=6, HitRate=100.0%
[STATS] hot_products: Hits=48, Misses=0, Sets=2, HitRate=100.0%

^C[INFO] Shutting down...

=== Final Cache Stats ===
[STATS] products: Hits=50, Misses=0, Sets=6, HitRate=100.0%
[STATS] hot_products: Hits=96, Misses=0, Sets=3, HitRate=100.0%

=== Refresh Job Stats ===
  Total Runs:   3
  Success:      3
  Failed:       0
  Total Time:   45ms

=== Application Exited ===
```

## 🏗️ 代码结构

### 商品服务 (`service/product.go`)

- **Product 模型**: ID, Name, Price, Stock, UpdatedAt
- **GetProduct(id)**: 获取单个商品（带 `@cacheable` 注解）
- **GetHotProducts()**: 获取热点商品（带 `@cacheable` 注解）
- **UpdatePrice(id, price)**: 更新价格（带 `@cacheput` 注解）

### 缓存预热 (`job/cache_warmup.go`)

- **WarmupCache()**: 执行缓存预热
- **WarmupCacheSimple()**: 简单预热（使用默认配置）
- **WarmupCacheWithBuilder()**: 使用构建器模式预热

### 缓存刷新 (`job/cache_refresh.go`)

- **Start()**: 启动定时刷新任务
- **Stop()**: 停止刷新任务
- **RunWithGracefulShutdown()**: 支持优雅关闭的运行方式
- **GetStats()**: 获取刷新统计

## 🎯 使用场景

### 1. 电商系统

```go
// 预热热门商品
warmer.Add("products", "hot:list", loadHotProducts, 5*time.Minute)

// 定时刷新库存
refreshJob.Start(ctx, &RefreshConfig{
    Interval: 1 * time.Minute, // 每分钟刷新库存
})
```

### 2. 内容管理系统

```go
// 预热热门文章
warmer.Add("articles", "featured", loadFeaturedArticles, 10*time.Minute)

// 定时刷新配置
refreshJob.Start(ctx, &RefreshConfig{
    Interval:      15 * time.Minute,
    RefreshConfig: true,
})
```

### 3. 用户系统

```go
// 预热 VIP 用户
warmer.Add("users", "vip:list", loadVIPUsers, 30*time.Minute)

// 定时刷新用户会话
refreshJob.Start(ctx, &RefreshConfig{
    Interval: 5 * time.Minute,
})
```

## 📝 最佳实践

### 1. 预热策略

- **启动时预热**: 核心业务数据
- **低峰期预热**: 全量数据预热
- **分层预热**: 先预热热点，再预热长尾

### 2. 刷新策略

- **高频数据**: 1-5 分钟刷新
- **中频数据**: 10-30 分钟刷新
- **低频数据**: 1 小时以上刷新

### 3. 监控告警

- 监控缓存命中率
- 监控刷新任务执行时间
- 设置失败告警阈值

## 🔍 调试技巧

### 1. 查看缓存内容

```go
cache, _ := manager.GetCache("products")
value, found, _ := cache.Get(ctx, "1")
fmt.Printf("Product: %v, Found: %v\n", value, found)
```

### 2. 查看统计信息

```go
stats := cache.Stats()
fmt.Printf("Hit Rate: %.2f%%\n", stats.HitRate*100)
```

### 3. 手动触发刷新

```go
refreshJob.refreshOnce(ctx, DefaultRefreshConfig())
```

## ⚠️ 注意事项

1. **预热并发**: 生产环境建议设置合理的并发数，避免数据库压力过大
2. **刷新间隔**: 根据业务容忍度设置，不要过于频繁
3. **优雅关闭**: 确保刷新任务完成后再退出应用
4. **错误处理**: 预热/刷新失败不应影响主业务流程

## 📚 相关文档

- [Go-Cache Framework 主文档](../../README.md)
- [缓存注解使用指南](../../docs/annotations.md)
- [缓存保护机制](../../docs/protection.md)

## 📄 License

MIT License
