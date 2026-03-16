package scan

import "time"

// Config 扫描配置
type Config struct {
	Dirs    []string // 扫描目录
	Force   bool     // 强制全量扫描
	Verbose bool     // 详细输出
}

// Result 扫描结果
type Result struct {
	ModulePath string                  // 模块路径（从 go.mod 获取）
	Packages   map[string]*PackageInfo // 包路径 -> 包信息
	Generated  time.Time               // 生成时间
}

// PackageInfo 包信息
type PackageInfo struct {
	ImportPath string               // 导入路径
	Dir        string               // 文件系统路径
	Name       string               // 包名
	Files      map[string]*FileInfo // 文件路径 -> 文件信息
}

// FileInfo 文件信息
type FileInfo struct {
	Path       string                    // 文件路径
	Imports    map[string]string         // 导入路径 -> 别名 (e.g., "github.com/xxx/model" -> "model")
	Interfaces map[string]*InterfaceInfo // 接口名 -> 接口信息
	Services   map[string]*ServiceInfo   // 类型名 -> 服务信息
}

// InterfaceInfo 接口信息
type InterfaceInfo struct {
	Name    string        // 接口名
	Methods []*MethodSpec // 方法定义
}

// ServiceInfo 服务实现信息
type ServiceInfo struct {
	TypeName    string                 // 类型名（小写开头）
	Implements  string                 // 实现的接口名（匹配后设置）
	Methods     map[string]*MethodInfo // 方法名 -> 方法信息
	Constructor *ConstructorInfo       // 构造函数信息（可选）
	Fields      []*FieldInfo           // 结构体字段信息（用于智能初始化）
}

// FieldInfo 字段信息
type FieldInfo struct {
	Name string // 字段名
	Type string // 字段类型
	Kind string // 字段种类：map/slice/struct/pointer/basic
}

// ConstructorInfo 构造函数信息
type ConstructorInfo struct {
	Name    string       // 构造函数名
	Params  []*ParamSpec // 参数
	Returns []*ParamSpec // 返回值
}

// MethodSpec 方法规格
type MethodSpec struct {
	Name    string       // 方法名
	Params  []*ParamSpec // 参数
	Returns []*ParamSpec // 返回值
}

// ParamSpec 参数规格
type ParamSpec struct {
	Name string // 参数名
	Type string // 参数类型
}

// MethodInfo 方法信息（含注解）
type MethodInfo struct {
	Name      string // 方法名
	Params    []*ParamSpec
	Returns   []*ParamSpec
	Operation string // cacheable/cacheput/cacheevict
	Cache     string // 缓存名
	Key       string // 缓存 key 表达式
	TTL       string // TTL
	Condition string // 条件表达式
	Unless    string // unless 表达式
	Before    bool   // cacheevict 的 before 标志
}

// MatchResult 匹配结果
type MatchResult struct {
	Interface *InterfaceInfo // 接口信息
	Service   *ServiceInfo   // 服务实现信息
}

// State 增量扫描状态
type State struct {
	LastScan time.Time            `json:"last_scan"`
	Files    map[string]FileState `json:"files"`
}

// FileState 文件状态
type FileState struct {
	ModTime time.Time `json:"mod_time"`
	Hash    string    `json:"hash"`
}

// AnnotationInfo 注解信息（从注释解析）
type AnnotationInfo struct {
	Type      string // cacheable/cacheput/cacheevict
	CacheName string
	Key       string
	TTL       string
	Condition string
	Unless    string
	Before    bool
}
