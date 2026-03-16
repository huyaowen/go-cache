package cache

import (
	"sync"

	"github.com/coderiser/go-cache/pkg/core"
)

var (
	globalManager core.CacheManager
	initOnce      sync.Once
)

// GetGlobalManager 获取全局管理器 (懒加载，线程安全)
// 
// 这是方案 G 的核心：用户无需手动传递 Manager，框架自动管理。
// 懒加载确保只在第一次调用时创建，避免资源浪费。
//
// 使用示例:
//   func main() {
//       // 零配置：直接使用，Manager 自动创建
//       svc := cache.NewProductService()
//       
//       // 或者自定义配置 (可选)
//       manager := core.NewCacheManager()
//       cache.SetGlobalManager(manager)
//       svc := cache.NewProductService()
//   }
func GetGlobalManager() core.CacheManager {
	initOnce.Do(func() {
		if globalManager == nil {
			globalManager = core.NewCacheManager()
		}
	})
	return globalManager
}

// SetGlobalManager 设置全局管理器 (可选，供高级用户)
//
// 99% 的用户不需要调用此函数。只有在需要自定义缓存配置
// (如 Redis 后端、自定义 TTL、指标导出等) 时才使用。
//
// 注意：此函数应该在应用启动时调用一次，之后不应再修改。
// 如果在 NewXxxService() 之后调用，已创建的服务不会使用新 Manager。
//
// 使用示例:
//   func main() {
//       // 自定义 Manager (Redis 后端)
//       manager := core.NewCacheManager()
//       // 配置 Redis...
//       cache.SetGlobalManager(manager)
//       
//       // 之后所有 NewXxxService() 都使用此 Manager
//       svc := cache.NewProductService()
//   }
func SetGlobalManager(manager core.CacheManager) {
	globalManager = manager
	// 重置 initOnce，允许 GetGlobalManager 返回新设置的值
	// 注意：这不是线程安全的，应该在单线程初始化时调用
	initOnce = sync.Once{}
}

// CloseGlobalManager 关闭全局管理器 (应用退出时调用)
//
// 释放缓存资源，关闭后端连接 (如 Redis)。
// 应该在应用优雅关闭时调用。
//
// 使用示例:
//   func main() {
//       defer cache.CloseGlobalManager()
//       // 应用逻辑...
//   }
func CloseGlobalManager() {
	if globalManager != nil {
		globalManager.Close()
		globalManager = nil
		initOnce = sync.Once{}
	}
}

// ResetGlobalManager 重置全局管理器 (主要用于测试)
//
// ⚠️ 仅用于测试场景，生产环境不应使用。
func ResetGlobalManager() {
	globalManager = nil
	initOnce = sync.Once{}
}
