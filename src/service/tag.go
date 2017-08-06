package service

import (
	"strings"
	"utils"
	"fmt"
	"errors"
	"time"
)

const (
	subscribeCountPrefix = "SUBC:"
	tagInfoExporeTime = time.Hour * 24
	feedTagPrefix = "feedu:"
	feedTagCountCachePrefix = "feedu_count:"
	feedTagCountCacheExpireTime = time.Minute * 10
)

// Tag ...
type Tag struct {
	Type
}

// TagArgs ...
type TagArgs struct {
	Common	CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Oid 	[]string 	`msgpack:"oid" must:"1" comment:"oid"`
	Uid 	int64 		`msgpack:"uid" must:"1" comment:"uid"`
	Imei 	string		`msgpack:"imei" must:"1" comment:"imei"`
}

// TagResult ...
type TagResult struct {
	Result Result
	Data   map[string]interface{}
}

// TagInfo ...
type TagInfo struct {
	Name 			string
	Type 			string
	Time 			int
	SubscribeCount	int
	Subscribed		int
}

// Get ...
func (t *Tag) Get(args *TagArgs, reply *TagResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	oids := []string{}
	for _, oid := range(args.Oid) {
		if strings.Contains(oid, ":") {   // TODO: add isValidOid() to utils
			oids = append(oids, oid)
		}
	}
	if len(oids) == 0 {
		t.Info("empty oid")
		reply.Result = Result{Status: 1, Error: "empty oid"}
		return errors.New("empty oid")
	}
	result, err := t.get(oids, args.Uid, args.Imei)
	if err != nil {
		t.Info("get tag data faild, err=%v", err)
		reply.Result = Result{Status: 1, Error: "failed"}
		return err
	}
	reply.Data = result
	return nil
}

func (t *Tag) getTagName(oid string) (string, error) {
	cacheKey := "TAG_NAME:" + oid
	info, found := t.Ctx.Cache.Get(cacheKey)
	if !found {
		if t.Ctx.Cfg.Switch.DisableGeObjectInfoBatch {  // 降级，不调用大数据接口
			return "", nil
		}
		ret, errAPI := t.Ctx.WeiboApiJson("i.api.weibo.com/2/darwin/platform/info/get_object_info_batch.json",
			map[string]interface{}{"object_ids": oid, "need_base_info": true, "retry": true})
		if errAPI != nil {
			t.Info("get_object_info_batch failed:%v", errAPI)
			return "", errAPI
		}
		if errStr, exists := ret["error"]; exists {
			t.Info("get_object_info_batch failed:%v", errStr)
			return "", errors.New(errStr.(string))
		}
		retData, exists := ret[oid]
		if !exists {
			t.Info("getTagName failed, ret=%v", ret)
			return "", errors.New("getTagName failed")
		}
		obj, exists := retData.(map[string]interface{})["baseInfo"]
		if !exists {
			t.Info("getTagName failed, ret=%v", ret)
			return "", errors.New("getTagName failed")
		}
		objMap := obj.(map[string]interface{})
		if !utils.KeysExists(objMap, []string{"displayName", "objectType"}) {
			t.Info("getTagName failed, ret=%v", ret)
			return "", errors.New("getTagName failed")
		}
		tagName := objMap["displayName"].(string)
		t.Ctx.Cache.Set(cacheKey, tagName, tagInfoExporeTime)
		return tagName, nil
	}
	return info.(string), nil
}

func (t *Tag) get(oids []string, uid int64, imei string) (data map[string]interface{}, err error) {
	if len(oids) == 0 || oids[0] == "" {
		t.Info("get_object_info_batch got empty oid(why?) oids=%v", oids)
		return
	}

	result := make(map[string]interface{})
	for _, oid := range oids {
		tagName, err := t.getTagName(oid)
		t.Info("getTagName %s:%s", oid, tagName)
		if err != nil {
			continue
		}
		result[oid] = &TagInfo{Name:tagName}
	}
	counts := t.getSubscribeCount(oids)
	for oid := range(result) {
		if count, exists := counts[oid]; exists {
			result[oid].(*TagInfo).SubscribeCount = count
		}
	}
	if uid !=0 || imei != "" {
		isSubscribed := (&User{t.Type}).isSubscribed("tag", uid, imei, oids)
		for oid := range(result) {
			if b, exists := isSubscribed[oid]; exists {
				result[oid].(*TagInfo).Subscribed = b
			}
		}
	}
	data = result
	return
}

func (t *Tag) getSubscribeCount(ids []string) (result map[string]int) {
	result = make(map[string]int)
	countKeys := []string{}
	for _, id := range(ids) {
		countKeys = append(countKeys, subscribeCountPrefix + id)
	}
	r, err := t.Ctx.Ssdb("subscribe_ssdb_r", 0)
	if err != nil {
		t.Error("get subscribe_ssdb_w failed:%v", err)
		return
	}
	counts, err := r.MultiGet(countKeys)
	if err != nil {
		t.Info("get counts failed:%v", err)
		return
	}
	for _, id := range(ids) {
		if _, exists := counts[subscribeCountPrefix +id]; !exists {
			counts[subscribeCountPrefix +id] = "0"
		}
	}
	for k, c := range(counts) {
		oid := k[len(subscribeCountPrefix):len(k)]
		result[oid] = utils.Atoi(c, 0)
	}
	return
}

func (t *Tag) updateSubscribeCount(ids []string, updateCount int) {
	countKeys := []string{}
	for _, id := range(ids) {
		countKeys = append(countKeys, subscribeCountPrefix + id)
	}
	w, err := t.Ctx.Ssdb("subscribe_ssdb_w", 0)
	if err != nil {
		t.Error("get subscribe_ssdb_w failed:%v", err)
		return
	}
	counts, err := w.MultiGet(countKeys)
	if err != nil {
		t.Info("get counts failed:%v", err)
		return
	}
	for _, id := range(ids) {
		if _, exists := counts[subscribeCountPrefix +id]; !exists {
			counts[subscribeCountPrefix +id] = "0"
		}
	}

	var keys, values []string
	for k, c := range(counts) {
		n := utils.Atoi(c, 0)
		n += updateCount
		if n < 0 {
			n = 0
		}
		keys = append(keys, k)
		values = append(values, fmt.Sprintf("%d", n))
	}

	err = w.MultiSet(keys, values)
	if err != nil {
		t.Info("set counts failed:%v", err)
		return
	}
}

// 标签流长度
func (t *Tag) getFeedCount(tagID string) int64 {
	if tagID == "" {
		return 0
	}
	feedTagCountCacheKey := feedTagCountCachePrefix + tagID
	if info, found := t.Ctx.Cache.Get(feedTagCountCacheKey); found {
		return info.(int64)
	}

	r, err := t.Ctx.Redis("feed")
	if err != nil {
		return 0
	}
	key := feedTagPrefix + tagID
	count, _ := r.ZCard(key)
	t.Ctx.Cache.Set(feedTagCountCacheKey, count, feedTagCountCacheExpireTime)
	t.Info("feed count for %s:%d", tagID, count)
	return count
}

