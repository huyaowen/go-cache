package proxy

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/coderiser/go-cache/pkg/core"
	"github.com/coderiser/go-cache/pkg/spel"
)

// TestService for proxy testing
type TestService struct {
	mu   sync.Mutex
	data map[string]interface{}
}

func NewTestService() *TestService {
	return &TestService{data: make(map[string]interface{})}
}

func (s *TestService) GetData(key string) interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if val, ok := s.data[key]; ok {
		return val
	}
	return nil
}

func (s *TestService) SetData(key string, value interface{}) interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return value
}

func (s *TestService) DeleteData(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *TestService) GetWithTTL(key string, ttl int) interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if val, ok := s.data[key]; ok {
		return val
	}
	return nil
}

// TestProxyFactory tests
func TestProxyFactory(t *testing.T) {
	t.Run("NewProxyFactory", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		factory := NewProxyFactory(manager)
		if factory == nil {
			t.Fatal("Expected non-nil factory")
		}
	})

	t.Run("Create - nil target", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		factory := NewProxyFactory(manager)
		_, err := factory.Create(nil)
		if err == nil {
			t.Error("Expected error for nil target")
		}
	})

	t.Run("Create - non-pointer target", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		factory := NewProxyFactory(manager)
		_, err := factory.Create("not a pointer")
		if err == nil {
			t.Error("Expected error for non-pointer target")
		}
	})

	t.Run("Create - pointer to non-struct", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		factory := NewProxyFactory(manager)
		var str = "test"
		_, err := factory.Create(&str)
		if err == nil {
			t.Error("Expected error for pointer to non-struct")
		}
	})

	t.Run("Create - valid target", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		factory := NewProxyFactory(manager)
		service := NewTestService()

		proxy, err := factory.Create(service)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if proxy == nil {
			t.Fatal("Expected non-nil proxy")
		}
	})
}

// TestProxyImpl tests
func TestProxyImpl(t *testing.T) {
	t.Run("Call - valid method without interceptor", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		factory := NewProxyFactory(manager)
		service := NewTestService()
		service.SetData("test-key", "test-value")

		proxy, err := factory.Create(service)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Call without any annotation - should invoke original
		args := []reflect.Value{reflect.ValueOf("test-key")}
		results := proxy.Call("GetData", args)

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("Call - valid method", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		factory := NewProxyFactory(manager)
		service := NewTestService()
		service.SetData("test-key", "test-value")

		proxy, err := factory.Create(service)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		args := []reflect.Value{reflect.ValueOf("test-key")}
		results := proxy.Call("GetData", args)

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		if results[0].Interface() != "test-value" {
			t.Errorf("Expected 'test-value', got %v", results[0].Interface())
		}
	})

	t.Run("Call - non-existent method", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		factory := NewProxyFactory(manager)
		service := NewTestService()

		proxy, _ := factory.Create(service)
		results := proxy.Call("NonExistentMethod", []reflect.Value{})

		if len(results) != 0 {
			t.Errorf("Expected 0 results for non-existent method, got %d", len(results))
		}
	})

	t.Run("GetTarget", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		factory := NewProxyFactory(manager)
		service := NewTestService()

		proxy, _ := factory.Create(service)
		target := proxy.GetTarget()

		if target != service {
			t.Error("Expected target to be the same service")
		}
	})

	t.Run("GetInterceptor", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		factory := NewProxyFactory(manager)
		service := NewTestService()

		proxy, _ := factory.Create(service)
		impl, ok := proxy.(*proxyImpl)
		if !ok {
			t.Fatal("Expected proxyImpl")
		}

		interceptor := impl.GetInterceptor()
		if interceptor == nil {
			t.Fatal("Expected non-nil interceptor")
		}
	})
}

