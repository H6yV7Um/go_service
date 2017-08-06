package service

import (
	"utils"
	"time"
	"fmt"
	"strings"
	"regexp"
	"strconv"
)

const (
	defaultCommentCount = 5
	commentExpireTime = 3 * time.Minute
	maxCommentNumber = 1000000
	commentCountsPrefix = "article_counts_"
	sourceOfComment = "c"
)

// Comment ...
type Comment struct {
	Type
}

// CommentArgs ...
type CommentArgs struct {
	Common	CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid 	int64  		`msgpack:"uid" must:"0" comment:"uid"`
	Oid   	string 		`msgpack:"oid" must:"1" comment:"oid"`
	Type  	string 		`msgpack:"type" must:"0" comment:"评论类型"`
	Max   	int    		`msgpack:"max" must:"0" comment:"上次最大评论id"`
	Count 	int    		`msgpack:"count" must:"0" comment:"返回结果数"`
	Source  string      	`msgpack:"source" must:"1" comment:"来源页"`
}

// CommentResult ...
type CommentResult struct {
	Result Result
	Data   struct {
		Comments []*CommentType
	}
}

// CommentCountArgs ...
type CommentCountArgs struct {
	Common		CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Oid   		[]string 	`msgpack:"oid" must:"1" comment:"oid"`
}

// CommentCountResult ...
type CommentCountResult struct {
	Result Result
	Data   map[string]int64
}

// CommentAddArgs ...
type CommentAddArgs struct {
	Common	 	CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid 		int64  		`msgpack:"uid" must:"1" comment:"uid"`
	Oid   		string 		`msgpack:"oid" must:"1" comment:"oid"`
	Type  		string 		`msgpack:"type" must:"0" comment:"评论类型"`
	Token 		string 		`msgpack:"token" must:"1" comment:"token"`
	Repost  	int    		`msgpack:"repost" must:"0" comment:"是否转发"`
	Text 		string 		`msgpack:"text" must:"1" comment:"评论内容"`
	Auth        string 		`msgpack:"auth" must:"1" comment:"认证方式(tauth,cookie)"`
}

// CommentAddResult ...
type CommentAddResult struct {
	Result Result
	Data   map[string]interface{}
}

// CommentDelArgs ...
type CommentDelArgs struct {
	Common	 	CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Uid 		int64  		`msgpack:"uid" must:"1" comment:"uid"`
	Wid   		string 		`msgpack:"wid" must:"1" comment:"wid"`
	Token 		string 		`msgpack:"token" must:"1" comment:"token"`
	Auth        string 		`msgpack:"auth" must:"1" comment:"认证方式(tauth,cookie)"`
}

// UserType ...
type UserType struct {
	VerifiedType 	uint64
	Verified 		bool
	Uid 			int64
	ProfileImageUrl string
	ScreenName		string
}

// CommentCountType ...
type CommentCountType struct {
	RepostsCount	uint64
	AttitudesCount	uint64
	CommentsCount	uint64
}

// CommentType ...
type CommentType struct {
	Count 			CommentCountType
	Text 			string
	CreatedAt 		uint64
	Mid 			string
	ObjectId 		string
	User 			UserType
	Type 			string
	Id uint64
	Wid 			uint64
	LikeCount 		uint64
	Liked 			uint64
}

// GetCommentCounterArgs 获取计数值
type GetCommentLikeNumArgs struct {
	Common CommonArgs   `msppack:"common" must:"0" comment:"公共参数" size:"40"`
	Cid    string       `msgpack:"cid" must:"1" comment:"评论ID"`
}

// CommentCounterResult 计数值
type CommentLikeNumResult struct {
	Result Result
	Data   int
}

var reShortURL *regexp.Regexp

func init() {
	reShortURL = regexp.MustCompile("(.*)(http://t\\.cn/\\w+)(.*)")
}

