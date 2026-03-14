package service

import (
	"github.com/coderiser/go-cache/pkg/proxy"
)

// 零配置服务初始化（gRPC 示例）

// 创建带缓存的服务实例
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
	// SimpleDecorateWithInterface 返回接口类型
	// 可以直接调用方法，无需 Invoke()
	_userService = proxy.SimpleDecorateWithInterface(NewUserService(), _userService)
	_orderService = proxy.SimpleDecorateWithInterface(NewOrderService(), _orderService)
}

// 使用说明:
// 1. 在方法前添加 @ 注解
// 2. 导入 cache 包触发自动扫描
// 3. 直接调用方法：user, _ := UserService().GetUser(123)
