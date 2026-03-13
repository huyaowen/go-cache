package service

// InitService 初始化服务（使用全局缓存管理器）
func InitService() UserServiceInterface {
	return NewUserService()
}
