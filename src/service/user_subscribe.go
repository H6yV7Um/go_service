package service

import (
	"fmt"
	"time"
	"encoding/json"
	"strconv"
	"libs/ssdb"
	"container/list"
	"utils"
	"strings"
)

const (
	defaultTargetCount = 2
	defaultListCount = 5
	userSubscribePrefix = "SUB:"
	feedRedisPrefix = "feedu_t:"
	extraDataKey = "config_61"
	disabledTagKey = "config_62"
	tagUpdateTimePrefix = "tag_updtime_"
)

// UserSubscribeResult ...
type UserSubscribeResult struct {
	Result Result
	Data   []interface{}
}

// UserSubscribeArgs ...
type UserSubscribeArgs struct {
	Common		CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid         int64      	`msgpack:"uid" must:"1" comment:"uid"`
	Imei        string   	`msgpack:"imei" must:"1" comment:"imei"`
	Targettype  string   	`msgpack:"target_type" must:"0" comment:"订阅类型，默认为tag"`
	Start       int      	`msgpack:"start" must:"0" comment:"标签列表start"`
	Count       int      	`msgpack:"count" must:"0" comment:"标签列表count"`
	Targetcount int      	`msgpack:"target_count" must:"0" comment:"每个标签返回文章数"`
	Needextra 	int      	`msgpack:"need_extra" must:"0" comment:"是否需要推荐标签"`
	Targetid 	[]string 	`msgpack:"target_id" must:"0" comment:"标签id（用于订阅）"`
}

// SubscribeTargetType ...
type SubscribeTargetType struct {
	Items 		[]string
	Id    		string
	FeedNew 	int
	Name  		string
	Extra 		int
	UpdateTime  int
	LastUpdateTime  int
}

// ExtraTargetType ...
type ExtraTargetType []struct {
	Id 		string
	Name	string
}

// ListSubscribeArgs ...
type ListSubscribeArgs struct {
	Common		CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid         int64      	`msgpack:"uid" must:"1" comment:"uid"`
	Imei        string   	`msgpack:"imei" must:"1" comment:"imei"`
	Targettype  string   	`msgpack:"target_type" must:"0" comment:"订阅类型(tag,source)，默认为全部"`
	Lasttime    int      	`msgpack:"last_time" must:"0" comment:"上一次最老的时间戳，默认0从最新开始取"`
	Count       int      	`msgpack:"count" must:"0" comment:"返回数量"`
}

// SubscribeListItemType ...
type SubscribeListItemType struct {
	Id    	string
	Name  	string
	Type  	string
	Time  	int
}

// ListSubscribeResult ...
type ListSubscribeResult struct {
	Result Result
	Data   []SubscribeListItemType
}

// CheckSubscribedResult ...
type CheckSubscribedResult struct {
	Result Result
	Data   []map[string]interface{}
}

// CheckSubscribedArgs ...
type CheckSubscribedArgs struct {
	Common		CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid         int64      	`msgpack:"uid" must:"1" comment:"uid"`
	Imei        string   	`msgpack:"imei" must:"1" comment:"imei"`
	Targetid 	[]string 	`msgpack:"target_id" must:"0" comment:"订阅id"`
}

func (u *User) getUserSubscribeKey(uid int64, imei string) (key string) {
	if uid == 0 {
		key = userSubscribePrefix + imei
	} else {
		key = fmt.Sprintf("%s%d", userSubscribePrefix, uid)
	}
	return
}

func (u *User) getExtraTarget(start int, count int, targetList *[]string, r *ssdb.SsdbPool, targetKey string) {
	extraData, err := u.Ctx.GetCmsConfig(extraDataKey)
	var extraDataJSON ExtraTargetType
	if err = json.Unmarshal(extraData.([]byte), &extraDataJSON); err == nil {
		extraIDs := []string{}
		for i := 0; i < len(extraDataJSON); i++ {
			extraIDs = append(extraIDs, extraDataJSON[i].Id)
		}
		existsInTarget, err := r.ZMultiGet(targetKey, extraIDs)
		if err != nil {
			return
		}
		skip := 0
		for _, extraID := range(extraIDs) {
			if len(*targetList) >= count {
				break
			}
			if _, exists := existsInTarget[extraID]; exists {
				continue
			}
			if skip < start {
				skip++
				continue
			}
			*targetList = append(*targetList, extraID)
		}
	}
}

