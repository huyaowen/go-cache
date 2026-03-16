package scan

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/coderiser/go-cache/pkg/logger"
)

var annotationRegex = regexp.MustCompile(`//\s*@(\w+)\s*\(`)

// Scanner 扫描器
type Scanner struct {
	config *Config
	fset   *token.FileSet
}

// NewScanner 创建扫描器
func NewScanner(config *Config) *Scanner {
	return &Scanner{
		config: config,
		fset:   token.NewFileSet(),
	}
}

// Scan 执行扫描
func (s *Scanner) Scan(dirs []string) (*Result, error) {
	modulePath, err := DetectModulePath()
	if err != nil {
		return nil, fmt.Errorf("detect module path: %w", err)
	}

	result := &Result{
		ModulePath: modulePath,
		Packages:   make(map[string]*PackageInfo),
		Generated:  Now(),
	}

	state := LoadState()
	if s.config.Force {
		state = &State{Files: make(map[string]FileState)}
	}

	for _, dir := range dirs {
		if err := s.scanDir(dir, result, state); err != nil {
			if s.config.Verbose {
				logger.Warn("scan %s: %v", dir, err)
			}
		}
	}

	SaveState(state)

	return result, nil
}

// scanDir 扫描目录
func (s *Scanner) scanDir(dir string, result *Result, state *State) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	return filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		if strings.HasSuffix(path, "_cached.go") {
			return nil
		}

		if strings.Contains(path, "gocache_gen") || strings.Contains(path, ".cache-gen") {
			return nil
		}

		if !s.config.Force && !IsFileModified(path, info, state) {
			return nil
		}

		fileInfo, pkgPath, pkgName, err := s.parseFile(path, result.ModulePath)
		if err != nil {
			if s.config.Verbose {
				logger.Warn("parse %s: %v", path, err)
			}
			return nil
		}

		if fileInfo == nil || (len(fileInfo.Interfaces) == 0 && len(fileInfo.Services) == 0) {
			return nil
		}

		pkgInfo, exists := result.Packages[pkgPath]
		if !exists {
			pkgInfo = &PackageInfo{
				ImportPath: pkgPath,
				Dir:        filepath.Dir(path),
				Name:       pkgName,
				Files:      make(map[string]*FileInfo),
			}
			result.Packages[pkgPath] = pkgInfo
		}

		pkgInfo.Files[path] = fileInfo

		if s.config.Verbose {
			logger.Debug("  📄 %s: %d interfaces, %d services",
				filepath.Base(path), len(fileInfo.Interfaces), len(fileInfo.Services))
		}

		return nil
	})
}

// parseFile 解析单个文件
func (s *Scanner) parseFile(path, modulePath string) (*FileInfo, string, string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, "", "", err
	}

	node, err := parser.ParseFile(s.fset, path, content, parser.ParseComments)
	if err != nil {
		return nil, "", "", err
	}

	commentMap := s.buildCommentMap(path, content)

	fileInfo := &FileInfo{
		Path:       path,
		Imports:    make(map[string]string),
		Interfaces: make(map[string]*InterfaceInfo),
		Services:   make(map[string]*ServiceInfo),
	}

	// 解析导入
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		alias := ""
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		fileInfo.Imports[importPath] = alias
	}

	pkgPath := GetImportPath(modulePath, path)
	pkgName := node.Name.Name

	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok == token.TYPE {
				s.parseTypeDecl(d, fileInfo, commentMap)
			}
		case *ast.FuncDecl:
			s.parseFuncDecl(d, fileInfo, commentMap, node.Name.Name)
		}
	}

	return fileInfo, pkgPath, pkgName, nil
}

// parseTypeDecl 解析类型声明（接口和服务定义）
func (s *Scanner) parseTypeDecl(decl *ast.GenDecl, fileInfo *FileInfo, commentMap map[int]string) {
	for _, spec := range decl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		// 解析接口
		ifaceType, ok := typeSpec.Type.(*ast.InterfaceType)
		if ok {
			ifaceName := typeSpec.Name.Name
			ifaceInfo := &InterfaceInfo{
				Name:    ifaceName,
				Methods: make([]*MethodSpec, 0),
			}

			for _, method := range ifaceType.Methods.List {
				funcType, ok := method.Type.(*ast.FuncType)
				if !ok {
					continue
				}

				methodSpec := s.parseMethodSpec(method, funcType)
				if methodSpec != nil {
					ifaceInfo.Methods = append(ifaceInfo.Methods, methodSpec)
				}
			}

			if len(ifaceInfo.Methods) > 0 {
				fileInfo.Interfaces[ifaceName] = ifaceInfo
			}
			continue
		}

		// 解析结构体（服务）
		structType, ok := typeSpec.Type.(*ast.StructType)
		if ok {
			svcName := typeSpec.Name.Name
			svcInfo := &ServiceInfo{
				TypeName: svcName,
				Methods:  make(map[string]*MethodInfo),
				Fields:   make([]*FieldInfo, 0),
			}

			// 解析字段
			if structType.Fields != nil {
				for _, field := range structType.Fields.List {
					fieldInfo := s.parseField(field)
					if fieldInfo != nil {
						svcInfo.Fields = append(svcInfo.Fields, fieldInfo)
					}
				}
			}

			fileInfo.Services[svcName] = svcInfo
		}
	}
}

