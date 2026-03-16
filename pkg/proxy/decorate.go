package proxy

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/coderiser/go-cache/pkg/core"
)

// AutoDecorate 自动装饰器接口
type AutoDecorate interface {
	Decorate(target interface{}) error
	DecorateAndReturn(target interface{}) (Proxy, error)
	Register(name string, target interface{}) error
	GetProxy(name string) (Proxy, error)
	GetAllProxies() map[string]Proxy
	Interceptor() *methodInterceptor
}

// autoDecorateImpl 自动装饰器实现
type autoDecorateImpl struct {
	mu          sync.RWMutex
	manager     core.CacheManager
	factory     ProxyFactory
	registered  map[string]interface{}
	proxies     map[string]Proxy
	interceptor *methodInterceptor
}

// Interceptor 获取拦截器（用于注册注解）
func (a *autoDecorateImpl) Interceptor() *methodInterceptor {
	return a.interceptor
}

var (
	globalAutoDecorate *autoDecorateImpl
	globalInitOnce     sync.Once
)

// GetAutoDecorate 获取全局自动装饰器
func GetAutoDecorate(manager core.CacheManager) AutoDecorate {
	globalInitOnce.Do(func() {
		globalAutoDecorate = &autoDecorateImpl{
			manager:     manager,
			factory:     NewProxyFactory(manager),
			registered:  make(map[string]interface{}),
			proxies:     make(map[string]Proxy),
			interceptor: newMethodInterceptor(manager),
		}
	})
	return globalAutoDecorate
}

// NewAutoDecorate 创建新的自动装饰器
func NewAutoDecorate(manager core.CacheManager) AutoDecorate {
	return &autoDecorateImpl{
		manager:     manager,
		factory:     NewProxyFactory(manager),
		registered:  make(map[string]interface{}),
		proxies:     make(map[string]Proxy),
		interceptor: newMethodInterceptor(manager),
	}
}

// Decorate 装饰目标对象
func (a *autoDecorateImpl) Decorate(target interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if target == nil {
		return fmt.Errorf("target cannot be nil")
	}

	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr || targetValue.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer")
	}

	proxy, err := a.factory.Create(target)
	if err != nil {
		return fmt.Errorf("failed to create proxy: %w", err)
	}

	targetType := reflect.TypeOf(target)
	name := targetType.Elem().Name()
	if name == "" {
		name = fmt.Sprintf("anonymous_%p", target)
	}

	// 注册注解到拦截器
	impl, ok := proxy.(*proxyImpl)
	if ok {
		interceptor := impl.GetInterceptor()
		annotations := GetRegisteredAnnotations(name)
		if annotations != nil {
			for methodName, annotation := range annotations {
				interceptor.RegisterAnnotation(methodName, annotation)
			}
		}
	}

	a.proxies[name] = proxy
	return nil
}

// Register 注册可装饰的对象
func (a *autoDecorateImpl) Register(name string, target interface{}) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.registered[name] != nil {
		return fmt.Errorf("name '%s' already registered", name)
	}

	a.registered[name] = target

	proxy, err := a.factory.Create(target)
	if err != nil {
		return fmt.Errorf("failed to create proxy for '%s': %w", name, err)
	}

	// 注册注解到拦截器
	impl, ok := proxy.(*proxyImpl)
	if ok {
		interceptor := impl.GetInterceptor()
		annotations := GetRegisteredAnnotations(name)
		if annotations != nil {
			for methodName, annotation := range annotations {
				interceptor.RegisterAnnotation(methodName, annotation)
			}
		}
	}

	a.proxies[name] = proxy
	return nil
}

// GetProxy 获取代理对象
func (a *autoDecorateImpl) GetProxy(name string) (Proxy, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	proxy, exists := a.proxies[name]
	if !exists {
		return nil, fmt.Errorf("no proxy registered for '%s'", name)
	}

	return proxy, nil
}

// GetAllProxies 获取所有代理对象
func (a *autoDecorateImpl) GetAllProxies() map[string]Proxy {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]Proxy, len(a.proxies))
	for k, v := range a.proxies {
		result[k] = v
	}
	return result
}

