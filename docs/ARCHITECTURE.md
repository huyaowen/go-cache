# Go-Cache Framework - 架构设计文档

## 1. 概述

Go-Cache Framework 是一个为 Go 语言设计的注解式缓存框架，通过编译时代码生成和运行时反射装饰，实现零侵入的缓存集成体验。

## 2. 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        用户应用层                                │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   UserService                            │    │
│  │  ┌─────────────────────────────────────────────────────┐│    │
│  │  │  // @cacheable(cache="users", key="#id", ttl="30m") ││    │
│  │  │  func GetUser(id string) (*User, error)             ││    │
│  │  └─────────────────────────────────────────────────────┘│    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      编译时处理层                                │
│  ┌──────────────────┐         ┌─────────────────────────────┐   │
│  │   go generate    │ ──────► │    go-cache-gen             │   │
│  │   //go:generate  │         │  (代码生成器)                │   │
│  └──────────────────┘         └─────────────┬───────────────┘   │
│                                              │                   │
│                                              ▼                   │
│                                    ┌─────────────────┐          │
│                                    │  cache_meta.go  │          │
│                                    │  (生成的元数据)  │          │
│                                    └─────────────────┘          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      运行时处理层                                │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   init() 初始化                          │    │
│  │              cache.AutoDecorate(&UserService)            │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                  CacheManager                            │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │    │
│  │  │ SpEL 解析器 │  │ 缓存拦截器  │  │  元数据管理器   │  │    │
│  │  └─────────────┘  └─────────────┘  └─────────────────┘  │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      缓存后端层                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   CacheBackend                           │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │    │
│  │  │  Memory  │ │  Redis   │ │   Memcached │  Custom  │   │    │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘   │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

## 3. 数据流

```
用户调用 GetUser("123")
        │
        ▼
┌───────────────────┐
│  装饰后的方法      │ ──► 检查缓存元数据
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│  SpEL 表达式解析   │ ──► key="#id" → "users:123"
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│   CacheBackend    │
│      Get(key)     │ ──┬──► Hit: 直接返回
└─────────┬─────────┘   │
          │             │
          │ Miss        │
          ▼             │
┌───────────────────┐   │
│  执行原始方法      │   │
│   db.FindUser()   │   │
└─────────┬─────────┘   │
          │             │
          ▼             │
┌───────────────────┐   │
│   CacheBackend    │   │
│    Set(key,val)   │ ◄─┘
└─────────┬─────────┘
          │
          ▼
     返回结果
```

## 4. 核心设计原则

### 4.1 零侵入 (Zero Intrusion)
- 用户代码无需继承特定基类
- 无需手动调用缓存 API
- 注解即配置，无额外样板代码

### 4.2 编译时优化 (Compile-Time Optimization)
- 注解元数据在编译时生成
- 运行时仅做轻量级反射查找
- 避免运行时解析开销

### 4.3 后端可插拔 (Pluggable Backend)
- 统一的 CacheBackend 接口
- 支持内存、Redis、Memcached 等
- 用户可自定义后端实现

### 4.4 表达式驱动 (Expression-Driven)
- 使用 SpEL 风格的表达式定义 key
- 支持方法参数引用 `#param`
- 支持简单的表达式计算

## 5. 技术选型说明

| 组件 | 选型 | 理由 |
|------|------|------|
| 注解解析 | 编译时代码生成 | Go 无原生注解，`go generate` 是最佳实践 |
| 表达式引擎 | 自研轻量 SpEL | 避免重型依赖，仅需参数引用功能 |
| 反射装饰 | 运行时方法替换 | 利用 Go 反射实现透明拦截 |
| 并发控制 | sync.RWMutex | 读写分离，高性能并发访问 |
| 序列化 | encoding/gob + json | 内置支持 + 灵活扩展 |

## 6. 与 Spring Cache 对比

