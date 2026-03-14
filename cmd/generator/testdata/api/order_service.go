package api

// Order represents an order entity
type Order struct {
	ID     int64
	UserID int64
	Total  float64
	Status string
}

// OrderService provides order-related operations
type OrderService struct{}

// @cacheable(cache="orders", key="#id", ttl="2h")
func (s *OrderService) GetOrder(id int64) (*Order, error) {
	return &Order{ID: id, UserID: 1, Total: 100.00, Status: "pending"}, nil
}

// @cacheput(cache="orders", key="#order.ID", ttl="2h")
func (s *OrderService) CreateOrder(order *Order) (*Order, error) {
	return order, nil
}

// @cacheevict(cache="orders", key="#id")
func (s *OrderService) CancelOrder(id int64) error {
	return nil
}
