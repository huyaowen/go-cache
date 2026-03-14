package boundary

// 只有方法没有 struct (使用空接口接收者)

// @cacheable(cache="test", key="#id", ttl="30m")
func StandaloneFunc(id int64) (string, error) {
	return "result", nil
}
