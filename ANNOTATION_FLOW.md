# 注解注册与使用流程说明

**文档目的**: 解释代码生成器生成的注解如何被 SimpleDecorate 使用

---

## 📋 完整流程

### 步骤 1: 代码生成器注册注解

**生成文件**: `auto_register.go`

```go
package service

import (
    gocache "github.com/coderiser/go-cache/pkg/cache"
    "github.com/coderiser/go-cache/pkg/proxy"
)

func init() {
    gocache.RegisterGlobalAnnotation("userService", "GetUser", &proxy.CacheAnnotation{
        Type:      "cacheable",
        CacheName: "users",
        Key:       "#id",
        TTL:       "30m",
    })
}
```

**关键点**: 注解注册到 **cache 包** 的 `globalAnnotations` 映射表

---

### 步骤 2: cache 包注册回调函数

**文件**: `pkg/cache/registry.go`

```go
func init() {
    // 注册回调函数到 proxy 包
    proxy.SetGlobalAnnotationGetter(GetAllAnnotations)
}
```

**作用**: 让 proxy 包能够获取 cache 包中的注解

---

### 步骤 3: SimpleDecorate 获取注解

**文件**: `pkg/proxy/auto.go`

```go
func SimpleDecorateWithManager[T any](service T, manager core.CacheManager) *DecoratedService[T] {
    // ... 创建代理 ...
    
    // 获取类型名
    serviceType := proxyImpl.GetTypeName()
    
    // 获取注册的注解
    annotations := GetRegisteredAnnotations(serviceType)
    
    // 注册到拦截器
    for methodName, annotation := range annotations {
        interceptor.RegisterAnnotation(methodName, annotation)
    }
}
```

**获取逻辑**: `pkg/proxy/decorate.go`

```go
func GetRegisteredAnnotations(typeName string) map[string]*CacheAnnotation {
    // 1. 优先从 cache 包读取（通过回调）
    if globalAnnotationGetter != nil {
        cacheAnnotations := globalAnnotationGetter(typeName)
        if cacheAnnotations != nil && len(cacheAnnotations) > 0 {
            return cacheAnnotations
        }
    }
    
    // 2. 回退到本地注册表（向后兼容）
    // ...
}
```

---

### 步骤 4: 拦截器使用注解

**文件**: `pkg/proxy/interceptor.go`

```go
func (i *methodInterceptor) Intercept(methodName string, args []interface{}) (interface{}, error) {
    // 获取该方法的注解
    annotation := i.GetAnnotation(methodName)
    
    if annotation != nil {
        // 执行缓存逻辑
        switch annotation.Type {
        case "cacheable":
            return i.executeCacheable(methodName, args, annotation)
        case "cacheput":
            return i.executeCachePut(methodName, args, annotation)
        case "cacheevict":
            return i.executeCacheEvict(methodName, args, annotation)
        }
    }
    
    // 无注解，直接调用原始方法
    return i.invokeTarget(methodName, args)
}
```

---

## 🔄 数据流图

```
代码生成器
    ↓
auto_register.go (调用 cache.RegisterGlobalAnnotation)
    ↓
cache.globalAnnotations (注册表)
    ↓
[init() 注册回调]
    ↓
proxy.globalAnnotationGetter (回调函数)
    ↓
SimpleDecorate (调用 proxy.GetRegisteredAnnotations)
    ↓
proxy 拦截器 (注册注解)
    ↓
方法调用时执行缓存逻辑
```

---

## ✅ 验证测试

```bash
# 运行集成测试
go run /tmp/test_integration.go
```

**预期输出**:
```
✅ 成功：注解注册和获取流程正常
   Type: cacheable
   CacheName: test
   Key: #id
   TTL: 5m
```

---

## 🎯 关键修复

### 问题
- cache 包和 proxy 包的注解注册表独立
- 代码生成器注册到 cache 包
- SimpleDecorate 从 proxy 包获取
- **导致非 demo 场景注解无法生效**

### 修复方案
- 在 proxy 包添加 `SetGlobalAnnotationGetter()` 
- cache 包在 `init()` 中注册回调
- proxy 优先从 cache 包读取注解
- 回退到本地注册表保持兼容

### 提交
```
121433f fix: 连接 cache 包和 proxy 包的注解注册表
```

---

## 📝 使用示例

### 用户代码
```go
package service

type UserService struct {
    // ...
}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id int64) (*User, error) {
    // 业务逻辑
}

// init 中自动装饰
var UserServiceInstance *UserService

func init() {
    UserServiceInstance = proxy.SimpleDecorate(&UserService{})
}
```

### 生成的代码 (auto_register.go)
```go
func init() {
    cache.RegisterGlobalAnnotation("userService", "GetUser", &proxy.CacheAnnotation{
        Type:      "cacheable",
        CacheName: "users",
        Key:       "#id",
        TTL:       "30m",
    })
}
```

### 使用
```go
// 直接调用，缓存自动生效
user, err := UserServiceInstance.GetUser(123)
```

---

**文档创建时间**: 2026-03-14  
**状态**: ✅ 流程验证通过
