package testdata

// UserService handles user operations
type UserService struct{}

// GetUser retrieves a user by ID
// @cacheable(cache="users", key="#id", ttl="30m")
func (s *UserService) GetUser(id string) (*User, error) {
	return &User{ID: id, Name: "Test User"}, nil
}

// GetUserByEmail retrieves a user by email
// @cacheable(cache="users", key="#email", ttl="1h")
func (s *UserService) GetUserByEmail(email string) (*User, error) {
	return &User{Email: email, Name: "Test User"}, nil
}

// ListUsers lists all users with pagination
// @cacheable(cache="users_list", key="#page_#pageSize", ttl="5m")
func (s *UserService) ListUsers(page, pageSize int) ([]*User, error) {
	return []*User{{ID: "1", Name: "User 1"}}, nil
}

// CreateUser creates a new user
// @cacheable(cache="users", key="#user.ID", ttl="30m")
func (s *UserService) CreateUser(user *User) (*User, error) {
	return user, nil
}

// DeleteUser deletes a user by ID
func (s *UserService) DeleteUser(id string) error {
	return nil
}

// User represents a user entity
type User struct {
	ID    string
	Name  string
	Email string
}
