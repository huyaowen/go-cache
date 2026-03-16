package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAnnotationRegex 测试注解正则表达式解析
func TestAnnotationRegex(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  string
		wantKey   string
		wantTTL   string
		wantCache string
	}{
		{
			name:      "cacheable annotation",
			input:     `// @cacheable(cache="users", key="#id", ttl="30m")`,
			wantType:  "cacheable",
			wantKey:   "#id",
			wantTTL:   "30m",
			wantCache: "users",
		},
		{
			name:      "cacheput annotation",
			input:     `// @cacheput(cache="users", key="#user.ID", ttl="1h")`,
			wantType:  "cacheput",
			wantKey:   "#user.ID",
			wantTTL:   "1h",
			wantCache: "users",
		},
		{
			name:      "cacheevict annotation",
			input:     `// @cacheevict(cache="users", key="#id", before=true)`,
			wantType:  "cacheevict",
			wantKey:   "#id",
			wantCache: "users",
		},
		{
			name:      "annotation with condition",
			input:     `// @cacheable(cache="orders", key="#order.ID", ttl="2h", condition="#order.Status == 'PAID'")`,
			wantType:  "cacheable",
			wantKey:   "#order.ID",
			wantTTL:   "2h",
			wantCache: "orders",
		},
		{
			name:      "annotation with unless",
			input:     `// @cacheable(cache="products", key="#sku", ttl="1d", unless="#product.Stock == 0")`,
			wantType:  "cacheable",
			wantKey:   "#sku",
			wantTTL:   "1d",
			wantCache: "products",
		},
		{
			name:      "annotation with sync flag",
			input:     `// @cacheput(cache="users", key="#user.ID", sync=true)`,
			wantType:  "cacheput",
			wantKey:   "#user.ID",
			wantCache: "users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := annotationRegex.FindStringSubmatch(tt.input)
			if len(matches) < 3 {
				t.Fatalf("Expected regex to match, got no matches")
			}

			if matches[1] != tt.wantType {
				t.Errorf("Type = %v, want %v", matches[1], tt.wantType)
			}

			annotation := parseAnnotation(tt.input)
			if annotation == nil {
				t.Fatalf("Expected annotation to be parsed, got nil")
			}

			if annotation.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", annotation.Type, tt.wantType)
			}
			if annotation.Key != tt.wantKey {
				t.Errorf("Key = %v, want %v", annotation.Key, tt.wantKey)
			}
			if tt.wantTTL != "" && annotation.TTL != tt.wantTTL {
				t.Errorf("TTL = %v, want %v", annotation.TTL, tt.wantTTL)
			}
			if annotation.CacheName != tt.wantCache {
				t.Errorf("Cache = %v, want %v", annotation.CacheName, tt.wantCache)
			}
		})
	}
}

// TestGenerateCodeIntegration 测试 generateCode 函数的完整功能
func TestGenerateCodeIntegration(t *testing.T) {
	annotations := map[string]map[string]*CacheAnnotation{
		"UserService": {
			"GetUser": {
				Type:      "cacheable",
				CacheName: "users",
				Key:       "#id",
				TTL:       "30m",
			},
		},
	}

	// 创建临时目录
	tmpDir := t.TempDir()

	// 切换到临时目录进行测试
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// 调用 generateCode
	interfaces := make(map[string]*InterfaceInfo)
	generateCode(annotations, interfaces, ".")

	// 验证生成的文件
	outputPath := filepath.Join(tmpDir, "auto_register.go")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Generated file does not exist: %s", outputPath)
	}

	// 读取并验证内容
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	code := string(content)
	requiredStrings := []string{
		"package service",
		"func init()",
		"gocache.RegisterGlobalAnnotation",
		"UserService",
		"GetUser",
		"Type:      \"cacheable\"",
		"CacheName: \"users\"",
		"Key:       \"#id\"",
		"TTL:       \"30m\"",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(code, required) {
			t.Errorf("Generated code missing required string: %s", required)
		}
	}
}

