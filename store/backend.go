package store

import "time"

const (
	REDIS  = "redis"
	MEMORY = "memory"
)

type backend interface {
	Get(key interface{}) (result interface{}, err error)

	Set(key, value interface{}, duration time.Duration) (err error)

	Delete(key interface{}) (err error)

	Flush() (err error)

	Keys() (keys interface{}, err error)
}

func Backend(backend string) backend {

	switch backend {
	case REDIS:
		return
	case MEMORY:
		return
	default:
		return
	}

}
