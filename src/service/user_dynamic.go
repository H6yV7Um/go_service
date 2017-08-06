package service

import (
	"fmt"
)

const (
	defaultDynamicCount = 20
	userDynamicExpireTime = 3600
	userDynamicArticleExpireTime = 3600*24*3
	dynamicMcKeyPrefix = "DYNAMIC"
	dynamicArticleMcKeyPrefix = "DARTICLE"
)

// UserDynamicResult ...
type UserDynamicResult struct {
	Result Result
	Data   interface{}
}

// UserDynamicArgs ...
type UserDynamicArgs struct {
	Common		CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid 		int64      	`msgpack:"uid" must:"1" comment:"uid"`
	Sinceid     string     	`msgpack:"start" must:"0" comment:"翻页时起始游标位置"`
	Count       int      	`msgpack:"count" must:"0" comment:"返回结果条数,最大50，默认20,大于50按20记"`
}

// UserDynamicActionListArgs ...
type UserDynamicActionListArgs struct {
	Common		CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid 		int64      	`msgpack:"uid" must:"1" comment:"uid"`
	Oid     	[]string    `msgpack:"oid" must:"1" comment:"oid"`
}

// UserDynamicActionListResult ...
type UserDynamicActionListResult struct {
	Result Result
	Data   []interface{}
}

func getDynamicMcKey(uid int64, sinceid string, count int) string {
	return fmt.Sprintf("%s:%d:%s:%d", dynamicMcKeyPrefix, uid, sinceid, count)
}

// GetDynamic ...
func (u *User) GetDynamic(args *UserDynamicArgs, reply *UserDynamicResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	if args.Uid <= 0 {
		reply.Result = Result{Status: 1, Error: "invalid uid"}
		return nil
	}
	count := args.Count
	if count == 0 {
		count = defaultDynamicCount
	}

	//mcClient, err := u.Ctx.MC("misc_mc")
	mcClient, err := u.Ctx.Memcache("misc_mc")
	if err != nil {
		u.Error("MC error", err)
		reply.Result = Result{Status: 1, Error: "failed to get user dynamic"}
		return err
	}
	mcKey := getDynamicMcKey(args.Uid, args.Sinceid, count)
	/*mcValue := mcClient.Get(mcKey)
	if mcValue != "" {
		u.Debug("got dynamic from mc:%d", args.Uid)
		json.Unmarshal([]byte(mcValue.(string)), &((*reply).Data))
		return nil
	}*/
	mcValue, err := mcClient.GetJson(mcKey)
	u.Debug("got dynamic from key=%s err=%v mcValue=%v", mcKey, err, mcValue)
	if err == nil && mcValue != nil {
		//json.Unmarshal([]byte(mcValue.(string)), &((*reply).Data))
		(*reply).Data = mcValue
		return nil
	}

	params := map[string]interface{}{
		"uid": args.Uid,
		"count": count,
		"is_filter_deleted": true,
		"select": "time",
	}
	if args.Sinceid != "" {
		params["since_id"] = args.Sinceid
	}
	result, err := u.Ctx.WeiboApiJson(
		"i2.api.weibo.com/2/darwin/platform/content/weibo_top_dynamic.json", params)
	if err != nil {
		u.Info("call weibo_top_dynamic faild, err=%s, result=%v", err.Error(), result)
		reply.Result = Result{Status: 1, Error: "failed to get user dynamic"}
		return err
	}
	results, exists := result["results"]
	if !exists {
		reply.Result = Result{Status: 1, Error: "failed to get user dynamic"}
		return nil

	}
	(*reply).Data = results
	articleList, ok := results.([]interface{})
	if !ok {
		u.Error("parse results failed")
		return nil
	}
	for _, article := range(articleList) {
		articleMap, ok := article.(map[string]interface{})
		if !ok || articleMap["action_list"] == nil || articleMap["article_id"] == nil {
			u.Error("parse article failed")
			continue
		}
		articleMcKey := fmt.Sprintf("%s%d%s", dynamicArticleMcKeyPrefix, args.Uid, articleMap["article_id"])
		/*s, err := json.Marshal(articleMap)
		if err != nil {
			u.Error("json marshal failed err=%v", err)
			continue
		}*/
		//err = mcClient.Set(articleMcKey, string(s), userDynamicArticleExpireTime)
		u.Debug("set mc article key=%v", articleMcKey)
		err = mcClient.SetJson(articleMcKey, articleMap, userDynamicArticleExpireTime)
		if err != nil {
			u.Error("MC set failed key=%s err=%v", articleMcKey, err)
		}
	}

	/*s, err := json.Marshal(results)
	if err != nil {
		u.Error("json marshal failed err=%v", err)
		return nil
	}
	err = mcClient.Set(mcKey, string(s), userDynamicExpireTime)*/
	err = mcClient.SetJson(mcKey, results, userDynamicExpireTime)
	if err != nil {
		u.Error("MC set failed key=%s err=%v", mcKey, err)
	} else {
		u.Debug("set mc key=%v", mcKey)
	}

	return nil
}

// GetDynamicActionList ...
func (u *User) GetDynamicActionList(args *UserDynamicActionListArgs, reply *UserDynamicActionListResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	if args.Uid <= 0 || len(args.Oid) == 0 {
		reply.Result = Result{Status: 1, Error: "invalid uid or oid"}
		return nil
	}

	//mcClient, err := u.Ctx.MC("misc_mc")
	mcClient, err := u.Ctx.Memcache("misc_mc")
	if err != nil {
		u.Error("MC error", err)
		reply.Result = Result{Status: 1, Error: "failed to get user dynamic"}
		return err
	}

	for _, oid := range(args.Oid) {
		if oid == "" {
			u.Info("dynamic action list:invalid oid")
			continue
		}
		// TODO: 批量查询mc
		articleMcKey := fmt.Sprintf("%s%d%s", dynamicArticleMcKeyPrefix, args.Uid, oid)
		/*mcValue := mcClient.Get(articleMcKey)
		if mcValue == "" {
			u.Debug("dynamic action list not found:uid=%d oid=%s", args.Uid, oid)
			continue
		}
		
		var action map[string]interface{}
		err := json.Unmarshal([]byte(mcValue.(string)), &action)*/
		action, err := mcClient.GetJson(articleMcKey)
		u.Debug("got dynamic action list from mc:uid=%d oid=%s err=%v", args.Uid, oid, err)
		if err != nil {
			u.Error("Get dynamic action list failed:%v", err)
		} else {
			(*reply).Data = append((*reply).Data, action)
		}
	}

	return nil
}
