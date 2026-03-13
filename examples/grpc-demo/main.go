package main

import (
	"fmt"
	"log"

	"github.com/coderiser/go-cache/examples/grpc-demo/service"
)

func main() {
	// 创建服务实例（自动使用全局缓存管理器）
	userSvc := service.NewUserService()
	orderSvc := service.NewOrderService()

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