func (u *User) getDisabledTags() map[string]int {
	disabledTagsMap := make(map[string]int)
	disabledTagsConfig, err := u.Ctx.GetCmsConfig(disabledTagKey)
	var disabledTagsJSON ExtraTargetType
	if disabledTagsConfig != nil && err == nil {
		if err = json.Unmarshal(disabledTagsConfig.([]byte), &disabledTagsJSON); err == nil {
			for i := 0; i < len(disabledTagsJSON); i++ {
				disabledTagsMap[disabledTagsJSON[i].Id] = 1
			}
		}
	}
	return disabledTagsMap
}

// GetSubscribe ...
func (u *User) GetSubscribe(args *UserSubscribeArgs, reply *UserSubscribeResult) error {
	if !utils.CheckImei(args.Imei) {
		reply.Result = Result{Status: 1, Error: "invalid argument"}
		return nil
	}
	reply.Result = Result{Status: 0, Error: "success"}
	if len(args.Targettype) == 0 {
		u.Error("empty target_type")
		reply.Result = Result{Status: 1, Error: "empty target_type"}
		return nil
	}
	start := args.Start
	if start == 0 {
		start = defaultStart
	}
	count := args.Count
	if count == 0 {
		count = defaultCount
	}
	targetCount := args.Targetcount
	if targetCount <= 0 {
		targetCount = defaultTargetCount
	}

	r, err := u.Ctx.Ssdb("subscribe_ssdb_r", 0)
	if err != nil {
		u.Error("get user ssdb failed:%v", err)
		reply.Result = Result{Status: 1, Error: fmt.Sprintf("system error:%v", err)}
		return err
	}
	redisKey := u.getUserSubscribeKey(args.Uid, args.Imei)
	subscribeUpdateOidsKey := redisKey + "_up_" + args.Targettype
	targetKey := redisKey+":"+args.Targettype
	subscribeUpdateOids := u.getSubscribeUpdateOids(subscribeUpdateOidsKey)
	totalCount, _ := r.Zcount(targetKey)
	var start2, count2 int
	if start > totalCount - 1 {
		start2 = start - totalCount
		count2 = count
	} else if start + count > totalCount - 1 {
		start2 = 0
		count2 = count - (totalCount - start)
	}

	disabledTagsMap := u.getDisabledTags()
	targetList := []string{}
	ret, err := r.Zrrange(targetKey, start, count)
	if err != nil {
		u.Error("get subscribe targets failed:%v", err)
		reply.Result = Result{Status: 1, Error: fmt.Sprintf("get subscribe targets failed:%v", err)}
		return nil
	}
	for _, target := range ret {
		if _, exists := disabledTagsMap[target]; !exists {
			targetList = append(targetList, target)
		} else {
			u.Debug("tag disabled tag=%v\n", target)
		}
	}

	if count2 == 0 && len(targetList) < count {
		count2 = count  // 对count2做修正
	}
	followedTarget := len(targetList)
	if count2 > 0 {
		// 后面有标签取不到名字的情况，所以推荐标签直接取count个，避免返回数据不够
		u.getExtraTarget(start2, count, &targetList, r, targetKey)
	}
	result := []interface{}{}
	feedRedis, err := u.Ctx.Redis("feed", 0)
	if err != nil {
		u.Error("get feed redis failed:%v", err)
		reply.Result = Result{Status: 1, Error: fmt.Sprintf("system error:%v", err)}
		return err
	}
	targetInfo, err := (&Tag{u.Type}).get(targetList, 0, "")
	sortedData := list.New()
	for i, target := range targetList {
		if len(result) > count - 1 {
			break
		}
		feedNew := 0
		targetIDList, scores, err := feedRedis.ZRevRange(feedRedisPrefix +target, 0, targetCount-1)
		if err != nil {
			u.Error("get subscribe targets failed:%v", err)
			reply.Result = Result{Status: 1, Error: fmt.Sprintf("get subscribe targets failed:%v", err)}
			continue
		}
		if len(targetIDList) == 0 {
			continue
		}
		feedUpdateTime := scores[0]
		lastUpdateOids := subscribeUpdateOids[target]
		if !strings.Contains(lastUpdateOids, "@") {
			feedNew = 1 // 对于之前按时间判断更新的,第一次出来标记为新更新,会操作客户端在第一次访问时误报更新
		} else {
			oids := strings.Split(lastUpdateOids, "@")
			targetIDListNum := len(targetIDList)
			oidsNum         := len(oids)
			if targetIDListNum != oidsNum {
				feedNew = 1
			} else {
				j := 0
				for j < oidsNum {
					if targetIDList[j] != oids[j] {
						feedNew = 1
						break
					}
					j++
				}
			}
		}
		if len(targetIDList) == 1 {
			subscribeUpdateOids[target] = targetIDList[0] + "@"
		} else {
			subscribeUpdateOids[target] = strings.Join(targetIDList, "@")
		}
		
		t := targetInfo[target]
		var targetName string
		if t == nil {
			u.Info("get target info failed:%v", target)
			continue
		}
		targetName = targetInfo[target].(*TagInfo).Name   // TODO: empty?
		extra := 0
		if i > followedTarget - 1 {
			extra = 1
		}

		newItem := &SubscribeTargetType{
			Items: targetIDList,
			Extra: extra,
			Id: target,
			FeedNew: feedNew,
			Name: targetName,
			UpdateTime: feedUpdateTime,
			LastUpdateTime: feedUpdateTime}

		u.Debug("%v:extra=%v feednew=%v feedUpdateTime=%v", targetName, extra, feedNew, feedUpdateTime)
		if extra == 0 {
			// 关注标签，整体按新旧排序,新旧内部再按标签下第一篇文章时间排序
			e := sortedData.Front()
			for e != nil{
				item := e.Value.(*SubscribeTargetType)
				if item.Extra == 1 { // 关注标签排在非关注标签之前
					sortedData.InsertBefore(newItem, e)
					u.Debug("    - insert_before %v", item.Name)
					break
				} else if feedNew == 1 { // 待插入关注标签有更新
					if item.FeedNew == 1 { // 已插入关注标签有更新,则根据时间排序
						if item.UpdateTime < feedUpdateTime {
							sortedData.InsertBefore(newItem, e)
							u.Debug("    - insert_before %v", item.Name)
							break
						}
					} else { // 已插入关注标签无更新,则根据更新与否排序
						sortedData.InsertBefore(newItem, e)
						u.Debug("    - insert_before %v", item.Name)
						break
					}
				} else { // 待插入标签无更新
					if item.FeedNew != 1 { // 只与无更新的关注标签比较,且按时间排序
						if item.UpdateTime < feedUpdateTime {
							sortedData.InsertBefore(newItem, e)
							u.Debug("    - insert_before %v", item.Name)
							break
						}
					}
				}
				e = e.Next()
			}
			if nil == e {
				u.Debug("    - push_back")
				sortedData.PushBack(newItem)
			}
		} else {
			u.Debug("    - push_back no extra")
			e := sortedData.Front()
			for e != nil{
				item := e.Value.(*SubscribeTargetType)
				if item.Extra == 1 { // 非关注标签只与非关注标签比较
					if feedNew == 1 { // 待插入非关注标签有更新
						if item.FeedNew == 1 { // 已插入非关注标签同样有更新,根据时间排序
							if item.UpdateTime < feedUpdateTime {
								sortedData.InsertBefore(newItem, e)
								u.Debug("    - insert_before %v", item.Name)
								break
							}
						} else { // 已插入非关注标签无更新,后面的非关注标签肯定也无更新
							sortedData.InsertBefore(newItem, e)
							u.Debug("    - insert_before %v", item.Name)
							break
						}
					} else { // 待插入非关注标签无更新
						if item.FeedNew != 1 { // 只与无更新的已插入非关注标签比较
							if item.UpdateTime < feedUpdateTime {
								sortedData.InsertBefore(newItem, e)
								u.Debug("    - insert_before %v", item.Name)
								break
							}
						}
					}
				}
				e = e.Next()
			}
			if nil == e {
				u.Debug("    - push_back")
				sortedData.PushBack(newItem)
			}
		}
	}

	e := sortedData.Front()
	for e != nil{
		result = append(result, e.Value.(*SubscribeTargetType))
		e = e.Next()
	}
	
	u.updateSubscribeUpdateOids(subscribeUpdateOidsKey, subscribeUpdateOids)
	reply.Data = result

	return nil
}

