package service

import "libs/cache"

// Cache ...
type Cache struct {
	Type
}

// CacheGetArgs ...
type CacheGetArgs struct {
	Common	CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Name   	string    	`msgpack:"name" must:"0" comment:"key"`
	Key   	string    	`msgpack:"key" must:"1" comment:"key"`
}

// CacheGetResult ...
type CacheGetResult struct {
	Result Result
	Data   interface{}
}

// Get ...
func (c *Cache) Get(args *CacheGetArgs, reply *CacheGetResult) error {
	var cachePtr *cache.Cache
	if args.Name == "" {
		cachePtr = c.Ctx.Cache
	} else {
		var exists bool
		cachePtr, exists = c.Ctx.Caches[args.Name]
		if !exists {
			reply.Result = Result{Status: 1, Error: "cache name not found"}
			return nil
		}
	}
	if args.Key == "" {
		reply.Result = Result{Status: 1, Error: "invalid argument"}
		return nil
	}
	reply.Result = Result{Status: 0, Error: "success"}
	result, _ := cachePtr.Get(args.Key)
	(*reply).Data = result
	return nil
}

// Del ...
func (c *Cache) Del(args *CacheGetArgs, reply *CacheGetResult) error {
	var cachePtr *cache.Cache
	if args.Name == "" {
		cachePtr = c.Ctx.Cache
	} else {
		var exists bool
		cachePtr, exists = c.Ctx.Caches[args.Name]
		if !exists {
			reply.Result = Result{Status: 1, Error: "cache name not found"}
			return nil
		}
	}
	if args.Key == "" {
		reply.Result = Result{Status: 1, Error: "invalid argument"}
		return nil
	}
	reply.Result = Result{Status: 0, Error: "success"}
	cachePtr.Delete(args.Key)
	return nil
}
