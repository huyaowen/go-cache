package cache

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/coderiser/go-cache/pkg/core"
	"github.com/coderiser/go-cache/pkg/proxy"
	"github.com/coderiser/go-cache/pkg/spel"
)

// GlobalInterceptor 全局缓存拦截器
//
// 这是方案 G 的核心执行引擎：在方法调用时拦截，根据注解执行缓存逻辑。
// 通过代码生成器在 init() 中自动注册，用户无感知。
type GlobalInterceptor struct {
	mu        sync.RWMutex
	manager   core.CacheManager
	evaluator *spel.SpELEvaluator
}

// globalInterceptorInstance 全局拦截器单例
var (
	globalInterceptorInstance *GlobalInterceptor
	interceptorOnce           sync.Once
)

// GetGlobalInterceptor 获取全局拦截器单例
func GetGlobalInterceptor() *GlobalInterceptor {
	interceptorOnce.Do(func() {
		globalInterceptorInstance = &GlobalInterceptor{
			evaluator: spel.NewSpELEvaluator(),
		}
	})
	return globalInterceptorInstance
}

// SetManager 设置缓存管理器
func (gi *GlobalInterceptor) SetManager(manager core.CacheManager) {
	gi.mu.Lock()
	defer gi.mu.Unlock()
	gi.manager = manager
}

// GetManager 获取缓存管理器
func (gi *GlobalInterceptor) GetManager() core.CacheManager {
	gi.mu.RLock()
	defer gi.mu.RUnlock()
	return gi.manager
}

// InterceptCacheable 拦截 @cacheable 方法
//
// 核心逻辑:
// 1. 构建 SpEL 上下文
// 2. 生成缓存 Key
// 3. 查询缓存 (命中则返回)
// 4. 执行原始方法
// 5. 写入缓存
//
// 参数:
//   - typeName: 类型名 (如 "ProductService")
//   - methodName: 方法名 (如 "GetProduct")
//   - args: 方法参数
//   - annotation: 缓存注解
//   - originalFunc: 原始方法调用函数
//
// 返回: 方法执行结果
func (gi *GlobalInterceptor) InterceptCacheable(
	typeName, methodName string,
	args []reflect.Value,
	annotation *proxy.CacheAnnotation,
	originalFunc func() ([]reflect.Value, error),
) ([]reflect.Value, error) {
	gi.mu.RLock()
	manager := gi.manager
	gi.mu.RUnlock()

	if manager == nil {
		log.Printf("[WARN] GlobalInterceptor: manager is nil, skipping cache")
		results, err := originalFunc()
		return results, err
	}

	// 1. 构建 SpEL 上下文
	ctx := gi.buildSpelContext(args, annotation)

	// 2. 生成缓存 Key
	cacheKey, err := gi.evaluator.EvaluateToString(annotation.Key, ctx)
	if err != nil {
		log.Printf("[DEBUG] SpEL key evaluation failed: %v, falling back to original", err)
		results, err := originalFunc()
		return results, err
	}

	// 3. 获取缓存
	cache, err := manager.GetCache(annotation.CacheName)
	if err != nil {
		log.Printf("[DEBUG] GetCache failed: %v, falling back to original", err)
		results, err := originalFunc()
		return results, err
	}

	// 4. 查询缓存
	cachedValue, found, err := cache.Get(context.Background(), cacheKey)
	if err == nil && found {
		log.Printf("[INFO] Cache HIT: %s:%s", annotation.CacheName, cacheKey)
		// 缓存命中，直接返回
		return []reflect.Value{reflect.ValueOf(cachedValue)}, nil
	}

	log.Printf("[DEBUG] Cache MISS: %s:%s", annotation.CacheName, cacheKey)

	// 5. 执行原始方法
	results, err := originalFunc()
	if err != nil {
		return nil, err
	}

	// 6. 写入缓存 (如果有返回值)
	if len(results) > 0 {
		resultValue := results[0].Interface()

		// 检查 unless 条件
		if annotation.Unless != "" {
			ctx.SetResult(resultValue)
			unlessResult, err := gi.evaluator.Evaluate(annotation.Unless, ctx)
			if err == nil && gi.isTruthy(unlessResult) {
				log.Printf("[DEBUG] Cache skipped due to unless condition")
				return results, nil
			}
		}

		// 检查 condition 条件 (执行后再次确认)
		if annotation.Condition != "" {
			conditionResult, err := gi.evaluator.Evaluate(annotation.Condition, ctx)
			if err != nil || !gi.isTruthy(conditionResult) {
				log.Printf("[DEBUG] Cache skipped due to condition")
				return results, nil
			}
		}

		// 解析 TTL
		ttl := gi.parseTTL(annotation.TTL, ctx)

		// 写入缓存
		err = cache.Set(context.Background(), cacheKey, resultValue, ttl)
		if err != nil {
			log.Printf("[WARN] Cache set failed: %v", err)
		} else {
			log.Printf("[INFO] Cache SET: %s:%s (TTL=%v)", annotation.CacheName, cacheKey, ttl)
		}
	}

	return results, nil
}

