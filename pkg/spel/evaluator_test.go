package spel

import (
	"testing"
)

func TestSpELEvaluator(t *testing.T) {
	evaluator := NewSpELEvaluator()
	defer evaluator.ClearCache()

	t.Run("Evaluate simple expression with named arg", func(t *testing.T) {
		ctx := NewEvaluationContext()
		ctx.SetArg("id", 123)

		// expr library uses variable names directly (not #prefix)
		result, err := evaluator.Evaluate("id", ctx)
		if err != nil {
			t.Fatalf("Evaluate failed: %v", err)
		}
		if result != 123 {
			t.Errorf("Expected 123, got %v", result)
		}
	})

	t.Run("EvaluateToString with named arg", func(t *testing.T) {
		ctx := NewEvaluationContext()
		ctx.SetArg("name", "Alice")

		result, err := evaluator.EvaluateToString("name", ctx)
		if err != nil {
			t.Fatalf("EvaluateToString failed: %v", err)
		}
		if result != "Alice" {
			t.Errorf("Expected 'Alice', got '%s'", result)
		}
	})

	t.Run("EvaluateToString with concatenation", func(t *testing.T) {
		ctx := NewEvaluationContext()
		ctx.SetArg("prefix", "user")
		ctx.SetArg("id", 123)

		result, err := evaluator.EvaluateToString("prefix + ':' + string(id)", ctx)
		if err != nil {
			t.Fatalf("EvaluateToString failed: %v", err)
		}
		if result != "user:123" {
			t.Errorf("Expected 'user:123', got '%s'", result)
		}
	})

	t.Run("EvaluateToInt", func(t *testing.T) {
		ctx := NewEvaluationContext()
		ctx.SetArg("count", 42)

		result, err := evaluator.EvaluateToInt("count", ctx)
		if err != nil {
			t.Fatalf("EvaluateToInt failed: %v", err)
		}
		if result != 42 {
			t.Errorf("Expected 42, got %d", result)
		}
	})

	t.Run("Empty expression error", func(t *testing.T) {
		ctx := NewEvaluationContext()
		_, err := evaluator.Evaluate("", ctx)
		if err != ErrEmptyExpression {
			t.Errorf("Expected ErrEmptyExpression, got %v", err)
		}
	})

	t.Run("Invalid expression error", func(t *testing.T) {
		ctx := NewEvaluationContext()
		_, err := evaluator.Evaluate("invalid syntax @@@@", ctx)
		if err == nil {
			t.Error("Expected error for invalid expression")
		}
	})

	t.Run("Expression caching", func(t *testing.T) {
		evaluator.ClearCache()
		ctx := NewEvaluationContext()
		ctx.SetArg("x", 10)

		// First evaluation (compiles)
		_, err := evaluator.Evaluate("x * 2", ctx)
		if err != nil {
			t.Fatalf("First evaluation failed: %v", err)
		}
		if evaluator.CacheSize() != 1 {
			t.Errorf("Expected cache size 1, got %d", evaluator.CacheSize())
		}

		// Second evaluation (uses cache)
		result, err := evaluator.Evaluate("x * 2", ctx)
		if err != nil {
			t.Fatalf("Second evaluation failed: %v", err)
		}
		if result != 20 {
			t.Errorf("Expected 20, got %v", result)
		}
		if evaluator.CacheSize() != 1 {
			t.Errorf("Expected cache size still 1, got %d", evaluator.CacheSize())
		}
	})

	t.Run("ClearCache", func(t *testing.T) {
		evaluator.ClearCache() // Start fresh
		ctx := NewEvaluationContext()
		ctx.SetArg("y", 5)

		_, _ = evaluator.Evaluate("y + 1", ctx)
		if evaluator.CacheSize() != 1 {
			t.Errorf("Expected cache size 1, got %d", evaluator.CacheSize())
		}

		evaluator.ClearCache()
		if evaluator.CacheSize() != 0 {
			t.Errorf("Expected cache size 0 after clear, got %d", evaluator.CacheSize())
		}
	})
}

