package core

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
)

// TestNilMarker 测试空值标记
func TestNilMarker(t *testing.T) {
	t.Run("WrapAndUnwrap", func(t *testing.T) {
		// 测试 nil 值包装
		wrapped := WrapNilMarker(nil)
		if !IsNilMarker(wrapped) {
			t.Error("Expected nil to be wrapped as nilMarker")
		}

		unwrapped := UnwrapNilMarker(wrapped)
		if unwrapped != nil {
			t.Error("Expected unwrapped value to be nil")
		}

		// 测试非 nil 值
		value := "test"
		wrapped2 := WrapNilMarker(value)
		if IsNilMarker(wrapped2) {
			t.Error("Expected non-nil value to not be wrapped")
		}

		unwrapped2 := UnwrapNilMarker(wrapped2)
		if unwrapped2 != "test" {
			t.Error("Expected unwrapped value to be 'test'")
		}
	})

	t.Run("IsNilMarker", func(t *testing.T) {
		if !IsNilMarker(nil) {
			t.Error("nil should be detected as nilMarker")
		}
		if !IsNilMarker(backend.NilMarker) {
			t.Error("NilMarker string should be detected")
		}
		if IsNilMarker("test") {
			t.Error("'test' should not be detected as nilMarker")
		}
	})
}

// TestCachePenetrationProtection 测试缓存穿透保护
func TestCachePenetrationProtection(t *testing.T) {
	config := DefaultProtectionConfig()
	config.EnablePenetrationProtection = true
	config.EmptyValueTTL = 5 * time.Minute

	protection := NewCacheProtection(config)

	t.Run("EmptyValueCaching", func(t *testing.T) {
		ctx := context.Background()
		callCount := int64(0)
		cache := make(map[string]interface{})
		cacheTTL := make(map[string]time.Time)

		// 模拟缓存未命中，返回 nil
		cacheMissFn := func() (interface{}, error) {
			atomic.AddInt64(&callCount, 1)
			return nil, nil
		}

		cacheGet := func() (interface{}, bool, error) {
			v, found := cache["test-key"]
			return v, found, nil
		}

		cacheSet := func(v interface{}, ttl time.Duration) error {
			cache["test-key"] = protection.WrapForStorage(v)
			cacheTTL["test-key"] = time.Now().Add(ttl)
			return nil
		}

		// 第一次调用 - 应该执行 fn 并缓存空值
		result1, err1 := protection.ProtectedGet(ctx, "test-key", cacheGet, cacheMissFn, cacheSet)
		if err1 != nil {
			t.Fatalf("Unexpected error: %v", err1)
		}
		if result1 != nil {
			t.Errorf("Expected nil result, got: %v", result1)
		}
		if callCount != 1 {
			t.Errorf("Expected 1 call, got: %d", callCount)
		}

		// 第二次调用 - 应该从缓存获取空值标记，不执行 fn
		result2, err2 := protection.ProtectedGet(ctx, "test-key", cacheGet, cacheMissFn, cacheSet)
		if err2 != nil {
			t.Fatalf("Unexpected error: %v", err2)
		}
		if result2 != nil {
			t.Errorf("Expected nil result from cache, got: %v", result2)
		}
		if callCount != 1 {
			t.Errorf("Expected still 1 call (cached), got: %d", callCount)
		}
	})

	t.Run("NormalValueCaching", func(t *testing.T) {
		ctx := context.Background()
		callCount := int64(0)
		cache := make(map[string]interface{})

		cacheMissFn := func() (interface{}, error) {
			atomic.AddInt64(&callCount, 1)
			return "real-data", nil
		}

		cacheGet := func() (interface{}, bool, error) {
			v, found := cache["test-key-2"]
			return v, found, nil
		}

		cacheSet := func(v interface{}, ttl time.Duration) error {
			cache["test-key-2"] = protection.WrapForStorage(v)
			return nil
		}

		// 第一次调用
		result1, _ := protection.ProtectedGet(ctx, "test-key-2", cacheGet, cacheMissFn, cacheSet)
		if result1 != "real-data" {
			t.Errorf("Expected 'real-data', got: %v", result1)
		}

		// 第二次调用 - 应该从缓存获取
		result2, _ := protection.ProtectedGet(ctx, "test-key-2", cacheGet, cacheMissFn, cacheSet)
		if result2 != "real-data" {
			t.Errorf("Expected 'real-data' from cache, got: %v", result2)
		}

		if callCount != 1 {
			t.Errorf("Expected 1 call (cached after first), got: %d", callCount)
		}
	})
}

