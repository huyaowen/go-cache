//go:generate go run ../../../cmd/generator/main.go .

package service

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/coderiser/go-cache/pkg/core"
	"github.com/coderiser/go-cache/pkg/proxy"
	"github.com/coderiser/go-cache/examples/cron-job/model"
)

// ProductServiceInterface 商品服务接口
// 
// 使用代码生成器（方案 A）：
//   1. 运行：go generate ./...
//   2. 自动生成：.cache-gen/product_decorated.go (实现此接口的包装器)
//   3. 使用：ProductServiceInterface 类型
type ProductServiceInterface interface {
	GetProduct(id int64) (*model.Product, error)
	GetHotProducts() ([]*model.Product, error)
	UpdatePrice(id int64, price float64) (*model.Product, error)
	GetAllProducts() []*model.Product
	GetProductByID(id int64) *model.Product
	UpdateStock(id int64, stock int) error
}

// productService 商品服务（带缓存注解）
type productService struct {
	mu       sync.RWMutex
	products map[int64]*model.Product
	manager  core.CacheManager
}

// NewProductService 创建商品服务实例
func NewProductService(manager core.CacheManager) *productService {
	s := &productService{
		products: make(map[int64]*model.Product),
		manager:  manager,
	}
	
	// 初始化一些测试数据
	s.initTestData()
	
	return s
}

// initTestData 初始化测试数据
func (s *productService) initTestData() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	for i := int64(1); i <= 10; i++ {
		s.products[i] = &model.Product{
			ID:        i,
			Name:      fmt.Sprintf("Product %d", i),
			Price:     float64(i) * 10.5,
			Stock:     100 + int(i)*10,
			UpdatedAt: now,
		}
	}
	log.Printf("[INFO] Initialized %d products", len(s.products))
}

// GetProduct 获取商品 - 带 @cacheable 注解
// @cacheable(cache="products", key="#id", ttl="1h")
func (s *productService) GetProduct(id int64) (*model.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log.Printf("[DEBUG] Querying database for product %d...", id)
	
	product, exists := s.products[id]
	if !exists {
		return nil, fmt.Errorf("product %d not found", id)
	}

	return product, nil
}

// GetHotProducts 获取热点商品 - 带 @cacheable 注解
// @cacheable(cache="hot_products", key="list", ttl="5m")
func (s *productService) GetHotProducts() ([]*model.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log.Printf("[DEBUG] Querying database for hot products...")
	
	// 返回前 5 个商品作为热点商品
	hotProducts := make([]*model.Product, 0, 5)
	for i := int64(1); i <= 5 && i <= int64(len(s.products)); i++ {
		hotProducts = append(hotProducts, s.products[i])
	}

	log.Printf("[DEBUG] Found %d hot products", len(hotProducts))
	return hotProducts, nil
}

// UpdatePrice 更新商品价格 - 带 @cacheput 注解
// @cacheput(cache="products", key="#id", ttl="1h")
func (s *productService) UpdatePrice(id int64, price float64) (*model.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product, exists := s.products[id]
	if !exists {
		return nil, fmt.Errorf("product %d not found", id)
	}

	log.Printf("[INFO] Updating product %d price: %.2f -> %.2f", id, product.Price, price)
	product.Price = price
	product.UpdatedAt = time.Now()

	return product, nil
}

// GetAllProducts 获取所有商品（用于预热）
func (s *productService) GetAllProducts() []*model.Product {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	products := make([]*model.Product, 0, len(s.products))
	for _, p := range s.products {
		products = append(products, p)
	}
	return products
}

// GetProductByID 根据 ID 获取商品（用于预热单个商品）
func (s *productService) GetProductByID(id int64) *model.Product {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.products[id]
}

// UpdateStock 更新商品库存（用于演示缓存刷新）
func (s *productService) UpdateStock(id int64, stock int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	product, exists := s.products[id]
	if !exists {
		return fmt.Errorf("product %d not found", id)
	}
	
	log.Printf("[INFO] Updating product %d stock: %d -> %d", id, product.Stock, stock)
	product.Stock = stock
	product.UpdatedAt = time.Now()
	return nil
}

// DecorateWithCache 使用缓存装饰服务
// 在 main 函数中调用，返回装饰后的服务实例（使用代码生成的包装器）
// 方案 A: 代码生成器生成类型安全的包装器
func DecorateWithCache(manager core.CacheManager) ProductServiceInterface {
	// 创建原始服务
	rawService := &productService{
		products: make(map[int64]*model.Product),
		manager:  manager,
	}
	rawService.initTestData()
	
	// 使用 SimpleDecorate 装饰服务
	decorated := proxy.SimpleDecorateWithManager(rawService, manager)
	
	// 使用代码生成的包装器（方案 A）
	// 生成命令：go generate ./...
	return NewDecoratedProductService(decorated)
}
