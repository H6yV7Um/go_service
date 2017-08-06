package service

import (
	"errors"
	"fmt"
	"hash/crc32"
	"strconv"
	"utils"
	"time"
)

const (
	collectionRedisNumber = 2
	collectionRedisPrefix = "coln_"
	collectionActionAdd = 0
	collectionActionDel = 1
)

// UserCollectionResult ...
type UserCollectionResult struct {
	Result Result
	Data   []map[string]interface{}
}

// UserCollectionExistResult ...
type UserCollectionExistResult struct {
	Result Result
	Data   int
}

type UserCollectionCountResult struct {
	Result Result
	Data   int64
}

// UserCollectionGetArgs ...
type UserCollectionGetArgs struct {
	Common CommonArgs    `msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid    int64          `msgpack:"uid" must:"1" comment:"uid"`
	Imei   string        `msgpack:"imei" must:"1" comment:"imei"`
	Offset int            `msgpack:"offset" must:"1" comment:"offset"`
	Num    int            `msgpack:"num" must:"1" comment:"num"`
}

// UserCollectionUpdateArgs ...
type UserCollectionUpdateArgs struct {
	Common CommonArgs    `msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid    int64          `msgpack:"uid" must:"1" comment:"uid"`
	Imei   string        `msgpack:"imei" must:"1" comment:"imei"`
	Action int            `msgpack:"action" must:"1" comment:"action"` //0:add;1:delete
	ID     []string        `msgpack:"id" must:"1" comment:"id"`
	Ctime  int64        `msgpack:"ctime" must:"1" comment:"ctime"`
}

// UserCollectionExistArgs ...
type UserCollectionExistArgs struct {
	Common CommonArgs    `msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid    int64         `msgpack:"uid" must:"1" comment:"uid"`
	Imei   string        `msgpack:"imei" must:"1" comment:"imei"`
	ID     string        `msgpack:"id" must:"1" comment:"id"`
}

type UserCollectionCountArgs struct {
	Common CommonArgs    `msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid    int64         `msgpack:"uid" must:"1" comment:"uid"`
	Imei   string        `msgpack:"imei" must:"1" comment:"imei"`
}

// GetCollection ...
func (u *User) GetCollection(args *UserCollectionGetArgs, reply *UserCollectionResult) error {
	data, err := u.getCollection(args)
	if err != nil {
		reply.Result = Result{Status: 1, Error: err.Error()}
		return err
	}
	reply.Result = Result{Status: 0, Error: "success"}
	reply.Data = data
	return nil
}

func (u *User) getCollection(args *UserCollectionGetArgs) (data []map[string]interface{}, err error) {
	r, _, key, err := u.getCollectionRedis(args.Uid, args.Imei)
	if err != nil {
		u.Error("get collection pika failed:%s", err.Error())
		err = errors.New("get collection pika failed")
		return
	}
	start := args.Offset
	stop := args.Offset + args.Num - 1
	data, err = getColDataTmp(r, key, start, stop)
	if err != nil {
		u.Error("pika operation failed:%s", err.Error())
		err = errors.New("pika operation failed")
		return
	}
	return
}

// UpdateCollection ...
func (u *User) UpdateCollection(args *UserCollectionUpdateArgs, reply *UserCollectionResult) error {
	if err := u.updateCollection(args); err != nil {
		u.Error("update collection failed:%s", err.Error())
		reply.Result = Result{Status: 1, Error: err.Error()}
		return err
	}
	reply.Result = Result{Status: 0, Error: "success"}
	return nil
}

func (u *User) updateCollection(args *UserCollectionUpdateArgs) error {
	_, w, key, err := u.getCollectionRedis(args.Uid, args.Imei)
	if err != nil {
		u.Error("[%d]get collection pika failed:%s", err.Error())
		return errors.New("get collection pika failed")
	}
	if err := updColData(w, key, args); err != nil {
		u.Error("pika operation update failed:%s", err.Error())
		return errors.New("pika operation failed")
	}
	return nil
}

// HasCollection ...
func (u *User) HasCollection(args *UserCollectionExistArgs, reply *UserCollectionExistResult) error {
	data, err := u.hasCollection(args)
	if err != nil {
		u.Error("HasCollection failed:%s", err.Error())
		reply.Result = Result{Status: 1, Error: err.Error()}
		return err
	}
	reply.Result = Result{Status: 0, Error: "success"}
	reply.Data = data
	return nil
}

func (u *User) hasCollection(args *UserCollectionExistArgs) (data int, err2 error) {
	r, _, key, err := u.getCollectionRedis(args.Uid, args.Imei)
	if err != nil {
		u.Error("get collection pika failed:%s", err.Error())
		err2 = errors.New("get collection pika failed")
		return
	}
	data, err2 = hasColData(r, key, args)
	if err2 != nil {
		u.Error("pika operation failed:%s", err2.Error())
		err2 = errors.New("pika operation failed")
		return
	}
	return
}

