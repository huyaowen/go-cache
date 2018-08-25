package store

import (
	"log"
	"strconv"

	"github.com/garyburd/redigo/redis"
)

const max_pool_size = 50

type redis_store struct {
	poolsize int
	password string
	dbNum    int
	poollist *redis.Pool
	savePath string
}

func instance() *redis_store {

	var rp *redis_store = new(redis_store)

	configs := []string{"", "", ""} //redis config strings

	if len(configs) > 0 {
		rp.savePath = configs[0]
	}

	if len(configs) > 1 {
		poolsize, err := strconv.Atoi(configs[1])
		if err != nil || poolsize < 0 {
			rp.poolsize = max_pool_size
		} else {
			rp.poolsize = poolsize
		}
	} else {
		rp.poolsize = max_pool_size
	}
	if len(configs) > 2 {
		rp.password = configs[2]
	}
	if len(configs) > 3 {
		dbnum, err := strconv.Atoi(configs[3])
		if err != nil || dbnum < 0 {
			rp.dbNum = 0
		} else {
			rp.dbNum = dbnum
		}
	} else {
		rp.dbNum = 0
	}
	rp.poollist = redis.NewPool(func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", rp.savePath)
		if err != nil {
			return nil, err
		}
		if rp.password != "" {
			if _, err = c.Do("AUTH", rp.password); err != nil {
				c.Close()
				return nil, err
			}
		}
		_, err = c.Do("SELECT", rp.dbNum)
		if err != nil {
			c.Close()
			return nil, err
		}
		return c, err
	}, rp.poolsize)

	err := rp.poollist.Get().Err()

	if err != nil {
		log.Fatalln(err)
	}

	return rp
}