func (u *User) getFeedUpdateTime(target string) int {
	r, err := u.Ctx.Redis("feed", 0)
	if err != nil {
		u.Error("get feed redis failed:%v", err)
		return 0
	}
	key := tagUpdateTimePrefix + target
	t := r.Get(key)
	if t == nil || err != nil {
		u.Info("getFeedUpdateTime failed:t=%v err=%v", t, err)
		return 0
	}
	i, _ := strconv.Atoi(string(t.([]byte)))
	return i
}

func (u *User) getSubscribeUpdateOids(subscribeUpdateOidsKey string) (result map[string]string) {
	result = make(map[string]string)
	r, err := u.Ctx.Ssdb("subscribe_ssdb_r", 0)
	if err != nil {
		u.Error("getSubscribeOids get subscribe_ssdb_r failed:%v", err)
		return
	}
	t, err := r.Get(subscribeUpdateOidsKey)
	if t == "" || err != nil {
		u.Info("getSubscribeOids failed:t=%v err=%v", t, err)
		return
	}
	err = json.Unmarshal([]byte(t), &result)
	if err != nil {
		u.Error("unmarshal subscribeUpdateOids failed:%v", err)
		return
	}
	return
}


func (u *User) updateSubscribeUpdateOids(subscribeUpdateOidsKey string, value map[string]string) {
	w, err := u.Ctx.Ssdb("subscribe_ssdb_w", 0)
	if err != nil {
		u.Error("updateSubscribeOids get subscribe_ssdb_w failed:%v", err)
		return
	}
	
	b, err := json.Marshal(value)
	if err != nil {
		u.Error("marshal subscribeUpdateOids failed:%v", err)
		return
	}
	
	u.Info("subscribeUpdateOidsKey:%v => %v", subscribeUpdateOidsKey, string(b))
	// TODO: setx 不起作用
	w.Do("setx", subscribeUpdateOidsKey, string(b), 3600*24)
}


