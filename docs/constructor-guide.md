# 服务构造函数指南

## 智能字段初始化（v2.1+）

**好消息！** 现在 `gocache` 可以自动初始化 `map` 类型字段，你不再需要提供构造函数（推荐但仍可选）。

### ✅ 自动生成初始化代码

```go
// 无需构造函数
type userService struct {
    users  map[int64]*User  // ✅ 自动生成：users: make(map[int64]*User)
    nextID int64            // ✅ 使用零值
}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *userService) GetUser(id int64) (*User, error) {
    // 可以安全使用 s.users
}
```

生成的代码：
```go
func NewCachedUserService() UserServiceInterface {
    raw := &userService{
        users: make(map[int64]*User),  // ✅ 自动生成
    }
    // ...
}
```

## 为什么需要构造函数？（高级场景）

虽然生成器可以自动初始化 `map` 字段，但在以下场景仍需要构造函数：

- 需要注入依赖（数据库、客户端等）
- 需要设置初始值（如 `nextID: 1`）
- 需要复杂的初始化逻辑
- 需要配置指针字段

## 使用示例

### ✅ 方式 1：不提供构造函数（简单场景）

```go
// service/user.go
package service

//go:generate gocache scan .

type userService struct {
    users  map[int64]*User  // ✅ 自动生成：users: make(map[int64]*User)
    nextID int64            // ✅ 使用零值 0
}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *userService) GetUser(id int64) (*User, error) {
    // 可以安全使用 s.users
    return s.users[id], nil
}
```

生成的代码：
```go
func NewCachedUserService() UserServiceInterface {
    raw := &userService{
        users: make(map[int64]*User),  // ✅ 自动生成
    }
    return &cachedUserService{raw: raw, ...}
}
```

**适用场景：** 简单服务，不需要依赖注入，不需要设置初始值。

### ✅ 方式 2：提供构造函数（推荐，完全控制）

```go
type userService struct {
    users  map[int64]*User
    nextID int64
    db     *sql.DB          // 需要注入依赖
}

// NewUserServiceRaw 创建原始服务
func NewUserServiceRaw(db *sql.DB) *userService {
    return &userService{
        users:  make(map[int64]*User),
        nextID: 1,            // 设置初始值
        db:     db,           // 注入依赖
    }
}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *userService) GetUser(id int64) (*User, error) {
    // 使用 s.db 查询数据库
}
```

生成的代码会调用你的构造函数：
```go
func NewCachedUserService(db *sql.DB) UserServiceInterface {
    raw := NewUserServiceRaw(db)  // ✅ 使用你的构造函数
    
    return &cachedUserService{
        raw:       raw,
        manager:   cache.GetGlobalManager(),
        evaluator: spel.NewSpELEvaluator(),
    }
}
```

**适用场景：** 需要依赖注入、设置初始值、复杂初始化逻辑。

## 最佳实践

1. **简单服务不需要构造函数** - `map` 字段会自动初始化
2. **需要依赖注入时使用构造函数** - 数据库、客户端等
3. **命名规范** - `New<ServiceName>Raw` 或 `New<ServiceName>`
4. **设置初始值时使用构造函数** - 如 `nextID: 1`

```go
// 简单服务 - 无需构造函数
type cacheService struct {
    data map[string]string  // ✅ 自动初始化
}

// 复杂服务 - 使用构造函数
type userService struct {
    users  map[int64]*User
    db     *sql.DB
}

func NewUserServiceRaw(db *sql.DB) *userService {
    return &userService{
        users: make(map[int64]*User),  // 或者在构造函数中初始化
        db:    db,
    }
}
```

## 常见问题

### Q: 构造函数必须叫 `NewXxxRaw` 吗？

**A:** 不必须，但推荐。生成器会自动检测返回 `*serviceType` 的 `New` 开头函数。

### Q: 构造函数可以有参数吗？

**A:** 可以。生成的 `NewCachedXxx` 会继承相同的参数。

### Q: 不提供构造函数会怎样？

**A:** 生成器会：
- ✅ 自动初始化 `map` 字段：`users: make(map[int64]*User)`
- ✅ 基本类型使用零值：`nextID: 0`
- ✅ 指针使用 nil：`db: nil`

### Q: 智能初始化的限制？

**A:** 当前版本：
- ✅ 支持 `map` 类型自动 `make()`
- ✅ 支持基本类型零值
- ❌ 不支持 slice 预分配（使用 nil）
- ❌ 不支持指针字段自动创建

需要这些功能时，请使用构造函数。

## 总结

**v2.1+：简单服务无需构造函数，复杂服务推荐使用。**
