package cache

import "time"

type cache interface {

	//get cache name
	GetName() (key interface{}, err error)

	//get current cache type
	GetNativeCache() (name string, err error)

	//get cache value by key
	Get(key interface{}) (result interface{}, err error)

	//put cache value
	Put(key, value interface{}, duration time.Duration) (err error)

	//delete cache key value
	Evict(key interface{}) (err error)

	//clear all cache data
	Clear() (err error)
}