// DecorateAndReturn 装饰目标对象并返回代理对象
// 返回的 Proxy 接口提供 Call 方法用于调用目标方法
//
// 使用示例：
//   type UserService struct { ... }
//   func (s *UserService) GetUser(id int) *User { ... }
//
//   var proxyService proxy.Proxy
//   func init() {
//       manager := core.NewCacheManager()
//       userService := &UserService{}
//       factory := proxy.NewProxyFactory(manager)
//       proxyObj, _ := factory.Create(userService)
//       proxyService = proxyObj.(proxy.Proxy)
//       // 注册注解
//       annotations := proxy.GetRegisteredAnnotations("UserService")
//       for name, ann := range annotations {
//           proxyService.RegisterAnnotation(name, ann)
//       }
//   }
//
//   // 调用方法
//   results := proxyService.Call("GetUser", []reflect.Value{reflect.ValueOf(1)})
//   user := results[0].Interface().(*User)
func (a *autoDecorateImpl) DecorateAndReturn(target interface{}) (Proxy, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if target == nil {
		return nil, fmt.Errorf("target cannot be nil")
	}

	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr || targetValue.IsNil() {
		return nil, fmt.Errorf("target must be a non-nil pointer")
	}

	proxy, err := a.factory.Create(target)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy: %w", err)
	}

	targetType := reflect.TypeOf(target)
	name := targetType.Elem().Name()
	if name == "" {
		name = fmt.Sprintf("anonymous_%p", target)
	}

	// 注册注解到拦截器
	impl, ok := proxy.(*proxyImpl)
	if ok {
		interceptor := impl.GetInterceptor()
		annotations := GetRegisteredAnnotations(name)
		if annotations != nil {
			for methodName, annotation := range annotations {
				interceptor.RegisterAnnotation(methodName, annotation)
			}
		}
	}

	a.proxies[name] = proxy

	// 返回 Proxy 接口（调用者使用 Call 方法调用）
	return a.proxies[name], nil
}

// 全局注解注册表
var (
	globalAnnotationRegistry = make(map[string]map[string]*CacheAnnotation)
	registryMu               sync.RWMutex
)

// RegisterAnnotation 注册方法注解（供代码生成使用）
func RegisterAnnotation(manager core.CacheManager, typeName, methodName string, annotation *CacheAnnotation) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if globalAnnotationRegistry[typeName] == nil {
		globalAnnotationRegistry[typeName] = make(map[string]*CacheAnnotation)
	}
	globalAnnotationRegistry[typeName][methodName] = annotation
}

// globalAnnotationGetter 全局注解获取函数（由 cache 包设置）
var globalAnnotationGetter func(string) map[string]*CacheAnnotation

// SetGlobalAnnotationGetter 设置全局注解获取函数
// 由 cache 包在 init() 中调用，注册获取注解的函数
func SetGlobalAnnotationGetter(getter func(string) map[string]*CacheAnnotation) {
	globalAnnotationGetter = getter
}

// GetRegisteredAnnotations 获取已注册的注解
// 优先从 cache 包的全局注册表读取（代码生成器注册的位置）
// 回退到 proxy 包的本地注册表（向后兼容）
func GetRegisteredAnnotations(typeName string) map[string]*CacheAnnotation {
	// 1. 优先从 cache 包读取（代码生成器注册的位置）
	if globalAnnotationGetter != nil {
		cacheAnnotations := globalAnnotationGetter(typeName)
		if cacheAnnotations != nil && len(cacheAnnotations) > 0 {
			return cacheAnnotations
		}
	}
	
	// 2. 回退到本地注册表（向后兼容）
	registryMu.RLock()
	defer registryMu.RUnlock()

	if annotations, exists := globalAnnotationRegistry[typeName]; exists {
		result := make(map[string]*CacheAnnotation, len(annotations))
		for k, v := range annotations {
			result[k] = v
		}
		return result
	}
	return nil
}

var _ AutoDecorate = (*autoDecorateImpl)(nil)
