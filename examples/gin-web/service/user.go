package service

//go:generate go run ../../../cmd/generator/main.go .

import (
	"fmt"
	"sync"
	"time"

	"github.com/coderiser/go-cache/examples/gin-web/model"
)

// UserServiceInterface 用户服务接口
type UserServiceInterface interface {
	GetUser(id int64) (*model.User, error)
	CreateUser(name, email string) (*model.User, error)
	UpdateUser(id int64, name, email string) (*model.User, error)
	DeleteUser(id int64) error
}

// userService 用户服务实现
type userService struct {
	mu     sync.RWMutex
	users  map[int64]*model.User
	nextID int64
}

// NewUserServiceRaw 创建原始用户服务（不带缓存）
func NewUserServiceRaw() *userService {
	return &userService{
		users:  make(map[int64]*model.User),
		nextID: 1,
	}
}

// GetUser 获取用户 - 带缓存
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *userService) GetUser(id int64) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[id]
	if !exists {
		return nil, fmt.Errorf("user %d not found", id)
	}

	return user, nil
}

// CreateUser 创建用户 - 带缓存更新
// @cacheput(cache="users", key="#result.ID", ttl="30m")
func (s *userService) CreateUser(name, email string) (*model.User, error) {
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

// UpdateUser 更新用户 - 带缓存更新
// @cacheput(cache="users", key="#id", ttl="30m")
func (s *userService) UpdateUser(id int64, name, email string) (*model.User, error) {
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

// DeleteUser 删除用户 - 带缓存清除
// @cacheevict(cache="users", key="#id")
func (s *userService) DeleteUser(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[id]; !exists {
		return fmt.Errorf("user %d not found", id)
	}

	delete(s.users, id)
	return nil
}