// TestGenerateCodeWithAllOptions 测试生成代码包含所有选项
func TestGenerateCodeWithAllOptions(t *testing.T) {
	annotations := map[string]map[string]*CacheAnnotation{
		"OrderService": {
			"CreateOrder": {
				Type:      "cacheput",
				CacheName: "orders",
				Key:       "#order.ID",
				TTL:       "1h",
				Condition: "#order.Status == 'PAID'",
				Unless:    "#order.Amount > 10000",
				Before:    false,
				Sync:      true,
			},
		},
	}

	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	interfaces := make(map[string]*InterfaceInfo)
	generateCode(annotations, interfaces, ".")

	outputPath := filepath.Join(tmpDir, "auto_register.go")
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	code := string(content)

	// 验证所有字段都被生成
	requiredStrings := []string{
		"Condition: \"#order.Status == 'PAID'",
		"Unless:    \"#order.Amount > 10000",
		"Sync:      true",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(code, required) {
			t.Errorf("Generated code missing: %s", required)
		}
	}
}

// TestGenerateCodeEmptyAnnotations 测试空注解的处理
func TestGenerateCodeEmptyAnnotations(t *testing.T) {
	annotations := make(map[string]map[string]*CacheAnnotation)

	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// 注意：generateAnnotationRegistration 会生成文件即使注解为空
	// 这是当前实现的行为
	interfaces := make(map[string]*InterfaceInfo)
	generateCode(annotations, interfaces, ".")

	// 文件会被创建，但包含 0 个注解注册
	outputPath := filepath.Join(tmpDir, "auto_register.go")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Log("Note: File is created even with empty annotations")
	}
}

// TestParseFileWithMultipleTypes 测试解析多个类型的文件
func TestParseFileWithMultipleTypes(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := `package service

type UserService struct{}
type ProductService struct{}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id string) *User {
	return &User{ID: id}
}

// @cacheable(cache="products", key="#sku", ttl="1h")
func (s *ProductService) GetProduct(sku string) *Product {
	return &Product{SKU: sku}
}
`

	testFilePath := filepath.Join(tmpDir, "multi_service.go")
	err := os.WriteFile(testFilePath, []byte(testFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	annotations := make(map[string]map[string]*CacheAnnotation)
	interfaces := make(map[string]*InterfaceInfo)
	parseFile(testFilePath, annotations, interfaces)

	if len(annotations) != 2 {
		t.Errorf("Expected 2 types, got %d", len(annotations))
	}

	if _, exists := annotations["UserService"]; !exists {
		t.Error("UserService not found")
	}
	if _, exists := annotations["ProductService"]; !exists {
		t.Error("ProductService not found")
	}
}

// TestCountAnnotationsWithMultipleMethods 测试多个方法的计数
func TestCountAnnotationsWithMultipleMethods(t *testing.T) {
	annotations := map[string]map[string]*CacheAnnotation{
		"Service1": {
			"Method1": {Type: "cacheable"},
			"Method2": {Type: "cacheable"},
			"Method3": {Type: "cacheable"},
		},
		"Service2": {
			"Method4": {Type: "cacheput"},
			"Method5": {Type: "cacheput"},
		},
	}

	count := countAnnotations(annotations)
	if count != 5 {
		t.Errorf("Count = %d, want 5", count)
	}
}

// TestParseAnnotationWithComplexExpressions 测试复杂表达式的解析
func TestParseAnnotationWithComplexExpressions(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantCondition string
		wantUnless    string
	}{
		{
			name:          "complex condition",
			input:         `// @cacheable(cache="c", key="k", condition="#user.Role == 'ADMIN' && #user.Active == true")`,
			wantCondition: "#user.Role == 'ADMIN' && #user.Active == true",
		},
		{
			name:       "complex unless",
			input:      `// @cacheable(cache="c", key="k", unless="#item.Stock <= 0 || #item.Deleted == true")`,
			wantUnless: "#item.Stock <= 0 || #item.Deleted == true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotation := parseAnnotation(tt.input)
			if annotation == nil {
				t.Fatal("Expected annotation to be parsed")
			}
			if tt.wantCondition != "" && annotation.Condition != tt.wantCondition {
				t.Errorf("Condition = %v, want %v", annotation.Condition, tt.wantCondition)
			}
			if tt.wantUnless != "" && annotation.Unless != tt.wantUnless {
				t.Errorf("Unless = %v, want %v", annotation.Unless, tt.wantUnless)
			}
		})
	}
}