// CheckLike ...
func (c *Comment) CheckLike(args *CheckLikeArgs, reply *CheckLikeResult) error {
	likeService := Like{c.Type}
	return likeService.CheckLikeBatch(args, reply, likeTypeComment)
}

// Like ...
func (c *Comment) Like(args *LikeArgs, reply *CommonResult) error {
	likeService := Like{c.Type}
	err := likeService.Like(args, reply, likeTypeComment)
	if err == nil {
		likeService.handleLikeCount(args.Oid, args.Liked, likeTypeComment)
	}
	
	return err
}

// GetCounterNum 获取评论某业务的计数值
func (c *Comment) GetLikeNum(args *GetCommentLikeNumArgs, reply *CommentLikeNumResult) error {
	likeService := Like{c.Type}
	reply.Data   = likeService.GetLikeNum(args.Cid, likeTypeComment)
	if reply.Data == -1 {
		reply.Result = Result{Status:0, Error:"operator error"}
		return nil
	}
	reply.Result = Result{Status:0, Error:"success"}
	return nil
}

// Get ...
func (c *Comment) Get(args *CommentArgs, reply *CommentResult) error {
	// TODO: 对于没有短链的微博文章，是否需要取mid对应的评论？如果是，所有接口都需要支持。
	if !utils.CheckOid(args.Oid) {
		reply.Result = Result{Status: 1, Error: "invalid argument"}
		return nil
	}
	reply.Result = Result{Status: 0, Error: "success"}
	count := args.Count
	if count == 0 {
		count = defaultCommentCount
	}

	reply.Data.Comments = []*CommentType{}
	var err error
	eoid := (&Article{c.Type}).getEffectiveOid(args.Oid)
	if args.Type == "hot" {
		err = c.getHotComment(args.Uid, eoid, args.Oid, args.Max, count, args.Source, &(reply.Data.Comments));
	} else {
		err = c.getComment(args.Uid, eoid, args.Oid, args.Max, count, &(reply.Data.Comments));
	}
	if err != nil {
		c.Info("get comment failed:%v", err)
		reply.Result = Result{Status: 1, Error: "get comment failed"}
	} else {
		if reply.Data.Comments != nil {
			for i, comment := range(reply.Data.Comments) {
				liked, _ := (&Like{c.Type}).checkLike(uint64(args.Uid), fmt.Sprintf("%d", comment.Wid), likeTypeComment)
				reply.Data.Comments[i].Liked = uint64(liked)
			}
			c.Info("%s get %d comments", args.Oid, len(reply.Data.Comments))
		}
	}

	return nil
}

func (c *Comment) filterComment(text *string) bool {
	filterText := []string{"工资日结", "加扣", "扣扣", "代开发票", "招网络兼职", "详情咨询",
		"贝兼", "扌召", "耳只", "力口",	"淘口令", "手机淘宝", "傔职", "涨渟", "亻訁", "亻言", "转发微博"}
	for _, t := range(filterText) {
		if strings.Contains(*text, t) {
			return true
		}
	}

	submatch := reShortURL.FindSubmatch([]byte(*text))
	if len(submatch) == 4 {
		sBefore := strings.Trim(string(submatch[1]), " ")
		sAfter := strings.Trim(string(submatch[3]), " ")
		if sAfter == "" {
			if sBefore == "" || sBefore == "分享图片" || sBefore == "转发" ||
				(strings.HasPrefix(sBefore, "【") && strings.HasSuffix(sBefore, "】")) {
				c.Info("comment filtered:%s", *text)
				return true
			}
		}
		*text = sBefore + " " + sAfter
	}
	return false
}