// parseField 解析结构体字段
func (s *Scanner) parseField(field *ast.Field) *FieldInfo {
	if len(field.Names) == 0 {
		return nil // 匿名字段
	}

	typeStr := getTypeString(field.Type, s.fset)
	kind := s.getFieldKind(field.Type)

	return &FieldInfo{
		Name: field.Names[0].Name,
		Type: typeStr,
		Kind: kind,
	}
}

// getFieldKind 获取字段类型种类
func (s *Scanner) getFieldKind(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.MapType:
		return "map"
	case *ast.ArrayType, *ast.StarExpr:
		if _, ok := t.(*ast.ArrayType); ok {
			return "slice"
		}
		return "pointer"
	case *ast.StructType:
		return "struct"
	case *ast.Ident:
		// 基本类型
		switch t.Name {
		case "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64",
			"float32", "float64", "string", "bool", "byte", "rune":
			return "basic"
		}
		return "custom"
	default:
		return "unknown"
	}
}

// parseFuncDecl 解析函数声明（服务方法和构造函数）
func (s *Scanner) parseFuncDecl(fn *ast.FuncDecl, fileInfo *FileInfo, commentMap map[int]string, pkgName string) {
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		s.parseMethod(fn, fileInfo, commentMap)
	} else if fn.Name != nil && strings.HasPrefix(fn.Name.Name, "New") {
		s.parseConstructor(fn, fileInfo, pkgName)
	}
}

// parseMethod 解析方法（服务实现）
func (s *Scanner) parseMethod(fn *ast.FuncDecl, fileInfo *FileInfo, commentMap map[int]string) {
	recvType := getReceiverTypeName(fn.Recv.List[0].Type)
	if recvType == "" {
		return
	}

	methodName := fn.Name.Name
	methodPos := s.fset.Position(fn.Pos()).Line

	var comment string
	for line := methodPos - 1; line >= 1; line-- {
		if c, exists := commentMap[line]; exists {
			comment = c
			break
		}
	}

	annotation := s.parseAnnotation(comment)
	if annotation == nil {
		return
	}

	svcInfo, exists := fileInfo.Services[recvType]
	if !exists {
		svcInfo = &ServiceInfo{
			TypeName: recvType,
			Methods:  make(map[string]*MethodInfo),
		}
		fileInfo.Services[recvType] = svcInfo
	}

	methodInfo := &MethodInfo{
		Name:      methodName,
		Params:    s.parseParams(fn.Type.Params),
		Returns:   s.parseParams(fn.Type.Results),
		Operation: annotation.Type,
		Cache:     annotation.CacheName,
		Key:       annotation.Key,
		TTL:       annotation.TTL,
		Condition: annotation.Condition,
		Unless:    annotation.Unless,
		Before:    annotation.Before,
	}

	svcInfo.Methods[methodName] = methodInfo
}

// parseConstructor 解析构造函数
func (s *Scanner) parseConstructor(fn *ast.FuncDecl, fileInfo *FileInfo, pkgName string) {
	if fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return
	}

	returnType := getTypeString(fn.Type.Results.List[0].Type, s.fset)

	var svcTypeName string
	if strings.HasPrefix(returnType, "*") {
		svcTypeName = returnType[1:]
	} else {
		svcTypeName = returnType
	}

	svcInfo, exists := fileInfo.Services[svcTypeName]
	if !exists {
		svcInfo = &ServiceInfo{
			TypeName: svcTypeName,
			Methods:  make(map[string]*MethodInfo),
		}
		fileInfo.Services[svcTypeName] = svcInfo
	}

	svcInfo.Constructor = &ConstructorInfo{
		Name:    fn.Name.Name,
		Params:  s.parseParams(fn.Type.Params),
		Returns: s.parseParams(fn.Type.Results),
	}
}

// parseMethodSpec 解析方法规格
func (s *Scanner) parseMethodSpec(field *ast.Field, funcType *ast.FuncType) *MethodSpec {
	if len(field.Names) == 0 {
		return nil
	}

	return &MethodSpec{
		Name:    field.Names[0].Name,
		Params:  s.parseParams(funcType.Params),
		Returns: s.parseParams(funcType.Results),
	}
}