| 特性 | Spring Cache (Java) | Go-Cache Framework |
|------|---------------------|---------------------|
| 注解语法 | `@Cacheable` | `// @cacheable` |
| 表达式语言 | SpEL | 轻量 SpEL 子集 |
| 配置方式 | XML/JavaConfig | 注解 + init() |
| 编译时检查 | 否 | 是 (go generate) |
| 运行时开销 | 中等 (AOP 代理) | 低 (直接方法调用) |
| 后端扩展 | CacheManager SPI | CacheBackend 接口 |
| 泛型支持 | 完善 | 通过 interface{} + 类型断言 |
| 错误处理 | 可配置异常策略 | 统一 error 返回 |

### 6.1 代码对比

**Spring Cache:**
```java
@Cacheable(cacheNames = "users", key = "#id", unless = "#result == null")
public User getUser(String id) {
    return userRepository.findById(id);
}
```

**Go-Cache Framework:**
```go
// @cacheable(cache="users", key="#id", unless="#result == nil")
func (s *UserService) GetUser(id string) (*User, error) {
    return db.FindUser(id)
}
```

## 7. 模块划分

```
go-cache/
├── core/
│   ├── cache_manager.go      # 缓存管理器
│   ├── cache_backend.go      # 后端接口定义
│   └── method_meta.go        # 方法元数据
├── annotation/
│   ├── parser.go             # 注解解析器
│   └── generator/            # 代码生成器
│       └── main.go           # go-cache-gen 入口
├── expression/
│   ├── spel.go               # SpEL 表达式引擎
│   └── evaluator.go          # 表达式求值器
├── backend/
│   ├── memory/               # 内存后端
│   ├── redis/                # Redis 后端
│   └── memcached/            # Memcached 后端
└── decorator/
    └── auto_decorate.go      # 运行时装饰器
```

## 8. 缓存异常保护架构

### 8.1 三层防护体系

```
┌─────────────────────────────────────────────────────────────────┐
│                    缓存异常保护层                                │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                  CacheProtection                         │    │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐    │    │
│  │  │  穿透保护    │ │  击穿保护    │ │  雪崩保护    │    │    │
│  │  │  (Nil Marker)│ │ (Singleflight)│ │  (TTL Jitter)│    │    │
│  │  └──────────────┘ └──────────────┘ └──────────────┘    │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      缓存后端层                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   CacheBackend                           │    │
│  │  ┌──────────┐ ┌──────────┐                              │    │
│  │  │  Memory  │ │  Redis   │                              │    │
│  │  └──────────┘ └──────────┘                              │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### 8.2 穿透保护流程

```
请求：GetUser("non-existent-id")
        │
        ▼
┌───────────────────┐
│   缓存未命中       │
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│  查询数据库        │ ──► 返回 nil / NotFound
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│  包装为空值标记    │ ──► "__GO_CACHE_NIL__"
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│  写入缓存 (TTL 5m)  │
└─────────┬─────────┘
          │
          ▼
  后续请求直接返回 nil
  （5 分钟内不再查询数据库）
```

### 8.3 击穿保护流程（Singleflight）

```
100 个并发请求：GetHotData("hot-key")
        │
        ├──────┬──────┬──────┬──────┐
        ▼      ▼      ▼      ▼      ▼
    ┌─────────────────────────────────┐
    │      Singleflight Group         │
    │  ┌───────────────────────────┐  │
    │  │   只执行一次数据库查询     │  │
    │  └─────────────┬─────────────┘  │
    └────────────────┼────────────────┘
                     │
        ┌────────────┼────────────┐
        ▼            ▼            ▼
    结果 1       结果 2       结果 100
    (共享)       (共享)       (共享)
```

### 8.4 雪崩保护原理

```
无保护情况：
Key1: TTL=30m, 过期时间 10:00:00
Key2: TTL=30m, 过期时间 10:00:00
Key3: TTL=30m, 过期时间 10:00:00
...
10:00:00 → 所有缓存同时过期 → 数据库压力激增 ❌