// TestCacheBreakdownProtection 测试缓存击穿保护（Singleflight）
func TestCacheBreakdownProtection(t *testing.T) {
	config := DefaultProtectionConfig()
	config.EnableBreakdownProtection = true

	protection := NewCacheProtection(config)

	t.Run("ConcurrentRequestsMerged", func(t *testing.T) {
		ctx := context.Background()
		callCount := int64(0)
		var wg sync.WaitGroup
		numGoroutines := 10

		// 模拟慢查询
		slowFn := func() (interface{}, error) {
			atomic.AddInt64(&callCount, 1)
			time.Sleep(100 * time.Millisecond)
			return "data", nil
		}

		results := make([]interface{}, numGoroutines)
		errors := make([]error, numGoroutines)
		shared := make([]bool, numGoroutines)

		// 启动多个并发请求
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				result, err, isShared := protection.ApplyBreakdownProtection(ctx, "same-key", slowFn)
				results[idx] = result
				errors[idx] = err
				shared[idx] = isShared
			}(i)
		}

		wg.Wait()

		// 验证所有请求都成功
		for i := 0; i < numGoroutines; i++ {
			if errors[i] != nil {
				t.Errorf("Goroutine %d got error: %v", i, errors[i])
			}
			if results[i] != "data" {
				t.Errorf("Goroutine %d got wrong result: %v", i, results[i])
			}
		}

		// 验证只有一个请求实际执行
		if callCount != 1 {
			t.Errorf("Expected 1 actual call (others merged), got: %d", callCount)
		}

		// 验证至少有一个请求是共享的（除了第一个）
		sharedCount := 0
		for i := 0; i < numGoroutines; i++ {
			if shared[i] {
				sharedCount++
			}
		}
		if sharedCount < numGoroutines-1 {
			t.Errorf("Expected at least %d shared requests, got: %d", numGoroutines-1, sharedCount)
		}
	})

	t.Run("DifferentKeysNotMerged", func(t *testing.T) {
		ctx := context.Background()
		callCount := int64(0)

		fn := func() (interface{}, error) {
			atomic.AddInt64(&callCount, 1)
			return "data", nil
		}

		// 不同 key 的请求不应该被合并
		protection.ApplyBreakdownProtection(ctx, "key-1", fn)
		protection.ApplyBreakdownProtection(ctx, "key-2", fn)
		protection.ApplyBreakdownProtection(ctx, "key-3", fn)

		if callCount != 3 {
			t.Errorf("Expected 3 calls for different keys, got: %d", callCount)
		}
	})

	t.Run("ForgetSingleFlight", func(t *testing.T) {
		ctx := context.Background()
		callCount := int64(0)

		fn := func() (interface{}, error) {
			atomic.AddInt64(&callCount, 1)
			return "data", nil
		}

		// 第一次调用
		protection.ApplyBreakdownProtection(ctx, "forget-key", fn)

		// 取消 singleflight
		protection.CancelSingleFlight("forget-key")

		// 第二次调用 - 应该重新执行
		protection.ApplyBreakdownProtection(ctx, "forget-key", fn)

		if callCount != 2 {
			t.Errorf("Expected 2 calls after Forget, got: %d", callCount)
		}
	})
}

