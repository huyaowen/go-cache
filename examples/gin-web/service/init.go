package service

import (
	"github.com/coderiser/go-cache/pkg/proxy"
)

// 零配置服务初始化
// 
// 无需运行代码生成器！
// cache 包的 init() 会自动扫描所有带 @ 注解的方法
// 并注册到缓存系统

// UserService 用户服务实例（带缓存）
// 通过 proxy.SimpleDecorate 自动装饰
var UserService = proxy.SimpleDecorate(NewUserServiceRaw())

// OrderService 订单服务实例（带缓存）
var OrderService = proxy.SimpleDecorate(NewOrderServiceRaw())

// 说明:
// 1. 在 user.go 和 order.go 中，方法前添加注解：
//    // @cacheable(cache="users", key="#id", ttl="30m")
//    func (s *userService) GetUser(id int64) (*User, error) { ... }
//
// 2. 导入 cache 包触发自动扫描（见 main.go）:
//    import _ "github.com/coderiser/go-cache/pkg/cache"
//
// 3. 直接使用装饰后的服务:
//    user, _ := UserService.GetUser(123)
//
// 就这么简单！无需 go generate，无需生成额外代码！