func (u *User)CountCollection(args *UserCollectionCountArgs, reply *UserCollectionCountResult) error {
	data, err := u.countCollection(args.Uid, args.Imei)
	if err != nil {
		reply.Result = Result{Status:1, Error:err.Error()}
		return err
	}
	reply.Result = Result{Status:0, Error:"success"}
	reply.Data = data
	return nil
}

func (u *User)countCollection(uid int64, imei string) (data int64, err error) {
	_, w, key, err := u.getCollectionRedis(uid, imei)
	if err != nil {
		u.Error("get collection pika error:%s", err.Error())
		err = errors.New("get collection pika error")
		return
	}

	//data, err = w.ZCard(key)
	cur_time := time.Now().Unix()
	past_time := cur_time - 5 * 30 * 24 * 60 * 60
	data, err = w.ZCount(key, past_time, cur_time)

	if err != nil {
		u.Error("count collection error:%s", err.Error())
		err = errors.New("count collection error")
		return
	}
	return
}

func (u *User) getCollectionRedis(uid int64, imei string) (r *utils.Redis, w *utils.Redis, key string, err error) {
	var index int
	if uid == 0 {
		index = int(crc32.ChecksumIEEE([]byte(imei)) % collectionRedisNumber)
		key = collectionRedisPrefix + imei
	} else {
		index = int(uid % collectionRedisNumber)
		key = fmt.Sprintf("%s%d", collectionRedisPrefix, uid)
	}
	r, err = u.Ctx.Redis("collection_pika_r", index)
	w, err = u.Ctx.Redis("collection_pika_w", index)
	return
}

func getColData(r *utils.Redis, key string, start int, stop int) (res []map[string]interface{}, err error) {
	if start < 0 || stop < 0 || start > stop {
		return nil, errors.New("invalid params")
	}
	resI, err := r.Do("ZREVRANGE", key, start, stop, "withscores")
	if err != nil {
		return
	}

	resS, ok := resI.([]interface{})
	if !ok || resS == nil || len(resS) % 2 != 0 {
		return nil, errors.New("inner error")
	}
	res = make([]map[string]interface{}, len(resS) / 2)
	var ctime int64
	for i := 0; i < len(resS); i += 2 {
		if ctime, err = strconv.ParseInt(string(resS[i + 1].([]byte)), 10, 64); err != nil {
			return nil, errors.New("ctime ParseInt error")
		}
		//提供http访问要以json输出,go Marshal方法对于[]byte默认添加base64编码,因此转换为string
		resB, _ := resS[i].([]byte)
		m := map[string]interface{}{
			"id":   string(resB),
			"ctime": ctime,
		}
		res[i / 2] = m
	}
	return
}

func getColDataTmp(r *utils.Redis, key string, start int, stop int) (res []map[string]interface{}, err error) {
	if start < 0 || stop < 0 || start > stop {
		return nil, errors.New("invalid params")
	}
	count := stop + 1 - start
	cur_time := time.Now().Unix()
	past_time := cur_time - 5 * 30 * 24 * 60 * 60

	resI, err := r.Do("ZREVRANGEBYSCORE", key, cur_time, past_time, "withscores", "limit", start, count)
	if err != nil {
		return
	}

	resS, ok := resI.([]interface{})
	if !ok || resS == nil || len(resS) % 2 != 0 {
		return nil, errors.New("inner error")
	}
	res = make([]map[string]interface{}, len(resS) / 2)
	var ctime int64
	for i := 0; i < len(resS); i += 2 {
		if ctime, err = strconv.ParseInt(string(resS[i + 1].([]byte)), 10, 64); err != nil {
			return nil, errors.New("ctime ParseInt error")
		}
		//提供http访问要以json输出,go Marshal方法对于[]byte默认添加base64编码,因此转换为string
		resB, _ := resS[i].([]byte)
		m := map[string]interface{}{
			"id":   string(resB),
			"ctime": ctime,
		}
		res[i / 2] = m
	}

	return
}

func updColData(w *utils.Redis, key string, args *UserCollectionUpdateArgs) (err error) {
	comArgs := make([]interface{}, 1)
	comArgs[0] = key
	if args.Action == collectionActionAdd {
		for _, oid := range args.ID {
			comArgs = append(comArgs, args.Ctime, oid)
		}
		_, err = w.Do("ZADD", comArgs...)

	} else if args.Action == collectionActionDel {
		for _, oid := range args.ID {
			comArgs = append(comArgs, oid)
		}
		_, err = w.Do("ZREM", comArgs...)
	}
	if err != nil {
		return
	}
	return nil
}

func hasColData(r *utils.Redis, key string, args *UserCollectionExistArgs) (data int, err error) {
	dataI, err := r.Do("ZRANK", key, args.ID)
	if err != nil {
		return 0, err
	}
	if _, ok := dataI.(int64); ok {
		data = 1
	} else {
		data = 0
	}

	return
}
