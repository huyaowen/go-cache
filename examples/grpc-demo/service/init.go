package service

// 零配置服务初始化（gRPC 示例）

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
// 1. 在方法前添加 @ 注解
// 2. 导入 cache 包触发自动扫描
// 3. 直接调用方法：user, _ := GetUserService().GetUser(123)
