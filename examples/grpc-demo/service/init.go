package service

import (
	"github.com/coderiser/go-cache/pkg/proxy"
)

// 零配置服务初始化（gRPC 示例）

// 创建带缓存的服务实例（小写变量名避免冲突）
var (
	_userService  = proxy.SimpleDecorate(NewUserService())
	_orderService = proxy.SimpleDecorate(NewOrderService())
)

// UserService 获取用户服务实例（带缓存）
func UserService() *proxy.DecoratedService[*UserService] {
	return _userService
}

// OrderService 获取订单服务实例（带缓存）
func OrderService() *proxy.DecoratedService[*OrderService] {
	return _orderService
}

// 使用说明:
// 1. 在方法前添加 @ 注解
// 2. 导入 cache 包触发自动扫描
// 3. 直接使用装饰后的服务
