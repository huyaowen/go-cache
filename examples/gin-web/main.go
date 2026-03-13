package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/coderiser/go-cache/examples/gin-web/handler"
	"github.com/coderiser/go-cache/examples/gin-web/service"
)

func main() {
	// 初始化日志
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[INFO] Starting Gin Web API with Go-Cache Framework...")

	// 初始化缓存和装饰服务
	service.InitCache()

	// 初始化 HTTP 处理器（使用已装饰的服务）
	// 方案 A: 代码生成器生成类型安全的包装器
	userHandler := handler.NewUserHandler(service.UserService)

	// 创建 Gin 路由
	r := gin.Default()

	// 注册路由
	api := r.Group("/api")
	{
		users := api.Group("/users")
		{
			users.GET("/:id", userHandler.GetUser)
			users.POST("/", userHandler.CreateUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
		}
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})

	// 启动服务
	log.Println("[INFO] Server listening on :8088")
	log.Println("[INFO] API Endpoints:")
	log.Println("[INFO]   GET    /api/users/:id    - 获取用户")
	log.Println("[INFO]   POST   /api/users        - 创建用户")
	log.Println("[INFO]   PUT    /api/users/:id    - 更新用户")
	log.Println("[INFO]   DELETE /api/users/:id    - 删除用户")
	log.Println("[INFO]")
	log.Println("[INFO] 缓存配置:")
	log.Println("[INFO]   - 后端：Memory")
	log.Println("[INFO]   - TTL: 30 分钟")
	log.Println("[INFO]   - 缓存键：users:{id}")
	log.Println("[INFO]")
	log.Println("[INFO] 架构方案：方案 A - 代码生成器（推荐）")
	log.Println("[INFO]   - 优点：业务代码零侵入，类型安全，IDE 友好")
	log.Println("[INFO]   - 使用：service.UserService.GetUser(id)")
	log.Println("[INFO]")
	log.Println("[INFO] 生成命令：go generate ./...")
	log.Println("[INFO]")
	log.Println("[INFO] 测试命令:")
	log.Println("[INFO]   curl http://localhost:8088/api/users/1")
	log.Println("[INFO]   curl -X POST http://localhost:8088/api/users -d '{\"name\":\"Alice\",\"email\":\"alice@example.com\"}'")

	if err := r.Run(":8088"); err != nil {
		log.Fatalf("[ERROR] Failed to start server: %v", err)
	}
}