// InterceptCachePut 拦截 @cacheput 方法
//
// 逻辑：先执行原始方法，然后更新缓存。
func (gi *GlobalInterceptor) InterceptCachePut(
	typeName, methodName string,
	args []reflect.Value,
	annotation *proxy.CacheAnnotation,
	originalFunc func() ([]reflect.Value, error),
) ([]reflect.Value, error) {
	gi.mu.RLock()
	manager := gi.manager
	gi.mu.RUnlock()

	if manager == nil {
		results, err := originalFunc()
		return results, err
	}

	// 执行原始方法
	results, err := originalFunc()
	if err != nil {
		return nil, err
	}

	// 更新缓存
	if len(results) > 0 {
		ctx := gi.buildSpelContext(args, annotation)
		ctx.SetResult(results[0].Interface())

		cacheKey, err := gi.evaluator.EvaluateToString(annotation.Key, ctx)
		if err != nil {
			return results, nil
		}

		cache, err := manager.GetCache(annotation.CacheName)
		if err != nil {
			return results, nil
		}

		ttl := gi.parseTTL(annotation.TTL, ctx)
		_ = cache.Set(context.Background(), cacheKey, results[0].Interface(), ttl)
	}

	return results, nil
}

// InterceptCacheEvict 拦截 @cacheevict 方法
//
// 逻辑：根据 beforeInvocation 决定在方法执行前或后清除缓存。
func (gi *GlobalInterceptor) InterceptCacheEvict(
	typeName, methodName string,
	args []reflect.Value,
	annotation *proxy.CacheAnnotation,
	originalFunc func() ([]reflect.Value, error),
) ([]reflect.Value, error) {
	gi.mu.RLock()
	manager := gi.manager
	gi.mu.RUnlock()

	if manager == nil {
		results, err := originalFunc()
		return results, err
	}

	cache, err := manager.GetCache(annotation.CacheName)
	if err != nil {
		results, err := originalFunc()
		return results, err
	}

	ctx := gi.buildSpelContext(args, annotation)

	// beforeInvocation=true: 方法执行前清除
	if annotation.Before {
		cacheKey, err := gi.evaluator.EvaluateToString(annotation.Key, ctx)
		if err == nil {
			_ = cache.Delete(context.Background(), cacheKey)
		}
	}

	// 执行原始方法
	results, err := originalFunc()
	if err != nil {
		return nil, err
	}

	// beforeInvocation=false (默认): 方法执行后清除
	if !annotation.Before {
		ctx.SetResult(results[0].Interface())
		cacheKey, err := gi.evaluator.EvaluateToString(annotation.Key, ctx)
		if err == nil {
			_ = cache.Delete(context.Background(), cacheKey)
		}
	}

	return results, nil
}

// buildSpelContext 构建 SpEL 求值上下文
func (gi *GlobalInterceptor) buildSpelContext(args []reflect.Value, annotation *proxy.CacheAnnotation) *spel.EvaluationContext {
	ctx := spel.NewEvaluationContext()

	// 设置参数
	for i, arg := range args {
		value := arg.Interface()
		ctx.SetArgByIndex(i, value)
		ctx.SetArg(fmt.Sprintf("p%d", i), value)
	}

	// 智能参数名映射 (常见模式)
	if len(args) >= 1 {
		ctx.SetArg("id", args[0].Interface())
	}
	if len(args) >= 2 {
		ctx.SetArg("user", args[1].Interface())
	}

	return ctx
}

// parseTTL 解析 TTL 字符串
func (gi *GlobalInterceptor) parseTTL(ttlExpr string, ctx *spel.EvaluationContext) time.Duration {
	if ttlExpr == "" {
		return 30 * time.Minute
	}

	// 尝试作为 SpEL 表达式求值
	result, err := gi.evaluator.EvaluateToInt(ttlExpr, ctx)
	if err == nil {
		return time.Duration(result) * time.Second
	}

	// 作为 duration 字符串解析
	d, err := time.ParseDuration(ttlExpr)
	if err == nil {
		return d
	}

	return 30 * time.Minute
}

// isTruthy 判断值是否为真
func (gi *GlobalInterceptor) isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return v != 0
	case float32, float64:
		return v != 0
	case string:
		return v != "" && v != "false"
	default:
		return true
	}
}
