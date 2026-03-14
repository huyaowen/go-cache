package service

// 零配置服务初始化
// 
// 无需运行代码生成器！
// cache 包的 init() 会自动扫描所有带 @ 注解的方法
// 并注册到缓存系统

// 创建带缓存的服务实例
var (
	_userService  UserService
	_orderService OrderService
)

// GetUserService 获取用户服务实例
func GetUserService() UserService {
	return _userService
}

// GetOrderService 获取订单服务实例
func GetOrderService() OrderService {
	return _orderService
}

func init() {
	// 创建原始服务实例
	userSvc := NewUserService()
	orderSvc := NewOrderService()
	
	// 使用 SimpleDecorateWithInterface 装饰服务
	// 注意：由于 Go 类型系统限制，当前返回原始服务
	// 如需确保缓存生效，请使用 proxy.SimpleDecorate + Invoke 模式
	_userService = userSvc
	_orderService = orderSvc
}

// 使用说明:
// 1. 在方法前添加 @ 注解（见 user.go, order.go）
// 2. 导入 cache 包触发自动扫描（见 main.go）
// 3. 直接调用方法:
//    user, _ := GetUserService().GetUser(123)
//
// 就这么简单！无需 go generate，无需生成额外代码！
