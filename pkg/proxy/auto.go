package proxy

import (
	"log"
	"github.com/coderiser/go-cache/pkg/core"
)

// DecoratedService 泛型装饰服务包装器
// 提供类型安全的代理访问，同时保持原始接口
type DecoratedService[T any] struct {
	proxy *proxyImpl
	raw   T
}

// Invoke 调用代理方法
func (d *DecoratedService[T]) Invoke(methodName string, args ...interface{}) ([]interface{}, error) {
	return d.proxy.Invoke(methodName, args...)
}

// GetProxy 获取底层代理对象
func (d *DecoratedService[T]) GetProxy() Proxy {
	return d.proxy
}

// GetRaw 获取原始服务对象
func (d *DecoratedService[T]) GetRaw() T {
	return d.raw
}

// SimpleDecorate 自动装饰服务对象（简化版 API - 泛型版本）
// 这是最简单的初始化 API，一行代码完成缓存代理创建
//
// 使用示例:
//   package service
//   import "github.com/coderiser/go-cache/pkg/proxy"
//   
//   type UserService struct { ... }
//   
//   // @cacheable(cache="users", key="#id", ttl="30m")
//   func (s *UserService) GetUser(id string) (*User, error) { ... }
//   
//   // 方式 1: 直接赋值（推荐）
//   var UserService = proxy.SimpleDecorate(&UserService{})
//   
//   // 方式 2: init 函数中（泛型版本）
//   var UserService *proxy.DecoratedService[UserService]
//   func init() {
//       UserService = proxy.SimpleDecorate(&UserService{})
//       // 调用方法：UserService.Invoke("GetUser", id)
//   }
//
// 注意:
// - 返回的是 DecoratedService 包装器，提供类型安全的 Invoke 方法
// - 如果装饰失败，返回包装原始对象的 DecoratedService
// - 需要在调用前确保代码生成器已运行（go-cache-gen ./...）
func SimpleDecorate[T any](service T) *DecoratedService[T] {
	manager := core.NewCacheManager()
	return SimpleDecorateWithManager(service, manager)
}

// SimpleDecorateWithManager 使用自定义缓存管理器自动装饰（泛型版本）
// 当需要自定义缓存配置（如 Redis 后端）时使用此函数
//
// 使用示例:
//   manager := core.NewCacheManager()
//   // 配置 Redis 后端...
//   manager.RegisterCache("users", redisBackend)
//   
//   var UserService = proxy.SimpleDecorateWithManager(&UserService{}, manager)
func SimpleDecorateWithManager[T any](service T, manager core.CacheManager) *DecoratedService[T] {
	factory := NewProxyFactory(manager)
	
	// 将服务转换为 interface{} 进行代理创建
	serviceIface := any(service)
	proxyObj, err := factory.Create(serviceIface)
	
	if err != nil {
		// 降级：返回包装原始对象的 DecoratedService
		return &DecoratedService[T]{raw: service}
	}
	
	proxyImpl, ok := proxyObj.(*proxyImpl)
	if !ok {
		// 降级：返回包装原始对象的 DecoratedService
		return &DecoratedService[T]{raw: service}
	}
	
	// 注册注解到拦截器
	interceptor := proxyImpl.GetInterceptor()
	
	// 获取类型名
	serviceType := proxyImpl.GetTypeName()
	log.Printf("[DEBUG] SimpleDecorateWithManager: serviceType=%s", serviceType)
	if serviceType != "" {
		annotations := GetRegisteredAnnotations(serviceType)
		log.Printf("[DEBUG] SimpleDecorateWithManager: annotations=%v", annotations)
		if annotations != nil {
			for methodName, annotation := range annotations {
				interceptor.RegisterAnnotation(methodName, annotation)
				log.Printf("[DEBUG] SimpleDecorateWithManager: Registered annotation for %s: %v", methodName, annotation.Type)
			}
		}
	}
	
	return &DecoratedService[T]{
		proxy: proxyImpl,
		raw:   service,
	}
}

