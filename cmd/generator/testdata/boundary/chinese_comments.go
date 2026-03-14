package boundary

// ChineseService 中文注释测试服务
// 这个服务用于测试中文注释的处理
type ChineseService struct{}

// @cacheable(cache="中文缓存", key="#id", ttl="30m")
// 获取用户信息 - 这是一个中文注释
func (s *ChineseService) GetUser(id int64) (string, error) {
	return "用户" + string(rune(id)), nil
}

// @cacheput(cache="中文缓存", key="#name", ttl="1h")
// 创建用户 - 支持特殊字符："'"\n\r\t
func (s *ChineseService) CreateUser(name string) (string, error) {
	return name, nil
}

// @cacheevict(cache="中文缓存", key="#id")
// 删除用户 - 测试 emoji 表情：😀 🎉 ✅ ❌
func (s *ChineseService) DeleteUser(id int64) error {
	return nil
}
