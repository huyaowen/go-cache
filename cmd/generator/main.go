package main

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
)

// MethodInfo 方法信息（用于生成包装器）
type MethodInfo struct {
	Name         string
	Params       []ParamInfo
	Results      []ParamInfo
	HasError     bool
	ResultType   string // 主要返回类型（第一个非 error 返回值）
}

// ParamInfo 参数信息
type ParamInfo struct {
	Name string
	Type string
}

// InterfaceInfo 接口信息
type InterfaceInfo struct {
	Name    string
	Methods []*MethodInfo
}

// CacheAnnotation 缓存注解
type CacheAnnotation struct {
	Type      string
	CacheName string
	Key       string
	TTL       string
	Condition string
	Unless    string
	Before    bool
	Sync      bool
}

var annotationRegex = regexp.MustCompile(`//\s*@(\w+)\s*\(([^)]+)\)`)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go-cache-gen <directory> [...]")
		fmt.Println("  Scans Go files for cache annotations and generates registration code")
		fmt.Println("  Output:")
		fmt.Println("    - .cache-gen/auto_register.go (annotation registration)")
		fmt.Println("    - <package>/service/generated_wrapper.go (interface wrappers)")
		os.Exit(1)
	}

	dirs := os.Args[1:]
	fmt.Printf("🔍 Scanning %d directory/directories...\n", len(dirs))

	allAnnotations := make(map[string]map[string]*CacheAnnotation) // typeName -> methodName -> annotation
	allInterfaces := make(map[string]*InterfaceInfo)                // typeName -> interface info

	for _, dir := range dirs {
		fmt.Printf("  📁 %s\n", dir)
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			// Skip generated files
			if strings.Contains(path, ".cache-gen") || strings.Contains(path, "generated_") {
				return nil
			}

			parseFile(path, allAnnotations, allInterfaces)
			return nil
		})

		if err != nil {
			fmt.Printf("  ❌ Error walking %s: %v\n", dir, err)
			continue
		}
	}

	if len(allAnnotations) == 0 {
		fmt.Println("⚠️  No annotations found, skipping code generation")
		return
	}

	generateCode(allAnnotations, allInterfaces)
	fmt.Println("✅ Code generation complete!")
}

func parseFile(path string, annotations map[string]map[string]*CacheAnnotation, interfaces map[string]*InterfaceInfo) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return
	}

	// 读取文件内容以获取注释
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	var currentComment string
	scanner := bufio.NewScanner(file)
	lineNum := 0
	commentLines := make(map[int]string)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			currentComment += line + "\n"
			commentLines[lineNum] = currentComment
		} else {
			currentComment = ""
		}
	}

	// 遍历 AST 查找接口和方法
	for _, decl := range node.Decls {
		// 查找接口定义
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
						interfaceName := typeSpec.Name.Name
						interfaceInfo := &InterfaceInfo{
							Name: interfaceName,
						}
						
						// 解析接口方法
						for _, method := range interfaceType.Methods.List {
							if funcType, ok := method.Type.(*ast.FuncType); ok {
								methodInfo := parseMethodType(method, funcType, fset)
								if methodInfo != nil {
									interfaceInfo.Methods = append(interfaceInfo.Methods, methodInfo)
								}
							}
						}
						
						if len(interfaceInfo.Methods) > 0 {
							interfaces[interfaceName] = interfaceInfo
						}
					}
				}
			}
		}
		
		// 查找带注解的方法
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
			continue
		}

		// 获取接收者类型名
		recvType := getReceiverTypeName(fn.Recv.List[0].Type)
		if recvType == "" {
			continue
		}

		// 查找方法前的注释
		methodName := fn.Name.Name
		methodPos := fset.Position(fn.Pos()).Line

		// 查找前一行的注释
		var comment string
		for line := methodPos - 1; line >= 1; line-- {
			if c, exists := commentLines[line]; exists {
				comment = c
				break
			}
		}

		if comment == "" {
			continue
		}

		// 解析注解
		annotation := parseAnnotation(comment)
		if annotation == nil {
			continue
		}

		if annotations[recvType] == nil {
			annotations[recvType] = make(map[string]*CacheAnnotation)
		}
		annotations[recvType][methodName] = annotation
	}
}

