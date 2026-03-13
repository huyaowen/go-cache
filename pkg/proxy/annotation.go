package proxy

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// AnnotationMeta 注解元数据
type AnnotationMeta struct {
	Type        string            // 注解类型
	CacheName   string            // 缓存名称
	Key         string            // 缓存键表达式
	TTL         string            // TTL 表达式
	Condition   string            // 条件表达式
	Unless      string            // 除非表达式
	Before      bool              // 是否在方法执行前执行
	Sync        bool              // 是否同步执行
	Attributes  map[string]string // 自定义属性
}

// AnnotationHandler 注解处理器接口
type AnnotationHandler interface {
	// Handle 处理注解
	// ctx: 上下文
	// meta: 注解元数据
	// args: 方法参数
	// 返回值：处理结果（如果有），错误
	Handle(ctx context.Context, meta *AnnotationMeta, args []reflect.Value) (interface{}, error)
	
	// GetPriority 获取优先级（数字越小优先级越高）
	GetPriority() int
}

// BaseAnnotationHandler 基础注解处理器（辅助实现）
type BaseAnnotationHandler struct {
	Priority int // 优先级
}

// GetPriority 获取优先级
func (b *BaseAnnotationHandler) GetPriority() int {
	return b.Priority
}

// AnnotationHandlerRegistry 注解处理器注册表
type AnnotationHandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[string]AnnotationHandler
	order    []string // 处理器执行顺序
}

// NewAnnotationHandlerRegistry 创建注解处理器注册表
func NewAnnotationHandlerRegistry() *AnnotationHandlerRegistry {
	return &AnnotationHandlerRegistry{
		handlers: make(map[string]AnnotationHandler),
		order:    make([]string, 0),
	}
}

// Register 注册注解处理器
func (r *AnnotationHandlerRegistry) Register(name string, handler AnnotationHandler) error {
	if name == "" {
		return fmt.Errorf("handler name cannot be empty")
	}
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查是否已存在
	if _, exists := r.handlers[name]; exists {
		return fmt.Errorf("handler '%s' already registered", name)
	}

	r.handlers[name] = handler
	r.order = append(r.order, name)
	return nil
}

// Unregister 注销注解处理器
func (r *AnnotationHandlerRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[name]; !exists {
		return fmt.Errorf("handler '%s' not found", name)
	}

	delete(r.handlers, name)
	
	// 从顺序列表中移除
	for i, n := range r.order {
		if n == name {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
	return nil
}

// GetHandler 获取注解处理器
func (r *AnnotationHandlerRegistry) GetHandler(name string) (AnnotationHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, exists := r.handlers[name]
	return handler, exists
}

// GetAllHandlers 获取所有处理器（按优先级排序）
func (r *AnnotationHandlerRegistry) GetAllHandlers() []AnnotationHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handlers := make([]AnnotationHandler, 0, len(r.handlers))
	for _, name := range r.order {
		handlers = append(handlers, r.handlers[name])
	}

	// 按优先级排序
	for i := 0; i < len(handlers)-1; i++ {
		for j := i + 1; j < len(handlers); j++ {
			if handlers[i].GetPriority() > handlers[j].GetPriority() {
				handlers[i], handlers[j] = handlers[j], handlers[i]
			}
		}
	}

	return handlers
}

// Handle 执行注解处理
func (r *AnnotationHandlerRegistry) Handle(ctx context.Context, meta *AnnotationMeta, args []reflect.Value) (interface{}, error) {
	handler, exists := r.GetHandler(meta.Type)
	if !exists {
		return nil, fmt.Errorf("no handler registered for annotation type '%s'", meta.Type)
	}
	return handler.Handle(ctx, meta, args)
}

// HasHandler 检查是否存在处理器
func (r *AnnotationHandlerRegistry) HasHandler(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.handlers[name]
	return exists
}

// ListHandlers 列出所有已注册的处理器名称
func (r *AnnotationHandlerRegistry) ListHandlers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]string, len(r.order))
	copy(result, r.order)
	return result
}

// Clear 清空所有处理器
func (r *AnnotationHandlerRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = make(map[string]AnnotationHandler)
	r.order = make([]string, 0)
}

// 全局注册表
var globalHandlerRegistry = NewAnnotationHandlerRegistry()

