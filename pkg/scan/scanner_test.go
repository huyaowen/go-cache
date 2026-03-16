package scan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectModulePath(t *testing.T) {
	path, err := DetectModulePath()
	if err != nil {
		t.Fatalf("DetectModulePath failed: %v", err)
	}
	if path == "" {
		t.Error("Expected non-empty module path")
	}
	t.Logf("Module path: %s", path)
}

func TestMatcher(t *testing.T) {
	matcher := NewMatcher()

	tests := []struct {
		name         string
		ifaceName    string
		expectedImpl string
	}{
		{"UserServiceInterface", "UserServiceInterface", "userService"},
		{"OrderService", "OrderService", "orderService"},
		{"UserRepository", "UserRepository", "userRepository"},
		{"CacheInterface", "CacheInterface", "cache"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.expectedImplName(tt.ifaceName)
			if result != tt.expectedImpl {
				t.Errorf("expectedImplName(%s) = %s, want %s", tt.ifaceName, result, tt.expectedImpl)
			}
		})
	}
}

func TestMatcherMatch(t *testing.T) {
	matcher := NewMatcher()

	interfaces := map[string]*InterfaceInfo{
		"UserServiceInterface": {
			Name: "UserServiceInterface",
			Methods: []*MethodSpec{
				{Name: "GetUser", Params: []*ParamSpec{{Name: "id", Type: "int64"}}, Returns: []*ParamSpec{{Type: "*User"}, {Type: "error"}}},
			},
		},
	}

	services := map[string]*ServiceInfo{
		"userService": {
			TypeName: "userService",
			Methods: map[string]*MethodInfo{
				"GetUser": {
					Name:      "GetUser",
					Operation: "cacheable",
					Cache:     "users",
					Key:       "#id",
					Params:    []*ParamSpec{{Name: "id", Type: "int64"}},
					Returns:   []*ParamSpec{{Type: "*User"}, {Type: "error"}},
				},
			},
		},
	}

	results := matcher.Match(interfaces, services)

	if len(results) != 1 {
		t.Fatalf("Expected 1 match, got %d", len(results))
	}

	result, exists := results["UserServiceInterface"]
	if !exists {
		t.Fatal("Expected UserServiceInterface match")
	}

	if result.Service.TypeName != "userService" {
		t.Errorf("Expected userService, got %s", result.Service.TypeName)
	}
}

func TestParseAnnotation(t *testing.T) {
	scanner := NewScanner(&Config{})

	tests := []struct {
		name     string
		comment  string
		expected *AnnotationInfo
	}{
		{
			name:    "cacheable",
			comment: "// @cacheable(cache=\"users\", key=\"#id\", ttl=\"30m\")",
			expected: &AnnotationInfo{
				Type:      "cacheable",
				CacheName: "users",
				Key:       "#id",
				TTL:       "30m",
			},
		},
		{
			name:    "cacheput",
			comment: "// @cacheput(cache=\"users\", key=\"#result.ID\", ttl=\"1h\")",
			expected: &AnnotationInfo{
				Type:      "cacheput",
				CacheName: "users",
				Key:       "#result.ID",
				TTL:       "1h",
			},
		},
		{
			name:    "cacheevict",
			comment: "// @cacheevict(cache=\"users\", key=\"#id\", before=true)",
			expected: &AnnotationInfo{
				Type:      "cacheevict",
				CacheName: "users",
				Key:       "#id",
				Before:    true,
			},
		},
		{
			name:     "no annotation",
			comment:  "// regular comment",
			expected: nil,
		},
		{
			name:     "missing cache",
			comment:  "// @cacheable(key=\"#id\")",
			expected: nil,
		},
		{
			name:     "missing key",
			comment:  "// @cacheable(cache=\"users\")",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scanner.parseAnnotation(tt.comment)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("Expected annotation, got nil")
			}

			if result.Type != tt.expected.Type {
				t.Errorf("Type = %s, want %s", result.Type, tt.expected.Type)
			}
			if result.CacheName != tt.expected.CacheName {
				t.Errorf("CacheName = %s, want %s", result.CacheName, tt.expected.CacheName)
			}
			if result.Key != tt.expected.Key {
				t.Errorf("Key = %s, want %s", result.Key, tt.expected.Key)
			}
			if result.TTL != tt.expected.TTL {
				t.Errorf("TTL = %s, want %s", result.TTL, tt.expected.TTL)
			}
			if result.Before != tt.expected.Before {
				t.Errorf("Before = %v, want %v", result.Before, tt.expected.Before)
			}
		})
	}
}

