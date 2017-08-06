package utils

import (
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"net/url"
	"time"
	"strconv"
)

var errorInvalidScheme = errors.New("invalid Redis database URI scheme")

type Redis struct {
	p *redis.Pool // redis connection pool
}

func NewReidsFromUriString(uriString string) (*Redis, error) {
	p, err := redisPoolFromUri(uriString)
	return &Redis{p: p}, err
}

func redisPoolFromUri(uriString string) (*redis.Pool, error) {
	uri, err := url.Parse(uriString)
	if err != nil {
		return nil, err
	}

	var network string
	var host string
	var password string
	var db string

	switch uri.Scheme {
	case "tcp":
		network = "tcp"
		host = uri.Host
		if uri.User != nil {
			password, _ = uri.User.Password()
		}

		if len(uri.Path) > 1 {
			db = uri.Path[1:]
		}

	case "unix":
		network = "unix"
		host = uri.Path
	default:
		return nil, errorInvalidScheme
	}

	dialFunc := func() (conn redis.Conn, err error) {
		conn, err = redis.Dial(network, host)
		if err != nil {
			return nil, err
		}

		if password != "" {
			_, err := conn.Do("AUTH", password)
			if err != nil {
				conn.Close()
				return nil, err
			}
		}

		if db != "" {
			_, err := conn.Do("SELECT", db)
			if err != nil {
				conn.Close()
				return nil, err
			}
		}

		return
	}

	maxIdle, maxActive, idleTimeout := redisParseQuery(uri.RawQuery)
	return &redis.Pool{
		MaxIdle:     maxIdle,
		MaxActive:   maxActive,
		IdleTimeout: time.Duration(idleTimeout) * time.Second, //按秒超时
		Dial:        dialFunc,
	}, nil
}

func redisParseQuery(uri string) (int, int, int) {
	query, err := url.ParseQuery(uri)
	if err != nil {
		return 2, 0, 0
	}

	var (
		maxIdle int = 8
		maxActive int = 0
		idleTimeout int = 0
	)

	if data, ok := query["maxIdle"]; ok {
		maxIdle = Atoi(data[0], maxIdle)
	}

	if data, ok := query["maxActive"]; ok {
		maxActive = Atoi(data[0], maxActive)
	}

	if data, ok := query["idleTimeout"]; ok {
		idleTimeout = Atoi(data[0], idleTimeout)
	}

	return maxIdle, maxActive, idleTimeout
}

func (r *Redis)Stats() *StorageStats {
	return &StorageStats{
		ActiveConn:r.p.ActiveCount(),
	}
}

func (r *Redis) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	return r.do(commandName, args...)
}

func (r *Redis) do(commandName string, args ...interface{}) (reply interface{}, err error) {
	c := r.p.Get()
	defer c.Close()

	return c.Do(commandName, args...)
}

func (r *Redis) Set(key string, args ...interface{}) interface{} {
	if v, err := r.do("SET", key, args); err == nil {
		return v
	}
	return nil
}

func (r *Redis) Get(key string) interface{} {
	if v, err := r.do("GET", key); err == nil {
		return v
	}
	return nil
}

func (r *Redis)Get1(key string) (reply string, err error) {
	return redis.String(r.do("GET", key))
}

func (r *Redis) GetMulti(keys []string) []interface{} {
	size := len(keys)
	var rv []interface{}
	c := r.p.Get()
	defer c.Close()
	var err error
	for _, key := range keys {
		err = c.Send("GET", key)
		if err != nil {
			goto ERROR
		}
	}

	if err = c.Flush(); err != nil {
		goto ERROR
	}
	for i := 0; i < size; i++ {
		if v, err := c.Receive(); err == nil {
			if v != nil {
				rv = append(rv, v.([]byte))
			} else {
				rv = append(rv, v)
			}
		} else {
			rv = append(rv, err)
		}
	}
	return rv

	ERROR:
	rv = rv[0:0]
	for i := 0; i < size; i++ {
		rv = append(rv, nil)
	}

	return rv
}

func (r *Redis) HGetAllMulti(keys []string) []interface{} {
	size := len(keys)
	var rv []interface{}
	c := r.p.Get()
	defer c.Close()
	var err error
	for _, key := range keys {
		err = c.Send("HGETALL", key)
		if err != nil {
			goto ERROR
		}
	}

	if err = c.Flush(); err != nil {
		goto ERROR
	}
	for i := 0; i < size; i++ {
		if v, err := c.Receive(); err == nil {
			tmp, tmperr := redis.StringMap(v, err)
			if tmperr == nil {
				rv = append(rv, tmp)
			} else {
				rv = append(rv, nil)
			}
		} else {
			rv = append(rv, nil)
		}
	}

	return rv

	ERROR:
	rv = rv[0:0]
	for i := 0; i < size; i++ {
		rv = append(rv, nil)
	}
	return rv
}

func (r *Redis) HMGet(key string, fields []string) (result map[string]interface{}, err error) {
	result = make(map[string]interface{})
	var ret interface{}
	params := []interface{}{key}
	for _, f := range fields {
		params = append(params, f)
	}
	if len(fields) == 0 {
		ret, err = r.Do("HGETALL", key)
	} else {
		ret, err = r.Do("HMGET", params...)
	}

	if err != nil {
		return
	}
	retList := ret.([]interface{})
	if len(fields) == 0 {
		for i := 0; i < len(retList); i += 2 {
			key, okKey := retList[i].([]byte)
			value, okValue := retList[i + 1].([]byte)
			if !okKey || !okValue {
				return nil, errors.New("redigo: ScanMap key not a bulk string value")
			}
			result[string(key)] = interface{}(string(value))
		}
		return
	}
	if len(retList) != len(fields) {
		err = errors.New(fmt.Sprintf("hmget result number(%d) not match field numer(%d)", len(retList), len(fields)))
		return
	}
	for i, f := range fields {
		if retList[i] != nil {
			//result[f] = ""
			//} else {
			result[f] = interface{}(string(retList[i].([]byte)))
		}
	}
	return
}