// RegisterHandler 注册全局注解处理器（便捷函数）
func RegisterHandler(name string, handler AnnotationHandler) error {
	return globalHandlerRegistry.Register(name, handler)
}

// UnregisterHandler 注销全局注解处理器
func UnregisterHandler(name string) error {
	return globalHandlerRegistry.Unregister(name)
}

// GetHandler 获取全局注解处理器
func GetHandler(name string) (AnnotationHandler, bool) {
	return globalHandlerRegistry.GetHandler(name)
}

// HandleAnnotation 执行全局注解处理
func HandleAnnotation(ctx context.Context, meta *AnnotationMeta, args []reflect.Value) (interface{}, error) {
	return globalHandlerRegistry.Handle(ctx, meta, args)
}

// ListHandlers 列出所有已注册的全局处理器
func ListHandlers() []string {
	return globalHandlerRegistry.ListHandlers()
}

// Example: 自定义注解处理器示例
// ```go
// // 定义自定义注解处理器
// type LoggingHandler struct {
//     BaseAnnotationHandler
// }
//
// func (h *LoggingHandler) Handle(ctx context.Context, meta *AnnotationMeta, args []reflect.Value) (interface{}, error) {
//     log.Printf("Calling method with annotation: %s", meta.Type)
//     log.Printf("Cache: %s, Key: %s", meta.CacheName, meta.Key)
//     
//     // 调用原始方法
//     // ... 实现逻辑
//     
//     return result, nil
// }
//
// // 注册处理器
// func init() {
//     proxy.RegisterHandler("logging", &LoggingHandler{
//         BaseAnnotationHandler{Priority: 100},
//     })
// }
// ```

// RateLimitHandler 限流注解处理器示例
type RateLimitHandler struct {
	BaseAnnotationHandler
	limits sync.Map // map[string]*RateLimiter
}

// RateLimiter 简单的限流器
type RateLimiter struct {
	mu       sync.Mutex
	count    int
	maxCount int
	resetAt  time.Time
}

// Handle 处理限流注解
func (h *RateLimitHandler) Handle(ctx context.Context, meta *AnnotationMeta, args []reflect.Value) (interface{}, error) {
	key := fmt.Sprintf("%s:%s", meta.CacheName, meta.Key)
	
	limiter := h.getLimiter(key, meta)
	if !limiter.Allow() {
		return nil, fmt.Errorf("rate limit exceeded for %s", key)
	}
	
	return nil, nil
}

func (h *RateLimitHandler) getLimiter(key string, meta *AnnotationMeta) *RateLimiter {
	if v, ok := h.limits.Load(key); ok {
		return v.(*RateLimiter)
	}
	
	maxCount := 100 // 默认 100 次
	if maxStr, ok := meta.Attributes["max"]; ok {
		// 解析配置
		_ = maxStr
	}
	
	limiter := &RateLimiter{
		maxCount: maxCount,
		resetAt:  time.Now().Add(time.Minute),
	}
	h.limits.Store(key, limiter)
	return limiter
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	if time.Now().After(rl.resetAt) {
		rl.count = 0
		rl.resetAt = time.Now().Add(time.Minute)
	}
	
	if rl.count >= rl.maxCount {
		return false
	}
	
	rl.count++
	return true
}

// RetryHandler 重试注解处理器示例
type RetryHandler struct {
	BaseAnnotationHandler
}

// Handle 处理重试注解
func (h *RetryHandler) Handle(ctx context.Context, meta *AnnotationMeta, args []reflect.Value) (interface{}, error) {
	// maxRetries := 3
	// if retriesStr, ok := meta.Attributes["maxRetries"]; ok {
	// 	// 解析重试次数
	// 	_ = retriesStr
	// }
	
	// 实现重试逻辑
	// ...
	
	return nil, nil
}

// CircuitBreakerHandler 熔断器注解处理器示例
type CircuitBreakerHandler struct {
	BaseAnnotationHandler
}

// Handle 处理熔断器注解
func (h *CircuitBreakerHandler) Handle(ctx context.Context, meta *AnnotationMeta, args []reflect.Value) (interface{}, error) {
	// 实现熔断器逻辑
	// ...
	
	return nil, nil
}
