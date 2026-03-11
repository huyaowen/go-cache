package proxy

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/go-cache-framework/pkg/core"
	"github.com/go-cache-framework/pkg/spel"
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
	manager       core.CacheManager
	evaluator     *spel.SpELEvaluator
	methodCache   map[string]*CacheAnnotation
	methodCacheMu sync.RWMutex
}

func newMethodInterceptor(manager core.CacheManager) *methodInterceptor {
	return &methodInterceptor{
		manager:     manager,
		evaluator:   spel.NewSpELEvaluator(),
		methodCache: make(map[string]*CacheAnnotation),
	}
}

// Intercept 拦截方法调用
func (i *methodInterceptor) Intercept(target interface{}, methodName string, args []reflect.Value) []reflect.Value {
	annotation := i.getAnnotation(methodName)
	if annotation == nil {
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

	for idx, arg := range args {
		value := arg.Interface()
		ctx.SetArgByIndex(idx, value)
		ctx.SetArg(fmt.Sprintf("p%d", idx), value)
	}

	return &callInfo{
		methodName: methodName,
		methodType: targetType,
		ctx:        ctx,
	}
}

func (i *methodInterceptor) getAnnotation(methodName string) *CacheAnnotation {
	i.methodCacheMu.RLock()
	defer i.methodCacheMu.RUnlock()
	return i.methodCache[methodName]
}

// RegisterAnnotation 注册方法注解
func (i *methodInterceptor) RegisterAnnotation(methodName string, annotation *CacheAnnotation) {
	i.methodCacheMu.Lock()
	defer i.methodCacheMu.Unlock()
	i.methodCache[methodName] = annotation
}

func (i *methodInterceptor) handleCacheable(target interface{}, callInfo *callInfo, annotation *CacheAnnotation, args []reflect.Value) []reflect.Value {
	cache, err := i.manager.GetCache(annotation.CacheName)
	if err != nil {
		return i.invokeOriginal(target, callInfo.methodName, args)
	}

	callInfo.ctx.CacheName = annotation.CacheName
	cacheKey, err := i.evaluator.EvaluateToString(annotation.Key, callInfo.ctx)
	if err != nil {
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
		return []reflect.Value{reflect.ValueOf(value)}
	}

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
	result, err := i.evaluator.EvaluateToInt(ttlExpr, ctx)
	if err != nil {
		return 30 * time.Minute
	}
	return time.Duration(result) * time.Second
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
