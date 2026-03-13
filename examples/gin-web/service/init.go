package service

import (
	"log"
	"time"

	"github.com/coderiser/go-cache/pkg/backend"
	"github.com/coderiser/go-cache/pkg/core"
	"github.com/coderiser/go-cache/pkg/proxy"
)

var (
	// cacheManager 全局缓存管理器
	cacheManager core.CacheManager
	
	// UserService 全局用户服务实例（已装饰）
	// 方案 A: 代码生成器生成类型安全的包装器
	// 业务代码零侵入，handler 直接使用接口调用
	UserService UserServiceInterface
)

// InitCache 初始化缓存和装饰服务
// 在 main 函数启动时调用
// 
// 方案 A 使用方式：
// 1. 在 service 文件中添加：//go:generate go-cache-gen ./...
// 2. 运行：go generate ./...
// 3. 生成的包装器自动实现 UserServiceInterface 接口
// 4. handler 直接使用：service.UserService.GetUser(id)
func InitCache() {
	log.Println("[INFO] Initializing cache manager...")
	
	// 1. 创建缓存管理器
	cacheManager = core.NewCacheManager()
	
	// 2. 配置 users 缓存（Memory 后端）
	memoryBackend, err := backend.NewMemoryBackend(&backend.CacheConfig{
		Name:          "users",
		DefaultTTL:    30 * time.Minute,
		MaxSize:       1000,
	})
	if err != nil {
		log.Fatalf("[ERROR] Failed to create memory backend: %v", err)
	}
	
	err = cacheManager.RegisterBackend("users", func(cfg *backend.CacheConfig) (backend.CacheBackend, error) {
		return memoryBackend, nil
	})
	if err != nil {
		log.Fatalf("[ERROR] Failed to register users backend: %v", err)
	}
	
	log.Println("[INFO] Cache manager initialized with memory backend")
	
	// 3. 创建原始服务并使用泛型装饰
	rawService := NewUserService()
	decorated := proxy.SimpleDecorateWithManager(rawService, cacheManager)
	
	// 4. 使用生成的包装器实现接口（方案 A: 代码生成）
	UserService = NewDecoratedUserService(decorated)
	
	log.Printf("[DEBUG] UserService type: %T, decorated=%v", UserService, UserService != nil)
	log.Println("[INFO] UserService decorated with cache annotations")
	log.Println("[INFO] Cache annotations registered:")
	log.Println("[INFO]   - GetUser: @cacheable(cache=\"users\", key=\"#id\", ttl=\"30m\")")
	log.Println("[INFO]   - CreateUser: @cacheput(cache=\"users\", key=\"#user.ID\", ttl=\"30m\")")
	log.Println("[INFO]   - UpdateUser: @cacheput(cache=\"users\", key=\"#id\", ttl=\"30m\")")
	log.Println("[INFO]   - DeleteUser: @cacheevict(cache=\"users\", key=\"#id\")")
	log.Println("[INFO]")
	log.Println("[INFO] 架构方案：方案 A - 代码生成器（推荐）")
	log.Println("[INFO]   - 优点：业务代码零侵入，类型安全，IDE 友好")
	log.Println("[INFO]   - 使用：service.UserService.GetUser(id)")
	log.Println("[INFO]")
	log.Println("[INFO] 生成命令：go generate ./...")
}

// GetCacheManager 获取全局缓存管理器
func GetCacheManager() core.CacheManager {
	return cacheManager
}