func (c *Comment) formatComment(uid int64, oid string, hot bool, item map[string]interface{}) *CommentType {
	text := utils.GetStringValueByPath(item, "string:text")
	if c.filterComment(&text) {
		return nil
	}
	commentType := utils.GetStringValueByPath(item, "string:type")
	createdAt := utils.GetStringValueByPath(item, "string:created_at")
	likeCount := uint64(0)
	repostsCount := uint64(0)
	commentsCount := uint64(0)
	attitudesCount := uint64(0)
	if hot {
		likeCount = utils.GetIntValueByPath(item, "float64:like_count")
		// TODO:这三个计数貌似这里都是0
		repostsCount = utils.GetIntValueByPath(item, "float64:status/reposts_count")
		commentsCount = utils.GetIntValueByPath(item, "float64:status/comments_count")
		attitudesCount = utils.GetIntValueByPath(item, "float64:status/attitudes_count")
	} else if commentType == "repost" {
		repostsCount = utils.GetIntValueByPath(item, "float64:reposts_count")
		commentsCount = utils.GetIntValueByPath(item, "float64:comments_count")
		attitudesCount = utils.GetIntValueByPath(item, "float64:attitudes_count")
	}
	if attitudesCount < likeCount {
		attitudesCount = likeCount
	}
	tm, err := time.Parse("Mon Jan 02 15:04:05 -0700 2006", createdAt)
	if err != nil {
		c.Error("parse comment time failed:%s", err.Error())
		tm = time.Now()
	}

	comment := CommentType{
		Count:CommentCountType{RepostsCount:repostsCount, CommentsCount:commentsCount, AttitudesCount:attitudesCount},
		User:UserType{
			Verified:utils.GetBoolValueByPath(item, "bool:user/verified"),
			VerifiedType:utils.GetIntValueByPath(item, "float64:user/verified_type"),
			ProfileImageUrl:utils.GetStringValueByPath(item, "string:user/profile_image_url"),
			ScreenName:utils.GetStringValueByPath(item, "string:user/name"),
			Uid:int64(utils.GetIntValueByPath(item, "float64:user/id")),
		},
		Text:text,
		CreatedAt:uint64(tm.Unix()),
		Mid:utils.GetStringValueByPath(item, "string:mid"),
		ObjectId:oid,
		Type:commentType,
		Id:utils.GetIntValueByPath(item, "float64:cid"),
		Wid:utils.GetIntValueByPath(item, "float64:id"),
		LikeCount:likeCount,
		Liked:0,
	}
	return &comment
}

func (c *Comment) getComment(uid int64, eoid string, oid string, max int, count int, result *[]*CommentType) error {
	if c.Ctx.Cfg.Switch.DisableCommentsObjectList {  // 降级，不调用平台接口
		return nil
	}
	page := 1
	maxPage := 20
	if max == 0 {
		maxPage = 3
	}
	for {
		ret, errAPI := c.Ctx.WeiboApiJson("i2.api.weibo.com/2/comments/object/list.json",
			map[string]interface{}{"object_id": eoid, "page":page, "count":20, "tauth":true, "retry":true})
		if errAPI != nil {
			c.Info("getComment failed:%v", errAPI)
			if strings.Contains(errAPI.Error(), "timeout") {
				page++
				continue
			}
			return errAPI
		}
		comments := ret["comments"]   // TODO: check
		if comments == nil {
			c.Error("comments=nil oid=%v", eoid)
			return nil
		}
		for _, comment := range(comments.([]interface{})) {
			src := comment.(map[string]interface{})
			if src == nil {
				c.Info("src=nil oid=%v", eoid)
				continue
			}
			var item map[string]interface{}
			if _, exists := src["comment"]; exists {
				item = src["comment"].(map[string]interface{})
				item["type"] = "comment"
			} else if _, exists := src["status"]; exists {
				item = src["status"].(map[string]interface{})
				item["type"] = "repost"
			} else {
				c.Error("invalid type oid=%v", eoid)
				continue
			}

			id := utils.GetIntValueByPath(item, "float64:id")
			if max > 0 && id >= uint64(max) {
				continue
			}
			item["cid"] = id
			comment := c.formatComment(uid, oid, false, item)
			if comment != nil {
				*result = append(*result, comment)
			}
			if len(*result) >= count {
				break
			}
		}
		if len(*result) >= count || page > maxPage {   // FIXME: 限制接口调用次数
			break
		}
		nextCursor, ok := ret["next_cursor"].(float64)
		if ok && nextCursor == 0 {
			break
		}
		page++
	}
	return nil
}

