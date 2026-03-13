//go:generate go run ../../../cmd/generator/main.go .

package service

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/coderiser/go-cache/examples/gin-web/model"
)

// UserServiceInterface 用户服务接口
// 
// 使用代码生成器（方案 A）：
//   1. 运行：go generate ./...
//   2. 自动生成：.cache-gen/user_decorated.go (实现此接口的包装器)
//   3. 使用：UserServiceInterface 类型
type UserServiceInterface interface {
	GetUser(id int64) (*model.User, error)
	CreateUser(user *model.User) (*model.User, error)
	UpdateUser(id int64, user *model.User) (*model.User, error)
	DeleteUser(id int64) error
}

// userService 用户服务实现（带缓存注解）
type userService struct {
	mu    sync.RWMutex
	users map[int64]*model.User
}

// NewUserService 创建用户服务实例
func NewUserService() *userService {
	return &userService{
		users: make(map[int64]*model.User),
	}
}

// GetUser 获取用户 - 带 @cacheable 注解
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *userService) GetUser(id int64) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log.Printf("[INFO] Querying database for user %d...", id)
	
	user, exists := s.users[id]
	if !exists {
		return nil, fmt.Errorf("user %d not found", id)
	}

	log.Printf("[INFO] Found user: %s", user.Name)
	return user, nil
}

// CreateUser 创建用户 - 带 @cacheput 注解
// @cacheput(cache="users", key="#user.ID", ttl="30m")
func (s *userService) CreateUser(user *model.User) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user.ID = time.Now().UnixNano() % 100000 // 简单 ID 生成
	user.CreatedAt = time.Now()

	log.Printf("[INFO] Creating user: %s (%s)", user.Name, user.Email)
	s.users[user.ID] = user

	return user, nil
}

// UpdateUser 更新用户 - 带 @cacheput 注解
// @cacheput(cache="users", key="#id", ttl="30m")
func (s *userService) UpdateUser(id int64, user *model.User) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.users[id]
	if !exists {
		return nil, fmt.Errorf("user %d not found", id)
	}

	log.Printf("[INFO] Updating user %d: %s -> %s", id, existing.Name, user.Name)
	
	existing.Name = user.Name
	existing.Email = user.Email
	s.users[id] = existing

	return existing, nil
}

// DeleteUser 删除用户 - 带 @cacheevict 注解
// @cacheevict(cache="users", key="#id")
func (s *userService) DeleteUser(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("[INFO] Deleting user %d...", id)
	
	_, exists := s.users[id]
	if !exists {
		return fmt.Errorf("user %d not found", id)
	}

	delete(s.users, id)
	return nil
}