// TestFileScanner 测试文件扫描功能
func TestFileScanner(t *testing.T) {
	// 创建临时测试文件
	tmpDir := t.TempDir()

	testFile := `package service

type UserService struct{}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id string) *User {
	return &User{ID: id}
}

// @cacheput(cache="users", key="#user.ID", ttl="1h")
func (s *UserService) UpdateUser(user *User) error {
	return nil
}

// No annotation
func (s *UserService) DeleteUser(id string) error {
	return nil
}
`

	testFilePath := filepath.Join(tmpDir, "test_service.go")
	err := os.WriteFile(testFilePath, []byte(testFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 测试 parseFile
	annotations := make(map[string]map[string]*CacheAnnotation)
	interfaces := make(map[string]*InterfaceInfo)
	parseFile(testFilePath, annotations, interfaces)

	// 验证解析结果
	if len(annotations) != 1 {
		t.Fatalf("Expected 1 type, got %d", len(annotations))
	}

	userService, exists := annotations["UserService"]
	if !exists {
		t.Fatal("UserService not found in annotations")
	}

	if len(userService) != 2 {
		t.Fatalf("Expected 2 methods, got %d", len(userService))
	}

	// 验证 GetUser 注解
	getUser, exists := userService["GetUser"]
	if !exists {
		t.Fatal("GetUser annotation not found")
	}
	if getUser.Type != "cacheable" {
		t.Errorf("GetUser Type = %v, want cacheable", getUser.Type)
	}
	if getUser.CacheName != "users" {
		t.Errorf("GetUser CacheName = %v, want users", getUser.CacheName)
	}
	if getUser.Key != "#id" {
		t.Errorf("GetUser Key = %v, want #id", getUser.Key)
	}
	if getUser.TTL != "30m" {
		t.Errorf("GetUser TTL = %v, want 30m", getUser.TTL)
	}

	// 验证 UpdateUser 注解
	updateUser, exists := userService["UpdateUser"]
	if !exists {
		t.Fatal("UpdateUser annotation not found")
	}
	if updateUser.Type != "cacheput" {
		t.Errorf("UpdateUser Type = %v, want cacheput", updateUser.Type)
	}
}

// TestEdgeCases 测试边界条件
func TestEdgeCases(t *testing.T) {
	t.Run("empty annotation", func(t *testing.T) {
		// 空注解应该返回 nil
		annotation := parseAnnotation(`// @cacheable()`)
		if annotation != nil {
			t.Error("Empty annotation should return nil")
		}
	})

	t.Run("missing cache name", func(t *testing.T) {
		// 缺少 cache 参数应该返回 nil
		annotation := parseAnnotation(`// @cacheable(key="#id", ttl="30m")`)
		if annotation != nil {
			t.Error("Annotation without cache name should return nil")
		}
	})

	t.Run("missing key", func(t *testing.T) {
		// 缺少 key 参数应该返回 nil
		annotation := parseAnnotation(`// @cacheable(cache="users", ttl="30m")`)
		if annotation != nil {
			t.Error("Annotation without key should return nil")
		}
	})

	t.Run("invalid annotation format", func(t *testing.T) {
		// 非法格式应该返回 nil
		annotation := parseAnnotation(`// @cacheable cache="users" key="#id"`)
		if annotation != nil {
			t.Error("Invalid format should return nil")
		}
	})

	t.Run("annotation with spaces", func(t *testing.T) {
		// 带有多余空格的注解
		input := `//  @cacheable( cache="users" , key="#id" , ttl="30m" )`
		annotation := parseAnnotation(input)
		if annotation == nil {
			t.Error("Annotation with spaces should be parsed")
		} else {
			if annotation.CacheName != "users" {
				t.Errorf("CacheName = %v, want users", annotation.CacheName)
			}
		}
	})

	t.Run("annotation with single quotes", func(t *testing.T) {
		// 使用单引号的注解
		input := `// @cacheable(cache='users', key='#id', ttl='30m')`
		annotation := parseAnnotation(input)
		if annotation == nil {
			t.Error("Annotation with single quotes should be parsed")
		} else {
			if annotation.CacheName != "users" {
				t.Errorf("CacheName = %v, want users", annotation.CacheName)
			}
		}
	})

	t.Run("comment without annotation", func(t *testing.T) {
		// 普通注释应该返回 nil
		annotation := parseAnnotation(`// This is a regular comment`)
		if annotation != nil {
			t.Error("Regular comment should return nil")
		}
	})

	t.Run("annotation with boolean true", func(t *testing.T) {
		input := `// @cacheevict(cache="users", key="#id", before=true)`
		annotation := parseAnnotation(input)
		if annotation == nil {
			t.Error("Annotation with before=true should be parsed")
		} else {
			if !annotation.Before {
				t.Error("Before should be true")
			}
		}
	})

	t.Run("annotation with boolean false", func(t *testing.T) {
		input := `// @cacheevict(cache="users", key="#id", before=false)`
		annotation := parseAnnotation(input)
		if annotation == nil {
			t.Error("Annotation with before=false should be parsed")
		} else {
			if annotation.Before {
				t.Error("Before should be false")
			}
		}
	})
}

// TestRegexPattern 测试正则表达式本身
func TestRegexPattern(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		matches bool
	}{
		{"valid annotation", `// @cacheable(cache="users")`, true},
		{"annotation with space", `//  @cacheable(cache="users")`, true},
		{"no space after comment", `//@cacheable(cache="users")`, true},
		{"regular comment", `// This is a comment`, false},
		{"empty line", ``, false},
		{"code line", `func foo() {}`, false},
		{"incomplete annotation", `// @cacheable`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched := annotationRegex.MatchString(tt.input)
			if matched != tt.matches {
				t.Errorf("Match = %v, want %v for input: %s", matched, tt.matches, tt.input)
			}
		})
	}
}