// TestMethodInterceptor tests
func TestMethodInterceptor(t *testing.T) {
	t.Run("newMethodInterceptor", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		interceptor := newMethodInterceptor(manager)
		if interceptor == nil {
			t.Fatal("Expected non-nil interceptor")
		}
	})

	t.Run("Intercept - no annotation invokes original", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		interceptor := newMethodInterceptor(manager)
		service := NewTestService()
		service.SetData("test-key", "test-value")

		args := []reflect.Value{reflect.ValueOf("test-key")}
		results := interceptor.Intercept(service, "GetData", args)

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		if results[0].Interface() != "test-value" {
			t.Errorf("Expected 'test-value', got %v", results[0].Interface())
		}
	})

	t.Run("Intercept - non-existent method", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		interceptor := newMethodInterceptor(manager)
		service := NewTestService()

		results := interceptor.Intercept(service, "NonExistent", []reflect.Value{})
		if len(results) != 0 {
			t.Errorf("Expected 0 results, got %d", len(results))
		}
	})

	t.Run("RegisterAnnotation and getAnnotation", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		interceptor := newMethodInterceptor(manager)

		annotation := &CacheAnnotation{
			Type:      "cacheable",
			CacheName: "test",
			Key:       "#id",
		}

		interceptor.RegisterAnnotation("TestMethod", annotation)

		cached := interceptor.getAnnotation("TestMethod")
		if cached != annotation {
			t.Error("Expected cached annotation to match registered one")
		}

		missing := interceptor.getAnnotation("NonExistent")
		if missing != nil {
			t.Error("Expected nil for non-existent annotation")
		}
	})

	t.Run("parseTTL", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		interceptor := newMethodInterceptor(manager)

		// Empty TTL
		ctx := spel.NewEvaluationContext()
		ttl := interceptor.parseTTL("", ctx)
		if ttl != 30*time.Minute {
			t.Errorf("Expected 30m default TTL, got %v", ttl)
		}

		// Valid TTL expression
		ctx.SetArg("ttl", int64(60))
		ttl = interceptor.parseTTL("ttl", ctx)
		if ttl != 60*time.Second {
			t.Errorf("Expected 60s TTL, got %v", ttl)
		}

		// Invalid TTL expression (should use default)
		ttl = interceptor.parseTTL("invalid_expr", ctx)
		if ttl != 30*time.Minute {
			t.Errorf("Expected 30m default TTL for invalid expr, got %v", ttl)
		}
	})

	t.Run("isTruthy", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		interceptor := newMethodInterceptor(manager)

		tests := []struct {
			value  interface{}
			expected bool
		}{
			{nil, false},
			{false, false},
			{true, true},
			{int(0), false},
			{int(1), true},
			{int(-1), true},
			{uint8(0), false},
			{uint8(1), true},
			{uint16(0), false},
			{uint16(1), true},
			{uint32(0), false},
			{uint32(1), true},
			{uint64(0), false},
			{uint64(1), true},
			{float32(0), false},
			{float32(1.5), true},
			{float64(0), false},
			{float64(1.5), true},
			{"", false},
			{"false", false},
			{"FALSE", false},
			{"true", true},
			{"anything", true},
		}

		for _, test := range tests {
			result := interceptor.isTruthy(test.value)
			if result != test.expected {
				t.Errorf("isTruthy(%v) = %v, expected %v", test.value, result, test.expected)
			}
		}
	})

	t.Run("handleCacheable - cache miss then hit", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		interceptor := newMethodInterceptor(manager)

		annotation := &CacheAnnotation{
			Type:      "cacheable",
			CacheName: "test-cache",
			Key:       "'test-key'",
			TTL:       "60",
		}
		interceptor.RegisterAnnotation("GetData", annotation)

		service := NewTestService()
		service.SetData("test-key", "cached-value")

		// First call - cache miss, should invoke original
		results := interceptor.Intercept(service, "GetData", []reflect.Value{reflect.ValueOf("test-key")})
		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		// Second call - cache hit, should return cached value
		results = interceptor.Intercept(service, "GetData", []reflect.Value{reflect.ValueOf("test-key")})
		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
		if results[0].Interface() != "cached-value" {
			t.Errorf("Expected 'cached-value', got %v", results[0].Interface())
		}
	})

	t.Run("handleCacheable - invalid cache name", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		interceptor := newMethodInterceptor(manager)

		annotation := &CacheAnnotation{
			Type:      "cacheable",
			CacheName: "non-existent-cache",
			Key:       "'test-key'",
		}
		interceptor.RegisterAnnotation("GetData", annotation)

		service := NewTestService()
		service.SetData("test-key", "original-value")

		// Should fall back to original invocation
		results := interceptor.Intercept(service, "GetData", []reflect.Value{reflect.ValueOf("test-key")})
		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("handleCacheable - condition false", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		interceptor := newMethodInterceptor(manager)

		annotation := &CacheAnnotation{
			Type:      "cacheable",
			CacheName: "test-cache",
			Key:       "'test-key'",
			Condition: "false",
		}
		interceptor.RegisterAnnotation("GetData", annotation)

		service := NewTestService()
		service.SetData("test-key", "original-value")

		// Condition is false, should invoke original
		results := interceptor.Intercept(service, "GetData", []reflect.Value{reflect.ValueOf("test-key")})
		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("handleCacheable - unless true", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		interceptor := newMethodInterceptor(manager)

		annotation := &CacheAnnotation{
			Type:      "cacheable",
			CacheName: "test-cache",
			Key:       "'test-key'",
			Unless:    "true",
		}
		interceptor.RegisterAnnotation("GetData", annotation)

		service := NewTestService()
		service.SetData("test-key", "original-value")

		// Unless is true, should not cache but return result
		results := interceptor.Intercept(service, "GetData", []reflect.Value{reflect.ValueOf("test-key")})
		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("handleCachePut", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		interceptor := newMethodInterceptor(manager)

		annotation := &CacheAnnotation{
			Type:      "cacheput",
			CacheName: "test-cache",
			Key:       "'put-key'",
			TTL:       "60",
		}
		interceptor.RegisterAnnotation("SetData", annotation)

		service := NewTestService()

		// Call should invoke original and cache result
		args := []reflect.Value{
			reflect.ValueOf("put-key"),
			reflect.ValueOf("put-value"),
		}
		results := interceptor.Intercept(service, "SetData", args)

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		// Verify value is cached
		cache, _ := manager.GetCache("test-cache")
		cached, found, _ := cache.Get(context.Background(), "put-key")
		if !found || cached != "put-value" {
			t.Error("Expected value to be cached")
		}
	})

	t.Run("handleCacheEvict - before", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		interceptor := newMethodInterceptor(manager)

		// Pre-populate cache
		cache, _ := manager.GetCache("test-cache")
		cache.Set(context.Background(), "evict-key", "cached-value", 5*time.Minute)

		annotation := &CacheAnnotation{
			Type:      "cacheevict",
			CacheName: "test-cache",
			Key:       "'evict-key'",
			Before:    true,
		}
		interceptor.RegisterAnnotation("GetData", annotation)

		service := NewTestService()

		// Should evict before invocation
		results := interceptor.Intercept(service, "GetData", []reflect.Value{reflect.ValueOf("evict-key")})
		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		// Verify cache is evicted
		_, found, _ := cache.Get(context.Background(), "evict-key")
		if found {
			t.Error("Expected cache to be evicted")
		}
	})

	t.Run("handleCacheEvict - after", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		interceptor := newMethodInterceptor(manager)

		annotation := &CacheAnnotation{
			Type:      "cacheevict",
			CacheName: "test-cache",
			Key:       "'evict-key'",
			Before:    false,
		}
		interceptor.RegisterAnnotation("GetData", annotation)

		service := NewTestService()

		// Should evict after invocation
		results := interceptor.Intercept(service, "GetData", []reflect.Value{reflect.ValueOf("evict-key")})
		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
	})
}

