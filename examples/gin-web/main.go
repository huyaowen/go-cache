package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/coderiser/go-cache/examples/gin-web/handler"
	"github.com/coderiser/go-cache/examples/gin-web/service"
	_ "github.com/coderiser/go-cache/pkg/cache" // 导入 cache 包，触发自动注解扫描
)

func main() {
	// 🎉 零配置！无需运行代码生成器
	// cache 包的 init() 会自动扫描 service 包中的注解
	// 并注册到代理系统
	
	// 初始化服务（自动应用缓存）
	userService := service.UserService()

	// 创建处理器
	userHandler := handler.NewUserHandler(userService)

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