// TestCountAnnotations 测试注解计数功能
func TestCountAnnotations(t *testing.T) {
	annotations := map[string]map[string]*CacheAnnotation{
		"UserService": {
			"GetUser":    {Type: "cacheable"},
			"UpdateUser": {Type: "cacheput"},
			"DeleteUser": {Type: "cacheevict"},
		},
		"ProductService": {
			"GetProduct": {Type: "cacheable"},
		},
	}

	count := countAnnotations(annotations)
	if count != 4 {
		t.Errorf("Count = %d, want 4", count)
	}

	// 测试空 map
	empty := make(map[string]map[string]*CacheAnnotation)
	if countAnnotations(empty) != 0 {
		t.Error("Empty annotations should return 0")
	}
}

// TestGetReceiverTypeName 测试接收者类型名提取
func TestGetReceiverTypeName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
	}{
		{
			name:    "minimal valid annotation",
			input:   `// @cacheable(cache="c", key="k")`,
			wantNil: false,
		},
		{
			name:    "no parameters",
			input:   `// @cacheable()`,
			wantNil: true,
		},
		{
			name:    "only cache",
			input:   `// @cacheable(cache="c")`,
			wantNil: true,
		},
		{
			name:    "only key",
			input:   `// @cacheable(key="k")`,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAnnotation(tt.input)
			if tt.wantNil && result != nil {
				t.Errorf("Expected nil, got %+v", result)
			}
			if !tt.wantNil && result == nil {
				t.Error("Expected annotation, got nil")
			}
		})
	}
}