// parseParams 解析参数列表
func (s *Scanner) parseParams(params *ast.FieldList) []*ParamSpec {
	if params == nil {
		return nil
	}

	var result []*ParamSpec
	for _, param := range params.List {
		typeStr := getTypeString(param.Type, s.fset)
		if len(param.Names) > 0 {
			for _, name := range param.Names {
				result = append(result, &ParamSpec{
					Name: name.Name,
					Type: typeStr,
				})
			}
		} else {
			result = append(result, &ParamSpec{
				Name: "_",
				Type: typeStr,
			})
		}
	}
	return result
}

// buildCommentMap 构建注释映射
func (s *Scanner) buildCommentMap(path string, content []byte) map[int]string {
	commentMap := make(map[int]string)

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	lineNum := 0
	var currentComment string

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "//") {
			currentComment += line + "\n"
			commentMap[lineNum] = currentComment
		} else if trimmed == "" {
			continue
		} else {
			currentComment = ""
		}
	}

	return commentMap
}

// parseAnnotation 解析注解
func (s *Scanner) parseAnnotation(comment string) *AnnotationInfo {
	if comment == "" {
		return nil
	}

	matches := annotationRegex.FindStringSubmatch(comment)
	if len(matches) < 2 {
		return nil
	}

	annotation := &AnnotationInfo{
		Type: matches[1],
	}

	// 从注解位置开始查找配对的括号
	// 先找到注解在 comment 中的位置
	annotationStart := strings.Index(comment, matches[0])
	if annotationStart == -1 {
		return nil
	}
	
	// 从注解位置查找括号
	rest := comment[annotationStart:]
	start := strings.Index(rest, "(")
	end := strings.LastIndex(rest, ")")
	if start == -1 || end == -1 || end <= start {
		return nil
	}

	paramsStr := rest[start+1 : end]
	params := parseAnnotationParams(paramsStr)

	for key, value := range params {
		switch key {
		case "cache":
			annotation.CacheName = value
		case "key":
			annotation.Key = value
		case "ttl":
			annotation.TTL = value
		case "condition":
			annotation.Condition = value
		case "unless":
			annotation.Unless = value
		case "before":
			annotation.Before = value == "true"
		}
	}

	if annotation.CacheName == "" || annotation.Key == "" {
		return nil
	}

	return annotation
}

// parseAnnotationParams 解析注解参数
func parseAnnotationParams(paramsStr string) map[string]string {
	result := make(map[string]string)
	paramsStr = strings.TrimSpace(paramsStr)

	for len(paramsStr) > 0 {
		eqIdx := strings.Index(paramsStr, "=")
		if eqIdx == -1 {
			break
		}

		key := strings.TrimSpace(paramsStr[:eqIdx])
		paramsStr = strings.TrimSpace(paramsStr[eqIdx+1:])

		if len(paramsStr) == 0 {
			break
		}

		var value string
		var consumed int

		if paramsStr[0] == '"' || paramsStr[0] == '\'' {
			quote := paramsStr[0]
			paramsStr = paramsStr[1:]
			consumed = 1

			for i := 0; i < len(paramsStr); i++ {
				if paramsStr[i] == quote && (i == 0 || paramsStr[i-1] != '\\') {
					value = paramsStr[:i]
					consumed = i + 1
					break
				}
			}

			for consumed < len(paramsStr) {
				if paramsStr[consumed] == ',' || paramsStr[consumed] == ')' {
					break
				}
				consumed++
			}
		} else {
			for i := 0; i < len(paramsStr); i++ {
				if paramsStr[i] == ',' || paramsStr[i] == ')' || paramsStr[i] == ' ' {
					value = paramsStr[:i]
					consumed = i
					break
				}
				if i == len(paramsStr)-1 {
					value = paramsStr
					consumed = len(paramsStr)
				}
			}
		}

		result[key] = value
		paramsStr = strings.TrimSpace(paramsStr[consumed:])
		if len(paramsStr) > 0 && paramsStr[0] == ',' {
			paramsStr = strings.TrimSpace(paramsStr[1:])
		}
	}

	return result
}

// getReceiverTypeName 获取接收者类型名
func getReceiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return ""
}

// getTypeString 获取类型字符串
func getTypeString(expr ast.Expr, fset *token.FileSet) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + getTypeString(t.X, fset)
	case *ast.SelectorExpr:
		return getTypeString(t.X, fset) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + getTypeString(t.Elt, fset)
	case *ast.MapType:
		return "map[" + getTypeString(t.Key, fset) + "]" + getTypeString(t.Value, fset)
	case *ast.ChanType:
		return "chan " + getTypeString(t.Value, fset)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + getTypeString(t.Elt, fset)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// Now 获取当前时间（便于测试）
var Now = func() time.Time {
	return time.Now()
}
