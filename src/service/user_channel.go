package service

import (
	"fmt"
	"encoding/json"
	"errors"
	"strconv"
	"time"
)

const (
	userChannelPrefix = "CH:"
	userWeiboChannelPrefix = "WCH:"
)

// UserChannelResult ...
type UserChannelResult struct {
	Result Result
	Data   	*UserChannelData
}

// UserChannelData ...
type UserChannelData struct {
	Attributes 		map[string]interface{}    `msgpack:"attributes"`
	Subscribed		[]map[string]interface{}  `msgpack:"subscribed"`
}

// UserWeiboChannelResult ...
type UserWeiboChannelResult struct {
	Result Result
	Data   string
}

// UserChannelGetArgs ...
type UserChannelGetArgs struct {
	Common		CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid         int64      	`msgpack:"uid" must:"1" comment:"uid"`
	Imei        string   	`msgpack:"imei" must:"1" comment:"imei"`
}

// UserWeiboChannelGetArgs ...
type UserWeiboChannelGetArgs struct {
	Common CommonArgs `msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid    int64      `msgpack:"uid" must:"1" comment:"uid"`
	Flag   bool       `msgpack:"flag" must:"0" comment:"flag"`
}

// UserChannelUpdateArgs ...
type UserChannelUpdateArgs struct {
	Common		CommonArgs 			`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid         int64      			`msgpack:"uid" must:"1" comment:"uid"`
	Imei        string   			`msgpack:"imei" must:"1" comment:"imei"`
	Data   		*UserChannelData    `msgpack:"data" must:"1" comment:"data"`
}

