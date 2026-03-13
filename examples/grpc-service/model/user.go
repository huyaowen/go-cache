package model

// User 用户模型
type User struct {
	ID    int32  `json:"id" msgpack:"id"`
	Name  string `json:"name" msgpack:"name"`
	Email string `json:"email" msgpack:"email"`
}
