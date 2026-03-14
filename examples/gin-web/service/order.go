package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/coderiser/go-cache/examples/gin-web/model"
)

// OrderService 订单服务（带缓存）
type OrderService struct {
	mu     sync.RWMutex
	orders map[int64]*model.Order
	nextID int64
}

// NewOrderService 创建订单服务实例
func NewOrderService() *OrderService {
	return &OrderService{
		orders: make(map[int64]*model.Order),
		nextID: 1,
	}
}

// GetOrder 获取订单
// @cacheable(cache="orders", key="#id", ttl="30m")
func (s *OrderService) GetOrder(id int64) (*model.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	order, exists := s.orders[id]
	if !exists {
		return nil, fmt.Errorf("order %d not found", id)
	}

	return order, nil
}

// CreateOrder 创建订单
// @cacheput(cache="orders", key="#result.ID", ttl="30m")
func (s *OrderService) CreateOrder(userID int64, total float64) (*model.Order, error) {
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

// UpdateOrderStatus 更新订单状态
// @cacheput(cache="orders", key="#id", ttl="30m")
func (s *OrderService) UpdateOrderStatus(id int64, status string) (*model.Order, error) {
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
