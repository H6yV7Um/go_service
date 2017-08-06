package service

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"models"
	"sort"
	"strings"
	"time"
	"utils"
)

const (
	defaultStart = 0
	defaultCount = 10
	userRedisNumber = 16
	userRedisPrefix = "up_"
	favorRedisPrefix = "uf_"
	userSubscribedRedisPrefix = "user_"
	imeiSubscribedRedisPrefix = "imei_"
	profileRedisPrefix = "T:"
	weiboInfoRedisPrefix = "U:"
	userExpireTime = 7 * 24 * time.Hour
	userNameExpireTime = 24 * time.Hour
)

var (
	basicFields = []string{"uid", "imei", "time", "updtime", "token", "location", "aid", "wm", "version",
		"device_brand", "device_model", "device_ua", "device_type"}
)

// User ...
type User struct {
	Type
}

// UserResult ...
type UserResult struct {
	Result Result
	Data   []map[string]interface{}
}

// UserArgs ...
type UserArgs struct {
	Common CommonArgs   `msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid    []int64      `msgpack:"uid" must:"1" comment:"uid"`
	Imei   []string     `msgpack:"imei" must:"0" comment:"imei"`
	Field  []string     `msgpack:"field" must:"0" comment:"fields"`
	Start  int          `msgpack:"start" must:"0" comment:"start"`
	Count  int          `msgpack:"count" must:"0" comment:"count"`
	From   string       `msgpack:"from" must:"0" comment:"from"`
}

// UserUpdateArgs ....
type UserUpdateArgs struct {
	Common CommonArgs                `msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid    int64                        `msgpack:"uid" must:"1" comment:"uid"`
	Imei   string                    `msgpack:"imei" must:"1" comment:"imei"`
	Values map[string]interface{}    `msgpack:"values" must:"1" comment:"values"`
}

// UserNoticeResult ...
type UserNoticeResult struct {
	Result Result
	Data   []map[string]NoticeData
}

// NoticeItemType ....
type NoticeItemType struct {
	Oid       string
	City      string
	Tag       string
	Time      int
	PushTime  int
	WhiteList bool
	PushType  string
}