// UserWeiboChannelUpdateArgs ...
type UserWeiboChannelUpdateArgs struct {
	Common      CommonArgs         `msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid         int64              `msgpack:"uid" must:"1" comment:"uid"`
	Data        string             `msgpack:"data" must:"1" comment:"data" disable_log:"1"`
}

func (u *User) getUserChannelKey(uid int64, imei string) (key string, oldKey string) {
	if uid == 0 {
		key = userChannelPrefix + imei
		oldKey = "imei_" + imei
	} else {
		key = fmt.Sprintf("%s%d", userChannelPrefix, uid)
		oldKey = fmt.Sprintf("user_%d", uid)
	}
	return
}

// GetWeiboChannel ...
func (u *User) GetWeiboChannel(args *UserWeiboChannelGetArgs, reply *UserWeiboChannelResult) error {
	reply.Result = Result{Status:0, Error:"success"}

	if args.Uid <= 0 {
		reply.Result = Result{Status:1, Error:"paras error"}
		return nil
	}

	// 直接取初始频道
	if args.Flag == true {
		initChannels, err := u.getWeiBoInitChannel()
		if err != nil {
			reply.Result = Result{Status:2, Error:"empty"}
			return nil
		}
		reply.Data = initChannels
		return nil
	}

	key := fmt.Sprintf("%s%d", userWeiboChannelPrefix, args.Uid)

	r, err := u.Ctx.Ssdb("subscribe_ssdb_r", 0)
	if err != nil {
		u.Error("get user ssdb failed:%v", err)
		reply.Result = Result{Status:1, Error:fmt.Sprintf("system error:%v", err)}
		return errors.New("system error")
	}

	result, err := r.Get(key)
	if err != nil {
		u.Error("get channl failed:%v uid=%d", err, args.Uid)
		reply.Result = Result{Status:1, Error:"failed"}
		return nil
	}

	if result == "" {
		initChannels, err := u.getWeiBoInitChannel()
		if err != nil {
			reply.Result = Result{Status:2, Error:"empty"}
			return nil
		}
		reply.Data = initChannels
	} else {
		reply.Data = result
	}

	return nil
}

// 从cms后台获取发现广场的初始频道
func (u *User) getWeiBoInitChannel() (string, error) {

	initChannelKey := "WeiBo_Init_Channel"
	returnVal := ""

	// 检查缓存
	cacheResult, found := (*u.Ctx).Cache.Get(initChannelKey)

	if !found {
		initKey := "hash:versions"
		redis, err := u.Ctx.Redis("cms_redis")
		if err != nil {
			u.Error("get cms_redis failed:%v", err)
			return "", errors.New("system error")
		}

		// 取出频道列表
		version, err := redis.HGetAll(initKey)
		if err != nil {
			u.Error("get hash:versions failed:%v", err)
			return "", errors.New("system error")
		}

		// 微博线上频道
		ol_weibo_channel, ok := version["ol_weibo_channel"]
		if !ok {
			return "", errors.New("ol_weibo_channel is empty")
		}

		retValue := make(map[string]interface{})

		err = json.Unmarshal([]byte(ol_weibo_channel.(string)), &retValue)
		if err != nil {
			u.Error("Unmarshal hash:versions failed:%v", err)
			return "", errors.New("Unmarshal Error")
		}

		timeUpdated := ""
		if ts, ok := retValue["time_updated"].(float64); !ok {
			timeUpdated = strconv.FormatInt(time.Now().Unix(), 10)
		} else {
			tsInt := int(ts)
			timeUpdated = strconv.Itoa(tsInt)
		}

		hashKey := "hash:version:ol_weibo_channel:" + timeUpdated

		allCategories, err := redis.HGetAll(hashKey)
		if err != nil {
			u.Error("get allCategories failed:%v", err)
			return "", errors.New("system error")
		}

		allCates, err := json.Marshal(allCategories)

		if err != nil {
			u.Error("Marshal allCategories failed:%v", err)
			return "", errors.New("system error")
		}

		returnVal = string(allCates)
		// 写入缓存
		u.Ctx.Cache.Set(initChannelKey, string(allCates), 3 * time.Minute)
	} else {
		returnVal = cacheResult.(string)
	}

	return returnVal, nil
}

// GetChannel ...
func (u *User) GetChannel(args *UserChannelGetArgs, reply *UserChannelResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	result, err := u.getChannel(args.Uid, args.Imei)
	if err != nil {
		u.Error("get channel failed:%v", err)
		reply.Result = Result{Status: 1, Error: err.Error()}
		return err
	}

	reply.Data = result
	return nil
}

func (u *User) getChannel(uid int64, imei string) (resultData *UserChannelData, err2 error) {
	r, err := u.Ctx.Ssdb("subscribe_ssdb_r", 0)
	if err != nil {
		u.Error("get user ssdb failed:%v", err)
		err2 = errors.New("system error")
		return
	}
	w, err := u.Ctx.Ssdb("subscribe_ssdb_w", 0)
	if err != nil {
		u.Error("get user ssdb failed:%v", err)
		err2 = errors.New("system error")
		return
	}
	channelRedis, err := u.Ctx.Redis("subscribed_w", 0)
	if err != nil {
		u.Error("get subscribed_r redis failed:%v", err)
		err2 = errors.New("system error")
		return
	}
	key, oldKey := u.getUserChannelKey(uid, imei)

	result, err := r.Get(key)

	if err != nil || result == "" {
		u.Info("not found in ssdb result=%v, err=%v", result, err)
		ret, err := channelRedis.HGet("subscribed", oldKey)
		u.Info("redis result=%v, oldKey=%v, err=%v", ret, oldKey, err)
		if err == nil && ret != nil && ret != "" && ret != "{}" {
			var result map[string]interface{}
			resultChnnel := &UserChannelData {
				Attributes:make(map[string]interface{}), Subscribed:[]map[string]interface{}{}}
			json.Unmarshal([]byte(ret.(string)), &result)
			for k, v := range(result) {
				if k == "subscribed" {
					subscribed := []interface{}{}
					switch vList := v.(type) {
					case map[string]interface {}:
						u.Error("other format oldKey=%v ret=%v", oldKey, ret)
						// 极个别的格式不对，需要修复。原因未知
						fixSubscribed := make([]interface{}, len(vList))
						for index,item := range(vList) {
							i, _ := strconv.Atoi(index)
							if i > len(fixSubscribed) - 1 {
								continue
							}
							fixSubscribed[i] = item
						}
						for _, ch := range(fixSubscribed) {
							if ch != nil {
								subscribed = append(subscribed, ch)
							}
						}
						result["subscribed"] = subscribed
						fixedResult, err := json.Marshal(result)
						if err != nil {
							channelRedis.HSet("subscribed", oldKey, fixedResult)
						}
					case []interface{}:
						subscribed = v.([]interface{})
					}

					for _, ch := range(subscribed) {
						resultChnnel.Subscribed = append(resultChnnel.Subscribed, ch.(map[string]interface{}))
					}
				} else {
					resultChnnel.Attributes[k] = v
				}
			}
			resultJSON, err := json.Marshal(resultChnnel)
			if err == nil {
				w.Set(key, string(resultJSON))
			} else {
				u.Info("json encode faled:%v", err)
			}
			resultData = resultChnnel
			return
		}
		u.Info("get old subscribed failed, ret=%v, err=%v", ret, err)
	} else {
		var resultChnnel UserChannelData
		err = json.Unmarshal([]byte(result), &resultChnnel)
		if err == nil {
			for _, ch := range(resultChnnel.Subscribed) {
				switch ch["cate_id"].(type) {
				case float64:
					ch["cate_id"] = fmt.Sprintf("%d", int64(ch["cate_id"].(float64)))
				}
			}
			resultData = &resultChnnel
			return
		}
	}

	return
}

func (u *User) updateChannel(uid int64, imei string, newData *UserChannelData) error {
	w, err := u.Ctx.Ssdb("subscribe_ssdb_w", 0)
	if err != nil {
		u.Error("get user ssdb failed:%v", err)
		return errors.New("system error")
	}
	//key, oldKey := this.getUserChannelKey(uid, imei)
	key, _ := u.getUserChannelKey(uid, imei)

	// 双写老redis
	/*
	channelRedis, err := this.Ctx.Redis("subscribed_w", 0)
	if err != nil {
		this.Ctx.Logger.Errorf("[%v]get subscribed_w redis failed:%v", seqid, err)
		return errors.New("system error")
	}

	oldChannelData := make(map[string]interface{})
	for k, v := range(newData.Attributes) {
		oldChannelData[k] = v
	}
	subscribed := []map[string]interface{}{}
	for _, v := range(newData.Subscribed) {
		subscribed = append(subscribed, v)
	}
	oldChannelData["subscribed"] = subscribed
	oldJson, err := json.Marshal(oldChannelData)
	if err == nil {
		this.Ctx.Logger.Infof("[%v]save redis key=%v value=%v err=%v\n", seqid, oldKey, string(oldJson), err)
		channelRedis.HSet("subscribed", oldKey, string(oldJson))
	} else {
		this.Ctx.Logger.Infof("[%v]json encode faled:%v\n", seqid, err)
	}*/

	// 写新ssdb
	newJSON, err := json.Marshal(newData)
	if err == nil {
		w.Set(key, string(newJSON))
	} else {
		u.Error("updateChannel json encode faled:%v", err)
	}

	return nil
}

// UpdateChannel ...
func (u *User) UpdateChannel(args *UserChannelUpdateArgs, reply *UserChannelResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	err := u.updateChannel(args.Uid, args.Imei, args.Data)
	if err != nil {
		u.Error("get channel failed:%v", err)
		reply.Result = Result{Status: 1, Error: err.Error()}
		return err
	}

	return nil
}

// UpdateWeiboChannel ...
func (u *User) UpdateWeiboChannel(args *UserWeiboChannelUpdateArgs, reply *UserWeiboChannelResult) error {
	if args.Uid <= 0 || args.Data == "" {
		reply.Result = Result{Status:1, Error:"paras error"}
		return nil
	}
	key := fmt.Sprintf("%s%d", userWeiboChannelPrefix, args.Uid)

	w, err := u.Ctx.Ssdb("subscribe_ssdb_w", 0)
	if err != nil {
		u.Error("get user ssdb failed:%v", err)
		reply.Result = Result{Status:1, Error:fmt.Sprintf("system error:%v", err)}
		return errors.New("system error")
	}

	reply.Result = Result{Status:0, Error:"success"}
	_, err = w.Do("setx", key, args.Data, 24*3600*3650)
	if err != nil {
		u.Error("update weibo channel failed:%v", err)
		reply.Result = Result{Status:1, Error:"update WeiboChannel failed"}
	}

	return nil
}
