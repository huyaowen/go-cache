package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/coderiser/go-cache/examples/gin-web/model"
)

// UserService 用户服务（带缓存）
// 
// 零配置使用说明:
// 1. 在方法前添加 @ 注解
// 2. 导入 cache 包触发自动扫描（见 main.go）
// 3. 直接使用，缓存自动生效
type UserService struct {
	mu     sync.RWMutex
	users  map[int64]*model.User
	nextID int64
}

// NewUserService 创建用户服务实例
// 通过 proxy.SimpleDecorate 自动应用缓存
func NewUserService() *UserService {
	return &UserService{
		users:  make(map[int64]*model.User),
		nextID: 1,
	}
}

// GetUser 获取用户
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id int64) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[id]
	if !exists {
		return nil, fmt.Errorf("user %d not found", id)
	}

	return user, nil
}

// CreateUser 创建用户
// @cacheput(cache="users", key="#result.ID", ttl="30m")
func (s *UserService) CreateUser(name, email string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user := &model.User{
		ID:        s.nextID,
		Name:      name,
		Email:     email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.nextID++
	s.users[user.ID] = user

	return user, nil
}

// UpdateUser 更新用户
// @cacheput(cache="users", key="#id", ttl="30m")
func (s *UserService) UpdateUser(id int64, name, email string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[id]
	if !exists {
		return nil, fmt.Errorf("user %d not found", id)
	}

	user.Name = name
	user.Email = email
	user.UpdatedAt = time.Now()

	return user, nil
}

// DeleteUser 删除用户
// @cacheevict(cache="users", key="#id")
func (s *UserService) DeleteUser(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[id]; !exists {
		return fmt.Errorf("user %d not found", id)
	}

	delete(s.users, id)
	return nil
}
