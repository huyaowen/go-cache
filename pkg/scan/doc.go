// Package scan 提供编译时扫描和代码生成功能
//
// 扫描 Go 源文件中的缓存注解，并生成对应的缓存包装器代码。
//
// 使用方式:
//
//	// 在业务代码中定义接口和实现
//	type UserServiceInterface interface {
//	    GetUser(id int64) (*User, error)
//	}
//
//	type userService struct {
//	    db *sql.DB
//	}
//
//	// @cacheable(cache="users", key="#id", ttl="30m")
//	func (s *userService) GetUser(id int64) (*User, error) {
//	    return s.db.QueryUser(id)
//	}
//
//	// 执行扫描命令
//	// $ gocache scan ./...
//
//	// 使用生成的代码
//	func main() {
//	    service.InitUserService(db)
//	    user, _ := service.UserService.GetUser(1)
//	}
package scan
