package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/coderiser/go-cache/examples/gin-web/handler"
	"github.com/coderiser/go-cache/examples/gin-web/service"
)

func main() {
	// 初始化服务 (零配置，使用全局缓存管理器)
	service.InitUserService()

	// 创建处理器
	userHandler := handler.NewUserHandler(service.UserService)

	// 设置 Gin 路由
	r := gin.Default()

	// 用户路由
	r.GET("/users/:id", userHandler.GetUser)
	r.POST("/users", userHandler.CreateUser)
	r.DELETE("/users/:id", userHandler.DeleteUser)

	// 启动服务
	log.Println("🚀 Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
