package testdata

// User represents a user entity
type User struct {
	ID   int64
	Name string
}

// Metadata represents additional metadata
type Metadata struct {
	Version   int
	Timestamp int64
}

// ComplexTypeService tests complex type handling
type ComplexTypeService struct{}

// @cacheable(cache="users", key="#ids", ttl="30m")
func (s *ComplexTypeService) ListUsers(ids []int64) ([]*User, error) {
	return []*User{{ID: 1, Name: "User1"}}, nil
}

// @cacheable(cache="usermap", key="#ids", ttl="30m")
func (s *ComplexTypeService) GetUserMap(ids []int64) (map[int64]*User, error) {
	return map[int64]*User{1: {ID: 1, Name: "User1"}}, nil
}

// @cacheable(cache="watch", key="#id", ttl="5m")
func (s *ComplexTypeService) WatchUser(id int64) (<-chan *User, error) {
	ch := make(chan *User)
	return ch, nil
}

// @cacheable(cache="usermeta", key="#id", ttl="30m")
func (s *ComplexTypeService) GetUserWithMeta(id int64) (*User, *Metadata, error) {
	return &User{ID: id, Name: "User"}, &Metadata{Version: 1, Timestamp: 0}, nil
}

// @cacheable(cache="users", key="#id", ttl="30m")
func (s *ComplexTypeService) GetByID(id int64) (*User, error) {
	return &User{ID: id, Name: "User"}, nil
}