启用雪崩保护（10% 抖动）：
Key1: TTL=28m15s, 过期时间 09:58:15
Key2: TTL=31m42s, 过期时间 10:01:42
Key3: TTL=29m03s, 过期时间 09:59:03
...
过期时间分散在 27-33 分钟之间 → 数据库压力平滑 ✅
```

### 8.5 保护机制配置

```go
type ProtectionConfig struct {
    // 穿透保护
    EnablePenetrationProtection bool
    EmptyValueTTL               time.Duration
    
    // 击穿保护
    EnableBreakdownProtection   bool
    
    // 雪崩保护
    EnableAvalancheProtection   bool
    TTLJitterFactor             float64  // 0.0-0.5
}
```

### 8.6 核心组件

| 组件 | 位置 | 功能 |
|------|------|------|
| `CacheProtection` | `pkg/core/protection.go` | 保护机制协调器 |
| `nilMarker` | `pkg/core/protection.go` | 空值标记常量 |
| `singleflight.Group` | `golang.org/x/sync` | 请求合并 |
| `WrapForStorage` | `pkg/core/protection.go` | 值包装（穿透保护） |
| `ApplyAvalancheProtection` | `pkg/core/protection.go` | TTL 抖动计算 |
| `ProtectedGet` | `pkg/core/protection.go` | 完整保护获取流程 |

## 9. Redis 后端架构

### 9.1 Redis 后端组件

```
┌─────────────────────────────────────────────────────────────────┐
│                    RedisBackend                                  │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐    │
│  │ Redis Client│ │  连接池     │ │   序列化/反序列化        │    │
│  │ (go-redis)  │ │ (PoolSize)  │ │   (JSON/Gob)            │    │
│  └─────────────┘ └─────────────┘ └─────────────────────────┘    │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐    │
│  │  Key 前缀   │ │  TTL 管理    │ │   统计计数器            │    │
│  │  (Prefix)   │ │  (Jitter)   │ │   (Hits/Misses)         │    │
│  └─────────────┘ └─────────────┘ └─────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### 9.2 Redis Key 结构

```
格式：{Prefix}:{CacheName}:{Key}

示例：
- myapp:users:123
- myapp:products:sku_456
- myapp:sessions:abc123
```

### 9.3 连接池管理

```go
type RedisConfig struct {
    PoolSize     int  // 最大连接数
    MinIdleConns int  // 最小空闲连接
    MaxRetries   int  // 失败重试次数
    
    DialTimeout  time.Duration  // 连接超时
    ReadTimeout  time.Duration  // 读取超时
    WriteTimeout time.Duration  // 写入超时
}
```

## 10. 版本规划

- **v0.1.0**: 核心接口 + 内存后端 + 基础注解支持 ✅
- **v0.2.0**: Redis 后端 + 完整 SpEL 支持 ✅
- **v0.3.0**: 缓存穿透/击穿/雪崩防护 ✅
- **v1.0.0**: 生产就绪，完整文档和测试

## 11. 缓存异常保护机制

### 11.1 保护机制架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                    缓存异常保护层                                │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                  CacheProtection                         │    │
│  │  ┌─────────────────┐ ┌─────────────────┐ ┌───────────┐  │    │
│  │  │ 穿透保护        │ │ 击穿保护        │ │ 雪崩保护  │  │    │
│  │  │ (Penetration)   │ │ (Breakdown)     │ │(Avalanche)│  │    │
│  │  │                 │ │                 │ │           │  │    │
│  │  │ • 空值缓存      │ │ • Singleflight  │ │ • TTL 抖动 │  │    │
│  │  │ • NilMarker     │ │ • 并发请求合并   │ │ • ±10%    │  │    │
│  │  │ • 5min TTL      │ │ • 减少 DB 压力    │ │           │  │    │
│  │  └─────────────────┘ └─────────────────┘ └───────────┘  │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### 11.2 ProtectedGet 流程

```
1. 尝试从缓存获取 ──► 命中？ ──┬── 是 ──► 检查 NilMarker
                               │
                               └── 否 ──► 执行 cacheMissFn
                                           │
                                           ▼
2. 应用击穿保护 ──► Singleflight.Do(key, fn)
                     │
                     ├──► 第一个请求：执行 fn
                     │       │
                     │       ▼
                     │   写入缓存 ──► 应用雪崩保护 (TTL + Jitter)
                     │
                     └──► 并发请求：等待结果（共享）
```

