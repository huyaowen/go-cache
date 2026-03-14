package service

import (
	"github.com/coderiser/go-cache/pkg/proxy"
)

// 零配置服务初始化
// 
// 无需运行代码生成器！
// cache 包的 init() 会自动扫描所有带 @ 注解的方法
// 并注册到缓存系统

// 创建带缓存的服务实例（小写变量名避免冲突）
var (
	_userService  = proxy.SimpleDecorate(NewUserService())
	_orderService = proxy.SimpleDecorate(NewOrderService())
)

// UserService 获取用户服务实例（带缓存）
// 返回装饰后的服务，可以直接调用方法
func UserService() *proxy.DecoratedService[*UserService] {
	return _userService
}

// OrderService 获取订单服务实例（带缓存）
func OrderService() *proxy.DecoratedService[*OrderService] {
	return _orderService
}

// 使用说明:
// 1. 在方法前添加 @ 注解（见 user.go, order.go）
// 2. 导入 cache 包触发自动扫描（见 main.go）
// 3. 直接使用装饰后的服务:
//    results, _ := UserService().Invoke("GetUser", 123)
//
// 就这么简单！无需 go generate，无需生成额外代码！