// NoticeArgs ....
type NoticeArgs struct {
	Common CommonArgs `msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid    int64          `msgpack:"uid" must:"1" comment:"uid"`
	Count  int        `msgpack:"count" must:"1" comment:"获取数量"`
}

// NoticeJSON ...
type NoticeJSON struct {
	PushTime int    `json:"t"`
	Oid      string `json:"o"`
}

// NoticeData ...
type NoticeData struct {
	PushTime int    `json:"pushtime" msgpack:"pushtime"`
}

// FavorItem ...
type FavorItem struct {
	Type  string
	Ftime int
}

// ByFavor ...
type ByFavor []FavorItem

func (b ByFavor) Len() int {
	return len(b)
}
func (b ByFavor) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
func (b ByFavor) Less(i, j int) bool {
	return b[i].Ftime < b[j].Ftime
}

var emptyResult map[string]interface{}

func (u *User) mapApiResult(item map[string]interface{}) (result map[string]interface{}) {
	result = make(map[string]interface{})
	result["idstr"] = utils.GetStringValueByPath(item, "user_show/idstr")
	result["screen_name"] = utils.GetStringValueByPath(item, "user_show/screen_name")
	result["avatar_large"] = utils.GetStringValueByPath(item, "user_show/avatar_large")
	result["id"] = utils.GetIntValueByPath(item, "user_show/id")
	result["uid"] = utils.GetStringValueByPath(item, "user_show/idstr")
	result["avatar_hd"] = utils.GetStringValueByPath(item, "user_show/avatar_hd")
	result["name"] = utils.GetStringValueByPath(item, "user_show/name")
	result["description"] = utils.GetStringValueByPath(item, "user_show/description")
	result["gender"] = utils.GetStringValueByPath(item, "user_show/gender")
	result["verified_type"] = utils.GetIntValueByPath(item, "user_show/verified_type")
	result["verified_reason"] = utils.GetStringValueByPath(item, "user_show/verified_reason")
	result["profile_image_url"] = utils.GetStringValueByPath(item, "user_show/profile_image_url")
	result["collection_count"] = utils.GetIntValueByPath(item, "collection_count")
	result["subscribe_count"] = utils.GetIntValueByPath(item, "subscribe_count")
	return
}

// Get ...
func (u *User) Get(args *UserArgs, reply *UserResult) error {
	if !utils.CheckImei(strings.Join(args.Imei, ",")) {
		reply.Result = Result{Status: 1, Error: "invalid argument"}
		return nil
	}
	reply.Result = Result{Status: 0, Error: "success"}
	start := args.Start
	if start == 0 {
		start = defaultStart
	}
	count := args.Count
	if count == 0 {
		count = defaultCount
	}
	expectedFields := basicFields
	needFavor := false
	needSubscribed := false
	needProfile := false
	needExtraInfo := false
	needCounter := false
	for _, f := range args.Field {
		switch f {
		case "favor":
			needFavor = true
		case "subscribed":
			needSubscribed = true
		case "profile":
			needProfile = true
		case "extra":
			needExtraInfo = true
		case "counter":
			needCounter = true
		case "settings":
		case "tags":
		}
	}

	var result []map[string]interface{}
	for i := 0; i < len(args.Uid); i++ {
		userData := make(map[string]interface{})
		u.getUserBasicAttributes(args.Uid[i], args.Imei[i], expectedFields, &userData)
		if needFavor {
			u.getUserFavor(args.Uid[i], args.Imei[i], start, count, &userData)
		}
		if needSubscribed {
			u.getUserSubscribed(args.Uid[i], args.Imei[i], &userData)
		}
		if needProfile {
			u.getUserProfile(args.Uid[i], args.Imei[i], &userData)
		}
		if needExtraInfo {
			u.getUserExtraInfo(args.Uid[i], args.Imei[i], &userData, args.From)
		}
		if needCounter {
			u.getUserCounters(args.Uid[i], args.Imei[i], &userData)
		}

		if len(userData) > 0 {
			result = append(result, userData)
		}
	}

	if args.From == "api" {
		for _, user := range (result) {
			reply.Data = append(reply.Data, u.mapApiResult(user))
		}
	} else {
		reply.Data = result
	}

	return nil
}

func (u *User) getUserRedis(uid int64, imei string) (r *utils.Redis, w *utils.Redis, key string, err error) {
	var index int
	if uid == 0 {
		index = int(crc32.ChecksumIEEE([]byte(imei)) % userRedisNumber)
		key = userRedisPrefix + imei
	} else {
		index = int(uid % userRedisNumber)
		key = fmt.Sprintf("%s%d", userRedisPrefix, uid)
	}
	r, err = u.Ctx.Redis("user_r", index)
	/*if err == nil {
		r.Do("SELECT", 12)
	}*/
	w, err = u.Ctx.Redis("user_w", index)
	/*if err == nil {
		w.Do("SELECT", 12)
	}*/

	return
}

func (u *User) getUserBasicAttributes(uid int64, imei string, fields []string, result *map[string]interface{}) error {
	r, w, redisKey, err := u.getUserRedis(uid, imei)
	if err != nil {
		u.Error("get user redis failed:%v", err)
		return err
	}

	var ret map[string]interface{}
	ret, err = r.HMGet(redisKey, fields)
	//r.Do("SELECT", 0) // TODO: 用户基本数据也放到db 0
	if err != nil {
		u.Error("get user redis data failed:%v", err)
		return err
	}

	// TODO: 部分数据在接口只写了redis，没有写mysql，问题解决后此处的兼容可以去掉
	if ret == nil || len(ret) == 0 {
		r.Do("SELECT", 12)
		ret, err = r.HMGet(redisKey, fields)
		u.Info("get user from old redis, ret=%v err=%v", ret, err)
		r.Do("SELECT", 0)
		if ret != nil && len(ret) > 0 {
			w.HMSet(redisKey, ret)
			w.Do("EXPIRE", userExpireTime)
		}
	}

	if ret == nil || len(ret) == 0 {
		sql := fmt.Sprintf("select * from toutiao_user where `uid`=%d and `imei`='%s'", uid, imei)
		rows, err := u.Ctx.GetDBInstance("toutiaouser", "toutiao_user_r").Query(sql)
		if err != nil || len(rows) == 0 {
			u.Info("get user data from db empty uid=%v imei=%v err=%v", uid, imei, err)
			return nil
		}
		dbRet, ok := rows[0].(*models.ToutiaoUser)
		if !ok {
			u.Info("get user data from db empty uid=%v imei=%v parse failed", uid, imei)
			return nil
		}
		ret = dbRet.Format()
		u.Info("get user data from db, err=%v ret=%v", err, ret)
		w.HMSet(redisKey, ret)
		w.Do("EXPIRE", userExpireTime)
	}

	if v, exists := ret["location"]; exists {
		var jsonValue map[string]interface{}
		var tVal []byte
		switch t := v.(type) {
		case string:
			tVal = []byte(t)
		case *string:
			tVal = []byte(*t)
		}
		if err := json.Unmarshal(tVal, &jsonValue); err == nil {
			ret["location"] = jsonValue
		}
	} else {
		(*result)["location"] = emptyResult
	}

	*result = ret
	return nil
}

func (u *User) setUserBasicAttributes(uid int64, imei string, values map[string]interface{}) error {
	_, w, redisKey, err := u.getUserRedis(uid, imei)
	if err != nil {
		u.Error("get user redis failed:%v", err)
		return err
	}

	oldValue := make(map[string]interface{})
	err = u.getUserBasicAttributes(uid, imei, basicFields, &oldValue)
	if err != nil {
		u.Error("get old value failed:%v", err)
		return err
	}

	for k, v := range values {
		oldValue[k] = v
	}
	err = w.HMSet(redisKey, oldValue)
	//r.Do("SELECT", 0) // TODO: 用户基本数据也放到db 0
	if err != nil {
		u.Error("set user redis data failed:%v", err)
		return err
	}
	w.Do("EXPIRE", userExpireTime)

	return nil
}

// SortFavor map按value某个字段排序（暂时只支持int），返回排序后的map的key的列表
func SortFavor(m map[string]interface{}) (result []string, err error) {
	sortedMap := make(map[int][]string)
	keysArray := []int{}
	for k, v := range m {
		var item FavorItem
		json.Unmarshal([]byte(v.(string)), &item)
		newKey := item.Ftime
		if _, exists := sortedMap[newKey]; !exists {
			sortedMap[newKey] = []string{}
		}
		sortedMap[newKey] = append(sortedMap[newKey], k)
		keysArray = append(keysArray, newKey)
	}
	sort.Ints(keysArray)
	for _, k := range keysArray {
		for _, k2 := range sortedMap[k] {
			result = append(result, k2)
		}
	}
	return
}

func (u *User) getUserFavor(uid int64, imei string, start int, count int, result *map[string]interface{}) error {
	(*result)["favor"] = emptyResult
	r, err := u.Ctx.Redis("favor_r")
	if err != nil {
		return err
	}
	r.Do("SELECT", 5)
	var key string
	if uid != 0 {
		key = fmt.Sprintf("%s%d", favorRedisPrefix, uid)
	} else if imei == "" {
		return errors.New("both uid and imei are empty")
	} else {
		key = favorRedisPrefix + imei
	}

	var ret map[string]interface{}
	ret, err = r.HGetAll(key)
	if err != nil {
		u.Error("get user redis data failed:%v", err)
		return err
	}
	if len(ret) == 0 || start > len(ret) - 1 {
		(*result)["favor"] = emptyResult
	} else {
		sortedKeys, _ := SortFavor(ret)
		sortedMap := make(map[string]interface{})
		end := start + count
		if end > len(sortedKeys) {
			end = len(sortedKeys)
		}
		for _, k := range sortedKeys[start:end] {
			if k == "" {
				continue
			}
			sortedMap[k] = ret[k]
		}
		(*result)["favor"] = sortedMap
	}

	return nil
}

func (u *User) getUserSubscribed(uid int64, imei string, result *map[string]interface{}) error {
	(*result)["subscribed"] = emptyResult
	r, err := u.Ctx.Redis("subscribed_r")
	if err != nil {
		return err
	}
	var key string
	if uid != 0 {
		key = fmt.Sprintf("%s%d", userSubscribedRedisPrefix, uid)
	} else if imei == "" {
		return errors.New("both uid and imei are empty")
	} else {
		key = imeiSubscribedRedisPrefix + imei
	}
	var ret map[string]interface{}
	ret, err = r.HMGet("subscribed", []string{key})
	if err != nil {
		u.Error("get user redis data failed:%v", err)
		return err
	}
	if len(ret) == 0 {
		(*result)["subscribed"] = emptyResult
	} else {
		var jsonValue map[string]interface{}
		if err := json.Unmarshal([]byte(ret[key].(string)), &jsonValue); err == nil {
			(*result)["subscribed"] = jsonValue
		}
	}

	return nil
}

func (u *User) getUserProfileRedis(uid int64, imei string, prefix string) (r *utils.Redis, w *utils.Redis, key string, err error) {
	var index int
	if uid == 0 {
		index = int(crc32.ChecksumIEEE([]byte(imei)) % userRedisNumber)
		key = prefix + imei
	} else {
		index = int(crc32.ChecksumIEEE([]byte(fmt.Sprintf("%d", uid))) % userRedisNumber)
		key = fmt.Sprintf("%s%d", prefix, uid)
	}
	r, err = u.Ctx.Redis("user_r", index)
	w, err = u.Ctx.Redis("user_w", index)
	return
}

func (u *User) getUserProfile(uid int64, imei string, result *map[string]interface{}) error {
	// 用户Profile数据，文档：
	// http://redmine.admin.t.sina.cn/projects/algorithm/wiki/User_Profile%E6%95%B0%E6%8D%AE%E5%AD%98%E5%82%A8%E5%92%8C%E6%A0%BC%E5%BC%8F
	(*result)["profile"] = emptyResult
	r, _, redisKey, err := u.getUserProfileRedis(uid, imei, profileRedisPrefix)
	if err != nil {
		u.Error("get user profile redis failed:%v", err)
		return err
	}
	var ret map[string]interface{}
	ret, err = r.HGetAll(redisKey)
	if err != nil {
		u.Error("get user profile data failed:%v", err)
		return err
	}
	(*result)["profile"] = ret
	return nil
}

func (u *User) getUserExtraInfo(uid int64, imei string, result *map[string]interface{}, from string) error {
	// TODO: 如果获取为空，需要从平台和大数据获取
	(*result)["user_show"] = emptyResult
	(*result)["big_data"] = emptyResult
	if uid == 0 {
		return nil
	}

	r, w, redisKey, err := u.getUserProfileRedis(uid, imei, weiboInfoRedisPrefix)
	if err != nil {
		return err
	}

	var ret map[string]interface{}
	ret, err = r.HGetAll(redisKey)
	if err != nil {
		u.Error("get user weibo data failed:%v", err)
		return err
	}
	var weiboInfo []map[string]interface{}
	if v, exists := ret["I"]; exists {
		b := bytes.NewReader([]byte(v.(string)))
		var out bytes.Buffer
		r, _ := zlib.NewReader(b)
		io.Copy(&out, r)
		json.Unmarshal(out.Bytes(), &weiboInfo)
		if len(weiboInfo) > 0 {
			(*result)["big_data"] = weiboInfo[0]
		}
		if len(weiboInfo) > 1 {
			(*result)["user_show"] = weiboInfo[1]
		}
	} else {
		userShowResult, err := u.Ctx.WeiboApiJson("i.api.weibo.com/2/users/show.json",
			map[string]interface{}{"uid": uid})
		if err != nil {
			u.Info("get user show faild, userShowResult=%v", userShowResult)
			return err
		}

		(*result)["user_show"] = userShowResult
		bigDataResult, err := u.Ctx.WeiboApiJson("i.api.weibo.com/2/darwin/platform/info/get_user_info.json",
			map[string]interface{}{"uid": uid})
		if err != nil {
			u.Info("get big data faild, bigDataResult=%v", bigDataResult)
			return err
		}
		(*result)["big_data"] = bigDataResult

		weiboInfo = append(weiboInfo, bigDataResult)
		weiboInfo = append(weiboInfo, userShowResult)
		weiboInfoJSON, err := json.Marshal(weiboInfo)
		if err != nil {
			u.Info("json encode faild, err=%v", err)
			return err
		}
		var in bytes.Buffer
		zw := zlib.NewWriter(&in)
		zw.Write(weiboInfoJSON)
		zw.Close()

		err = w.HSet(redisKey, "I", in.Bytes())
		if err != nil {
			u.Info("redis set failed for user data, err=%v", err)
		}
		w.Expire(redisKey, 3 * 3600 * 24)
	}
	if from != "api" {
		err = u.getUserInterestTag(uid, result)
	}

	return nil
}

func (u *User) getUserScreenInfo(uid int64) (result map[string]interface{}, err error) {
	info := make(map[string]interface{})
	err = u.getUserExtraInfo(uid, "", &info, "")
	if err != nil {
		return
	}
	info = u.mapApiResult(info)
	if info["uid"] == "" || info["screen_name"] == "" || info["profile_image_url"] == "" {
		err = errors.New("invalid user info")
	}
	result = map[string]interface{}{
		"uid": info["uid"],
		"screen_name": info["screen_name"],
		"profile_image_url": info["profile_image_url"],
		"verified_type": info["verified_type"],
	}
	return
}

func (u *User) getUserCounters(uid int64, imei string, result *map[string]interface{}) error {
	if uid == 0 {
		return nil
	}
	colCount, _ := u.countCollection(uid, imei)
	(*result)["collection_count"] = colCount
	(*result)["subscribe_count"] = u.getSubscribeCount(uid, imei, []string{"source", "tag"})
	return nil
}

func (u *User) getUserInterestTag(uid int64, result *map[string]interface{}) error {
	if uid == 0 {
		return nil
	}
	if profile, exists := (*result)["profile"]; exists {
		w1, existsW1 := (profile.(map[string]interface{}))["W1"]
		w2, existsW2 := (profile.(map[string]interface{}))["W2"]
		w3, existsW3 := (profile.(map[string]interface{}))["W3"]
		if existsW1 && existsW2 && existsW3 {
			(*result)["interest_tag"] = map[string]string{
				"W1": w1.(string), "W2": w2.(string), "W3": w3.(string),
			}
			return nil
		}
	}
	interestTag := make(map[string]string)
	_, w, redisKey, err := u.getUserProfileRedis(uid, "-", profileRedisPrefix)
	if err != nil {
		u.Error("get user profile redis failed:%v", err)
		return err
	}
	for _, level := range []int{1, 2, 3} {
		wData, err := u.Ctx.WeiboApiJson("i2.api.weibo.com/2/darwin/platform/object/user_interest_tag.json",
			map[string]interface{}{"uid": uid, "level": level})
		if err != nil {
			u.Info("get user interest tag faild, result=%v", wData)
			return err
		}
		if results, exists := wData["results"]; exists {
			wValue := ""
			for _, tag := range results.([]interface{}) {
				tagMap := tag.(map[string]interface{})
				if wValue == "" {
					wValue = fmt.Sprintf("%s~%f", tagMap["object_id"].(string), tagMap["weight"].(float64))
				} else {
					wValue = fmt.Sprintf("%s,%s~%f", wValue, tagMap["object_id"].(string), tagMap["weight"].(float64))
				}
			}
			wKey := fmt.Sprintf("W%d", level)
			w.HSet(redisKey, wKey, wValue)
			interestTag[wKey] = wValue
		}
	}
	(*result)["interest_tag"] = interestTag
	return nil
}

// Clear ...
func (u *User) Clear(args *UserArgs, reply *UserResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	reply.Data = []map[string]interface{}{}

	uid := int64(0)
	imei := ""
	if len(args.Uid) > 0 {
		uid = args.Uid[0]
	} else if len(args.Imei) > 0 {
		imei = args.Imei[0]
	} else {
		reply.Result = Result{Status: 1, Error: "invalid uid or imei"}
		return nil
	}

	for _, f := range args.Field {
		result := make(map[string]interface{})
		switch f {
		case "favor":

		case "channel":
			result["field"] = "频道订阅"
			w, err := u.Ctx.Ssdb("subscribe_ssdb_w", 0)
			if err != nil {
				result["msg"] = "系统错误：" + err.Error()
				continue
			}
			channelRedis, err := u.Ctx.Redis("subscribed_w", 0)
			if err != nil {
				result["msg"] = "系统错误：" + err.Error()
				continue
			}
			key, oldKey := u.getUserChannelKey(uid, imei)
			err = channelRedis.HDel("subscribed", oldKey)
			if err != nil {
				result["msg"] = "删除旧格式数据出错：" + err.Error()
				continue
			}
			channelData, _ := w.Get(key)
			err = w.Del(key)
			if err != nil {
				result["msg"] = "删除数据出错：" + err.Error()
				continue
			}
			result["msg"] = "清除成功，data=" + channelData
		case "tags":

		case "profile":

		case "extra":

		case "history":
		}
		if len(result) > 0 {
			reply.Data = append(reply.Data, result)
		}
	}

	return nil
}

// Update ...
func (u *User) Update(args *UserUpdateArgs, reply *UserResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	if args.Uid == 0 && args.Imei == "" {
		reply.Result = Result{Status: 1, Error: "uid and imei both empty"}
		return nil
	}

	args.Values["uid"] = args.Uid
	args.Values["imei"] = args.Imei
	args.Values["updtime"] = time.Now().Unix()
	dbFields := (&models.ToutiaoUser{}).Format()
	for k := range(args.Values) {
		if _, exists := dbFields[k]; !exists {
			delete(args.Values, k)
		}
	}
	_, err := u.Ctx.GetDBInstance("toutiaouser", "toutiao_user_w").Update("toutiao_user", args.Values)
	if err != nil {
		u.Info("update user failed:uid=%v imei=%v err=%v", args.Uid, args.Imei, err)
		return nil
	}

	err = u.setUserBasicAttributes(args.Uid, args.Imei, args.Values)
	if err != nil {
		u.Error("setUserBasicAttributes failed:%v", err)
	}

	return nil
}

// GetNotice ....
func (u *User) GetNotice(args *NoticeArgs, reply *UserNoticeResult) error {
	reply.Result = Result{Status: 0, Error: "success"}

	if args.Uid == 0 || args.Count == 0 {
		u.Info("args failds uid=%v count=%v", args.Uid, args.Count)
		return errors.New("uid is not exists or num is not exists")
	}

	uid := fmt.Sprintf("%d", args.Uid)
	count := args.Count
	key := "push_" + uid

	r, err := u.Ctx.Redis("notice_r")
	if err != nil {
		u.Error("get notice redis failed: %v", err)
		return err
	}

	list, err := r.LRange(key, -count, -1)
	if err != nil {
		u.Info("get user notice list failed:%v", err)
		return err
	}

	// 反转slice
	for i, j := 0, len(list) - 1; i < j; i, j = i + 1, j - 1 {
		list[i], list[j] = list[j], list[i]
	}

	listJSON := NoticeJSON{}
	listData := []NoticeJSON{}

	for _, v := range list {
		err := json.Unmarshal([]byte(v), &listJSON)
		if err != nil {
			continue
		}
		listData = append(listData, listJSON)
	}

	reply.Data = []map[string]NoticeData{}
	for _, v := range listData {
		node := make(map[string]NoticeData)
		if !utils.CheckOid(v.Oid) {
			continue
		}
		oid := v.Oid
		node[oid] = NoticeData{
			PushTime: v.PushTime,
		}
		reply.Data = append(reply.Data, node)
	}

	return nil
}

func (u *User) getName(uid int64) string {
	if uid <= 0 {
		return ""
	}
	cacheKey := fmt.Sprintf("UNAME:%d", uid)
	if info, found := u.Ctx.Cache.Get(cacheKey); found {
		name, ok := info.(string)
		if ok {
			u.Info("getName from cache:%s", name)
			return name
		} else {
			u.Info("getName from cache failed:%v", info)
		}
	}

	userShowResult, err := u.Ctx.WeiboApiJson("i.api.weibo.com/2/users/show.json",
		map[string]interface{}{"uid": uid})
	if err != nil {
		u.Error("get user show faild, userShowResult=%v", userShowResult)
		return ""
	}
	name, ok := userShowResult["screen_name"].(string)
	if !ok {
		return ""
	}
	u.Ctx.Cache.Set(cacheKey, name, userNameExpireTime)
	return name
}