func TestEvaluationContext(t *testing.T) {
	t.Run("SetArg and GetArg", func(t *testing.T) {
		ctx := NewEvaluationContext()
		ctx.SetArg("key", "value")

		val, ok := ctx.GetArg("key")
		if !ok {
			t.Error("Expected to find key")
		}
		if val != "value" {
			t.Errorf("Expected 'value', got %v", val)
		}
	})

	t.Run("SetArgByIndex", func(t *testing.T) {
		ctx := NewEvaluationContext()
		ctx.SetArgByIndex(0, "first")
		ctx.SetArgByIndex(1, "second")

		val0, ok0 := ctx.GetArgByIndex(0)
		if !ok0 || val0 != "first" {
			t.Errorf("Expected 'first' at index 0, got %v", val0)
		}

		val1, ok1 := ctx.GetArgByIndex(1)
		if !ok1 || val1 != "second" {
			t.Errorf("Expected 'second' at index 1, got %v", val1)
		}
	})

	t.Run("SetResult", func(t *testing.T) {
		ctx := NewEvaluationContext()
		ctx.SetResult("result_value")

		if ctx.Result != "result_value" {
			t.Errorf("Expected 'result_value', got %v", ctx.Result)
		}
	})

	t.Run("BuildVariables", func(t *testing.T) {
		ctx := NewEvaluationContext()
		ctx.SetArg("name", "test")
		ctx.SetArgByIndex(0, 42)
		ctx.SetResult("return_val")
		ctx.CacheName = "mycache"
		ctx.Method = "GetUser"

		vars := ctx.BuildVariables()

		if vars["name"] != "test" {
			t.Errorf("Expected name='test', got %v", vars["name"])
		}
		if vars["#0"] != 42 {
			t.Errorf("Expected #0=42, got %v", vars["#0"])
		}
		if vars["result"] != "return_val" {
			t.Errorf("Expected result='return_val', got %v", vars["result"])
		}
		if vars["cacheName"] != "mycache" {
			t.Errorf("Expected cacheName='mycache', got %v", vars["cacheName"])
		}
		if vars["method"] != "GetUser" {
			t.Errorf("Expected method='GetUser', got %v", vars["method"])
		}
	})

	t.Run("SetExtra and GetExtra", func(t *testing.T) {
		ctx := NewEvaluationContext()
		ctx.SetExtra("custom", "data")

		val, ok := ctx.GetExtra("custom")
		if !ok || val != "data" {
			t.Errorf("Expected 'data', got %v", val)
		}
	})
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected int64
		hasError bool
	}{
		{int(42), 42, false},
		{int8(42), 42, false},
		{int16(42), 42, false},
		{int32(42), 42, false},
		{int64(42), 42, false},
		{uint(42), 42, false},
		{uint8(42), 42, false},
		{uint16(42), 42, false},
		{uint32(42), 42, false},
		{uint64(42), 42, false},
		{float32(42.0), 42, false},
		{float64(42.0), 42, false},
		{"42", 42, false},
		{"invalid", 0, true},
		{nil, 0, true},
	}

	for _, tt := range tests {
		result, err := toInt64(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("Expected error for input %v, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %v: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("Expected %d for input %v, got %d", tt.expected, tt.input, result)
			}
		}
	}
}

func TestCompilationError(t *testing.T) {
	err := &CompilationError{
		Expression: "test_expr",
		Err:        ErrEmptyExpression,
	}

	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}

	unwrapped := err.Unwrap()
	if unwrapped != ErrEmptyExpression {
		t.Errorf("Expected ErrEmptyExpression, got %v", unwrapped)
	}
}

func TestEvaluationError(t *testing.T) {
	err := &EvaluationError{
		Expression: "test_expr",
		Err:        ErrEmptyExpression,
	}

	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}

	unwrapped := err.Unwrap()
	if unwrapped != ErrEmptyExpression {
		t.Errorf("Expected ErrEmptyExpression, got %v", unwrapped)
	}
}

func BenchmarkSpELEvaluator(b *testing.B) {
	evaluator := NewSpELEvaluator()
	defer evaluator.ClearCache()

	ctx := NewEvaluationContext()
	ctx.SetArg("id", 123)
	ctx.SetArg("name", "test")

	// Warm up cache
	_, _ = evaluator.Evaluate("#id", ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate("#id", ctx)
	}
}

func BenchmarkSpELEvaluatorComplex(b *testing.B) {
	evaluator := NewSpELEvaluator()
	defer evaluator.ClearCache()

	ctx := NewEvaluationContext()
	ctx.SetArg("prefix", "user")
	ctx.SetArg("id", 123)

	// Warm up cache
	_, _ = evaluator.Evaluate("#prefix + ':' + #id", ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate("#prefix + ':' + #id", ctx)
	}
}
