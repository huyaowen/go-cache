package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coderiser/go-cache/examples/grpc-service/server"
	"github.com/coderiser/go-cache/examples/grpc-service/service"
	pb "github.com/coderiser/go-cache/examples/grpc-service/proto"
	"github.com/coderiser/go-cache/pkg/backend"
	"github.com/coderiser/go-cache/pkg/core"
	"github.com/coderiser/go-cache/pkg/proxy"
	"google.golang.org/grpc"
)

// 全局用户服务实例（使用接口类型）
// 方案 A: 代码生成器生成类型安全的包装器
var userService service.UserServiceInterface

// initUserService 初始化用户服务（使用代码生成的包装器）
func initUserService(manager core.CacheManager) {
	// 创建原始服务
	rawService := service.NewUserService()
	
	// 使用 SimpleDecorate 装饰服务
	decorated := proxy.SimpleDecorateWithManager(rawService, manager)
	
	// 使用代码生成的包装器（方案 A）
	userService = service.NewDecoratedUserService(decorated)
	
	log.Printf("[INFO] UserService decorated with code-generated wrapper")
	log.Printf("[INFO] Cache annotations registered:")
	log.Printf("[INFO]   - GetUser: @cacheable(cache=\"users\", key=\"#id\", ttl=\"30m\")")
	log.Printf("[INFO]   - CreateUser: @cacheput(cache=\"users\", key=\"#result.ID\", ttl=\"30m\")")
	log.Printf("[INFO]   - UpdateUser: @cacheput(cache=\"users\", key=\"#user.ID\", ttl=\"30m\")")
	log.Printf("[INFO]   - DeleteUser: @cacheevict(cache=\"users\", key=\"#id\")")
}

func main() {
	// 初始化日志
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	// 1. 创建缓存管理器
	manager := core.NewCacheManager()
	defer manager.Close()
	
	// 2. 配置 users 缓存
	usersConfig := &backend.CacheConfig{
		Name:       "users",
		DefaultTTL: 30 * time.Minute,
		MaxSize:    1000,
	}
	
	memoryBackend, err := backend.NewMemoryBackend(usersConfig)
	if err != nil {
		log.Fatalf("[ERROR] Failed to create memory backend: %v", err)
	}
	
	// 注册缓存
	_ = manager
	_ = memoryBackend
	
	usersCache, err := manager.GetCache("users")
	if err != nil {
		log.Fatalf("[ERROR] Failed to get users cache: %v", err)
	}
	
	log.Printf("[INFO] Cache initialized with memory backend")
	
	// 3. 初始化用户服务（使用代码生成的包装器）
	initUserService(manager)
	log.Printf("[INFO] UserService initialized with sample data")
	
	// 4. 创建 gRPC 服务器
	grpcServer := grpc.NewServer()
	
	// 5. 注册用户服务（使用接口类型，通过适配器）
	pb.RegisterUserServiceServer(grpcServer, server.NewUserServiceServer(server.NewUserServiceClient(userService)))
	log.Printf("[INFO] UserService registered with gRPC server")
	
	// 6. 启动 gRPC 服务器
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("[ERROR] Failed to listen: %v", err)
	}
	
	// 7. 优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		
		log.Printf("[INFO] Shutting down gRPC server...")
		printCacheStats(usersCache)
		grpcServer.GracefulStop()
	}()
	
	log.Printf("[INFO] gRPC server starting on :50051")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("[ERROR] Failed to serve: %v", err)
	}
}

// printCacheStats 打印缓存统计
func printCacheStats(cache core.CacheBackend) {
	stats := cache.Stats()
	
	log.Printf("=== Cache Statistics ===")
	log.Printf("Total Requests: %d", stats.Hits+stats.Misses)
	log.Printf("Cache Hits: %d", stats.Hits)
	log.Printf("Cache Misses: %d", stats.Misses)
	if stats.HitRate > 0 {
		log.Printf("Hit Rate: %.2f%%", stats.HitRate*100)
	}
	log.Printf("Cache Size: %d / %d", stats.Size, stats.MaxSize)
	log.Printf("========================")
}
