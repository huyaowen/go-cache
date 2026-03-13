package proxy

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/coderiser/go-cache/pkg/core"
)

// ProxyFactory 代理工厂接口
type ProxyFactory interface {
	Create(target interface{}) (Proxy, error)
}

// Proxy 代理接口
type Proxy interface {
	Call(methodName string, args []reflect.Value) []reflect.Value
	GetTarget() interface{}
	GetTypeName() string
	RegisterAnnotation(methodName string, annotation *CacheAnnotation)
}

// proxyImpl 代理实现
type proxyImpl struct {
	target      interface{}
	targetType  reflect.Type
	interceptor *methodInterceptor
	manager     core.CacheManager
	mu          sync.RWMutex
}

// proxyFactoryImpl 代理工厂实现
type proxyFactoryImpl struct {
	manager core.CacheManager
}

// NewProxyFactory 创建代理工厂
func NewProxyFactory(manager core.CacheManager) ProxyFactory {
	return &proxyFactoryImpl{manager: manager}
}

// Create 创建代理
func (f *proxyFactoryImpl) Create(target interface{}) (Proxy, error) {
	if target == nil {
		return nil, fmt.Errorf("target cannot be nil")
	}

	targetType := reflect.TypeOf(target)
	if targetType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("target must be a pointer")
	}

	elemType := targetType.Elem()
	if elemType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("target must be a pointer to struct")
	}

	interceptor := newMethodInterceptor(f.manager)

	return &proxyImpl{
		target:      target,
		targetType:  targetType,
		interceptor: interceptor,
		manager:     f.manager,
	}, nil
}

// Call 调用目标方法（带拦截）
func (p *proxyImpl) Call(methodName string, args []reflect.Value) []reflect.Value {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.target == nil {
		return []reflect.Value{}
	}
	
	return p.interceptor.Intercept(p.target, methodName, args)
}

// GetTarget 获取目标对象
func (p *proxyImpl) GetTarget() interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.target
}

// GetInterceptor 获取拦截器（用于注册注解）
func (p *proxyImpl) GetInterceptor() *methodInterceptor {
	return p.interceptor
}

// GetTypeName 获取类型名称
func (p *proxyImpl) GetTypeName() string {
	if p.targetType.Kind() == reflect.Ptr {
		return p.targetType.Elem().Name()
	}
	return p.targetType.Name()
}

// RegisterAnnotation 注册方法注解
func (p *proxyImpl) RegisterAnnotation(methodName string, annotation *CacheAnnotation) {
	p.interceptor.RegisterAnnotation(methodName, annotation)
}

// Invoke 通过反射调用代理方法（便捷方法）
func (p *proxyImpl) Invoke(methodName string, args ...interface{}) ([]interface{}, error) {
	targetValue := reflect.ValueOf(p.target)
	method := targetValue.MethodByName(methodName)
	
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}
	
	methodType := method.Type()
	
	// 转换参数
	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		if i < methodType.NumIn() {
			paramType := methodType.In(i)
			argValue := reflect.ValueOf(arg)
			if argValue.Type().AssignableTo(paramType) {
				in[i] = argValue
			} else {
				// 尝试转换
				if argValue.CanConvert(paramType) {
					in[i] = argValue.Convert(paramType)
				} else {
					return nil, fmt.Errorf("argument %d type mismatch: got %T, want %s", i, arg, paramType)
				}
			}
		}
	}
	
	// 调用拦截器
	results := p.interceptor.Intercept(p.target, methodName, in)
	
	// 转换返回值
	out := make([]interface{}, len(results))
	for i, result := range results {
		out[i] = result.Interface()
	}
	
	return out, nil
}

var _ Proxy = (*proxyImpl)(nil)
var _ ProxyFactory = (*proxyFactoryImpl)(nil)
