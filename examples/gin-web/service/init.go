package service

import (
	"github.com/coderiser/go-cache/pkg/core"
	"github.com/coderiser/go-cache/examples/gin-web/service/cached"
)

// InitService 初始化服务（使用全局缓存管理器）
func InitService() UserServiceInterface {
	return cached.NewUserService()
}