func (u *User) isSubscribed(targetType string, uid int64, imei string, ids []string) (result map[string]int) {
	result = make(map[string]int)
	r, err := u.Ctx.Ssdb("subscribe_ssdb_r", 0)
	if err != nil {
		u.Error("get subscribe_ssdb_r failed:%v", err)
		return
	}
	redisKey := u.getUserSubscribeKey(uid, imei)
	ret, err := r.ZMultiGet(redisKey+":"+targetType, ids)
	if err != nil {
		u.Error("get subscribe targets failed:%v", err)
		return nil
	}

	for _, id := range ids {
		if _, exists := ret[id]; exists {
			result[id] = 1
		} else {
			result[id] = 0
		}
	}
	return
}

func (u *User) checkSubscribed(targetType string, uid int64, imei string, ids []string, result *[]map[string]interface{}) {
	subscribed := u.isSubscribed(targetType, uid, imei, ids)
	for _, id := range(ids) {
		if s, exists := subscribed[id]; exists {
			*result = append(*result, map[string]interface{}{
				"id": id,
				"subscribed": s,
				"type": targetType,
			})
		}
	}
}

// CheckSubscribed ...
func (u *User) CheckSubscribed(args *CheckSubscribedArgs, reply *CheckSubscribedResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	if args.Uid == 0 && args.Imei == "" {
		u.Info("invalid uid or imei")
		reply.Result = Result{Status: 1, Error: "invalid uid or imei"}
		return nil
	}

	var tagIDs, sourceIDs []string
	for _, id := range(args.Targetid) {
		if utils.CheckOid(id) {
			tagIDs = append(tagIDs, id)
		} else {
			sourceIDs = append(sourceIDs, id)
		}
	}
	u.Info("uid=%v tags=%v ids=%v", args.Uid, tagIDs, args.Targetid)
	u.checkSubscribed("tag", args.Uid, args.Imei, tagIDs, &(reply.Data))
	u.checkSubscribed("source", args.Uid, args.Imei, sourceIDs, &(reply.Data))

	return nil
}

// AddSubscribe ...
func (u *User) AddSubscribe(args *UserSubscribeArgs, reply *UserSubscribeResult) error {
	reply.Result = Result{Status: 1, Error: "failed"}
	w, err := u.Ctx.Ssdb("subscribe_ssdb_w", 0)
	if err != nil {
		u.Error("get subscribe_ssdb_w failed:%v", err)
		return err
	}
	r, err := u.Ctx.Ssdb("subscribe_ssdb_r", 0)
	if err != nil {
		u.Error("get subscribe_ssdb_r failed:%v", err)
		return err
	}
	
	redisKey := u.getUserSubscribeKey(args.Uid, args.Imei)
	key := redisKey+ ":" + args.Targettype
	existTargets, err := r.ZMultiGet(key, args.Targetid)
	if err != nil {
		u.Info("load subscribe info for type %v failed:%v", args.Targettype, err)
		return nil
	}

	disabledTargetIDs := make(map[string]int)
	if args.Targettype == "tag" {
		disabledTargetIDs = u.getDisabledTags()
	}

	targetIds := []string{}
	for _, id := range(args.Targetid) {
		if _, exists := disabledTargetIDs[id]; exists {
			u.Debug("tag disabled id=%v\n", id)
			continue
		}
		if _, exists := existTargets[id]; !exists {
			targetIds = append(targetIds, id)
		}
	}
	u.Debug("existTargets=%v targetIds=%v\n", existTargets, targetIds)
	
	if len(targetIds) == 0 {
		u.Info("add subscribe for type %v failed, empty targetIds")
		return nil
	}
	
	err = w.ZMultiSet(key, time.Now().UnixNano(), targetIds)
	if err != nil {
		u.Info("add subscribe for type %v failed:%v", args.Targettype, err)
		return nil
	}
	(&Tag{u.Type}).updateSubscribeCount(targetIds, 1)
	//_, err = w.Do("expire", redisKey+":"+args.Targettype, 3600*24*7)
	
	reply.Result = Result{Status: 0, Error: "success"}
	return nil
}