// parseMethodType 解析方法类型
func parseMethodType(field *ast.Field, funcType *ast.FuncType, fset *token.FileSet) *MethodInfo {
	methodInfo := &MethodInfo{
		Name: field.Names[0].Name,
	}
	
	// 解析参数
	if funcType.Params != nil {
		for _, param := range funcType.Params.List {
			paramType := getTypeString(param.Type, fset)
			paramName := "_"
			if len(param.Names) > 0 {
				paramName = param.Names[0].Name
			}
			methodInfo.Params = append(methodInfo.Params, ParamInfo{
				Name: paramName,
				Type: paramType,
			})
		}
	}
	
	// 解析返回值
	if funcType.Results != nil {
		for _, result := range funcType.Results.List {
			resultType := getTypeString(result.Type, fset)
			resultName := ""
			if len(result.Names) > 0 {
				resultName = result.Names[0].Name
			}
			
			methodInfo.Results = append(methodInfo.Results, ParamInfo{
				Name: resultName,
				Type: resultType,
			})
			
			// 检查是否为 error 类型
			if resultType == "error" {
				methodInfo.HasError = true
			} else {
				methodInfo.ResultType = resultType
			}
		}
	}
	
	return methodInfo
}

// getTypeString 获取类型的字符串表示
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
	default:
		return fset.Position(expr.Pos()).String()
	}
}

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

func parseAnnotation(comment string) *CacheAnnotation {
	matches := annotationRegex.FindStringSubmatch(comment)
	if len(matches) < 3 {
		return nil
	}

	annotation := &CacheAnnotation{
		Type: matches[1],
	}

	// 解析参数
	params := strings.Split(matches[2], ",")
	for _, param := range params {
		parts := strings.SplitN(strings.TrimSpace(param), "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// 去除引号
		value = strings.Trim(value, "\"'")

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
		case "sync":
			annotation.Sync = value == "true"
		}
	}

	// 验证必需字段
	if annotation.CacheName == "" || annotation.Key == "" {
		return nil
	}

	return annotation
}

func generateCode(annotations map[string]map[string]*CacheAnnotation, interfaces map[string]*InterfaceInfo) {
	// 1. 生成注解注册代码
	generateAnnotationRegistration(annotations)
	
	// 2. 生成接口包装器代码
	generateInterfaceWrappers(annotations, interfaces)
}

