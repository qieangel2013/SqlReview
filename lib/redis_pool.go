package julive_com

import (
	"../redis"
	"errors"
	"time"
)

var (
	REDIS_CONN_ERROR = errors.New("redis conn error")
)

const (
	SCRIPT_LIST_PUSH = `
local q = KEYS[1]
return redis.call("RPUSH", q, ARGV[1])
`
	SCRIPT_GET = `
local q = KEYS[1]
return redis.call("GET", q)
`
	SCRIPT_SET = `
local q = KEYS[1]
return redis.call("SET", q, ARGV[1])
`
	SCRIPT_DEL = `
local q = KEYS[1]
return redis.call("DEL", q)
`

	SCRIPT_LIST_COUNT = `
local q = KEYS[1]
return redis.call("LLEN", q)
`

	SCRIPT_LIST_POP = `
local q = KEYS[1]
local v = redis.call("LPOP", q)
if v ~= ""
then
	return v
end
return 0
`
)

type RedisConfType struct {
	RedisPw          string
	RedisHost        string
	RedisDb          int
	RedisMaxActive   int
	RedisMaxIdle     int
	RedisIdleTimeOut int
}

func NewRedisPool(redis_conf RedisConfType) *redis.Pool {
	redis_client_pool := &redis.Pool{
		MaxIdle:     redis_conf.RedisMaxIdle,
		MaxActive:   redis_conf.RedisMaxActive,
		IdleTimeout: time.Duration(redis_conf.RedisIdleTimeOut) * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", redis_conf.RedisHost)
			if err != nil {
				return nil, err
			}

			if redis_conf.RedisPw == "" {
				return c, nil
			}

			_, err = c.Do("AUTH", redis_conf.RedisPw)
			if err != nil {
				Error.Println("redis password error")
			}

			// 选择db
			if _, err := c.Do("SELECT", redis_conf.RedisDb); err != nil {
				c.Close()
				Error.Println("redis select db error")
			}

			return c, nil
		},
	}
	return redis_client_pool
}

func ListPush(redis_client *redis.Pool, q string, body string) (int, error) {
	rc := redis_client.Get()
	defer rc.Close()

	script := redis.NewScript(1, SCRIPT_LIST_PUSH)
	resp, err := redis.Int(script.Do(rc, q, body))
	if err == redis.ErrNil {
		err = nil
	}
	return resp, err
}

func ListPop(redis_client *redis.Pool, q string) (resp string, err error) {
	rc := redis_client.Get()
	defer rc.Close()

	script := redis.NewScript(1, SCRIPT_LIST_POP)
	resp, err = redis.String(script.Do(rc, q))
	if err == redis.ErrNil {
		err = nil
	}
	return resp, err
}

func GetKey(redis_client *redis.Pool, q string) (resp string, err error) {
	rc := redis_client.Get()
	defer rc.Close()

	script := redis.NewScript(1, SCRIPT_GET)
	resp, err = redis.String(script.Do(rc, q))
	if err == redis.ErrNil {
		err = nil
	}
	return resp, err
}

func DelKey(redis_client *redis.Pool, q string) (string, error) {
	rc := redis_client.Get()
	defer rc.Close()

	script := redis.NewScript(1, SCRIPT_DEL)
	resp, err := redis.String(script.Do(rc, q))
	if err == redis.ErrNil {
		err = nil
	}
	return resp, err
}

func SetKey(redis_client *redis.Pool, q string, body string) (int, error) {
	rc := redis_client.Get()
	defer rc.Close()

	script := redis.NewScript(1, SCRIPT_SET)
	resp, err := redis.Int(script.Do(rc, q, body))
	if err == redis.ErrNil {
		err = nil
	}
	return resp, err
}

func ListCount(redis_client *redis.Pool, q string) (int, error) {
	rc := redis_client.Get()
	defer rc.Close()

	script := redis.NewScript(1, SCRIPT_LIST_COUNT)
	resp, err := redis.Int(script.Do(rc, q))
	if err == redis.ErrNil {
		err = nil
	}
	return resp, err
}