### 11.3 三大保护机制

| 机制 | 问题 | 解决方案 | 关键实现 |
|------|------|---------|---------|
| **穿透保护** | 查询不存在的数据，缓存无法命中 | 空值缓存（NilMarker） | `WrapNilMarker()`, `UnwrapNilMarker()` |
| **击穿保护** | 热点 key 失效，并发请求直达数据库 | Singleflight 合并请求 | `singleflight.Group.Do()` |
| **雪崩保护** | 大量 key 同时到期，集体失效 | TTL 随机抖动 | `ApplyAvalancheProtection()` |

### 11.4 配置示例

```go
config := &ProtectionConfig{
    EnablePenetrationProtection: true,  // 穿透保护
    EmptyValueTTL:               5 * time.Minute,
    EnableBreakdownProtection:   true,  // 击穿保护
    EnableAvalancheProtection:   true,  // 雪崩保护
    TTLJitterFactor:             0.1,   // ±10% 抖动
}

manager := core.NewCacheManager()
manager.SetProtectionConfig(config)
```

### 11.5 性能影响

| 保护机制 | 额外开销 |
|---------|---------|
| 穿透保护 | < 1μs |
| 击穿保护 | ~10μs (Singleflight) |
| 雪崩保护 | < 1μs (随机数生成) |

---

## 12. P2 新增组件

P2 阶段引入了多个新组件，增强了框架的可观测性和易用性。

### 12.1 代码生成器 (go-cache-gen)

**位置**: `cmd/generator/main.go`

**职责**:
- 扫描 Go 源代码中的缓存注解
- 生成类型安全的注册代码
- 输出到 `.cache-gen/auto_register.go`

**工作流程**:
```
源代码 (.go 文件)
    │
    ▼
AST 解析 (go/parser)
    │
    ▼
注解提取 (正则匹配)
    │
    ▼
元数据收集 (map[typeName][methodName]Annotation)
    │
    ▼
代码生成 (Go 代码模板)
    │
    ▼
输出文件 (.cache-gen/auto_register.go)
```

**生成的代码结构**:
```go
package registry

import "github.com/coderiser/go-cache/pkg/proxy"

func init() {
    proxy.RegisterAnnotation(nil, "UserService", "GetUser", &proxy.CacheAnnotation{
        Type:      "cacheable",
        CacheName: "users",
        Key:       "#id",
        TTL:       "30m",
    })
}
```

### 12.2 简化装饰 API (SimpleDecorate)

**位置**: `pkg/proxy/auto.go`

**设计目标**: 一行代码完成缓存代理创建

**API 层次**:
```
SimpleDecorate(service)                          // 最简模式
    │
    ├──► SimpleDecorateWithManager(service, manager)  // 自定义管理器
    │
    ├──► SimpleDecorateWithError(service)              // 带错误处理
    │
    └──► SimpleDecorateWithManagerAndError(service, manager)  // 完整模式
```

**内部流程**:
```
1. 创建 CacheManager (默认或传入)
    │
    ▼
2. 创建 ProxyFactory
    │
    ▼
3. 创建代理对象 (Proxy)
    │
    ▼
4. 获取类型名 (reflect.Type)
    │
    ▼
5. 查找已注册注解 (GetRegisteredAnnotations)
    │
    ▼
6. 注册到拦截器 (RegisterAnnotation)
    │
    ▼
7. 返回代理对象
```

### 12.3 混合后端 (HybridBackend)

**位置**: `pkg/backend/hybrid.go`

**架构**:
```
                    HybridBackend
                    /           \
                   /             \
                  ▼               ▼
            L1: Memory      L2: Redis
            (快速)          (持久)
            
读取流程:
1. 查询 L1 ──► 命中？── 是 ──► 返回
               │
               └── 否 ──► 查询 L2 ──► 命中？── 是 ──► 回写 L1 ──► 返回
                                        │
                                        └── 否 ──► 返回 miss
                                        
写入流程:
1. 写入 L1
2. 异步写入 L2
```

