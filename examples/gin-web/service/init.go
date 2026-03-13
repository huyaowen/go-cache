package service

import (
	"github.com/coderiser/go-cache/pkg/core"
	"github.com/coderiser/go-cache/examples/gin-web/service/cached"
)

// InitService 初始化服务
func InitService(manager core.CacheManager) UserServiceInterface {
	return cached.NewUserService()
}
