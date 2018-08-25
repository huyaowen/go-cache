package store

import "time"

const (
	REDIS  = "redis"
	MEMORY = "memory"
	MONGO  = "mongo"
)

type backend interface {
	Get(key interface{}) (result interface{}, err error)

	Set(key, value interface{}, duration time.Duration) (err error)

	SetE(key, value interface{}, duration time.Duration) (err error)

	Delete(key interface{}) (v interface{}, err error)

	DeleteExpired() (err error)

	Flush() (err error)

	Keys() (keys []interface{}, err error)
}

func Backend(backend string) backend {

	switch backend {
	case REDIS:
		return nil
	case MEMORY:
		return NewMemoryStore(DefaultExpiration, DefaultCleanupInterval)
	case MONGO:
		return nil
	default:
		return nil
	}

}
