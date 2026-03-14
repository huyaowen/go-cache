package cache

import (
	"sync"

	"github.com/coderiser/go-cache/pkg/proxy"
)

var (
	globalAnnotations = make(map[string]map[string]*proxy.CacheAnnotation)
	registryMu        sync.RWMutex
)

// init 初始化时注册全局注解获取函数到 proxy 包，并启动自动扫描
func init() {
	proxy.SetGlobalAnnotationGetter(GetAllAnnotations)
	// 自动扫描并注册注解（运行时解析源代码）
	// 这替代了代码生成器生成的 auto_register.go
	go AutoScanAndRegister()
}

// RegisterGlobalAnnotation 注册全局注解 (供代码生成器调用)
//
// 这是方案 G 的核心机制：通过 init() 自动注册所有带注解的方法，
// 用户无需手动注册，实现"注解后直接使用"的体验。
//
// 此函数由 go generate 生成的代码调用，用户不应手动调用。
//
// 生成的代码示例:
//   // .cache-gen/auto_register.go
//   func init() {
//       cache.RegisterGlobalAnnotation("ProductService", "GetProduct", &proxy.CacheAnnotation{
//           Type:      "cacheable",
//           CacheName: "products",
//           Key:       "#id",
//           TTL:       "1h",
//       })
//   }
func RegisterGlobalAnnotation(typeName, methodName string, annotation *proxy.CacheAnnotation) {
	if annotation == nil {
		return
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if globalAnnotations[typeName] == nil {
		globalAnnotations[typeName] = make(map[string]*proxy.CacheAnnotation)
	}
	globalAnnotations[typeName][methodName] = annotation
}

// GetGlobalAnnotation 获取全局注解
//
// 根据类型名和方法名获取已注册的注解。
// 返回 nil 表示该方法没有缓存注解。
//
// 使用示例:
//   ann := cache.GetGlobalAnnotation("ProductService", "GetProduct")
//   if ann != nil {
//       // 该方法有缓存注解
//   }
func GetGlobalAnnotation(typeName, methodName string) *proxy.CacheAnnotation {
	registryMu.RLock()
	defer registryMu.RUnlock()

	if methods, ok := globalAnnotations[typeName]; ok {
		return methods[methodName]
	}
	return nil
}

// GetAllAnnotations 获取某类型的所有注解
//
// 返回指定类型的所有方法注解映射。
// 主要用于代码生成和调试。
func GetAllAnnotations(typeName string) map[string]*proxy.CacheAnnotation {
	registryMu.RLock()
	defer registryMu.RUnlock()

	if methods, ok := globalAnnotations[typeName]; ok {
		// 返回副本，避免外部修改
		result := make(map[string]*proxy.CacheAnnotation, len(methods))
		for k, v := range methods {
			result[k] = v
		}
		return result
	}
	return nil
}

// HasGlobalAnnotation 检查某方法是否有全局注解
//
// 快速判断方法是否需要缓存拦截。
func HasGlobalAnnotation(typeName, methodName string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()

	if methods, ok := globalAnnotations[typeName]; ok {
		_, exists := methods[methodName]
		return exists
	}
	return false
}

// ListGlobalTypes 列出所有已注册的类型
//
// 主要用于调试和统计。
func ListGlobalTypes() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	types := make([]string, 0, len(globalAnnotations))
	for typeName := range globalAnnotations {
		types = append(types, typeName)
	}
	return types
}

// ClearGlobalAnnotations 清空所有全局注解 (仅用于测试)
//
// ⚠️ 仅用于测试场景，生产环境不应使用。
func ClearGlobalAnnotations() {
	registryMu.Lock()
	defer registryMu.Unlock()
	globalAnnotations = make(map[string]map[string]*proxy.CacheAnnotation)
}
