//go:generate go run ../../../cmd/generator/main.go .

package service

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/coderiser/go-cache/examples/grpc-service/model"
)

// UserServiceInterface 用户服务接口
// 
// 使用代码生成器（方案 A）：
//   1. 运行：go generate ./...
//   2. 自动生成：.cache-gen/user_decorated.go (实现此接口的包装器)
//   3. 使用：UserServiceInterface 类型
type UserServiceInterface interface {
	GetUser(id int32) (*model.User, error)
	CreateUser(name, email string) (*model.User, error)
	UpdateUser(id int32, name, email string) (*model.User, error)
	DeleteUser(id int32) error
}

// userService 用户服务（带缓存注解）
type userService struct {
	mu   sync.RWMutex
	db   map[int32]*model.User
	next int32
}

// NewUserService 创建用户服务实例
func NewUserService() *userService {
	s := &userService{
		db:   make(map[int32]*model.User),
		next: 1,
	}
	
	// 初始化一些测试数据
	s.db[1] = &model.User{ID: 1, Name: "Alice", Email: "alice@example.com"}
	s.db[2] = &model.User{ID: 2, Name: "Bob", Email: "bob@example.com"}
	s.db[3] = &model.User{ID: 3, Name: "Charlie", Email: "charlie@example.com"}
	s.next = 4
	
	return s
}

// GetUser 获取用户信息
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *userService) GetUser(id int32) (*model.User, error) {
	log.Printf("[INFO] Cache miss: users:%d", id)
	log.Printf("[INFO] Querying database...")
	
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// 模拟数据库查询延迟
	time.Sleep(100 * time.Millisecond)
	
	user, exists := s.db[id]
	if !exists {
		return nil, fmt.Errorf("user not found: %d", id)
	}
	
	return user, nil
}

// CreateUser 创建用户
// @cacheput(cache="users", key="#result.ID", ttl="30m")
func (s *userService) CreateUser(name, email string) (*model.User, error) {
	log.Printf("[INFO] Creating new user: %s", name)
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	user := &model.User{
		ID:    s.next,
		Name:  name,
		Email: email,
	}
	s.next++
	s.db[user.ID] = user
	
	log.Printf("[INFO] User created with ID: %d", user.ID)
	return user, nil
}

// UpdateUser 更新用户信息
// @cacheput(cache="users", key="#user.ID", ttl="30m")
func (s *userService) UpdateUser(id int32, name, email string) (*model.User, error) {
	log.Printf("[INFO] Updating user: %d", id)
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	user, exists := s.db[id]
	if !exists {
		return nil, fmt.Errorf("user not found: %d", id)
	}
	
	user.Name = name
	user.Email = email
	
	log.Printf("[INFO] User updated: %d", id)
	return user, nil
}

// DeleteUser 删除用户
// @cacheevict(cache="users", key="#id", before=false)
func (s *userService) DeleteUser(id int32) error {
	log.Printf("[INFO] Deleting user: %d", id)
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.db[id]; !exists {
		return fmt.Errorf("user not found: %d", id)
	}
	
	delete(s.db, id)
	log.Printf("[INFO] User deleted: %d", id)
	return nil
}
