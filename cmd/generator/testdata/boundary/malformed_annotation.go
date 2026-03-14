package boundary

// MalformedService 测试注解格式错误
type MalformedService struct{}

// @cacheable(cache="test"  // 缺少闭合括号
func (s *MalformedService) GetMalformed(id int64) (string, error) {
	return "result", nil
}

// @cacheable(cache="test", key=)  // 参数值缺失
func (s *MalformedService) GetMissingValue(id int64) (string, error) {
	return "result", nil
}

// @cacheable  // 参数完全缺失
func (s *MalformedService) GetNoParams(id int64) (string, error) {
	return "result", nil
}

// @cacheable(cache="test", key="#id", ttl="30m")  // 正确的注解
func (s *MalformedService) GetValid(id int64) (string, error) {
	return "result", nil
}
