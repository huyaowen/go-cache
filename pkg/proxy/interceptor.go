package proxy

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/coderiser/go-cache/pkg/core"
	"github.com/coderiser/go-cache/pkg/spel"
)

// MethodInterceptor 方法拦截器接口
type MethodInterceptor interface {
	Intercept(target interface{}, methodName string, args []reflect.Value) []reflect.Value
}

// CacheAnnotation 缓存注解元数据
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

// methodInterceptor 方法拦截器实现
type methodInterceptor struct {
	manager     core.CacheManager
	evaluator   *spel.SpELEvaluator
	methodCache sync.Map // map[string]*CacheAnnotation
}

func newMethodInterceptor(manager core.CacheManager) *methodInterceptor {
	return &methodInterceptor{
		manager:   manager,
		evaluator: spel.NewSpELEvaluator(),
		// methodCache is a sync.Map, no initialization needed
	}
}

// Intercept 拦截方法调用
func (i *methodInterceptor) Intercept(target interface{}, methodName string, args []reflect.Value) []reflect.Value {
	log.Printf("[DEBUG] Intercept: method=%s, target=%T", methodName, target)
	annotation := i.getAnnotation(methodName)
	log.Printf("[DEBUG] Intercept: annotation=%v", annotation)
	if annotation == nil {
		log.Printf("[DEBUG] Intercept: No annotation, invoking original")
		return i.invokeOriginal(target, methodName, args)
	}

	callInfo := i.getCallInfo(target, methodName, args)

	switch annotation.Type {
	case "cacheable":
		return i.handleCacheable(target, callInfo, annotation, args)
	case "cacheput":
		return i.handleCachePut(target, callInfo, annotation, args)
	case "cacheevict":
		return i.handleCacheEvict(target, callInfo, annotation, args)
	default:
		return i.invokeOriginal(target, methodName, args)
	}
}

type callInfo struct {
	methodName string
	methodType reflect.Type
	ctx        *spel.EvaluationContext
}

func (i *methodInterceptor) getCallInfo(target interface{}, methodName string, args []reflect.Value) *callInfo {
	targetValue := reflect.ValueOf(target)
	targetType := targetValue.Type()

	ctx := spel.NewEvaluationContext()
	ctx.Target = target
	ctx.TargetType = targetType
	ctx.Method = methodName

	// 设置参数：支持多种 SpEL 语法
	// #0, #1, ... (索引)
	// #p0, #p1, ... (带前缀的索引)
	// p0, p1, ... (不带#的参数)
	for idx, arg := range args {
		value := arg.Interface()
		ctx.SetArgByIndex(idx, value)
		ctx.SetArg(fmt.Sprintf("p%d", idx), value)
		// 支持 #0, #1 语法（通过 BuildVariables 自动设置）
	}
	
	// 特殊处理：从注解中提取参数名并映射
	// 对于常见模式如 #id, #user，尝试智能匹配
	if len(args) >= 1 {
		// 第一个参数映射为 #id（如果注解使用 #id）
		ctx.SetArg("id", args[0].Interface())
	}
	if len(args) >= 2 {
		// 第二个参数映射为 #user（如果注解使用 #user）
		ctx.SetArg("user", args[1].Interface())
	}

	return &callInfo{
		methodName: methodName,
		methodType: targetType,
		ctx:        ctx,
	}
}

func (i *methodInterceptor) getAnnotation(methodName string) *CacheAnnotation {
	if v, ok := i.methodCache.Load(methodName); ok {
		return v.(*CacheAnnotation)
	}
	return nil
}

// RegisterAnnotation 注册方法注解
func (i *methodInterceptor) RegisterAnnotation(methodName string, annotation *CacheAnnotation) {
	i.methodCache.Store(methodName, annotation)
}

