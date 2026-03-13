package server

import (
	"context"
	"log"

	"github.com/coderiser/go-cache/examples/grpc-service/model"
	"github.com/coderiser/go-cache/examples/grpc-service/service"
	pb "github.com/coderiser/go-cache/examples/grpc-service/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserServiceClient gRPC 用户服务客户端接口
// 使用 service 包定义的接口（方案 A: 代码生成）
// 此接口与 service.UserServiceInterface 兼容（添加 context 参数）
type UserServiceClient interface {
	GetUser(ctx context.Context, id int32) (*model.User, error)
	CreateUser(ctx context.Context, name, email string) (*model.User, error)
	UpdateUser(ctx context.Context, id int32, name, email string) (*model.User, error)
	DeleteUser(ctx context.Context, id int32) error
}

// adapterUserService 适配器，将 service.UserServiceInterface 转换为 UserServiceClient
type adapterUserService struct {
	service.UserServiceInterface
}

func (a *adapterUserService) GetUser(ctx context.Context, id int32) (*model.User, error) {
	return a.UserServiceInterface.GetUser(id)
}

func (a *adapterUserService) CreateUser(ctx context.Context, name, email string) (*model.User, error) {
	return a.UserServiceInterface.CreateUser(name, email)
}

func (a *adapterUserService) UpdateUser(ctx context.Context, id int32, name, email string) (*model.User, error) {
	return a.UserServiceInterface.UpdateUser(id, name, email)
}

func (a *adapterUserService) DeleteUser(ctx context.Context, id int32) error {
	return a.UserServiceInterface.DeleteUser(id)
}

// NewUserServiceClient 创建适配的客户端
func NewUserServiceClient(svc service.UserServiceInterface) UserServiceClient {
	return &adapterUserService{UserServiceInterface: svc}
}

// UserServiceServer gRPC 用户服务实现
type UserServiceServer struct {
	pb.UnimplementedUserServiceServer
	client UserServiceClient
}

// NewUserServiceServer 创建 gRPC 服务实例
func NewUserServiceServer(client UserServiceClient) *UserServiceServer {
	return &UserServiceServer{
		client: client,
	}
}

// GetUser 获取用户信息
func (s *UserServiceServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	log.Printf("[gRPC] GetUser called with id: %d", req.Id)
	
	user, err := s.client.GetUser(ctx, req.Id)
	if err != nil {
		log.Printf("[gRPC] GetUser error: %v", err)
		return nil, status.Error(codes.NotFound, err.Error())
	}
	
	log.Printf("[gRPC] GetUser success: %+v", user)
	return &pb.User{
		Id:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}, nil
}

// CreateUser 创建用户
func (s *UserServiceServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
	log.Printf("[gRPC] CreateUser called with name: %s, email: %s", req.Name, req.Email)
	
	user, err := s.client.CreateUser(ctx, req.Name, req.Email)
	if err != nil {
		log.Printf("[gRPC] CreateUser error: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	
	log.Printf("[gRPC] CreateUser success: %+v", user)
	return &pb.User{
		Id:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}, nil
}

// UpdateUser 更新用户信息
func (s *UserServiceServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.User, error) {
	log.Printf("[gRPC] UpdateUser called with id: %d, name: %s, email: %s", req.Id, req.Name, req.Email)
	
	user, err := s.client.UpdateUser(ctx, req.Id, req.Name, req.Email)
	if err != nil {
		log.Printf("[gRPC] UpdateUser error: %v", err)
		return nil, status.Error(codes.NotFound, err.Error())
	}
	
	log.Printf("[gRPC] UpdateUser success: %+v", user)
	return &pb.User{
		Id:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}, nil
}

// DeleteUser 删除用户
func (s *UserServiceServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.Empty, error) {
	log.Printf("[gRPC] DeleteUser called with id: %d", req.Id)
	
	err := s.client.DeleteUser(ctx, req.Id)
	if err != nil {
		log.Printf("[gRPC] DeleteUser error: %v", err)
		return nil, status.Error(codes.NotFound, err.Error())
	}
	
	log.Printf("[gRPC] DeleteUser success")
	return &pb.Empty{}, nil
}