func (r *Redis)HMGet1(key string, args...interface{}) (map[string]string, error) {
	if len(args) == 0 {
		return nil, errors.New("invalid params:no fields")
	}
	argsCombine := append([]interface{}{key}, args...)

	values, err := redis.Strings(r.do("HMGET", argsCombine...))
	if err != nil {
		return nil, errors.New(err.Error())
	}
	if len(values) != len(args) {
		return nil, errors.New("fields and values don't match")
	}
	result := make(map[string]string)
	for k := range args {
		field, ok := args[k].(string)
		if !ok {
			return nil, errors.New("arg type is not string")
		}
		result[field] = values[k]
	}
	return result, nil
}

func (r *Redis) HGet(key string, field string) (result interface{}, err error) {
	ret, err := r.HMGet(key, []string{field})
	if err != nil || len(ret) == 0 {
		return
	}
	if v, exists := ret[field]; exists {
		result = v
	}
	return
}

func (r *Redis) HGetAll(key string) (result map[string]interface{}, err error) {
	return r.HMGet(key, []string{})
}

func (r *Redis) HSet(key string, field string, value interface{}) error {
	_, err := r.do("HSET", key, field, value)
	if err == nil {
		return nil
	}
	return err
}

func (r *Redis) HDel(key string, field string) error {
	_, err := r.do("HDEL", key, field)
	if err == nil {
		return nil
	}
	return err
}

func (r *Redis) HMSet(key string, values map[string]interface{}) error {
	args := []interface{}{key}
	for field, value := range values {
		args = append(args, field)
		args = append(args, value)
	}
	_, err := r.do("HMSET", args...)
	if err == nil {
		return nil
	}
	return err
}

func (r *Redis) ZADDMulti(key string, score int64, values []string) error {
	args := []interface{}{key}
	for _, value := range values {
		args = append(args, score)
		args = append(args, value)
	}
	_, err := r.do("ZADD", args...)
	if err == nil {
		return nil
	}
	return err
}

func (r *Redis) ZRevRange(key string, start int, stop int) (values []string, scores []int, err error) {
	ret, err := r.Do("ZREVRANGE", key, start, stop, "withscores")
	if err != nil {
		return
	}
	items := ret.([]interface{})
	if items == nil || len(items) % 2 != 0 {
		err = errors.New("invalid items")
		return
	}
	for i := 0; i < len(items); i += 2 {
		values = append(values, string(items[i].([]byte)))
		n, _ := strconv.Atoi(string(items[i + 1].([]byte)))
		scores = append(scores, n)
	}
	return
}

func (r *Redis)ZRevRangeWithScore(key string, start, stop int) (values []string, err error) {
	values, err = redis.Strings(r.do("ZREVRANGE", key, start, stop, "WITHSCORES"))
	return
}

func (r *Redis)ZRevRangeWithoutScore(key string, start, stop int) (values []string, err error) {
	resI, err := r.do("ZREVRANGE", key, start, stop)
	if err != nil {
		return
	}
	resS, ok := resI.([]interface{})
	if !ok {
		return nil, errors.New("empty data")
	}

	values = make([]string, len(resS))
	for k := range resS {
		if item, ok := resS[k].([]byte); ok {
			values[k] = string(item)
		}
	}
	return
}

func (r *Redis)ZCard(key string) (count int64, err error) {
	count, err = redis.Int64(r.do("ZCARD", key))
	return
}

func (r *Redis)ZCount(key string, min, max int64) (count int64, err error) {
	count, err = redis.Int64(r.do("ZCOUNT", key, min, max))
	return
}

func (r *Redis)LTrim(key string, start, stop int) (err error) {
	_, err = r.do("LTRIM", key, start, stop)
	return
}

func (r *Redis)LRange(key string, start, stop int) (values []string, err error) {
	values, err = redis.Strings(r.do("LRANGE", key, start, stop))
	return
}

func (r *Redis)LPush(key string, item interface{}) (err error) {
	_, err = r.do("LPUSH", key, item)
	return
}

func (r *Redis)ZRem(key string, item interface{}) (err error) {
	_, err = r.do("ZREM", key, item)
	return
}

func (r *Redis) Expire(key string, t int) error {
	_, err := r.do("EXPIRE", key, t)
	if err == nil {
		return nil
	}
	return err
}

func (r *Redis) Delete(key string) error {
	var err error
	if _, err = r.do("DEL", key); err != nil {
		return err
	}
	return nil
}

func (r *Redis) IsExist(key string) bool {
	v, err := redis.Bool(r.do("EXISTS", key))
	if err != nil {
		return false
	}

	return v
}

func (r *Redis) Incr(key string) error {
	_, err := redis.Bool(r.do("INCRBY", key, 1))
	return err
}

func (r *Redis) Decr(key string) error {
	_, err := redis.Bool(r.do("INCRBY", key, -1))
	return err
}

func (r *Redis) Conn() (interface{}, error) {
	if r.p != nil {
		return r.p.Get(), nil
	} else {
		return nil, fmt.Errorf("redis pool is nil")
	}
}

func (r *Redis) Close() {
	r.p.Close()
}
