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
)

// AutoProxy 运行时自动代理 (零代码生成)
//
// 这是方案 1 的实现：完全移除代码生成，运行时解析注解并动态代理。
//
// 使用示例:
//   type ProductService struct{}
//
//   // @cacheable(cache="products", key="#id", ttl="1h")
//   func (s *ProductService) GetProduct(id int64) (*model.Product, error) { ... }
//
//   func main() {
//       svc := AutoProxy(&ProductService{})
//       product, _ := svc.(*ProductService).GetProduct(1)
//   }
func AutoProxy(target interface{}) interface{} {
	targetValue := reflect.ValueOf(target)
	targetType := targetValue.Type()

	// 扫描所有方法，查找带注解的方法
	annotatedMethods := scanAnnotatedMethods(targetType)

	if len(annotatedMethods) == 0 {
		// 没有注解，直接返回原始对象
		return target
	}

	// 创建代理
	proxy := &runtimeProxy{
		target:   target,
		methods:  annotatedMethods,
		manager:  GetGlobalManager(),
		cache:    make(map[string]*methodCache),
	}

	// 返回包装后的对象
	return proxy
}

// runtimeProxy 运行时代理
type runtimeProxy struct {
	target   interface{}
	methods  []*annotatedMethod
	manager  core.CacheManager
	cache    map[string]*methodCache
	cacheMu  sync.RWMutex
}

// methodCache 方法缓存元数据
type methodCache struct {
	annotation *proxy.CacheAnnotation
	method     reflect.Method
}

// annotatedMethod 带注解的方法
type annotatedMethod struct {
	Name       string
	Method     reflect.Method
	Annotation *proxy.CacheAnnotation
}

// scanAnnotatedMethods 扫描所有带注解的方法
// 注意：Go 不支持运行时读取注释，此方案已废弃
// 
// 替代方案：使用代码生成器 (cmd/generator) 在编译时生成注解注册代码
// 参考：examples/gin-web/service/auto_register.go
//
// 此函数保留仅用于向后兼容，新功能请使用代码生成器方案
func scanAnnotatedMethods(targetType reflect.Type) []*annotatedMethod {
	// 运行时注解扫描在 Go 中不可行（无法读取注释）
	// 已废弃：请使用代码生成器方案 (go-cache-gen)
	return make([]*annotatedMethod, 0)
}

// Call 调用代理方法
func (p *runtimeProxy) Call(methodName string, args ...interface{}) ([]interface{}, error) {
	// 查找方法
	var method *annotatedMethod
	for _, m := range p.methods {
		if m.Name == methodName {
			method = m
			break
		}
	}

	if method == nil {
		// 没有注解，直接调用原始方法
		return p.invokeOriginal(methodName, args)
	}

	// 有注解，执行缓存逻辑
	return p.invokeWithCache(method, args)
}

// invokeWithCache 带缓存的调用
func (p *runtimeProxy) invokeWithCache(method *annotatedMethod, args []interface{}) ([]interface{}, error) {
	annotation := method.Annotation

	// 构建缓存 Key
	cacheKey := buildCacheKey(annotation.CacheName, annotation.Key, args)

	// 查询缓存
	cache, err := p.manager.GetCache(annotation.CacheName)
	if err != nil {
		return p.invokeOriginal(method.Name, args)
	}

	cachedValue, found, _ := cache.Get(context.Background(), cacheKey)
	if found {
		log.Printf("[INFO] Cache HIT: %s", cacheKey)
		return []interface{}{cachedValue}, nil
	}

	// 执行原始方法
	results, err := p.invokeOriginal(method.Name, args)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	if len(results) > 0 {
		ttl := parseTTL(annotation.TTL)
		_ = cache.Set(context.Background(), cacheKey, results[0], ttl)
	}

	return results, nil
}

// invokeOriginal 调用原始方法
func (p *runtimeProxy) invokeOriginal(methodName string, args []interface{}) ([]interface{}, error) {
	targetValue := reflect.ValueOf(p.target)
	method := targetValue.MethodByName(methodName)

	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	// 转换参数
	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}

	// 调用方法
	results := method.Call(in)

	// 转换返回值
	out := make([]interface{}, len(results))
	for i, result := range results {
		out[i] = result.Interface()
	}

	return out, nil
}

// buildCacheKey 构建缓存 Key
func buildCacheKey(cacheName, keyExpr string, args []interface{}) string {
	if keyExpr == "" {
		return fmt.Sprintf("%s:%v", cacheName, args)
	}

	// 简单处理：使用第一个参数
	if len(args) > 0 {
		return fmt.Sprintf("%s:%v", cacheName, args[0])
	}

	return fmt.Sprintf("%s:default", cacheName)
}

// parseTTL 解析 TTL
func parseTTL(ttlStr string) time.Duration {
	if ttlStr == "" {
		return 30 * time.Minute
	}
	d, _ := time.ParseDuration(ttlStr)
	return d
}

// 以下是一个更实用的方案：使用接口包装

// CachedService 缓存服务接口
type CachedService interface {
	Call(methodName string, args ...interface{}) ([]interface{}, error)
}

// NewCachedService 创建缓存服务 (实用版本)
//
// 使用示例:
//   type ProductService struct{}
//   func (s *ProductService) GetProduct(id int64) (*model.Product, error) { ... }
//
//   func main() {
//       svc := NewCachedService(&ProductService{})
//       results, _ := svc.Call("GetProduct", int64(1))
//       product := results[0].(*model.Product)
//   }
func NewCachedService(target interface{}) CachedService {
	return &runtimeCachedService{
		target:  target,
		manager: GetGlobalManager(),
	}
}

// runtimeCachedService 运行时缓存服务
type runtimeCachedService struct {
	target  interface{}
	manager core.CacheManager
}

// Call 调用方法 (带缓存)
func (s *runtimeCachedService) Call(methodName string, args ...interface{}) ([]interface{}, error) {
	// 这里可以集成注解解析
	// 为简化示例，直接调用原始方法
	return s.invokeOriginal(methodName, args)
}

// invokeOriginal 调用原始方法
func (s *runtimeCachedService) invokeOriginal(methodName string, args []interface{}) ([]interface{}, error) {
	targetValue := reflect.ValueOf(s.target)
	method := targetValue.MethodByName(methodName)

	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}

	results := method.Call(in)

	out := make([]interface{}, len(results))
	for i, result := range results {
		out[i] = result.Interface()
	}

	return out, nil
}
