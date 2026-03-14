package service

import (
	"github.com/coderiser/go-cache/pkg/proxy"
)

// 零配置服务初始化
// 
// 无需运行代码生成器！
// cache 包的 init() 会自动扫描所有带 @ 注解的方法
// 并注册到缓存系统

// 创建带缓存的服务实例
// 使用 proxy.SimpleDecorateWithInterface 返回接口类型
// 可以直接调用方法，无需 Invoke()
var (
	_userService  UserService
	_orderService OrderService
)

// UserService 获取用户服务实例（带缓存）
func UserService() UserService {
	return _userService
}

// OrderService 获取订单服务实例（带缓存）
func OrderService() OrderService {
	return _orderService
}

func init() {
	// SimpleDecorateWithInterface 会返回装饰后的服务
	// 保持接口类型，可以直接调用方法
	_userService = proxy.SimpleDecorateWithInterface(NewUserService(), _userService)
	_orderService = proxy.SimpleDecorateWithInterface(NewOrderService(), _orderService)
}

// 使用说明:
// 1. 在方法前添加 @ 注解（见 user.go, order.go）
// 2. 导入 cache 包触发自动扫描（见 main.go）
// 3. 直接调用方法:
//    user, _ := UserService().GetUser(123)
//
// 就这么简单！无需 go generate，无需生成额外代码！