func (i *methodInterceptor) handleCacheable(target interface{}, callInfo *callInfo, annotation *CacheAnnotation, args []reflect.Value) []reflect.Value {
	log.Printf("[DEBUG] handleCacheable: method=%s, cache=%s, key=%s, args=%v", callInfo.methodName, annotation.CacheName, annotation.Key, args)
	
	cache, err := i.manager.GetCache(annotation.CacheName)
	if err != nil {
		log.Printf("[DEBUG] handleCacheable: Failed to get cache: %v", err)
		return i.invokeOriginal(target, callInfo.methodName, args)
	}
	log.Printf("[DEBUG] handleCacheable: Got cache backend: %T", cache)

	callInfo.ctx.CacheName = annotation.CacheName
	cacheKey, err := i.evaluator.EvaluateToString(annotation.Key, callInfo.ctx)
	log.Printf("[DEBUG] handleCacheable: cacheKey='%v', err=%v, ctx.args=%v", cacheKey, err, callInfo.ctx.Args)
	if err != nil {
		log.Printf("[DEBUG] handleCacheable: SpEL evaluation failed: %v", err)
		return i.invokeOriginal(target, callInfo.methodName, args)
	}

	if annotation.Condition != "" {
		conditionResult, err := i.evaluator.Evaluate(annotation.Condition, callInfo.ctx)
		if err != nil || !i.isTruthy(conditionResult) {
			return i.invokeOriginal(target, callInfo.methodName, args)
		}
	}

	ctx := context.Background()
	value, found, err := cache.Get(ctx, cacheKey)
	if err == nil && found {
		log.Printf("[INFO] Cache HIT: %s:%s", annotation.CacheName, cacheKey)
		return []reflect.Value{reflect.ValueOf(value)}
	}
	log.Printf("[DEBUG] Cache MISS: %s:%s", annotation.CacheName, cacheKey)

	results := i.invokeOriginal(target, callInfo.methodName, args)

	if len(results) > 0 {
		resultValue := results[0].Interface()
		callInfo.ctx.SetResult(resultValue)

		if annotation.Unless != "" {
			unlessResult, err := i.evaluator.Evaluate(annotation.Unless, callInfo.ctx)
			if err == nil && i.isTruthy(unlessResult) {
				return results
			}
		}

		ttl := i.parseTTL(annotation.TTL, callInfo.ctx)
		_ = cache.Set(ctx, cacheKey, resultValue, ttl)
	}

	return results
}

func (i *methodInterceptor) handleCachePut(target interface{}, callInfo *callInfo, annotation *CacheAnnotation, args []reflect.Value) []reflect.Value {
	results := i.invokeOriginal(target, callInfo.methodName, args)

	cache, err := i.manager.GetCache(annotation.CacheName)
	if err != nil {
		return results
	}

	callInfo.ctx.CacheName = annotation.CacheName
	if len(results) > 0 {
		callInfo.ctx.SetResult(results[0].Interface())
	}

	cacheKey, err := i.evaluator.EvaluateToString(annotation.Key, callInfo.ctx)
	if err != nil {
		return results
	}

	if len(results) > 0 {
		ctx := context.Background()
		ttl := i.parseTTL(annotation.TTL, callInfo.ctx)
		_ = cache.Set(ctx, cacheKey, results[0].Interface(), ttl)
	}

	return results
}

func (i *methodInterceptor) handleCacheEvict(target interface{}, callInfo *callInfo, annotation *CacheAnnotation, args []reflect.Value) []reflect.Value {
	cache, err := i.manager.GetCache(annotation.CacheName)
	if err != nil {
		return i.invokeOriginal(target, callInfo.methodName, args)
	}

	if annotation.Before {
		callInfo.ctx.CacheName = annotation.CacheName
		cacheKey, err := i.evaluator.EvaluateToString(annotation.Key, callInfo.ctx)
		if err == nil {
			ctx := context.Background()
			_ = cache.Delete(ctx, cacheKey)
		}
	}

	results := i.invokeOriginal(target, callInfo.methodName, args)

	if !annotation.Before {
		callInfo.ctx.CacheName = annotation.CacheName
		if len(results) > 0 {
			callInfo.ctx.SetResult(results[0].Interface())
		}
		cacheKey, err := i.evaluator.EvaluateToString(annotation.Key, callInfo.ctx)
		if err == nil {
			ctx := context.Background()
			_ = cache.Delete(ctx, cacheKey)
		}
	}

	return results
}

func (i *methodInterceptor) invokeOriginal(target interface{}, methodName string, args []reflect.Value) []reflect.Value {
	targetValue := reflect.ValueOf(target)
	method := targetValue.MethodByName(methodName)

	if !method.IsValid() {
		return []reflect.Value{}
	}

	return method.Call(args)
}

func (i *methodInterceptor) parseTTL(ttlExpr string, ctx *spel.EvaluationContext) time.Duration {
	if ttlExpr == "" {
		return 30 * time.Minute
	}

	// 尝试作为 SpEL 表达式求值
	result, err := i.evaluator.EvaluateToInt(ttlExpr, ctx)
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

func (i *methodInterceptor) isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case int8:
		return v != 0
	case int16:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case uint:
		return v != 0
	case uint8:
		return v != 0
	case uint16:
		return v != 0
	case uint32:
		return v != 0
	case uint64:
		return v != 0
	case float32:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v != "" && strings.ToLower(v) != "false"
	default:
		return true
	}
}

var _ MethodInterceptor = (*methodInterceptor)(nil)
