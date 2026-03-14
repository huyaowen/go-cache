package services

// User represents a user entity
type User struct {
	ID    int64
	Name  string
	Email string
}

// UserService provides user-related operations
type UserService struct{}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id int64) (*User, error) {
	return &User{ID: id, Name: "Test User"}, nil
}

// @cacheput(cache="users", key="#user.ID", ttl="30m")
func (s *UserService) CreateUser(user *User) (*User, error) {
	return user, nil
}

// @cacheevict(cache="users", key="#id")
func (s *UserService) DeleteUser(id int64) error {
	return nil
}