func (c *Comment) getHotComment(uid int64, eoid string, oid string, max int, count int, source string, result *[]*CommentType) error {
	cacheKey := fmt.Sprintf("HOT_COMMENT:%s_%d_%d_%s", oid, max, count, source)
	cacheResult, found := c.Ctx.Cache.Get(cacheKey)
	if !found {
		if c.Ctx.Cfg.Switch.DisableObjectsHotTimeline {  // 降级，不调用平台接口
			return nil
		}
		page := 1
		total := 0
		start := 0
		if max > 0 {
			total  = max / maxCommentNumber
			maxid := max % maxCommentNumber
			start  = total - maxid + 1
			page   = start / count + 1
		}
		
		newCount := count
		if source == sourceOfComment && page == 1 && newCount < 10 { // 平台第1页,个数小于10,返回随机结果;当评论页时,需要有序,取10条记录以保持有序
			newCount = 10
		}
		ret, errAPI := c.Ctx.WeiboApiJson("i.api.weibo.com/2/objects/hot_timeline.json",
			map[string]interface{}{"object_id": eoid, "page":page, "count":newCount}) // 通过平台接口调用,没有保证有序??
		if errAPI != nil {
			c.Info("hot_timeline failed:%v", errAPI)
			return errAPI
		}
		comments := ret["comments"]   // TODO: check
		if comments == nil {
			return nil
		}
		totalNumber := utils.GetIntValueByPath(ret, "float64:total_number")
		if page == 1 {  // 第一页使用实时的total_number，后续沿用第一次的值
			total = int(totalNumber)
		}
		for i, comment := range(comments.([]interface{})) {
			item := comment.(map[string]interface{})
			if item == nil {
				continue
			}
			cid := int(totalNumber) - i - start
			if cid < 0 {
				cid = 0
			}
			item["cid"]  = total * maxCommentNumber + cid
			item["type"] = "comment"
			comment     := c.formatComment(uid, oid, true, item)
			if comment != nil {
				*result = append(*result, comment)
			}
			
			if len(*result) >= count {
				break
			}
		}
		c.Ctx.Cache.Set(cacheKey, *result, commentExpireTime)
	} else {
		*result = cacheResult.([]*CommentType)
		return nil
	}
	return nil
}

func (c *Comment) getCountByAPI(oids []string) (counts map[string]int64, err error) {
	counts = make(map[string]int64)
	if len(oids) == 0 {
		return
	}
	params := map[string]interface{}{
		"object_ids": strings.Join(oids, ",")}
	ret, errAPI := c.Ctx.WeiboApiJson("i2.api.weibo.com/2/comments/object/counts.json", params)
	if errAPI != nil || ret == nil || ret["error_code"] != nil {
		c.Info("getCount failed:%v ret=%v", errAPI, ret)
		err = errAPI
		return
	}
	for k, v := range(ret) {
		counts[k] = int64(v.(float64))
	}
	return
}

func (c *Comment) getCount(oids []string) (counts map[string]int64, err error) {
	counts = make(map[string]int64)
	if len(oids) == 0 {
		return
	}
	keys := []string{}
	oidsForKeys := []string{}
	for _, oid := range(oids) {
		keys = append(keys, commentCountsPrefix + oid)
		oidsForKeys = append(oidsForKeys, oid)
	}
	r, err := c.Ctx.Redis("comment_r")
	if err != nil {
		c.Info("get redis failed:%v", err)
		return
	}
	r.Do("SELECT", 11)
	ret := r.GetMulti(keys)
	for k, v := range(ret) {
		oid := oidsForKeys[k]
		if v == nil {
			counts[oid] = 0
		} else {
			v2, ok := v.([]byte)
			if !ok {
				c.Error("get comment count failed:%v", v)
				counts[oid] = 0
			} else {
				counts[oid], _ = strconv.ParseInt(string(v2), 10, 0)
			}
		}
	}
	return
}

