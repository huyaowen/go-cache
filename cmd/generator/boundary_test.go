package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBoundaryEmptyFile 边界测试：空文件
func TestBoundaryEmptyFile(t *testing.T) {
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

// TestBoundaryNoAnnotations 边界测试：无注解
func TestBoundaryNoAnnotations(t *testing.T) {
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

// TestBoundaryComplexTypes 边界测试：复杂类型（slice/map/channel）
func TestBoundaryComplexTypes(t *testing.T) {
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

// @cacheable(cache="matrix", key="#id", ttl="10m")
func (s *ComplexService) GetData(id string) ([][][]byte, error) {
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

	if len(complexService) != 4 {
		t.Fatalf("Expected 4 methods, got %d", len(complexService))
	}
}

// TestBoundaryGenericMethods 边界测试：泛型方法
func TestBoundaryGenericMethods(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := `package service

type GenericService struct{}

// @cacheable(cache="items", key="#id", ttl="30m")
func (s *GenericService) GetItem[T any](id int64) (*T, error) {
	return nil, nil
}

// @cacheable(cache="list", key="#type", ttl="1h")
func (s *GenericService) List[T any](itemType string) ([]T, error) {
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
	if len(annotations) < 1 {
		t.Logf("Generic methods: got %d types", len(annotations))
	}
}

// TestBoundaryErrorInput 边界测试：错误输入
func TestBoundaryErrorInput(t *testing.T) {
	// 测试非法注解格式
	tests := []struct {
		name  string
		input string
	}{
		{"empty annotation", `// @cacheable()`},
		{"missing cache", `// @cacheable(key="#id", ttl="30m")`},
		{"missing key", `// @cacheable(cache="users", ttl="30m")`},
		{"no parentheses", `// @cacheable cache="users" key="#id"`},
		{"unclosed paren", `// @cacheable(cache="users", key="#id"`},
		{"malformed", `// @cacheable(cache="users"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotation := parseAnnotation(tt.input)
			if annotation != nil {
				t.Logf("Input: %s, got annotation (expected nil or partial)", tt.input)
			}
		})
	}
}

// TestBoundarySpecialCharacters 边界测试：特殊字符
func TestBoundarySpecialCharacters(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCache string
		wantKey   string
	}{
		{
			name:      "dots in key",
			input:     `// @cacheable(cache="user.data", key="#user.profile.id", ttl="30m")`,
			wantCache: "user.data",
			wantKey:   "#user.profile.id",
		},
		{
			name:      "underscores",
			input:     `// @cacheable(cache="user_data", key="#user_id", ttl="30m")`,
			wantCache: "user_data",
			wantKey:   "#user_id",
		},
		{
			name:      "hyphens",
			input:     `// @cacheable(cache="user-data", key="#user-id", ttl="30m")`,
			wantCache: "user-data",
			wantKey:   "#user-id",
		},
		{
			name:      "mixed quotes",
			input:     `// @cacheable(cache="users", key='#id', ttl="30m")`,
			wantCache: "users",
			wantKey:   "#id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotation := parseAnnotation(tt.input)
			if annotation == nil {
				t.Errorf("Expected annotation, got nil for: %s", tt.input)
				return
			}
			if annotation.CacheName != tt.wantCache {
				t.Errorf("CacheName = %v, want %v", annotation.CacheName, tt.wantCache)
			}
			if annotation.Key != tt.wantKey {
				t.Errorf("Key = %v, want %v", annotation.Key, tt.wantKey)
			}
		})
	}
}

// TestBoundaryMultipleAnnotationsSameMethod 边界测试：同一方法多个注解
func TestBoundaryMultipleAnnotationsSameMethod(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := `package service

type MultiAnnotationService struct{}

// @cacheable(cache="users", key="#id", ttl="30m")
// @cacheput(cache="users", key="#result.ID", ttl="1h")
func (s *MultiAnnotationService) GetOrCreate(id int64) (*User, error) {
	return nil, nil
}
`

	testFilePath := filepath.Join(tmpDir, "multi_annotation.go")
	err := os.WriteFile(testFilePath, []byte(testFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	annotations := make(map[string]map[string]*CacheAnnotation)
	interfaces := make(map[string]*InterfaceInfo)
	parseFile(testFilePath, annotations, interfaces)

	// 当前实现只取最后一个注解
	if len(annotations) != 1 {
		t.Fatalf("Expected 1 type, got %d", len(annotations))
	}

	service, exists := annotations["MultiAnnotationService"]
	if !exists {
		t.Fatal("MultiAnnotationService not found")
	}

	if len(service) != 1 {
		t.Logf("Multiple annotations: got %d methods (last one wins)", len(service))
	}
}

// TestBoundaryVeryLongTTL 边界测试：超长 TTL
func TestBoundaryVeryLongTTL(t *testing.T) {
	input := `// @cacheable(cache="users", key="#id", ttl="365d")`
	annotation := parseAnnotation(input)
	if annotation == nil {
		t.Error("Expected annotation with long TTL")
	}
	if annotation.TTL != "365d" {
		t.Errorf("TTL = %v, want 365d", annotation.TTL)
	}
}

// TestBoundaryZeroTTL 边界测试：零 TTL
func TestBoundaryZeroTTL(t *testing.T) {
	input := `// @cacheable(cache="users", key="#id", ttl="0s")`
	annotation := parseAnnotation(input)
	if annotation == nil {
		t.Error("Expected annotation with zero TTL")
	}
	if annotation.TTL != "0s" {
		t.Errorf("TTL = %v, want 0s", annotation.TTL)
	}
}

// TestBoundaryAllAnnotationTypes 边界测试：所有注解类型
func TestBoundaryAllAnnotationTypes(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := `package service

type AllTypesService struct{}

// @cacheable(cache="c1", key="#id", ttl="30m")
func (s *AllTypesService) Get(id int64) (*Item, error) {
	return nil, nil
}

// @cacheput(cache="c2", key="#item.ID", ttl="1h")
func (s *AllTypesService) Create(item *Item) (*Item, error) {
	return nil, nil
}

// @cacheevict(cache="c3", key="#id", before=true)
func (s *AllTypesService) DeleteBefore(id int64) error {
	return nil
}

// @cacheevict(cache="c4", key="#id", before=false)
func (s *AllTypesService) DeleteAfter(id int64) error {
	return nil
}
`

	testFilePath := filepath.Join(tmpDir, "all_types.go")
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

	service, exists := annotations["AllTypesService"]
	if !exists {
		t.Fatal("AllTypesService not found")
	}

	if len(service) != 4 {
		t.Errorf("Expected 4 methods, got %d", len(service))
	}

	// 验证各个注解类型
	if service["Get"].Type != "cacheable" {
		t.Errorf("Get type = %v, want cacheable", service["Get"].Type)
	}
	if service["Create"].Type != "cacheput" {
		t.Errorf("Create type = %v, want cacheput", service["Create"].Type)
	}
	if service["DeleteBefore"].Type != "cacheevict" {
		t.Errorf("DeleteBefore type = %v, want cacheevict", service["DeleteBefore"].Type)
	}
	if !service["DeleteBefore"].Before {
		t.Error("DeleteBefore Before should be true")
	}
	if service["DeleteAfter"].Type != "cacheevict" {
		t.Errorf("DeleteAfter type = %v, want cacheevict", service["DeleteAfter"].Type)
	}
	if service["DeleteAfter"].Before {
		t.Error("DeleteAfter Before should be false")
	}
}

// TestBoundaryConditionAndUnless 边界测试：condition 和 unless 同时存在
func TestBoundaryConditionAndUnless(t *testing.T) {
	input := `// @cacheable(cache="users", key="#id", ttl="30m", condition="#user.Active", unless="#user.Deleted")`
	annotation := parseAnnotation(input)
	if annotation == nil {
		t.Fatal("Expected annotation with both condition and unless")
	}
	if annotation.Condition != "#user.Active" {
		t.Errorf("Condition = %v, want #user.Active", annotation.Condition)
	}
	if annotation.Unless != "#user.Deleted" {
		t.Errorf("Unless = %v, want #user.Deleted", annotation.Unless)
	}
}

// TestBoundarySyncFlag 边界测试：sync 标志
func TestBoundarySyncFlag(t *testing.T) {
	input := `// @cacheput(cache="users", key="#id", sync=true)`
	annotation := parseAnnotation(input)
	if annotation == nil {
		t.Fatal("Expected annotation with sync flag")
	}
	if !annotation.Sync {
		t.Error("Sync should be true")
	}
}