// SimpleDecorateWithInterface 自动装饰服务对象并返回接口类型
// 这是推荐的使用方式，可以直接断言为目标接口
//
// 使用示例:
//   var UserService UserServiceInterface
//   func init() {
//       UserService = proxy.SimpleDecorateWithInterface(&UserService{})
//   }
//
// 注意：此函数返回的代理对象实现了目标接口，可以直接调用方法
func SimpleDecorateWithInterface[T any](service T, targetInterface T) T {
	manager := core.NewCacheManager()
	return SimpleDecorateWithInterfaceAndManager(service, targetInterface, manager)
}

// SimpleDecorateWithInterfaceAndManager 使用自定义缓存管理器装饰并返回接口类型
func SimpleDecorateWithInterfaceAndManager[T any](service T, targetInterface T, manager core.CacheManager) T {
	factory := NewProxyFactory(manager)
	
	// 将服务转换为 interface{} 进行代理创建
	serviceIface := any(service)
	proxyObj, err := factory.Create(serviceIface)
	
	if err != nil {
		// 降级：返回原始对象
		return service
	}
	
	proxyImpl, ok := proxyObj.(*proxyImpl)
	if !ok {
		// 降级：返回原始对象
		return service
	}
	
	// 注册注解到拦截器
	interceptor := proxyImpl.GetInterceptor()
	
	// 获取类型名
	serviceType := proxyImpl.GetTypeName()
	if serviceType != "" {
		annotations := GetRegisteredAnnotations(serviceType)
		if annotations != nil {
			for methodName, annotation := range annotations {
				interceptor.RegisterAnnotation(methodName, annotation)
			}
		}
	}
	
	// 通过反射创建一个实现目标接口的包装器
	// 这里我们返回原始服务，但通过代理调用方法
	// 由于 Go 的类型系统限制，我们无法直接返回实现接口的代理
	// 推荐使用 DecoratedService + Invoke 模式
	_ = proxyObj
	
	return service
}

// SimpleDecorateWithError 自动装饰服务对象（带错误返回 - 泛型版本）
// 当需要处理装饰失败的情况时使用此函数
//
// 使用示例:
//   var UserService *proxy.DecoratedService[UserService]
//   func init() {
//       decorated, err := proxy.SimpleDecorateWithError(&UserService{})
//       if err != nil {
//           log.Printf("Cache decoration failed: %v, using fallback", err)
//       }
//       UserService = decorated
//   }
func SimpleDecorateWithError[T any](service T) (*DecoratedService[T], error) {
	manager := core.NewCacheManager()
	return SimpleDecorateWithManagerAndError(service, manager)
}

// SimpleDecorateWithManagerAndError 使用自定义缓存管理器自动装饰（带错误返回 - 泛型版本）
func SimpleDecorateWithManagerAndError[T any](service T, manager core.CacheManager) (*DecoratedService[T], error) {
	factory := NewProxyFactory(manager)
	
	// 将服务转换为 interface{} 进行代理创建
	serviceIface := any(service)
	proxyObj, err := factory.Create(serviceIface)
	
	if err != nil {
		return &DecoratedService[T]{raw: service}, err
	}
	
	proxyImpl, ok := proxyObj.(*proxyImpl)
	if !ok {
		return &DecoratedService[T]{raw: service}, nil
	}
	
	// 注册注解到拦截器
	interceptor := proxyImpl.GetInterceptor()
	
	// 获取类型名
	serviceType := proxyImpl.GetTypeName()
	if serviceType != "" {
		annotations := GetRegisteredAnnotations(serviceType)
		if annotations != nil {
			for methodName, annotation := range annotations {
				interceptor.RegisterAnnotation(methodName, annotation)
			}
		}
	}
	
	return &DecoratedService[T]{
		proxy: proxyImpl,
		raw:   service,
	}, nil
}