**配置示例**:
```go
hybrid := backend.NewHybridBackend(
    "users",
    memoryBackend,  // L1
    redisBackend,   // L2
    &backend.HybridConfig{
        AsyncL2Write: true,      // 异步写 L2
        L1OnlyOnMiss: false,     // L2 miss 时也写 L1
    },
)
```

### 12.4 指标系统 (Metrics)

**位置**: `pkg/metrics/prometheus.go`

**指标类型**:
```
Counter:
  - go_cache_hits_total{cache, backend}
  - go_cache_misses_total{cache, backend}
  
Histogram:
  - go_cache_duration_seconds{cache, backend, operation}
  
Gauge:
  - go_cache_size{cache, backend}
```

**集成流程**:
```
CacheManager
    │
    ├──► EnableMetrics()
    │       │
    │       ▼
    │   创建 Collector
    │       │
    │       ▼
    │   注册到 Prometheus
    │
    └──► 每次缓存操作
            │
            ▼
        更新指标
            │
            ▼
        Prometheus 抓取 (/metrics)
```

### 12.5 追踪系统 (Tracing)

**位置**: `pkg/tracing/opentelemetry.go`

**Span 层级**:
```
用户请求 Span
    │
    └──► Cache Operation Span
            │
            ├──► cache.operation: "get" | "set" | "evict"
            ├──► cache.key: "user:123"
            ├──► cache.backend: "memory" | "redis"
            ├──► cache.hit: true | false
            └──► cache.duration: 1.234ms
```

**集成方式**:
```go
import (
    "go.opentelemetry.io/otel"
    "github.com/coderiser/go-cache/pkg/tracing"
)

tracer := otel.Tracer("go-cache")
wrapper := tracing.NewOpenTelemetryWrapper(tracer)
manager.SetTracingWrapper(wrapper)
```

### 12.6 组件交互图

```
┌──────────────────────────────────────────────────────────────┐
│                        用户代码                               │
│  var UserService = proxy.SimpleDecorate(&UserService{})      │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                    go-cache-gen (编译时)                      │
│  扫描注解 ──► 生成 .cache-gen/auto_register.go                │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                    运行时初始化                               │
│  init() 执行 ──► 注册注解 ──► 创建代理                        │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                    方法调用拦截                               │
│  UserService.GetUser() ──► Proxy ──► Interceptor             │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                    缓存处理                                   │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ SpEL 求值   │  │ 保护机制    │  │ HybridBackend       │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
│         │                │                    │              │
│         ▼                ▼                    ▼              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              Metrics & Tracing                       │    │
│  └─────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────┘
```

---

## 13. 最佳实践

### 13.1 初始化顺序

```go
func init() {
    // 1. 创建后端
    backend := createRedisBackend()
    
    // 2. 创建管理器并注册后端
    manager := core.NewCacheManager()
    manager.RegisterCache("users", backend)
    
    // 3. 启用监控（可选）
    manager.EnableMetrics()
    manager.EnableTracing(otel.Tracer("go-cache"))
    
    // 4. 配置保护机制
    manager.SetProtection(core.NewCacheProtection(core.DefaultProtectionConfig()))
    
    // 5. 装饰服务
    UserService = proxy.SimpleDecorateWithManager(&UserService{}, manager).(*UserService)
}
```

### 13.2 代码生成集成

```go
// 在包级别添加 go:generate 指令
//go:generate go-cache-gen ./...

package service

// ... 服务定义
```

然后在构建前运行:
```bash
go generate ./...
go build .
```

### 13.3 监控配置

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'go-cache'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
```

```go
// 暴露 metrics 端点
http.Handle("/metrics", promhttp.Handler())
go http.ListenAndServe(":8080", nil)
```

---

**文档版本**: 2.0 (P2)  
**最后更新**: 2026-03-13
| 击穿保护 | < 10μs |
| 雪崩保护 | < 1μs |

**总结**：保护机制开销极小，生产环境建议全部启用。

---

**实现文件**：`pkg/core/protection.go`  
**测试文件**：`pkg/core/protection_test.go`