// TestCacheAvalancheProtection 测试缓存雪崩保护（TTL 随机偏移）
func TestCacheAvalancheProtection(t *testing.T) {
	config := DefaultProtectionConfig()
	config.EnableAvalancheProtection = true
	config.TTLJitterFactor = 0.1

	protection := NewCacheProtection(config)

	t.Run("TTLJitter", func(t *testing.T) {
		baseTTL := 30 * time.Minute
		jitterFactor := 0.1

		// 多次计算 TTL，验证随机性
		ttls := make([]time.Duration, 100)
		for i := 0; i < 100; i++ {
			ttls[i] = protection.ApplyAvalancheProtection(baseTTL)
		}

		// 验证 TTL 在预期范围内
		minExpected := time.Duration(float64(baseTTL) * (1 - jitterFactor))
		maxExpected := time.Duration(float64(baseTTL) * (1 + jitterFactor))

		for i, ttl := range ttls {
			if ttl < minExpected || ttl > maxExpected {
				t.Errorf("TTL[%d] %v out of expected range [%v, %v]", i, ttl, minExpected, maxExpected)
			}
		}

		// 验证 TTL 有变化（不是固定值）
		hasVariation := false
		for i := 1; i < len(ttls); i++ {
			if ttls[i] != ttls[0] {
				hasVariation = true
				break
			}
		}
		if !hasVariation {
			t.Error("Expected TTL variation, but all values were identical")
		}
	})

	t.Run("JitterFactorBoundary", func(t *testing.T) {
		baseTTL := 10 * time.Minute

		// 测试 0.5 边界
		config2 := &ProtectionConfig{
			EnableAvalancheProtection: true,
			TTLJitterFactor:           0.6, // 超过 0.5 应该被限制
		}
		protection2 := NewCacheProtection(config2)

		ttl := protection2.ApplyAvalancheProtection(baseTTL)
		minExpected := time.Duration(float64(baseTTL) * 0.5)
		maxExpected := time.Duration(float64(baseTTL) * 1.5)

		if ttl < minExpected || ttl > maxExpected {
			t.Errorf("TTL %v out of bounded range [%v, %v]", ttl, minExpected, maxExpected)
		}
	})

	t.Run("MinimumTTL", func(t *testing.T) {
		baseTTL := 100 * time.Millisecond
		config3 := &ProtectionConfig{
			EnableAvalancheProtection: true,
			TTLJitterFactor:           0.5,
		}
		protection3 := NewCacheProtection(config3)

		// 即使抖动后很小，TTL 也应该至少为 1 秒
		ttl := protection3.ApplyAvalancheProtection(baseTTL)
		if ttl < time.Second {
			t.Errorf("Expected minimum TTL of 1s, got: %v", ttl)
		}
	})

	t.Run("DisabledJitter", func(t *testing.T) {
		baseTTL := 30 * time.Minute
		config4 := &ProtectionConfig{
			EnableAvalancheProtection: false,
		}
		protection4 := NewCacheProtection(config4)

		ttl := protection4.ApplyAvalancheProtection(baseTTL)
		if ttl != baseTTL {
			t.Errorf("Expected original TTL when disabled, got: %v", ttl)
		}
	})

	t.Run("CalculateTTLWithJitter", func(t *testing.T) {
		baseTTL := 60 * time.Minute
		customFactor := 0.2

		ttl := protection.CalculateTTLWithJitter(baseTTL, customFactor)

		minExpected := time.Duration(float64(baseTTL) * (1 - customFactor))
		maxExpected := time.Duration(float64(baseTTL) * (1 + customFactor))

		if ttl < minExpected || ttl > maxExpected {
			t.Errorf("TTL %v out of expected range [%v, %v]", ttl, minExpected, maxExpected)
		}
	})
}

// TestProtectionConfig 测试保护配置
func TestProtectionConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultProtectionConfig()

		if !config.EnablePenetrationProtection {
			t.Error("Expected penetration protection enabled by default")
		}
		if config.EmptyValueTTL != 5*time.Minute {
			t.Errorf("Expected empty value TTL of 5m, got: %v", config.EmptyValueTTL)
		}
		if !config.EnableBreakdownProtection {
			t.Error("Expected breakdown protection enabled by default")
		}
		if !config.EnableAvalancheProtection {
			t.Error("Expected avalanche protection enabled by default")
		}
		if config.TTLJitterFactor != 0.1 {
			t.Errorf("Expected jitter factor of 0.1, got: %v", config.TTLJitterFactor)
		}
	})

	t.Run("CustomConfig", func(t *testing.T) {
		config := &ProtectionConfig{
			EnablePenetrationProtection: false,
			EmptyValueTTL:               10 * time.Minute,
			EnableBreakdownProtection:   true,
			EnableAvalancheProtection:   false,
			TTLJitterFactor:             0.2,
		}

		protection := NewCacheProtection(config)
		retrieved := protection.GetProtectionConfig()

		if retrieved.EnablePenetrationProtection {
			t.Error("Expected penetration protection disabled")
		}
		if retrieved.EmptyValueTTL != 10*time.Minute {
			t.Errorf("Expected empty value TTL of 10m, got: %v", retrieved.EmptyValueTTL)
		}
	})

	t.Run("NilConfig", func(t *testing.T) {
		protection := NewCacheProtection(nil)
		config := protection.GetProtectionConfig()

		if config == nil {
			t.Error("Expected default config when nil passed")
		}
	})
}

