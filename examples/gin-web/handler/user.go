package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/coderiser/go-cache/examples/gin-web/service"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetUser 获取用户
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// 使用 Invoke 调用装饰后的方法
	results, err := h.userService.Invoke("GetUser", id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if len(results) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no result"})
		return
	}

	// 检查错误
	if errResult, ok := results[1].(error); ok && errResult != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errResult.Error()})
		return
	}

	user, ok := results[0].(*service.UserService)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid response type"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// CreateUser 创建用户
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := h.userService.Invoke("CreateUser", req.Name, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(results) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no result"})
		return
	}

	c.JSON(http.StatusCreated, results[0])
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	results, err := h.userService.Invoke("DeleteUser", id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if errResult, ok := results[0].(error); ok && errResult != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errResult.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}