// TestCacheAnnotation tests
func TestCacheAnnotation(t *testing.T) {
	annotation := &CacheAnnotation{
		Type:      "cacheable",
		CacheName: "users",
		Key:       "#id",
		TTL:       "30m",
		Condition: "#result != nil",
		Unless:    "#result == nil",
		Before:    false,
		Sync:      true,
	}

	if annotation.Type != "cacheable" {
		t.Errorf("Expected type 'cacheable', got '%s'", annotation.Type)
	}
	if annotation.CacheName != "users" {
		t.Errorf("Expected cacheName 'users', got '%s'", annotation.CacheName)
	}
	if annotation.Key != "#id" {
		t.Errorf("Expected key '#id', got '%s'", annotation.Key)
	}
}

// TestAutoDecorate tests
func TestAutoDecorate(t *testing.T) {
	t.Run("NewAutoDecorate", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		decorator := NewAutoDecorate(manager)
		if decorator == nil {
			t.Fatal("Expected non-nil decorator")
		}
	})

	t.Run("GetAutoDecorate - singleton", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		d1 := GetAutoDecorate(manager)
		d2 := GetAutoDecorate(manager)

		if d1 == nil || d2 == nil {
			t.Fatal("Expected non-nil decorators")
		}
		// Note: sync.Once ensures same instance across calls
	})

	t.Run("Decorate - nil target", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		decorator := NewAutoDecorate(manager)
		err := decorator.Decorate(nil)
		if err == nil {
			t.Error("Expected error for nil target")
		}
	})

	t.Run("Decorate - non-pointer", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		decorator := NewAutoDecorate(manager)
		err := decorator.Decorate("not a pointer")
		if err == nil {
			t.Error("Expected error for non-pointer")
		}
	})

	t.Run("Decorate - valid target", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		decorator := NewAutoDecorate(manager)
		service := NewTestService()

		err := decorator.Decorate(service)
		if err != nil {
			t.Fatalf("Decorate failed: %v", err)
		}
	})

	t.Run("Register - empty name", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		decorator := NewAutoDecorate(manager)
		err := decorator.Register("", NewTestService())
		if err == nil {
			t.Error("Expected error for empty name")
		}
	})

	t.Run("Register - valid", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		decorator := NewAutoDecorate(manager)
		service := NewTestService()

		err := decorator.Register("testService", service)
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}
	})

	t.Run("Register - duplicate name", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		decorator := NewAutoDecorate(manager)
		service := NewTestService()

		err := decorator.Register("dupService", service)
		if err != nil {
			t.Fatalf("First register failed: %v", err)
		}

		err = decorator.Register("dupService", service)
		if err == nil {
			t.Error("Expected error for duplicate name")
		}
	})

	t.Run("GetProxy - non-existent", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		decorator := NewAutoDecorate(manager)
		proxy, err := decorator.GetProxy("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent proxy")
		}
		if proxy != nil {
			t.Error("Expected nil proxy for non-existent")
		}
	})

	t.Run("GetProxy - existing", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		decorator := NewAutoDecorate(manager)
		service := NewTestService()

		err := decorator.Register("testService", service)
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		proxy, err := decorator.GetProxy("testService")
		if err != nil {
			t.Fatalf("GetProxy failed: %v", err)
		}
		if proxy == nil {
			t.Fatal("Expected non-nil proxy")
		}
	})

	t.Run("GetAllProxies", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		decorator := NewAutoDecorate(manager)
		service1 := NewTestService()
		service2 := NewTestService()

		decorator.Register("service1", service1)
		decorator.Register("service2", service2)

		proxies := decorator.GetAllProxies()
		if len(proxies) != 2 {
			t.Errorf("Expected 2 proxies, got %d", len(proxies))
		}
	})

	t.Run("Interceptor", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		decorator := NewAutoDecorate(manager)
		interceptor := decorator.Interceptor()
		if interceptor == nil {
			t.Fatal("Expected non-nil interceptor")
		}
	})
}

