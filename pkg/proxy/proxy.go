package proxy

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/go-cache-framework/pkg/core"
)

// ProxyFactory 代理工厂接口
type ProxyFactory interface {
	Create(target interface{}) (Proxy, error)
}

// Proxy 代理接口
type Proxy interface {
	Call(methodName string, args []reflect.Value) []reflect.Value
	GetTarget() interface{}
}

// proxyImpl 代理实现 - 简化版本，不使用 reflect.MakeFunc
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

var _ Proxy = (*proxyImpl)(nil)
var _ ProxyFactory = (*proxyFactoryImpl)(nil)