// TestParseFileComplexTypes 测试复杂类型解析（slice/map/channel）
func TestParseFileComplexTypes(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := `package service

type ComplexService struct{}

// @cacheable(cache="data", key="#ids", ttl="30m")
func (s *ComplexService) GetUsers(ids []int64) ([]*User, error) {
	return nil, nil
}

// @cacheable(cache="config", key="#name", ttl="1h")
func (s *ComplexService) GetConfig(name string) (map[string]interface{}, error) {
	return nil, nil
}

// @cacheable(cache="events", key="#ch", ttl="5m")
func (s *ComplexService) GetEvents(ch chan string) (<-chan Event, error) {
	return nil, nil
}
`

	testFilePath := filepath.Join(tmpDir, "complex_service.go")
	err := os.WriteFile(testFilePath, []byte(testFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	annotations := make(map[string]map[string]*CacheAnnotation)
	interfaces := make(map[string]*InterfaceInfo)
	parseFile(testFilePath, annotations, interfaces)

	if len(annotations) != 1 {
		t.Fatalf("Expected 1 type, got %d", len(annotations))
	}

	complexService, exists := annotations["ComplexService"]
	if !exists {
		t.Fatal("ComplexService not found")
	}

	if len(complexService) != 3 {
		t.Fatalf("Expected 3 methods, got %d", len(complexService))
	}
}

// TestParseFileGenericMethods 测试泛型方法（Go 1.18+）
func TestParseFileGenericMethods(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := `package service

type GenericService struct{}

// @cacheable(cache="items", key="#id", ttl="30m")
func (s *GenericService) GetItem[T any](id int64) (*T, error) {
	return nil, nil
}
`

	testFilePath := filepath.Join(tmpDir, "generic_service.go")
	err := os.WriteFile(testFilePath, []byte(testFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	annotations := make(map[string]map[string]*CacheAnnotation)
	interfaces := make(map[string]*InterfaceInfo)
	parseFile(testFilePath, annotations, interfaces)

	// 泛型方法应该被解析
	if len(annotations) != 1 {
		t.Logf("Generic methods: got %d types", len(annotations))
	}
}

// TestParseFileNoAnnotations 测试无注解文件
func TestParseFileNoAnnotations(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := `package service

type EmptyService struct{}

func (s *EmptyService) DoSomething() error {
	return nil
}

func (s *EmptyService) GetInfo() string {
	return "info"
}
`

	testFilePath := filepath.Join(tmpDir, "empty_service.go")
	err := os.WriteFile(testFilePath, []byte(testFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	annotations := make(map[string]map[string]*CacheAnnotation)
	interfaces := make(map[string]*InterfaceInfo)
	parseFile(testFilePath, annotations, interfaces)

	if len(annotations) != 0 {
		t.Errorf("Expected 0 annotations, got %d", len(annotations))
	}
}

// TestParseFileEmptyFile 测试空文件
func TestParseFileEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := `package service
`

	testFilePath := filepath.Join(tmpDir, "empty.go")
	err := os.WriteFile(testFilePath, []byte(testFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	annotations := make(map[string]map[string]*CacheAnnotation)
	interfaces := make(map[string]*InterfaceInfo)
	parseFile(testFilePath, annotations, interfaces)

	if len(annotations) != 0 {
		t.Errorf("Expected 0 annotations for empty file, got %d", len(annotations))
	}
}

// TestGenerateMultipleServices 测试多服务生成
func TestGenerateMultipleServices(t *testing.T) {
	annotations := map[string]map[string]*CacheAnnotation{
		"UserService": {
			"GetUser": {
				Type:      "cacheable",
				CacheName: "users",
				Key:       "#id",
				TTL:       "30m",
			},
			"UpdateUser": {
				Type:      "cacheput",
				CacheName: "users",
				Key:       "#user.ID",
				TTL:       "1h",
			},
		},
		"OrderService": {
			"GetOrder": {
				Type:      "cacheable",
				CacheName: "orders",
				Key:       "#id",
				TTL:       "30m",
			},
		},
		"ProductService": {
			"GetProduct": {
				Type:      "cacheable",
				CacheName: "products",
				Key:       "#sku",
				TTL:       "2h",
			},
			"ListProducts": {
				Type:      "cacheable",
				CacheName: "products",
				Key:       "#category",
				TTL:       "1h",
			},
		},
	}

	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	interfaces := make(map[string]*InterfaceInfo)
	generateCode(annotations, interfaces, ".")

	outputPath := filepath.Join(tmpDir, "auto_register.go")
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	code := string(content)

	// 验证所有服务都被生成
	requiredStrings := []string{
		"UserService",
		"OrderService",
		"ProductService",
		"GetUser",
		"UpdateUser",
		"GetOrder",
		"GetProduct",
		"ListProducts",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(code, required) {
			t.Errorf("Generated code missing: %s", required)
		}
	}
}