// TestAnnotationRegistry tests
func TestAnnotationRegistry(t *testing.T) {
	t.Run("RegisterAnnotation and GetRegisteredAnnotations", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		annotation := &CacheAnnotation{
			Type:      "cacheable",
			CacheName: "test",
			Key:       "#id",
		}

		RegisterAnnotation(manager, "TestType", "TestMethod", annotation)

		annotations := GetRegisteredAnnotations("TestType")
		if annotations == nil {
			t.Fatal("Expected non-nil annotations")
		}
		if len(annotations) != 1 {
			t.Errorf("Expected 1 annotation, got %d", len(annotations))
		}
	})

	t.Run("GetRegisteredAnnotations - non-existent", func(t *testing.T) {
		annotations := GetRegisteredAnnotations("NonExistent")
		if annotations != nil {
			t.Errorf("Expected nil for non-existent type, got %v", annotations)
		}
	})
}

// TestProxyIntegration tests
func TestProxyIntegration(t *testing.T) {
	t.Run("Full proxy flow", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		factory := NewProxyFactory(manager)
		service := NewTestService()
		service.SetData("integration-key", "integration-value")

		proxy, err := factory.Create(service)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		args := []reflect.Value{reflect.ValueOf("integration-key")}
		results := proxy.Call("GetData", args)

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
		if results[0].Interface() != "integration-value" {
			t.Errorf("Expected 'integration-value', got %v", results[0].Interface())
		}
	})

	t.Run("AutoDecorate with annotation registration", func(t *testing.T) {
		manager := core.NewCacheManager()
		defer manager.Close()

		// Register annotation
		annotation := &CacheAnnotation{
			Type:      "cacheable",
			CacheName: "users",
			Key:       "#id",
			TTL:       "60",
		}
		RegisterAnnotation(manager, "TestService", "GetData", annotation)

		// Create auto decorate
		decorator := NewAutoDecorate(manager)
		service := NewTestService()
		service.SetData("user-1", "user-data")

		err := decorator.Decorate(service)
		if err != nil {
			t.Fatalf("Decorate failed: %v", err)
		}

		// Get proxy and call
		proxy, err := decorator.GetProxy("TestService")
		if err != nil {
			t.Fatalf("GetProxy failed: %v", err)
		}

		args := []reflect.Value{reflect.ValueOf("user-1")}
		results := proxy.Call("GetData", args)

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
	})
}

// Benchmarks
func BenchmarkProxyCall(b *testing.B) {
	manager := core.NewCacheManager()
	defer manager.Close()

	factory := NewProxyFactory(manager)
	service := NewTestService()
	service.SetData("key", "value")

	proxy, _ := factory.Create(service)
	args := []reflect.Value{reflect.ValueOf("key")}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = proxy.Call("GetData", args)
	}
}

func BenchmarkMethodInterceptorCall(b *testing.B) {
	manager := core.NewCacheManager()
	defer manager.Close()

	interceptor := newMethodInterceptor(manager)
	service := NewTestService()
	service.SetData("key", "value")

	args := []reflect.Value{reflect.ValueOf("key")}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = interceptor.Intercept(service, "GetData", args)
	}
}

func BenchmarkAutoDecorate(b *testing.B) {
	manager := core.NewCacheManager()
	defer manager.Close()

	decorator := NewAutoDecorate(manager)
	service := NewTestService()
	decorator.Register("service", service)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = decorator.GetProxy("service")
	}
}
