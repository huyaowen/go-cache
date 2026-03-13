package proxy

import (
	"context"
	"reflect"
	"testing"
)

func TestAnnotationHandlerRegistry_Basic(t *testing.T) {
	registry := NewAnnotationHandlerRegistry()

	// Create a test handler
	handler := &TestHandler{BaseAnnotationHandler: BaseAnnotationHandler{Priority: 10}}

	// Register
	err := registry.Register("test", handler)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Get
	retrieved, exists := registry.GetHandler("test")
	if !exists {
		t.Fatal("Expected handler to exist")
	}
	if retrieved.GetPriority() != 10 {
		t.Errorf("Expected priority 10, got %d", retrieved.GetPriority())
	}

	// HasHandler
	if !registry.HasHandler("test") {
		t.Fatal("Expected HasHandler to return true")
	}

	// List
	handlers := registry.ListHandlers()
	if len(handlers) != 1 {
		t.Errorf("Expected 1 handler, got %d", len(handlers))
	}
	if handlers[0] != "test" {
		t.Errorf("Expected 'test', got '%s'", handlers[0])
	}
}

func TestAnnotationHandlerRegistry_Unregister(t *testing.T) {
	registry := NewAnnotationHandlerRegistry()

	// Register and unregister
	_ = registry.Register("test", &TestHandler{BaseAnnotationHandler: BaseAnnotationHandler{Priority: 10}})
	err := registry.Unregister("test")
	if err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	// Verify
	_, exists := registry.GetHandler("test")
	if exists {
		t.Fatal("Expected handler to be unregistered")
	}

	if registry.HasHandler("test") {
		t.Fatal("Expected HasHandler to return false")
	}
}

func TestAnnotationHandlerRegistry_Duplicate(t *testing.T) {
	registry := NewAnnotationHandlerRegistry()

	// Register twice should fail
	_ = registry.Register("test", &TestHandler{BaseAnnotationHandler: BaseAnnotationHandler{Priority: 10}})
	err := registry.Register("test", &TestHandler{BaseAnnotationHandler: BaseAnnotationHandler{Priority: 20}})
	if err == nil {
		t.Fatal("Expected error for duplicate registration")
	}
}

func TestAnnotationHandlerRegistry_EmptyName(t *testing.T) {
	registry := NewAnnotationHandlerRegistry()

	err := registry.Register("", &TestHandler{BaseAnnotationHandler: BaseAnnotationHandler{Priority: 10}})
	if err == nil {
		t.Fatal("Expected error for empty name")
	}
}

func TestAnnotationHandlerRegistry_NilHandler(t *testing.T) {
	registry := NewAnnotationHandlerRegistry()

	err := registry.Register("test", nil)
	if err == nil {
		t.Fatal("Expected error for nil handler")
	}
}

func TestAnnotationHandlerRegistry_Priority(t *testing.T) {
	registry := NewAnnotationHandlerRegistry()

	// Register handlers with different priorities
	_ = registry.Register("high", &TestHandler{BaseAnnotationHandler: BaseAnnotationHandler{Priority: 1}})
	_ = registry.Register("low", &TestHandler{BaseAnnotationHandler: BaseAnnotationHandler{Priority: 100}})
	_ = registry.Register("medium", &TestHandler{BaseAnnotationHandler: BaseAnnotationHandler{Priority: 50}})

	// GetAllHandlers should return in priority order
	handlers := registry.GetAllHandlers()
	if len(handlers) != 3 {
		t.Fatalf("Expected 3 handlers, got %d", len(handlers))
	}

	if handlers[0].GetPriority() != 1 {
		t.Errorf("Expected first handler priority 1, got %d", handlers[0].GetPriority())
	}
	if handlers[1].GetPriority() != 50 {
		t.Errorf("Expected second handler priority 50, got %d", handlers[1].GetPriority())
	}
	if handlers[2].GetPriority() != 100 {
		t.Errorf("Expected third handler priority 100, got %d", handlers[2].GetPriority())
	}
}

func TestAnnotationHandlerRegistry_Handle(t *testing.T) {
	registry := NewAnnotationHandlerRegistry()

	_ = registry.Register("test", &TestHandler{BaseAnnotationHandler: BaseAnnotationHandler{Priority: 10}})

	ctx := context.Background()
	meta := &AnnotationMeta{
		Type:      "test",
		CacheName: "test-cache",
		Key:       "test-key",
	}

	result, err := registry.Handle(ctx, meta, nil)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if result != "test-result" {
		t.Errorf("Expected 'test-result', got '%v'", result)
	}
}

func TestAnnotationHandlerRegistry_NoHandler(t *testing.T) {
	registry := NewAnnotationHandlerRegistry()

	ctx := context.Background()
	meta := &AnnotationMeta{
		Type: "nonexistent",
	}

	_, err := registry.Handle(ctx, meta, nil)
	if err == nil {
		t.Fatal("Expected error for nonexistent handler")
	}
}

func TestAnnotationHandlerRegistry_Clear(t *testing.T) {
	registry := NewAnnotationHandlerRegistry()

	// Register some handlers
	_ = registry.Register("test1", &TestHandler{BaseAnnotationHandler: BaseAnnotationHandler{Priority: 10}})
	_ = registry.Register("test2", &TestHandler{BaseAnnotationHandler: BaseAnnotationHandler{Priority: 20}})

	// Clear
	registry.Clear()

	// Verify
	if len(registry.ListHandlers()) != 0 {
		t.Errorf("Expected 0 handlers after clear, got %d", len(registry.ListHandlers()))
	}
}

func TestGlobalRegistry(t *testing.T) {
	// Test global registry functions
	handler := &TestHandler{BaseAnnotationHandler: BaseAnnotationHandler{Priority: 10}}
	
	err := RegisterHandler("global-test", handler)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	// Get
	retrieved, exists := GetHandler("global-test")
	if !exists {
		t.Fatal("Expected handler to exist")
	}
	if retrieved.GetPriority() != 10 {
		t.Errorf("Expected priority 10, got %d", retrieved.GetPriority())
	}

	// List
	handlers := ListHandlers()
	found := false
	for _, h := range handlers {
		if h == "global-test" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Expected 'global-test' in handler list")
	}

	// Cleanup
	_ = UnregisterHandler("global-test")
}

func TestAnnotationMeta(t *testing.T) {
	meta := &AnnotationMeta{
		Type:      "cacheable",
		CacheName: "users",
		Key:       "#id",
		TTL:       "30m",
		Condition: "#id != ''",
		Unless:    "#result == nil",
		Before:    false,
		Sync:      true,
		Attributes: map[string]string{
			"maxRetries": "3",
			"timeout":    "5s",
		},
	}

	if meta.Type != "cacheable" {
		t.Errorf("Expected type 'cacheable', got '%s'", meta.Type)
	}
	if meta.CacheName != "users" {
		t.Errorf("Expected cache name 'users', got '%s'", meta.CacheName)
	}
	if meta.Attributes["maxRetries"] != "3" {
		t.Errorf("Expected maxRetries '3', got '%s'", meta.Attributes["maxRetries"])
	}
}

func TestBaseAnnotationHandler(t *testing.T) {
	handler := &BaseAnnotationHandler{Priority: 42}
	
	if handler.GetPriority() != 42 {
		t.Errorf("Expected priority 42, got %d", handler.GetPriority())
	}
}

// TestHandler is a mock handler for testing
type TestHandler struct {
	BaseAnnotationHandler
}

func (h *TestHandler) Handle(ctx context.Context, meta *AnnotationMeta, args []reflect.Value) (interface{}, error) {
	return "test-result", nil
}