// DelSubscribe ...
func (u *User) DelSubscribe(args *UserSubscribeArgs, reply *UserSubscribeResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	w, err := u.Ctx.Ssdb("subscribe_ssdb_w", 0)
	if err != nil {
		u.Error("get user profile redis failed:%v", err)
		return err
	}
	redisKey := u.getUserSubscribeKey(args.Uid, args.Imei)
	for _, targetID := range(args.Targetid) {
		_, err = w.Do("zdel", redisKey+":"+args.Targettype, targetID)
		if err != nil {
			u.Info("del subscribe failed:%v", err)
			continue
		}
	}
	(&Tag{u.Type}).updateSubscribeCount(args.Targetid, -1)

	return nil
}

// ListSubscribe ...
func (u *User) ListSubscribe(args *ListSubscribeArgs, reply *ListSubscribeResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	if args.Uid == 0 {  // 未登录直接返回空
		return nil
	}
	count := args.Count
	if count == 0 {
		count = defaultListCount
	}
	startTime := ""
	if args.Lasttime != 0 {
		startTime = fmt.Sprintf("%d", args.Lasttime * 1e9)   // score存的纳秒
	}
	targetTypes := []string{"tag", "source"}
	if args.Targettype != "" {
		targetTypes = []string{args.Targettype}
	}
	w, err := u.Ctx.Ssdb("subscribe_ssdb_r", 0)
	if err != nil {
		u.Error("get feed ssdb failed:%v", err)
		reply.Result = Result{Status: 1, Error: "system error"}
		return err
	}
	sortedData := list.New()
	for _, t := range(targetTypes) {
		key := fmt.Sprintf("SUB:%d:%s", args.Uid, t)
		ret, scores, err := w.Zrscan(key, startTime, "", count)
		if err != nil {
			u.Error("scan subscribe for type %v failed:%v", t, err)
			continue
		}
		targetList := []string{}
		for _, s := range(ret) {
			targetList = append(targetList, s)
		}
		targetNames := make(map[string]string)
		if t == "tag" {
			targetInfo, err := (&Tag{u.Type}).get(targetList, 0, "")
			if err == nil {
				for _, target := range(targetList) {
					tagInfo, ok := targetInfo[target].(*TagInfo)
					if !ok || tagInfo == nil {
						u.Info("get tag name failed:%v", targetInfo[target])
						continue
					}
					targetNames[target] = tagInfo.Name
				}
			}
		} else if t == "source" {
			for _, target := range(targetList) {
				uid, _ := strconv.ParseInt(target, 10, 0)
				if uid == 0 {
					continue
				}
				name := u.getName(uid)
				if name != "" {
					targetNames[target] = name
				}
			}
		}
		for i, s := range(ret) {
			if _, ok := targetNames[s]; !ok {
				continue
			}
			targetTime := scores[i]/1e9
			newItem := &SubscribeListItemType {
				Id:s,
				Name:targetNames[s],
				Type:t,
				Time:scores[i]/1e9,
			}
			e := sortedData.Front()
			for e != nil{
				item := e.Value.(*SubscribeListItemType)
				if item.Time < targetTime {
					sortedData.InsertBefore(newItem, e)
					break
				}
				e = e.Next()
			}
			if nil == e {
				sortedData.PushBack(newItem)
			}
		}

	}
	e := sortedData.Front()
	for e != nil && len(reply.Data) < count {
		item := e.Value.(*SubscribeListItemType)
		reply.Data = append(reply.Data, *item)
		e = e.Next()
	}

	return nil
}

func (u *User) getSubscribeCount(uid int64, imei string, targetTypes []string) (count int) {
	if uid == 0 {  // 未登录直接返回0
		return 0
	}
	
	w, err := u.Ctx.Ssdb("subscribe_ssdb_r", 0)
	if err != nil {
		u.Error("get feed ssdb failed:%v", err)
		return
	}
	for _, t := range(targetTypes) {
		key := fmt.Sprintf("SUB:%d:%s", uid, t)
		c, err := w.Zcount(key)
		if err != nil {
			u.Error("get count for type %v failed:%v", t, err)
			continue
		}
		count += c
	}

	return
}