// GetCount ...
func (c *Comment) GetCount(args *CommentCountArgs, reply *CommentCountResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	if len(args.Oid) == 0 {
		reply.Result = Result{Status: 0, Error: "success"}
		return nil
	}
	result := make(map[string]int64)
	oids := []string{}
	article := &Article{c.Type}
	for _, oid := range(args.Oid) {
		eoid := article.getEffectiveOid(oid)
		oids = append(oids, eoid)
	}

	counts, err := c.getCount(oids)
	if err != nil {
		reply.Result = Result{Status: 1, Error: err.Error()}
		return nil
	}
	for k, v := range(counts) {
		result[article.getOriginalOid(k)] = v
	}
	reply.Data = result

	return nil
}

// Add ...
func (c *Comment) Add(args *CommentAddArgs, reply *CommentAddResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	if len(args.Oid) == 0 {
		reply.Result = Result{Status: 0, Error: "success"}
		return nil
	}
	oid := (&Article{c.Type}).getEffectiveOid(args.Oid)
	params := map[string]interface{}{
		"method": "POST",
		"object_id": oid,
		"access_token":args.Token,
		"comment":args.Text}
	if args.Auth == "tauth" {
		params["tauth"] = true
	} else if args.Auth == "cookie" {
		params["cookie"] = args.Token
		delete(params, "access_token")
	} else {
		reply.Result = Result{Status: 1, Error: "invalid auth type"}
		return nil
	}
	ret, errAPI := c.Ctx.WeiboApiJson("i2.api.weibo.com/2/comments/create.json", params)
	if errAPI != nil || ret == nil || ret["error_code"] != nil {
		c.Error("create comment failed:%v ret=%v uid=%d oid=%s", errAPI, ret, args.Uid, args.Oid)
		reply.Result = Result{Status: 1, Error: utils.MapGetKey(ret, "error", "create comment failed").(string)}
		return errAPI
	}
	(*reply).Data = ret

	if args.Repost == 1 {
		params = map[string]interface{}{
			"method": "POST",
			"status": args.Text,
			"access_token":args.Token}
		if args.Auth == "tauth" {
			params["tauth"] = true
		} else if args.Auth == "cookie" {
			params["cookie"] = args.Token
			delete(params, "access_token")
		}
		ret, errAPI := c.Ctx.WeiboApiJson("i2.api.weibo.com/2/statuses/update.json", params)
		if errAPI != nil || ret == nil || ret["error_code"] != nil {
			c.Error("create repost failed:%v ret=%v", errAPI, ret)
			reply.Result = Result{Status: 1, Error: utils.MapGetKey(ret, "error", "create repost failed").(string)}
			return errAPI
		}
	}

	return nil
}

// Del ...
func (c *Comment) Del(args *CommentDelArgs, reply *CommonResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	if len(args.Wid) == 0 {
		reply.Result = Result{Status: 0, Error: "success"}
		return nil
	}
	params := map[string]interface{}{
		"method": "POST",
		"cid": args.Wid,
		"access_token":args.Token}
	if args.Auth == "tauth" {
		params["tauth"] = true
	} else if args.Auth == "cookie" {
		params["cookie"] = args.Token
		delete(params, "access_token")
	} else {
		reply.Result = Result{Status: 1, Error: "invalid auth type"}
		return nil
	}
	ret, errApi := c.Ctx.WeiboApiJson("i2.api.weibo.com/2/comments/destroy.json", params)
	if errApi != nil || ret == nil || ret["error_code"] != nil {
		c.Error("destroy comment failed:%v ret=%v", errApi, ret)
		reply.Result = Result{Status: 1, Error: utils.MapGetKey(ret, "error", "destroy comment failed").(string)}
		return errApi
	}
	c.Debug("destroy comment success:%v", ret)

	return nil
}
