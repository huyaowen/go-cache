// Package cache 提供 Go-Cache 框架的全局缓存功能 (方案 G - Beego 融合版)
//
// # 核心特性
//
//   - 全局管理器懒加载：无需手动传递 Manager
//   - 全局注解注册表：init() 自动注册，用户无感知
//   - 全局拦截器：方法调用时自动拦截，执行缓存逻辑
//
// # 快速开始
//
// 1. 在 Service 方法上添加缓存注解:
//
//	// @cacheable(cache="products", key="#id", ttl="1h")
//	func (s *ProductService) GetProduct(id int64) (*model.Product, error) {
//	    // 业务逻辑
//	}
//
// 2. 执行代码生成:
//
//	gocache scan ./service
//
// 3. 直接使用 (零配置):
//
//	func main() {
//	    svc := cache.NewProductService()
//	    product, err := svc.GetProduct(1)
//	}
//
// # 高级用法
//
// 自定义缓存管理器 (可选):
//
//	func main() {
//	    manager := core.NewCacheManager()
//	    // 配置 Redis 后端...
//	    cache.SetGlobalManager(manager)
//
//	    svc := cache.NewProductService()
//	    product, err := svc.GetProduct(1)
//	}
//
// # 架构设计
//
// 方案 G 借鉴了 Beego 路由注解的设计:
//   - init() 自动注册所有注解到全局表
//   - GetGlobalManager() 懒加载创建管理器
//   - 代码生成器生成 NewXxxService() 函数
//
// 用户体验: "注解后直接使用"，无需手动装饰或传递 Manager。
package cache
