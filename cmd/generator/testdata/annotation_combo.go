package testdata

// User represents a user entity
type User struct {
	ID    int64
	Name  string
	Email string
}

// AnnotationComboService tests various annotation combinations
type AnnotationComboService struct{}

// @cacheable(cache="users", key="#id", ttl="30m", condition="#id > 0")
func (s *AnnotationComboService) GetUser(id int64) (*User, error) {
	return &User{ID: id, Name: "Test User"}, nil
}

// @cacheput(cache="users", key="#user.ID", ttl="30m", unless="#user == nil")
func (s *AnnotationComboService) CreateUser(user *User) (*User, error) {
	return user, nil
}

// @cacheevict(cache="users", key="#id", before=true)
func (s *AnnotationComboService) DeleteUser(id int64) error {
	return nil
}

// @cacheable(cache="users", key="#id", ttl="1h", condition="#id > 0 && #id < 1000")
func (s *AnnotationComboService) GetUserWithRange(id int64) (*User, error) {
	return &User{ID: id, Name: "Test User"}, nil
}

// @cacheput(cache="users", key="#user.ID", ttl="30m", unless="#user.ID <= 0")
func (s *AnnotationComboService) UpdateUser(user *User) (*User, error) {
	return user, nil
}

// This method has no annotation and should be skipped
func (s *AnnotationComboService) HelperMethod() string {
	return "helper"
}

// Another method without annotation
func (s *AnnotationComboService) InternalLogic() int {
	return 42
}
