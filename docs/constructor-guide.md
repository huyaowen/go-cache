# 服务构造函数指南

## 为什么需要构造函数？

`gocache` 生成的缓存包装器需要创建服务实例。如果你的服务包含以下字段，**必须提供构造函数**：

- `map` 类型字段
- `slice` 类型字段（需要预分配）
- 需要初始化的指针字段
- 需要配置的连接/客户端

## 正确示例

### ✅ 提供构造函数（推荐）

```go
// service/user.go
package service

//go:generate gocache scan .

type userService struct {
    users  map[int64]*User  // map 需要初始化
    nextID int64            // 需要设置初始值
    db     *sql.DB          // 指针需要赋值
}

// NewUserServiceRaw 创建原始服务
func NewUserServiceRaw(db *sql.DB) *userService {
    return &userService{
        users:  make(map[int64]*User),
        nextID: 1,
        db:     db,
    }
}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *userService) GetUser(id int64) (*User, error) {
    // 安全使用 s.users
}
```

生成的代码会调用你的构造函数：
```go
func NewCachedUserService(db *sql.DB) UserServiceInterface {
    raw := NewUserServiceRaw(db)  // ✅ 正确初始化
    
    return &cachedUserService{
        raw:       raw,
        manager:   cache.GetGlobalManager(),
        evaluator: spel.NewSpELEvaluator(),
    }
}
```

### ❌ 不提供构造函数（危险）

```go
type userService struct {
    users map[int64]*User
}

// 没有提供构造函数

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *userService) GetUser(id int64) (*User, error) {
    // ❌ panic: assignment to entry in nil map
    s.users[id] = user
}
```

生成的代码会直接实例化：
```go
func NewCachedUserService() UserServiceInterface {
    raw := &userService{}  // ❌ users 是 nil map
    
    return &cachedUserService{raw: raw, ...}
}
```

## 零值可用的情况

如果你的服务所有字段都可以使用零值，可以不提供构造函数：

```go
// ✅ 可以不提供构造函数
type statelessService struct {
    // 无字段，或只有基本类型
}

func (s *statelessService) GetData(key string) (string, error) {
    // 不依赖任何状态
    return getDataFromDB(key)
}
```

## 最佳实践

1. **总是提供构造函数** - 即使当前不需要初始化
2. **命名规范** - `New<ServiceName>Raw` 或 `New<ServiceName>`
3. **初始化所有集合** - `map`、`slice` 使用 `make()`
4. **注入依赖** - 数据库、客户端等通过构造函数传入

```go
func NewUserServiceRaw(db *sql.DB, redis *redis.Client) *userService {
    return &userService{
        db:     db,
        redis:  redis,
        users:  make(map[int64]*User),
        orders: make(map[int64]*Order),
    }
}
```

## 常见问题

### Q: 构造函数必须叫 `NewXxxRaw` 吗？

**A:** 不必须，但推荐。生成器会自动检测返回 `*serviceType` 的 `New` 开头函数。

### Q: 构造函数可以有参数吗？

**A:** 可以。生成的 `NewCachedXxx` 会继承相同的参数。

### Q: 我忘了提供构造函数，会怎样？

**A:** 生成的代码使用 `&serviceType{}` 实例化。如果服务有未初始化的字段（如 nil map），会在运行时 panic。

## 总结

**一句话：提供构造函数，确保字段正确初始化。**
