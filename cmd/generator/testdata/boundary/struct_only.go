package boundary

// OnlyStruct 只有 struct 没有方法
type OnlyStruct struct {
	ID   int64
	Name string
}

// AnotherStruct 另一个只有 struct 的类型
type AnotherStruct struct {
	Value int
}
