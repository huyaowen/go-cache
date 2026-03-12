---
title: "Building a Spring Cache-Style Annotation-Based Cache Framework in Go"
published: true
tags: ['go', 'caching', 'opensource', 'backend', 'architecture']
series: 'Go-Cache Framework'
---

# Building a Spring Cache-Style Annotation-Based Cache Framework in Go

> Zero intrusion into business code, one annotation to handle caching

**Project**: https://github.com/coderiser/go-cache

---

## Background

In the Java ecosystem, Spring Cache is the de facto standard for caching. Just add `@Cacheable` annotation to a method, and caching logic is automatically handled with zero intrusion to business code.

However, in Go, implementing similar functionality has been challenging due to the lack of annotation system and runtime AOP mechanisms. Existing Go caching libraries mostly require manual cache logic or intrusive business code changes.

As a developer pursuing极致 development experience, I decided to tackle this challenge — **building a truly elegant annotation-based cache framework in Go**.

---

## Goals

The framework needed to satisfy:

1. **Zero Intrusion** - No cache logic in business code
2. **Annotation-Driven** - Declare caching behavior with simple annotations
3. **Automatic Proxy** - Runtime method interception
4. **Multi-Backend** - Pluggable Memory / Redis support
5. **High Performance** - Cache hit latency < 1ms

---

## Technical Challenges

### Challenge 1: Go Has No Annotations

Java's `@Cacheable` is a language-level annotation, but Go doesn't have this feature.

**Solution**: **Comments + AST Parsing**

```go
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id string) (*User, error) {
    // Business logic
}
```

Using `go/packages` and `go/ast` to scan comments and extract cache configuration at compile time.

---

### Challenge 2: Go Has No Runtime AOP

Spring can dynamically proxy methods at runtime, but Go doesn't support this mechanism.

**Solution**: **Reflection + Dynamic Proxy**

```go
type Proxy interface {
    Call(methodName string, args []reflect.Value) []reflect.Value
}

func (p *proxyImpl) Call(methodName string, args []reflect.Value) []reflect.Value {
    // 1. Check cache
    // 2. Return if hit
    // 3. Call original method if miss
    // 4. Write to cache
}
```

---

### Challenge 3: Minimizing Integration Cost

The initial design required users to manually call `cache.Decorate()`, which wasn't elegant enough.

**Final Solution**: **Interface Pattern + DecorateAndReturn**

```go
// 1. Define interface
type UserServiceInterface interface {
    GetUser(id string) (*User, error)
}

// 2. Implement interface with annotation
type UserService struct {
    db *gorm.DB
}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id string) (*User, error) {
    var u User
    err := s.db.Where("id = ?", id).First(&u).Error
    return &u, err
}

// 3. init() creates proxy
var UserService UserServiceInterface

func init() {
    manager := core.NewCacheManager()
    autoDecorate := proxy.GetAutoDecorate(manager)
    decorated, err := autoDecorate.DecorateAndReturn(&UserService{})
    if err != nil {
        UserService = &UserService{}
        return
    }
    UserService = decorated.(UserServiceInterface)
}

// Usage (completely transparent)
user, _ := UserService.GetUser("123")  // Automatic caching!
```

---

## Core Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Go-Cache Framework                      │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │  Annotation  │  │   Code Gen   │  │   Runtime Core   │  │
│  │   Parser     │  │  (go generate)│  │                  │  │
│  │  Scan Ann    │  │  Gen Meta    │  │  Proxy + SpEL    │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Cache Manager (Core Coordinator)          │  │
│  └─────────────────────────┬────────────────────────────┘  │
│                            │                                │
│  ┌─────────────────────────┴────────────────────────────┐  │
│  │              Cache Backend (Pluggable)                 │  │
│  │     Memory Backend  │  Redis Backend  │  Custom      │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## Core Features

### 1. @Cacheable - Cache Read

```go
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id string) (*User, error) {
    var u User
    s.db.First(&u, id)
    return &u, nil
}
```

- First call executes original method and writes to cache
- Subsequent calls return cached result directly
- SpEL expression support for dynamic key generation

---

### 2. @CachePut - Force Update

```go
// @cacheput(cache="users", key="#user.Id")
func (s *UserService) UpdateUser(user *User) error {
    return s.db.Save(user).Error
}
```

