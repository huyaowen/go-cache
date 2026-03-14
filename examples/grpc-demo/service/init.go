package service

import (
	"github.com/coderiser/go-cache/pkg/proxy"
)

// 零配置服务初始化（gRPC 示例）
// 与 gin-web 示例相同，无需特殊配置

// UserService 用户服务实例（带缓存）
var UserService = proxy.SimpleDecorate(NewUserServiceRaw())

// OrderService 订单服务实例（带缓存）
var OrderService = proxy.SimpleDecorate(NewOrderServiceRaw())

// 使用说明参考 gin-web 示例
