package main

import (
	"fmt"
	"log"

	"github.com/coderiser/go-cache/examples/grpc-demo/service"
	_ "github.com/coderiser/go-cache/pkg/cache" // 导入 cache 包，触发自动注解扫描
)

func main() {
	// 🎉 零配置！无需运行代码生成器
	// cache 包的 init() 会自动扫描 service 包中的注解
	// 并注册到代理系统
	
	// 获取装饰后的服务实例（带缓存）
	userSvc := service.GetUserService()
	orderSvc := service.GetOrderService()

	// 测试用户服务
	user, err := userSvc.GetUser(1)
	if err != nil {
		log.Printf("GetUser error: %v", err)
	} else {
		fmt.Printf("User: %+v\n", user)
	}

	// 测试订单服务
	order, err := orderSvc.GetOrder(100)
	if err != nil {
		log.Printf("GetOrder error: %v", err)
	} else {
		fmt.Printf("Order: %+v\n", order)
	}

	fmt.Println("gRPC Demo completed!")
}
