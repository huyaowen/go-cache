package main

import (
	"context"
	"flag"
	"log"
	"time"

	pb "github.com/coderiser/go-cache/examples/grpc-service/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	addr = flag.String("addr", "localhost:50051", "gRPC server address")
)

func main() {
	flag.Parse()

	log.Printf("Connecting to %s...", *addr)
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("=== Testing gRPC UserService ===")
	log.Println()

	// 1. 获取用户（缓存未命中）
	log.Println("1. GetUser(1) - Cache Miss Expected")
	resp1, err := client.GetUser(ctx, &pb.GetUserRequest{Id: 1})
	if err != nil {
		log.Printf("   Error: %v", err)
	} else {
		log.Printf("   User: id=%d, name=%s, email=%s", resp1.Id, resp1.Name, resp1.Email)
	}
	log.Println()

	// 等待一下
	time.Sleep(200 * time.Millisecond)

	// 2. 再次获取用户（缓存命中）
	log.Println("2. GetUser(1) - Cache Hit Expected")
	resp2, err := client.GetUser(ctx, &pb.GetUserRequest{Id: 1})
	if err != nil {
		log.Printf("   Error: %v", err)
	} else {
		log.Printf("   User: id=%d, name=%s, email=%s", resp2.Id, resp2.Name, resp2.Email)
	}
	log.Println()

	// 3. 创建用户
	log.Println("3. CreateUser - Create new user")
	resp3, err := client.CreateUser(ctx, &pb.CreateUserRequest{
		Name:  "David",
		Email: "david@example.com",
	})
	if err != nil {
		log.Printf("   Error: %v", err)
	} else {
		log.Printf("   Created: id=%d, name=%s, email=%s", resp3.Id, resp3.Name, resp3.Email)
	}
	log.Println()

	// 4. 更新用户
	log.Println("4. UpdateUser(1) - Update Alice")
	resp4, err := client.UpdateUser(ctx, &pb.UpdateUserRequest{
		Id:    1,
		Name:  "Alice Updated",
		Email: "alice.updated@example.com",
	})
	if err != nil {
		log.Printf("   Error: %v", err)
	} else {
		log.Printf("   Updated: id=%d, name=%s, email=%s", resp4.Id, resp4.Name, resp4.Email)
	}
	log.Println()

	// 5. 获取更新后的用户
	log.Println("5. GetUser(1) - Verify Update")
	resp5, err := client.GetUser(ctx, &pb.GetUserRequest{Id: 1})
	if err != nil {
		log.Printf("   Error: %v", err)
	} else {
		log.Printf("   User: id=%d, name=%s, email=%s", resp5.Id, resp5.Name, resp5.Email)
	}
	log.Println()

	// 6. 删除用户
	log.Println("6. DeleteUser(3) - Delete Charlie")
	_, err = client.DeleteUser(ctx, &pb.DeleteUserRequest{Id: 3})
	if err != nil {
		log.Printf("   Error: %v", err)
	} else {
		log.Printf("   Deleted successfully")
	}
	log.Println()

	log.Println("=== Test Complete ===")
}