func TestParseAnnotationParams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:  "simple",
			input: `cache="users", key="#id"`,
			expected: map[string]string{
				"cache": "users",
				"key":   "#id",
			},
		},
		{
			name:  "with spaces",
			input: `cache = "users" , key = "#id"`,
			expected: map[string]string{
				"cache": "users",
				"key":   "#id",
			},
		},
		{
			name:  "with condition",
			input: `cache="orders", key="#order.ID", condition="#order.Status == 'PAID'"`,
			expected: map[string]string{
				"cache":     "orders",
				"key":       "#order.ID",
				"condition": "#order.Status == 'PAID'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAnnotationParams(tt.input)

			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("params[%s] = %s, want %s", k, result[k], v)
				}
			}
		})
	}
}

func TestScanner(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := `package service

type User struct {
	ID   int64
	Name string
}

type UserServiceInterface interface {
	GetUser(id int64) (*User, error)
	CreateUser(name string, email string) (*User, error)
	DeleteUser(id int64) error
}

type userService struct {
	db string
}

func NewUserService(db string) *userService {
	return &userService{db: db}
}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *userService) GetUser(id int64) (*User, error) {
	return &User{ID: id}, nil
}

// @cacheput(cache="users", key="#result.ID", ttl="30m")
func (s *userService) CreateUser(name string, email string) (*User, error) {
	return &User{ID: 1, Name: name}, nil
}

// @cacheevict(cache="users", key="#id")
func (s *userService) DeleteUser(id int64) error {
	return nil
}
`

	testPath := filepath.Join(tmpDir, "user.go")
	if err := os.WriteFile(testPath, []byte(testFile), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config := &Config{
		Dirs:    []string{tmpDir},
		Force:   true,
		Verbose: true,
	}

	scanner := NewScanner(config)
	result, err := scanner.Scan([]string{tmpDir})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(result.Packages) == 0 {
		t.Fatal("Expected at least one package")
	}

	for _, pkg := range result.Packages {
		for _, file := range pkg.Files {
			if len(file.Interfaces) == 0 {
				t.Error("Expected at least one interface")
			}
			if len(file.Services) == 0 {
				t.Error("Expected at least one service")
			}

			svc, exists := file.Services["userService"]
			if !exists {
				t.Fatal("Expected userService")
			}

			if len(svc.Methods) != 3 {
				t.Errorf("Expected 3 methods, got %d", len(svc.Methods))
			}

			for name, method := range svc.Methods {
				t.Logf("Method: %s, Operation: %s, Cache: %s", name, method.Operation, method.Cache)
			}
		}
	}
}

func TestGenerator(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := `package service

type User struct {
	ID   int64
	Name string
}

type UserServiceInterface interface {
	GetUser(id int64) (*User, error)
}

type userService struct {
	db string
}

func NewUserService(db string) *userService {
	return &userService{db: db}
}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *userService) GetUser(id int64) (*User, error) {
	return &User{ID: id}, nil
}
`

	testPath := filepath.Join(tmpDir, "user.go")
	if err := os.WriteFile(testPath, []byte(testFile), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config := &Config{
		Dirs:    []string{tmpDir},
		Force:   true,
		Verbose: true,
	}

	scanner := NewScanner(config)
	result, err := scanner.Scan([]string{tmpDir})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	generator := NewGenerator(config)
	if err := generator.Generate(result); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	outputPath := filepath.Join(tmpDir, "user_cached.go")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Generated file not found: %s", outputPath)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	requiredStrings := []string{
		"package service",
		"type cachedUserService struct",
		"var UserService UserServiceInterface",
		"func NewCachedUserService",
		"func InitUserService",
		"func (c *cachedUserService) GetUser",
		"GetGlobalManager",
		"spel.NewSpELEvaluator",
	}

	for _, s := range requiredStrings {
		if !contains(contentStr, s) {
			t.Errorf("Generated file missing: %s", s)
		}
	}

	t.Logf("Generated file:\n%s", contentStr[:500])
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
