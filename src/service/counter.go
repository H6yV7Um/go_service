package service

import (
	"strconv"
)

// Counter 计数类
type Counter struct {
	Type
}

// CounterType ...
type CounterType int

// ContentType 模块类型
type ContentType int

const (
	// CounterTypeComment 评论计数器
	CounterTypeComment CounterType = 1
	
	// ContentTypeLike 赞模块
	ContentTypeLike ContentType = 1
	
	// redis存储
	redisCounterW        = "m7018"
	redisCounterR        = "r7018"
	
	// 计数器类型前缀
	commentCounterPrefix = "comment_"
	
	// 模块类型前缀
	likePrefix           = "like_"
)

// Increase 增加计数 id :  文章ID或评论ID; counterType : 计数器类型; contentType : 业务类型
func (c *Counter) Increase(id string, counterType CounterType, contentType ContentType) bool {
	if id == "" || !c.checkCounterType(counterType) || !c.checkContentType(contentType) {
		return false
	}

	w, err := c.Ctx.Redis(redisCounterW)
	if err != nil {
		c.Error("Counter get redis failed:%v", err)
		return false
	}

	key := c.getCounterKey(id, counterType, contentType)
	err = w.Incr(key)
	if err != nil {
		c.Error("Counter inc failed, id:%v", id)
		return false
	}

	return true
}

// Decrease 减少计数
func (c *Counter) Decrease(id string, counterType CounterType, contentType ContentType) bool {
	if id == "" || !c.checkCounterType(counterType) || !c.checkContentType(contentType) {
		return false
	}

	w, err := c.Ctx.Redis(redisCounterW)
	if err != nil {
		c.Error("Counter get redis failed:%v", err)
		return false
	}

	key := c.getCounterKey(id, counterType, contentType)
	err = w.Decr(key)
	if err != nil {
		c.Error("Counter dec failed, id:%v", id)
		return false
	}

	return true
}

// Get RETURN: -1 失败; >= 0 成功
func (c *Counter) Get(id string, counterType CounterType, contentType ContentType) int {
	if id == "" || !c.checkCounterType(counterType) || !c.checkContentType(contentType) {
		return -1
	}

	r, err := c.Ctx.Redis(redisCounterR)
	if err != nil {
		c.Error("Counter get redis failed:%v", err)
		return -1
	}

	key := c.getCounterKey(id, counterType, contentType)
	ret := r.Get(key)
	
	data := 0 // 无结果时,默认返回0
	if ret != nil {
		data, _ = strconv.Atoi(string(ret.([]byte)))
	}
	
	if data < 0 {
		c.recoveryCounter(id, counterType, contentType)
		data = 0
	}
	
	return data
}

// GetBatch 批量获取
func (c *Counter) GetBatch(ids []string, counterType CounterType, contentType ContentType) []int {
	counters := make([]int, len(ids))
	if len(ids) == 0 || !c.checkCounterType(counterType) || !c.checkContentType(contentType) {
		return counters
	}
	
	r, err := c.Ctx.Redis(redisCounterR)
	if err != nil {
		c.Error("Counter get redis failed:%v", err)
		return counters
	}
	
	keys := c.getCounterBatchKeys(ids, counterType, contentType)
	ret := r.GetMulti(keys)
	if err != nil {
		c.Error("Counter get faield, ids:%v", ids)
		return counters
	}
	
	for i := range(ret) {
		data, _ := strconv.Atoi(ret[i].(string))
		if data < 0 {
			c.recoveryCounter(ids[i], counterType, contentType)
			data = 0
		}
		counters = append(counters, data)
	}
	
	return counters
}

func (c *Counter) recoveryCounter(id string, counterType CounterType, contentType ContentType) {
	w, err := c.Ctx.Redis(redisCounterW)
	if err != nil {
		c.Error("Counter get redis failed:%v", err)
		return
	}
	
	key := c.getCounterKey(id, counterType, contentType)
	_, err = w.Do("set", key, 0)
	if err != nil {
		c.Error("Counter set failed, id:%v", id)
		return
	}
	
	return
}

func (c *Counter) checkCounterType(counterType CounterType) bool {
	if counterType != CounterTypeComment {
		return false
	}

	return true
}

func (c *Counter) checkContentType(contentType ContentType) bool {
	if contentType != ContentTypeLike {
		return false
	}
	
	return true
}

func (c *Counter) getCounterKey(id string, counterType CounterType, contentType ContentType) string {
	var key string
	
	var contentTypeString string
	if contentType == ContentTypeLike {
		contentTypeString = likePrefix
	}
	
	var counterTypeString string
	if counterType == CounterTypeComment {
		counterTypeString = commentCounterPrefix
	}
	
	key = counterTypeString + contentTypeString + id

	return key
}

func (c *Counter) getCounterBatchKeys(ids []string, counterType CounterType, contentType ContentType) []string {
	var keys []string
	
	for i := range(ids) {
		key := c.getCounterKey(ids[i], counterType, contentType)
		keys = append(keys, key)
	}
	
	return keys
}