// TestProtectionIntegration 测试保护机制集成
func TestProtectionIntegration(t *testing.T) {
	t.Run("FullProtectionFlow", func(t *testing.T) {
		ctx := context.Background()
		config := DefaultProtectionConfig()
		protection := NewCacheProtection(config)

		cache := make(map[string]interface{})
		callCount := int64(0)

		// 模拟完整的缓存流程
		cacheMissFn := func() (interface{}, error) {
			atomic.AddInt64(&callCount, 1)
			time.Sleep(50 * time.Millisecond) // 模拟慢查询
			return "cached-data", nil
		}

		cacheGet := func() (interface{}, bool, error) {
			v, found := cache["integration-key"]
			return v, found, nil
		}

		cacheSet := func(v interface{}, ttl time.Duration) error {
			cache["integration-key"] = v
			return nil
		}

		// 第一次调用 - 缓存未命中
		result1, err1 := protection.ProtectedGet(ctx, "integration-key", cacheGet, cacheMissFn, cacheSet)
		if err1 != nil {
			t.Fatalf("Unexpected error: %v", err1)
		}
		if result1 != "cached-data" {
			t.Errorf("Expected 'cached-data', got: %v", result1)
		}

		// 第二次调用 - 缓存命中
		result2, err2 := protection.ProtectedGet(ctx, "integration-key", cacheGet, cacheMissFn, cacheSet)
		if err2 != nil {
			t.Fatalf("Unexpected error: %v", err2)
		}
		if result2 != "cached-data" {
			t.Errorf("Expected 'cached-data' from cache, got: %v", result2)
		}

		if callCount != 1 {
			t.Errorf("Expected 1 call (cached after first), got: %d", callCount)
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		ctx := context.Background()
		protection := NewCacheProtection(nil)

		cacheMissFn := func() (interface{}, error) {
			return nil, context.DeadlineExceeded
		}

		cacheGet := func() (interface{}, bool, error) {
			return nil, false, nil
		}

		cacheSet := func(v interface{}, ttl time.Duration) error {
			return nil
		}

		_, err := protection.ProtectedGet(ctx, "error-key", cacheGet, cacheMissFn, cacheSet)
		if err != context.DeadlineExceeded {
			t.Errorf("Expected context.DeadlineExceeded, got: %v", err)
		}
	})
}

// TestProtectionStats 测试保护统计
func TestProtectionStats(t *testing.T) {
	protection := NewCacheProtection(nil)
	stats := protection.GetStats()

	if stats == nil {
		t.Error("Expected non-nil stats")
	}
	// 当前实现返回空统计，未来可扩展
}

// BenchmarkBreakdownProtection 性能测试 - 击穿保护
func BenchmarkBreakdownProtection(b *testing.B) {
	ctx := context.Background()
	protection := NewCacheProtection(nil)

	callCount := int64(0)
	fn := func() (interface{}, error) {
		atomic.AddInt64(&callCount, 1)
		time.Sleep(10 * time.Millisecond)
		return "data", nil
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			protection.ApplyBreakdownProtection(ctx, "bench-key", fn)
		}
	})

	// 验证 singleflight 有效
	if callCount > int64(b.N) {
		b.Errorf("Expected call count <= %d, got: %d", b.N, callCount)
	}
}

// BenchmarkAvalancheProtection 性能测试 - 雪崩保护
func BenchmarkAvalancheProtection(b *testing.B) {
	protection := NewCacheProtection(nil)
	baseTTL := 30 * time.Minute

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		protection.ApplyAvalancheProtection(baseTTL)
	}
}

// BenchmarkPenetrationProtection 性能测试 - 穿透保护
func BenchmarkPenetrationProtection(b *testing.B) {
	protection := NewCacheProtection(nil)

	b.Run("WrapNil", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			protection.WrapForStorage(nil)
		}
	})

	b.Run("WrapValue", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			protection.WrapForStorage("data")
		}
	})

	b.Run("Unwrap", func(b *testing.B) {
		wrapped := protection.WrapForStorage(nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			protection.UnwrapFromStorage(wrapped)
		}
	})
}
