package cache

import (
	"testing"

	"github.com/coderiser/go-cache/pkg/core"
	"github.com/coderiser/go-cache/pkg/proxy"
	"github.com/stretchr/testify/assert"
)

func TestGetGlobalManager_LazyLoading(t *testing.T) {
	// 重置全局状态
	ResetGlobalManager()

	// 第一次调用应该创建 Manager
	manager1 := GetGlobalManager()
	assert.NotNil(t, manager1)

	// 第二次调用应该返回同一个 Manager
	manager2 := GetGlobalManager()
	assert.NotNil(t, manager2)
	assert.Equal(t, manager1, manager2)
}

func TestSetGlobalManager_Custom(t *testing.T) {
	// 重置全局状态
	ResetGlobalManager()

	// 创建自定义 Manager
	customManager := core.NewCacheManager()

	// 设置自定义 Manager
	SetGlobalManager(customManager)

	// GetGlobalManager 应该返回自定义的 Manager
	manager := GetGlobalManager()
	assert.Equal(t, customManager, manager)
}

func TestCloseGlobalManager(t *testing.T) {
	// 重置全局状态
	ResetGlobalManager()

	// 获取 Manager (触发创建)
	manager := GetGlobalManager()
	assert.NotNil(t, manager)

	// 关闭
	CloseGlobalManager()

	// 再次获取应该创建新的 Manager
	newManager := GetGlobalManager()
	assert.NotNil(t, newManager)
	assert.NotEqual(t, manager, newManager)
}

func TestRegisterGlobalAnnotation(t *testing.T) {
	// 重置全局状态
	ClearGlobalAnnotations()

	// 注册注解
	annotation := &proxy.CacheAnnotation{
		Type:      "cacheable",
		CacheName: "products",
		Key:       "#id",
		TTL:       "1h",
	}
	RegisterGlobalAnnotation("ProductService", "GetProduct", annotation)

	// 获取注解
	retrieved := GetGlobalAnnotation("ProductService", "GetProduct")
	assert.NotNil(t, retrieved)
	assert.Equal(t, "cacheable", retrieved.Type)
	assert.Equal(t, "products", retrieved.CacheName)
	assert.Equal(t, "#id", retrieved.Key)
	assert.Equal(t, "1h", retrieved.TTL)
}

func TestGetGlobalAnnotation_NotFound(t *testing.T) {
	// 重置全局状态
	ClearGlobalAnnotations()

	// 查询不存在的注解
	annotation := GetGlobalAnnotation("NonExistent", "Method")
	assert.Nil(t, annotation)
}

func TestGetAllAnnotations(t *testing.T) {
	// 重置全局状态
	ClearGlobalAnnotations()

	// 注册多个注解
	RegisterGlobalAnnotation("ProductService", "GetProduct", &proxy.CacheAnnotation{
		Type:      "cacheable",
		CacheName: "products",
	})
	RegisterGlobalAnnotation("ProductService", "UpdateProduct", &proxy.CacheAnnotation{
		Type:      "cacheput",
		CacheName: "products",
	})
	RegisterGlobalAnnotation("UserService", "GetUser", &proxy.CacheAnnotation{
		Type:      "cacheable",
		CacheName: "users",
	})

	// 获取 ProductService 的所有注解
	annotations := GetAllAnnotations("ProductService")
	assert.NotNil(t, annotations)
	assert.Len(t, annotations, 2)
	assert.Contains(t, annotations, "GetProduct")
	assert.Contains(t, annotations, "UpdateProduct")
}

func TestHasGlobalAnnotation(t *testing.T) {
	// 重置全局状态
	ClearGlobalAnnotations()

	// 注册注解
	RegisterGlobalAnnotation("ProductService", "GetProduct", &proxy.CacheAnnotation{
		Type: "cacheable",
	})

	// 测试存在
	assert.True(t, HasGlobalAnnotation("ProductService", "GetProduct"))

	// 测试不存在
	assert.False(t, HasGlobalAnnotation("ProductService", "NonExistent"))
	assert.False(t, HasGlobalAnnotation("NonExistent", "Method"))
}

func TestListGlobalTypes(t *testing.T) {
	// 重置全局状态
	ClearGlobalAnnotations()

	// 注册注解
	RegisterGlobalAnnotation("ProductService", "GetProduct", &proxy.CacheAnnotation{})
	RegisterGlobalAnnotation("UserService", "GetUser", &proxy.CacheAnnotation{})

	// 获取所有类型
	types := ListGlobalTypes()
	assert.Len(t, types, 2)
	assert.Contains(t, types, "ProductService")
	assert.Contains(t, types, "UserService")
}

func TestClearGlobalAnnotations(t *testing.T) {
	// 重置全局状态
	ClearGlobalAnnotations()

	// 注册注解
	RegisterGlobalAnnotation("ProductService", "GetProduct", &proxy.CacheAnnotation{})

	// 验证已注册
	assert.NotNil(t, GetGlobalAnnotation("ProductService", "GetProduct"))

	// 清空
	ClearGlobalAnnotations()

	// 验证已清空
	assert.Nil(t, GetGlobalAnnotation("ProductService", "GetProduct"))
}

func TestGlobalInterceptor_Singleton(t *testing.T) {
	// 获取单例
	interceptor1 := GetGlobalInterceptor()
	assert.NotNil(t, interceptor1)

	// 再次获取应该返回同一个实例
	interceptor2 := GetGlobalInterceptor()
	assert.Equal(t, interceptor1, interceptor2)
}

func TestGlobalInterceptor_SetManager(t *testing.T) {
	interceptor := GetGlobalInterceptor()
	manager := core.NewCacheManager()
	defer manager.Close()

	// 设置 Manager
	interceptor.SetManager(manager)

	// 获取 Manager
	retrieved := interceptor.GetManager()
	assert.Equal(t, manager, retrieved)
}

// 注：parseTTL 和 isTruthy 是私有方法，测试需要重构
// 暂时跳过这些测试，后续通过集成测试验证功能
