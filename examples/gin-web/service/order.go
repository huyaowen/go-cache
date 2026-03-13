package service

//go:generate go run ../../../cmd/generator/main.go .

import (
	"fmt"
	"sync"
	"time"

	"github.com/coderiser/go-cache/examples/gin-web/model"
)

// OrderServiceInterface 订单服务接口
type OrderServiceInterface interface {
	GetOrder(id int64) (*model.Order, error)
	CreateOrder(userID int64, total float64) (*model.Order, error)
	UpdateOrderStatus(id int64, status string) (*model.Order, error)
}

// orderService 订单服务实现
type orderService struct {
	mu     sync.RWMutex
	orders map[int64]*model.Order
	nextID int64
}

// NewOrderServiceRaw 创建原始订单服务（不带缓存）
func NewOrderServiceRaw() *orderService {
	return &orderService{
		orders: make(map[int64]*model.Order),
		nextID: 1,
	}
}

// GetOrder 获取订单 - 带缓存
// @cacheable(cache="orders", key="#id", ttl="30m")
func (s *orderService) GetOrder(id int64) (*model.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	order, exists := s.orders[id]
	if !exists {
		return nil, fmt.Errorf("order %d not found", id)
	}

	return order, nil
}

// CreateOrder 创建订单 - 带缓存更新
// @cacheput(cache="orders", key="#result.ID", ttl="30m")
func (s *orderService) CreateOrder(userID int64, total float64) (*model.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order := &model.Order{
		ID:        s.nextID,
		UserID:    userID,
		Total:     total,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.nextID++
	s.orders[order.ID] = order

	return order, nil
}

// UpdateOrderStatus 更新订单状态 - 带缓存更新
// @cacheput(cache="orders", key="#id", ttl="30m")
func (s *orderService) UpdateOrderStatus(id int64, status string) (*model.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.orders[id]
	if !exists {
		return nil, fmt.Errorf("order %d not found", id)
	}

	order.Status = status
	order.UpdatedAt = time.Now()

	return order, nil
}
