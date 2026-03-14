package services

// Product represents a product entity
type Product struct {
	ID    int64
	Name  string
	Price float64
}

// ProductService provides product-related operations
type ProductService struct{}

// @cacheable(cache="products", key="#id", ttl="1h")
func (s *ProductService) GetProduct(id int64) (*Product, error) {
	return &Product{ID: id, Name: "Test Product", Price: 99.99}, nil
}

// @cacheput(cache="products", key="#product.ID", ttl="1h")
func (s *ProductService) UpdateProduct(product *Product) (*Product, error) {
	return product, nil
}

// @cacheevict(cache="products", key="#id")
func (s *ProductService) DeleteProduct(id int64) error {
	return nil
}