- Always executes original method
- Forces cache update after method execution

---

### 3. @CacheEvict - Delete Cache

```go
// @cacheevict(cache="users", key="#userId", before=true)
func (s *UserService) DeleteUser(userId string) error {
    return s.db.Delete(&User{}, userId).Error
}
```

- Supports `before=true` to delete before method execution
- Supports `allEntries=true` to clear entire cache

---

## SpEL Expressions

The framework integrates the `expr` engine with powerful expression syntax:

```go
// Reference parameters
@cacheable(cache="orders", key="#userId + '_' + #status")

// Reference return value (condition filter)
@cacheable(cache="data", key="#id", unless="#result == nil")

// Complex expressions
@cacheable(cache="products", key="category:#catId:page:#page")

// Ternary expressions
@cacheable(cache="config", key="#env == 'prod' ? 'p_' + #id : 't_' + #id")
```

---

## Cache Protection

### Cache Penetration (Querying Non-Existent Data)

```go
// Automatic nil caching
if result == nil {
    cache.Set(key, nilMarker, 5*time.Minute)
}
```

### Cache Breakdown (Hot Key Expiration)

```go
// Singleflight pattern
group.Do(key, func() (interface{}, error) {
    // Only one request hits the database
})
```

### Cache Avalanche (Mass Key Expiration)

```go
// TTL random jitter
ttl := baseTTL + time.Duration(rand.Int63n(jitter))
```

---

## Performance

| Scenario | Latency |
|----------|---------|
| Memory Hit | < 1ms |
| Redis Hit | < 5ms |
| SpEL Evaluation | < 50μs |
| Code Generation | < 1s (100 methods) |

Test Coverage: **83%+**

---

## Quick Start

### Installation

```bash
go get github.com/coderiser/go-cache
```

### 1. Define Service Interface

```go
package service

// Define interface
type UserServiceInterface interface {
    GetUser(id string) (*User, error)
}

// Implement interface
type UserService struct {
    db *gorm.DB
}

// Add cache annotation
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id string) (*User, error) {
    var u User
    err := s.db.Where("id = ?", id).First(&u).Error
    return &u, err
}
```

### 2. Initialize (using DecorateAndReturn)

```go
package service

import (
    "github.com/coderiser/go-cache/pkg/core"
    "github.com/coderiser/go-cache/pkg/proxy"
)

var UserService UserServiceInterface

func init() {
    manager := core.NewCacheManager()
    autoDecorate := proxy.GetAutoDecorate(manager)
    decorated, err := autoDecorate.DecorateAndReturn(&UserService{})
    if err != nil {
        UserService = &UserService{}
        return
    }
    UserService = decorated.(UserServiceInterface)
}
```

### 3. Generate Metadata

```bash
# Install code generator
go install github.com/coderiser/go-cache/cmd/generator@latest

# Generate annotation metadata
go-cache-gen ./...
```

### 4. Usage (Transparent)

```go
// Call through interface, caching applied automatically
user, err := UserService.GetUser("123")  // Automatic caching!
```

---

## Tech Stack

| Component | Technology | Reason |
|-----------|------------|--------|
| SpEL Engine | `expr` | Pure Go, excellent performance |
| Redis Client | `go-redis/v9` | Active community, feature-complete |
| AST Parsing | `go/packages` | Official standard library |
| Dynamic Proxy | `reflect` | Runtime reflection |

---

## Project Links

**GitHub**: https://github.com/coderiser/go-cache

**Documentation**: 
- [Architecture Design](docs/ARCHITECTURE.md)
- [Interface Specification](docs/INTERFACE_SPEC.md)
- [Integration Guide](docs/INTEGRATION_GUIDE.md)

---

## Summary

Implementing an annotation-based cache framework in Go is challenging, but through reasonable technology selection and architecture design, we successfully achieved:

✅ Zero intrusion into business code  
✅ Annotation-driven caching logic  
✅ Runtime automatic proxy  
✅ Multi-backend support  
✅ High performance  

While Go doesn't have Java-style annotations and AOP, we found the optimal solution in Go — **one line of init() + go generate**, maintaining Go's simplicity while achieving a development experience close to Spring Cache.

---

*Welcome to Star, Fork, and submit Issues!*

---

**(End)**