// generateAnnotationRegistration 生成注解注册代码
func generateAnnotationRegistration(annotations map[string]map[string]*CacheAnnotation) {
	// 创建输出目录
	outputDir := ".cache-gen"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("❌ Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	total := countAnnotations(annotations)
	timestamp := time.Now().Format(time.RFC3339)

	code := fmt.Sprintf(`// Code generated by go-cache-gen. DO NOT EDIT.
// Generated at: %s
// Total annotations: %d

package cached

import (
	"github.com/coderiser/go-cache/pkg/proxy"
	gocache "github.com/coderiser/go-cache/pkg/cache"
)

func init() {
`, timestamp, total)

	for typeName, methods := range annotations {
		for methodName, annotation := range methods {
			code += fmt.Sprintf("\tgocache.RegisterGlobalAnnotation(\"%s\", \"%s\", &proxy.CacheAnnotation{\n", typeName, methodName)
			code += fmt.Sprintf("\t\tType:      \"%s\",\n", annotation.Type)
			code += fmt.Sprintf("\t\tCacheName: \"%s\",\n", annotation.CacheName)
			code += fmt.Sprintf("\t\tKey:       \"%s\",\n", annotation.Key)
			if annotation.TTL != "" {
				code += fmt.Sprintf("\t\tTTL:       \"%s\",\n", annotation.TTL)
			}
			if annotation.Condition != "" {
				code += fmt.Sprintf("\t\tCondition: \"%s\",\n", annotation.Condition)
			}
			if annotation.Unless != "" {
				code += fmt.Sprintf("\t\tUnless:    \"%s\",\n", annotation.Unless)
			}
			if annotation.Before {
				code += fmt.Sprintf("\t\tBefore:    true,\n")
			}
			if annotation.Sync {
				code += fmt.Sprintf("\t\tSync:      true,\n")
			}
			code += "\t})\n"
		}
	}

	code += "}\n"

	outputPath := filepath.Join(outputDir, "auto_register.go")
	err := os.WriteFile(outputPath, []byte(code), 0644)
	if err != nil {
		fmt.Printf("❌ Error writing generated code: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📝 Generated: %s\n", outputPath)
	fmt.Printf("📊 Total annotations: %d\n", total)
	
	// Print summary
	fmt.Println("\n📋 Summary by type:")
	for typeName, methods := range annotations {
		fmt.Printf("  - %s: %d method(s)\n", typeName, len(methods))
		for methodName := range methods {
			fmt.Printf("    • %s\n", methodName)
		}
	}
}

// generateInterfaceWrappers 生成接口包装器代码（方案 G：Beego 融合版）
func generateInterfaceWrappers(annotations map[string]map[string]*CacheAnnotation, interfaces map[string]*InterfaceInfo) {
	// 找出有注解的接口，为它们生成包装器
	for interfaceName, interfaceInfo := range interfaces {
		// 检查是否有对应的实现类型有注解
		// 简单匹配：如果接口名是 UserServiceInterface，实现类型可能是 userService
		// 这里我们假设接口名去掉 "Interface" 后缀就是实现类型名（小写开头）
		expectedImpl := strings.TrimSuffix(interfaceName, "Interface")
		if expectedImpl == interfaceName {
			continue // 不是 Interface 结尾的接口，跳过
		}
		// 首字母小写：UserService -> userService
		expectedImpl = strings.ToLower(string(expectedImpl[0])) + expectedImpl[1:]
		
		hasAnnotations := false
		for typeName := range annotations {
			if typeName == expectedImpl {
				hasAnnotations = true
				break
			}
		}
		
		if !hasAnnotations || len(interfaceInfo.Methods) == 0 {
			continue
		}
		
		// 生成包装器代码（方案 G）
		generateCachedServiceForInterface(interfaceName, interfaceInfo, expectedImpl, annotations[expectedImpl])
	}
}

// generateCachedServiceForInterface 为单个接口生成缓存服务（方案 G）
func generateCachedServiceForInterface(interfaceName string, interfaceInfo *InterfaceInfo, implName string, typeAnnotations map[string]*CacheAnnotation) {
	// serviceName 是大写开头的服务名（用于类型名，如 UserService）
	serviceName := strings.TrimSuffix(interfaceName, "Interface")
	// cachedTypeName 是缓存装饰器类型名（如 cachedProductService）
	cachedTypeName := "cached" + serviceName
	// rawServiceFuncName 是原始服务构造函数名（如 NewProductService）
	// 从 implName 推导：productService -> NewProductService
	rawServiceFuncName := "New" + strings.ToUpper(string(implName[0])) + implName[1:]
	
	// 收集需要的导入 (key: 包路径，value: 别名)
	importsMap := make(map[string]string)
	importsMap["context"] = ""
	importsMap["fmt"] = ""
	importsMap["time"] = ""
	importsMap["github.com/coderiser/go-cache/pkg/core"] = ""
	// 使用别名避免包名冲突
	importsMap["github.com/coderiser/go-cache/pkg/cache"] = "gocache"
	
	// 检查是否需要 model 包导入
	currentDir, _ := os.Getwd()
	dirName := filepath.Base(currentDir)
	modelPath := fmt.Sprintf("github.com/coderiser/go-cache/examples/%s/model", dirName)
	
	for _, method := range interfaceInfo.Methods {
		for _, param := range method.Params {
			if strings.Contains(param.Type, "model.") {
				importsMap[modelPath] = ""
			}
		}
		for _, result := range method.Results {
			if strings.Contains(result.Type, "model.") {
				importsMap[modelPath] = ""
			}
		}
	}
	
	// 构建导入语句 (支持别名)
	var imports []string
	for imp, alias := range importsMap {
		if alias != "" {
			imports = append(imports, fmt.Sprintf("\t%s \"%s\"", alias, imp))
		} else {
			imports = append(imports, fmt.Sprintf("\t\"%s\"", imp))
		}
	}
	importsStr := strings.Join(imports, "\n")
	
	timestamp := time.Now().Format(time.RFC3339)
	
	// 输出到 .cache-gen 目录
	outputDir := ".cache-gen"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("❌ Error creating output directory: %v\n", err)
		return
	}
	
	// 文件名：<type>_cached.go（如 product_cached.go）
	fileNameBase := strings.TrimSuffix(interfaceName, "Interface")
	fileNameBase = strings.ToLower(string(fileNameBase[0])) + fileNameBase[1:]
	// 按大写字母分割，取第一个部分
	for i, ch := range fileNameBase {
		if i > 0 && ch >= 'A' && ch <= 'Z' {
			fileNameBase = fileNameBase[:i]
			break
		}
	}
	outputFileName := fileNameBase + "_cached.go"
	outputPath := filepath.Join(outputDir, outputFileName)
	
	// 生成 NewXxxService 函数
	newServiceFuncs := fmt.Sprintf(`
// New%[1]s 创建带缓存的%[1]s 实例（使用全局 Manager）
func New%[1]s() service.%[2]s {
	return New%[1]sWithManager(gocache.GetGlobalManager())
}

// New%[1]sWithManager 创建带缓存的%[1]s 实例（使用指定 Manager）
func New%[1]sWithManager(manager core.CacheManager) service.%[2]s {
	raw := service.%[3]s(manager)
	return &%[4]s{
		decorated: raw,
		manager:   manager,
	}
}
`, serviceName, interfaceName, rawServiceFuncName, cachedTypeName)
	
	code := fmt.Sprintf(`// Code generated by go-cache-gen. DO NOT EDIT.
// Generated at: %s
// Interface: %s
// Package: cached (generated cache wrappers)

package cached

import (
%s
	"github.com/coderiser/go-cache/examples/cron-job/service"
)

// %s 带缓存的包装实现
// 实现 service.%s 接口
type %s struct {
	decorated service.%s
	manager   core.CacheManager
}

%s
`, timestamp, interfaceName, importsStr, cachedTypeName, interfaceName, cachedTypeName, interfaceName, newServiceFuncs)

	// 生成每个接口方法的实现
	for _, method := range interfaceInfo.Methods {
		ann := typeAnnotations[method.Name]
		if ann != nil {
			code += generateCachedMethodWithAnnotation(method, ann, cachedTypeName)
		} else {
			code += generateSimplePassthroughMethod(method, cachedTypeName)
		}
	}

	// 添加接口实现检查
	code += fmt.Sprintf("\n// 确保实现接口\nvar _ service.%s = (*%s)(nil)\n", interfaceName, cachedTypeName)

	// 写入文件
	if err := os.WriteFile(outputPath, []byte(code), 0644); err != nil {
		fmt.Printf("❌ Error writing wrapper code: %v\n", err)
		return
	}

	fmt.Printf("📝 Generated cached service: %s (for %s)\n", outputPath, interfaceName)
}

// generateMethodImplementation 生成单个方法的实现
func generateMethodImplementation(method *MethodInfo, decoratedTypeName, serviceName string) string {
	// 构建参数列表字符串
	var params []string
	var argNames []string
	for i, param := range method.Params {
		name := param.Name
		if name == "_" || name == "" {
			name = fmt.Sprintf("arg%d", i)
		}
		params = append(params, fmt.Sprintf("%s %s", name, param.Type))
		argNames = append(argNames, name)
	}
	paramsStr := strings.Join(params, ", ")
	
	// 构建返回值类型字符串
	var results []string
	for _, result := range method.Results {
		results = append(results, result.Type)
	}
	resultsStr := ""
	if len(results) == 1 {
		// 单个返回值不需要括号
		resultsStr = results[0]
	} else if len(results) > 1 {
		resultsStr = "(" + strings.Join(results, ", ") + ")"
	}
	
	// 生成方法签名
	code := fmt.Sprintf("\n// %s 自动生成的方法实现\nfunc (d *%s) %s(%s) %s {\n", 
		method.Name, decoratedTypeName, method.Name, paramsStr, resultsStr)
	
	// 生成 Invoke 调用
	argsStr := strings.Join(argNames, ", ")
	if argsStr == "" {
		argsStr = fmt.Sprintf(`"%s"`, method.Name)
	} else {
		argsStr = fmt.Sprintf(`"%s", %s`, method.Name, argsStr)
	}
	
	// 处理错误返回：根据方法是否有非 error 返回值决定
	if method.ResultType != "" && method.ResultType != "error" {
		// 有非 error 返回值：(T, error) 或 (T)
		code += fmt.Sprintf(`	results, err := d.decorated.Invoke(%s)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no result")
	}
`, argsStr)
	} else if method.HasError {
		// 只有 error 返回值
		code += fmt.Sprintf(`	results, err := d.decorated.Invoke(%s)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return fmt.Errorf("no result")
	}
`, argsStr)
	} else {
		// 没有 error 返回值（罕见情况）
		code += fmt.Sprintf(`	results, err := d.decorated.Invoke(%s)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no result")
	}
`, argsStr)
	}
	
	// 处理返回值
	if method.ResultType != "" && method.ResultType != "error" {
		// 生成变量名：从类型名推导，如 *model.User -> user
		resultVarName := "result"
		cleanType := strings.TrimPrefix(method.ResultType, "*")
		if strings.Contains(cleanType, ".") {
			// 处理包名。类型名 的情况，如 model.User -> user
			parts := strings.Split(cleanType, ".")
			typeName := parts[len(parts)-1]
			resultVarName = strings.ToLower(string(typeName[0])) + typeName[1:]
		} else {
			resultVarName = strings.ToLower(string(cleanType[0])) + cleanType[1:]
		}
		
		code += fmt.Sprintf("	%s, ok := results[0].(%s)\n", resultVarName, method.ResultType)
		code += "	if !ok {\n"
		code += "		return nil, fmt.Errorf(\"wrong type\")\n"
		code += "	}\n"
		
		// 构建返回语句
		if method.HasError {
			code += fmt.Sprintf("	return %s, nil\n", resultVarName)
		} else {
			code += fmt.Sprintf("	return %s\n", resultVarName)
		}
	} else if method.HasError {
		// 只有 error 返回值（如 DeleteUser）
		code += "	if errResult, ok := results[0].(error); ok {\n"
		code += "		return errResult\n"
		code += "	}\n"
		code += "	return nil\n"
	}
	
	code += "}\n"
	
	return code
}

// generateCachedMethodWithAnnotation 生成带缓存注解的方法实现
func generateCachedMethodWithAnnotation(method *MethodInfo, ann *CacheAnnotation, cachedTypeName string) string {
	switch ann.Type {
	case "cacheable":
		return generateCacheableMethod(method, ann, cachedTypeName)
	case "cacheput":
		return generateCachePutMethod(method, ann, cachedTypeName)
	case "cacheevict":
		return generateCacheEvictMethod(method, ann, cachedTypeName)
	default:
		return generateSimplePassthroughMethod(method, cachedTypeName)
	}
}

// generateCacheableMethod 生成 @cacheable 方法实现
func generateCacheableMethod(method *MethodInfo, ann *CacheAnnotation, cachedTypeName string) string {
	params := buildParamList(method.Params)
	argNames := buildArgNames(method.Params)
	resultTypes := buildResultTypes(method.Results)
	firstArg := extractFirstArg(method.Params)
	
	code := fmt.Sprintf(`
// %s 带缓存的实现（@cacheable）
func (c *%s) %s(%s) %s {
	// 1. 获取缓存
	cache, err := c.manager.GetCache("%s")
	if err != nil {
		return c.decorated.%s(%s)
	}
	
	// 2. 生成缓存 Key
	key := fmt.Sprintf("%s:%%v", %s)
	
	// 3. 查询缓存
	if val, found, _ := cache.Get(context.Background(), key); found {
		return val.(%s), nil
	}
	
	// 4. 执行原始方法
	result, err := c.decorated.%s(%s)
	if err != nil {
		return nil, err
	}
	
	// 5. 写入缓存
	ttl, _ := time.ParseDuration("%s")
	_ = cache.Set(context.Background(), key, result, ttl)
	
	return result, nil
}
`, method.Name, cachedTypeName, method.Name, params, resultTypes,
		ann.CacheName, method.Name, argNames,
		ann.CacheName, firstArg,
		method.ResultType,
		method.Name, argNames,
		ann.TTL)
	
	return code
}

// generateCachePutMethod 生成 @cacheput 方法实现
func generateCachePutMethod(method *MethodInfo, ann *CacheAnnotation, cachedTypeName string) string {
	params := buildParamList(method.Params)
	argNames := buildArgNames(method.Params)
	resultTypes := buildResultTypes(method.Results)
	firstArg := extractFirstArg(method.Params)
	
	code := fmt.Sprintf(`
// %s 带缓存更新的实现（@cacheput）
func (c *%s) %s(%s) %s {
	// 1. 执行原始方法
	result, err := c.decorated.%s(%s)
	if err != nil {
		return nil, err
	}
	
	// 2. 更新缓存
	cache, err := c.manager.GetCache("%s")
	if err != nil {
		return result, nil
	}
	
	// 3. 生成缓存 Key
	key := fmt.Sprintf("%s:%%v", %s)
	
	// 4. 写入缓存
	ttl, _ := time.ParseDuration("%s")
	_ = cache.Set(context.Background(), key, result, ttl)
	
	return result, nil
}
`, method.Name, cachedTypeName, method.Name, params, resultTypes,
		method.Name, argNames,
		ann.CacheName,
		ann.CacheName, firstArg,
		ann.TTL)
	
	return code
}

// generateCacheEvictMethod 生成 @cacheevict 方法实现
func generateCacheEvictMethod(method *MethodInfo, ann *CacheAnnotation, cachedTypeName string) string {
	params := buildParamList(method.Params)
	argNames := buildArgNames(method.Params)
	resultTypes := buildResultTypes(method.Results)
	firstArg := extractFirstArg(method.Params)
	
	code := fmt.Sprintf(`
// %s 带缓存清除的实现（@cacheevict）
func (c *%s) %s(%s) %s {
	// 1. 执行原始方法
	result, err := c.decorated.%s(%s)
	if err != nil {
		return result, err
	}
	
	// 2. 清除缓存
	cache, err := c.manager.GetCache("%s")
	if err != nil {
		return result, nil
	}
	
	// 3. 生成缓存 Key
	key := fmt.Sprintf("%s:%%v", %s)
	
	// 4. 删除缓存
	_ = cache.Delete(context.Background(), key)
	
	return result, nil
}
`, method.Name, cachedTypeName, method.Name, params, resultTypes,
		method.Name, argNames,
		ann.CacheName,
		ann.CacheName, firstArg)
	
	return code
}

// generateSimplePassthroughMethod 生成简单透传方法（无缓存注解）
func generateSimplePassthroughMethod(method *MethodInfo, cachedTypeName string) string {
	params := buildParamList(method.Params)
	argNames := buildArgNames(method.Params)
	resultTypes := buildResultTypes(method.Results)
	
	code := fmt.Sprintf(`
// %s 透传方法（无缓存）
func (c *%s) %s(%s) %s {
	return c.decorated.%s(%s)
}
`, method.Name, cachedTypeName, method.Name, params, resultTypes,
		method.Name, argNames)
	
	return code
}

// buildParamList 构建参数列表字符串
func buildParamList(params []ParamInfo) string {
	var paramStrings []string
	for i, param := range params {
		name := param.Name
		if name == "_" || name == "" {
			name = fmt.Sprintf("arg%d", i)
		}
		paramStrings = append(paramStrings, fmt.Sprintf("%s %s", name, param.Type))
	}
	return strings.Join(paramStrings, ", ")
}

// buildArgNames 构建参数名称字符串（用于调用）
func buildArgNames(params []ParamInfo) string {
	var argNames []string
	for i, param := range params {
		name := param.Name
		if name == "_" || name == "" {
			name = fmt.Sprintf("arg%d", i)
		}
		argNames = append(argNames, name)
	}
	return strings.Join(argNames, ", ")
}

// buildResultTypes 构建返回值类型字符串
func buildResultTypes(results []ParamInfo) string {
	if len(results) == 0 {
		return ""
	}
	if len(results) == 1 {
		return results[0].Type
	}
	var resultTypes []string
	for _, result := range results {
		resultTypes = append(resultTypes, result.Type)
	}
	return "(" + strings.Join(resultTypes, ", ") + ")"
}

// extractFirstArg 提取第一个参数（用于生成缓存 Key）
func extractFirstArg(params []ParamInfo) string {
	if len(params) == 0 {
		return "\"default\""
	}
	name := params[0].Name
	if name == "_" || name == "" {
		return "arg0"
	}
	return name
}

func countAnnotations(annotations map[string]map[string]*CacheAnnotation) int {
	total := 0
	for _, methods := range annotations {
		total += len(methods)
	}
	return total
}
