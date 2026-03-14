package boundary

// SpecialCharsService 特殊字符测试
type SpecialCharsService struct{}

// @cacheable(cache="test\"quotes\"", key="#id", ttl="30m")
func (s *SpecialCharsService) GetWithQuotes(id int64) (string, error) {
	return "result", nil
}

// @cacheable(cache="test\nnewline", key="#id", ttl="30m")
func (s *SpecialCharsService) GetWithNewline(id int64) (string, error) {
	return "result", nil
}

// @cacheable(cache="test\ttab", key="#id", ttl="30m")
func (s *SpecialCharsService) GetWithTab(id int64) (string, error) {
	return "result", nil
}

// @cacheable(cache="test\\backslash", key="#id", ttl="30m")
func (s *SpecialCharsService) GetWithBackslash(id int64) (string, error) {
	return "result", nil
}
