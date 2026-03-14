package testdata

// User represents a user entity
type User struct {
	ID   int64
	Name string
}

// Product represents a product entity
type Product struct {
	ID    int64
	Name  string
	Price float64
}

// Order represents an order entity
type Order struct {
	ID     int64
	UserID int64
	Total  float64
}

// UserService provides user-related operations
type UserService struct{}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id int64) (*User, error) {
	return &User{ID: id, Name: "Test User"}, nil
}

// @cacheput(cache="users", key="#user.ID", ttl="30m")
func (s *UserService) CreateUser(user *User) (*User, error) {
	return user, nil
}

// ProductService provides product-related operations
type ProductService struct{}

// @cacheable(cache="products", key="#id", ttl="1h")
func (s *ProductService) GetProduct(id int64) (*Product, error) {
	return &Product{ID: id, Name: "Test Product", Price: 99.99}, nil
}

// @cacheevict(cache="products", key="#id")
func (s *ProductService) DeleteProduct(id int64) error {
	return nil
}

// OrderService provides order-related operations
type OrderService struct{}

// @cacheable(cache="orders", key="#id", ttl="2h")
func (s *OrderService) GetOrder(id int64) (*Order, error) {
	return &Order{ID: id, UserID: 1, Total: 100.00}, nil
}
