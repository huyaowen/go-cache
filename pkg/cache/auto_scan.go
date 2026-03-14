package cache

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/coderiser/go-cache/pkg/proxy"
)

var (
	autoScanOnce sync.Once
	scanPaths    = []string{"."} // 默认扫描当前目录
)

// SetAutoScanPaths 设置自动扫描的路径
// 应该在 main() 最开始调用
func SetAutoScanPaths(paths ...string) {
	scanPaths = paths
}

// AutoScanAndRegister 自动扫描并注册注解
// 在程序启动时自动调用一次
func AutoScanAndRegister() {
	autoScanOnce.Do(func() {
		for _, path := range scanPaths {
			scanDirectory(path)
		}
	})
}

// 注解正则表达式
var annotationRegex = regexp.MustCompile(`//\s*@(\w+)\s*\(([^)]+)\)`)

// scanDirectory 扫描目录中的所有 Go 文件
func scanDirectory(dir string) {
	absDir, _ := filepath.Abs(dir)
	
	filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		// 只处理 .go 文件，跳过测试文件和生成文件
		if !strings.HasSuffix(path, ".go") || 
		   strings.HasSuffix(path, "_test.go") ||
		   strings.Contains(path, ".cache-gen") ||
		   strings.Contains(path, "generated_") ||
		   strings.Contains(path, "auto_scan") {
			return nil
		}
		
		parseFile(path)
		return nil
	})
}

// parseFile 解析单个 Go 文件
func parseFile(path string) {
	fset := token.NewFileSet()
	
	// 解析文件（包含注释）
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return
	}
	
	count := 0
	// 遍历 AST 查找带注解的方法
	for _, decl := range file.Decls {
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
		annotation := parseMethodComments(fn, fset)
		if annotation == nil {
			continue
		}
		
		// 注册注解
		RegisterGlobalAnnotation(recvType, fn.Name.Name, annotation)
		count++
	}
	
	if count > 0 {
		fmt.Printf("[DEBUG] Parsed %s: found %d annotations\n", path, count)
	}
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

// parseMethodComments 解析方法前的注释（使用 AST）
func parseMethodComments(fn *ast.FuncDecl, fset *token.FileSet) *proxy.CacheAnnotation {
	// 直接使用 AST 中的注释
	if fn.Doc == nil || len(fn.Doc.List) == 0 {
		return nil
	}
	
	// 合并所有注释行
	var comments []string
	for _, comment := range fn.Doc.List {
		comments = append(comments, comment.Text)
	}
	
	comment := strings.Join(comments, "\n")
	return parseAnnotation(comment)
}

// parseAnnotation 解析注解字符串
func parseAnnotation(comment string) *proxy.CacheAnnotation {
	matches := annotationRegex.FindStringSubmatch(comment)
	if len(matches) < 3 {
		return nil
	}
	
	annotation := &proxy.CacheAnnotation{
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

// ScanDirectoryForTest 测试用扫描函数（绕过 sync.Once）
func ScanDirectoryForTest(dir string) {
	scanDirectory(dir)
}
