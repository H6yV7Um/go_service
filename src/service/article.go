package service

import (
	"contexts"
	"fmt"
	"strings"
	"time"
	"errors"
	"utils"
	"models"
	"strconv"
	"encoding/json"
	"libs/very_simple_json"
	"reflect"
)

const (
	articleMcPrefix        = "a_go_"
	articleMcBatchPrefix   = "b_go_"
	articleExpireTime      = 2 * 24 * 3600
	effectiveOidExpireTime = 3 * 24 * time.Hour
	oidByIDExpireTime      = 5 * 24 * time.Hour
	articleCountsPrefix    = "ac_"
	articleCountsExpireTime = 1 * time.Minute
	famousActiveKeyPrefix  = "FAMOUS_ACTIVE:"
	activityTypeStatus     = 1
	activityTypeLike       = 3
)

var tagsJsonLayout = map[string]interface{} {
	"category_id": reflect.String,
	"category_name": reflect.String,
	"display_name": reflect.String,
	"object_id": reflect.String,
	"object_type": reflect.String,
	"weight": reflect.Float64,
}

// Article 文章服务类
type Article struct {
	Type
}

// ArticlesType 文章列表类型
type ArticlesType map[string]map[string]interface{}

// ArticleType 文章类型
type ArticleType map[string]interface{}

// ArticleClearCacheArgs 清除文章缓存接口参数
type ArticleClearCacheArgs struct {
	Common        CommonArgs     `msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Oid           []string       `msgpack:"oid" must:"1" comment:"oid" size:"40"`
}

// ArticleFetchArgs 获取文章详情接口参数
type ArticleFetchArgs struct {
	Common        CommonArgs    `msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Oid           string        `msgpack:"oid" must:"1" comment:"oid" size:"40"`
	Time          int           `msgpack:"time" must:"0" comment:"文章的发布时间"`
	Needcounts    int           `msgpack:"need_counts" must:"0" comment:"是否返回计数"`
}

// ArticleFetchResult 获取文章详情接口返回类型
type ArticleFetchResult struct {
	Result Result
	Data   map[string]interface{}
}

// ArticleMultiFetchArgs 批量获取文章详情接口参数
type ArticleMultiFetchArgs struct {
	Common        CommonArgs    `msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Oid           []string      `msgpack:"oid" must:"1" comment:"oid" size:"40"`
	Time          int           `msgpack:"time" must:"0" comment:"文章的发布时间"`
	Needcounts    int           `msgpack:"need_counts" must:"0" comment:"是否返回计数"`
	Uid   		  uint64 		`msgpack:"uid" must:"1" comment:"uid"`
	Needliked     int           `msgpack:"need_liked" must:"0" comment:"是否赞状态"`
	Fields        []string      `msgpack:"fields" must:"0" comment:"需要的字段（可选:counts,liked,famous）"`
}

// ArticleMultiFetchResult 批量获取文章详情接口返回类型
type ArticleMultiFetchResult struct {
	Result Result
	Data   map[string]map[string]interface{}
}

func (a *Article) getOidByID(id string) string {
	cacheKey := "OID_BY_ID:" + id
	cacheResult, found := (*a.Ctx).Cache.Get(cacheKey)
	if !found {
		ts, err := utils.GetTimeFromIdString(id)
		if ts == 0 {
			return ""
		}
		sql := fmt.Sprintf("select oid from articles_%s where `id`='%s'", time.Unix(int64(ts), 0).Format("200601"), id)
		rows, err := a.Ctx.GetDBInstance("articles", "article_r").Query(sql)
		if rows == nil || err != nil {
			a.Error("query failed:%v", err)
			return ""
		}
		if len(rows) == 0 {
			a.Error("getOidByID not found in db:%s", id)
			return ""
		}
		article, ok := rows[0].(*models.Articles)
		if ok {
			a.Ctx.Cache.Set(cacheKey, article.Oid, oidByIDExpireTime)
			return article.Oid
		}
		a.Info("getOidByID failed:%s", id)
		return ""
	}
	return cacheResult.(string)
}

func (a *Article) combineArticles (result *ArticlesType, cached *ArticlesType) {
	if result == nil {
		return
	}
	if cached != nil {
		for k, v := range(*cached) {
			(*result)[k] = v
		}
	}
}

type articleCountsType struct {
	Comments 	int
	Playnum 	int
	Readnum		int
	Likenum		int
}

func getCount(v map[string]string, f string) int {
	value, ok := v[f]
	if !ok {
		return 0
	}
	n, _ := strconv.Atoi(value)
	if n < 0 {
		n = 0
	}
	
	return n
}

// 获取文章的评论数、阅读数、视频播放数等
func (a *Article) fetchCounts(oids []string, oidIDMap map[string]string, result *ArticlesType) error {
	counts := make(map[string]articleCountsType)
	if len(oids) == 0 {
		a.Info("oids empty")
		return nil
	}
	r, err := a.Ctx.Ssdb("subscribe_ssdb_r", 0)
	if err != nil {
		a.Info("get redis failed:%v", err)
		return nil
	}
	for _, oid := range(oids) {
		cache := a.Ctx.GetCache("article")
		item, ok := cache.HGet(oid, "counts")
		var cacheResult articleCountsType
		if ok {
			cacheResult, ok = item.(articleCountsType)
		}
		
		if !ok {
			v, err := r.HGetAll(articleCountsPrefix + oid)
			if v == nil || err != nil {
				a.Error("article get empty counts for %s err=%v", oid, err)
				cacheResult = articleCountsType{Comments:0, Playnum:0, Readnum:0, Likenum:0}
			} else {
				cacheResult = articleCountsType{
					Comments:getCount(v, "c"),
					Playnum:getCount(v, "p"),
					Readnum:getCount(v, "r"),
					Likenum:getCount(v, "l"),
				}
			}
			cache.HSet(oid, "counts", cacheResult, articleCountsExpireTime)
		}
		counts[oid] = cacheResult
	}
	for _, oid := range (oids) {
		id, exists := oidIDMap[oid]
		if !exists {
			a.Info("%s not in oidIdMap", oid)
			continue
		}
		_, exists = (*result)[id]
		if !exists {
			a.Info("%s not in result", id)
			continue
		}
		if v, exists := counts[oid]; exists {
			(*result)[id]["comment_count"] = v.Comments
			(*result)[id]["play_count"] = v.Playnum
			(*result)[id]["read_count"] = v.Readnum
			(*result)[id]["like_count"] = v.Likenum
		} else {
			(*result)[id]["comment_count"] = 0
			(*result)[id]["play_count"] = 0
			(*result)[id]["read_count"] = 0
			(*result)[id]["like_count"] = 0
		}
	}

	return nil
}

func (a *Article) fetchCommentCountBatch(oids []string, oidIDMap map[string]string, result *ArticlesType) error {
	commentService := Comment{a.Type}
	ret, err := commentService.getCount(oids)
	if err != nil {
		for _, oid := range (oids) {
			id, exists := oidIDMap[oid]
			if !exists {
				a.Info("%s not in oidIdMap", oid)
				continue
			}
			(*result)[id]["comment_count"] = 0
		}
		return err
	}
	for _, oid := range (oids) {
		id, exists := oidIDMap[oid]
		if !exists {
			a.Info("%s not in oidIdMap", oid)
			continue
		}
		_, exists = (*result)[id]
		if !exists {
			a.Info("%s not in result", id)
			continue
		}

		if v, exists := ret[oid]; exists {
			(*result)[id]["comment_count"] = v
		} else {
			(*result)[id]["comment_count"] = 0
		}
	}
	return nil
}

func (a *Article) fetchCommentCount(oid string, id string) int64 {
	commentService := Comment{a.Type}
	ret, _ := commentService.getCount([]string{oid})
	if v, exists := ret[oid]; exists {
		return v
	}
	return 0
}

// 获取文章的赞状态
func (a *Article) fetchLiked(uid uint64, oids []string, oidIDMap map[string]string, result *ArticlesType) error {
	likeService := &(Like{a.Type})
	likedResult:= likeService.checkLikeBatch(uid, oids, likeTypeArticle)
	for _, oid := range (oids) {
		id, exists := oidIDMap[oid]
		if !exists {
			a.Info("%s not in oidIdMap", oid)
			continue
		}
		_, exists = (*result)[id]
		if !exists {
			a.Info("%s not in result", id)
			continue
		}
		liked, _ := likedResult[oid]
		(*result)[id]["liked"] = liked
	}
	
	return nil
}

func (a *Article) getToutiaoArticle(oid string, uid uint64, ip string, ua string) error {
	// TODO: device固定传mobile
	ret, err := a.Ctx.WeiboApiJson("i.api.article.card.weibo.com/article/show",
		map[string]interface{}{
			"source":       contexts.APPKEY,
			"device":       "mobile",
			"uid":          uid,
			"skip_content": 1,
			"ip":           ip,
			"ua":           ua,
			"object_id":    oid,
			"set_read":     1})
	if err != nil {
		a.Info("get weibo info faild, ret=%v", ret)
		return errors.New("get weibo info faild")
	}
	return nil
}

// ClearCache 清除文章缓存
func (a *Article) ClearCache(args *ArticleClearCacheArgs, reply *CommonResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	err := a.Ctx.Article().(*Article).clearArticleInfoInCache(args.Oid)
	if err != nil {
		reply.Result = Result{Status: 1, Error: "failed:" + err.Error()}
	}
	return err
}

func (a *Article) getArticleInfoFromCache(oids []string, params ...uint64) (articles ArticlesType, missed []string) {
	var keys []string

	articles = make(map[string]map[string]interface{})
	for _, oid := range oids {
		keys = append(keys, articleMcPrefix + oid)
	}

	if len(keys) == 0 {
		a.Info("no keys")
		return
	}

	mc, err := a.Ctx.Memcache("article_mc")
	if err != nil {
		a.Error("MC error", err)
		missed = oids
		return
	}
	res, err := mc.GetJsonMulti(keys)
	if err != nil {
		a.Info("GetMultiMap failed err=%v", err)
		missed = oids
		return
	}
	for _, key := range keys {
		oid := key[len(articleMcPrefix):]
		resultValue, exists := res[key]
		if !exists {
			missed = append(missed, oid)
			a.Debug("key %v not found, oid=%s err=%v", key, oid, err)
		} else {
			articles[oid] = resultValue.(map[string]interface{})
		}
	}
	return
}

func (a *Article) setArticleInfoToCache(articles ArticlesType, isNewID bool) (retError error) {
	mcClient, err := a.Ctx.Memcache("article_mc")
	if err != nil {
		a.Error("MC error", err)
		retError = err
		return
	}
	var key string
	for _, article := range articles {
		if !utils.KeysExists(article, []string{"id", "oid"}) {
			continue
		}
		var id string
		if isNewID {
			id = article["id"].(string)
		} else {
			id = article["oid"].(string)
		}
		key = articleMcPrefix + id
		
		err = mcClient.SetJson(key, article, articleExpireTime)
		a.Debug("set mc key=%v err=%v expire=%ds", key, err, articleExpireTime)
		if err != nil {
			retError = err
		}
	}

	return
}

func (a *Article) setArticleInfoToCacheSingle(article map[string]interface {}, isNewID bool, isCardArticle bool) (retError error) {
	if !utils.KeysExists(article, []string{"id"}) {
		return nil
	}
	mcClient, err := a.Ctx.Memcache("article_mc")
	if err != nil {
		a.Error("MC error", err)
		retError = err
		return
	}
	var key string
	id := utils.GetStringValueByPath(article, "string:id")
	oid := utils.GetStringValueByPath(article, "string:oid")
	if isNewID {
		key = articleMcPrefix + id
	} else {
		key = articleMcPrefix + oid
	}
	if key == articleMcPrefix {
		return
	}

	if isCardArticle {
		err = mcClient.SetJson(key, article, articleExpireTime)
	} else {
		err = mcClient.SetArticle(key, article, articleExpireTime)
	}
	a.Debug("set mc key=%v err=%v expire=%d", key, err, articleExpireTime)
	if err != nil {
		retError = err
	}
	topGroupID := utils.GetStringValueByPath(article, "string:top_groupid")
	if oid != "" && topGroupID != "" {
		err = mcClient.SetString(topGroupidMcPrefix + oid, topGroupID, articleExpireTime)
	}

	return
}

func (a *Article) getArticleBatchInfoFromCache(oids []string, params ...uint64) (articles ArticlesType, missed []string) {
	var keys []string

	articles = make(map[string]map[string]interface{})
	for _, oid := range oids {
		keys = append(keys, articleMcBatchPrefix + oid)
	}

	if len(keys) == 0 {
		return
	}

	mcClient, err := a.Ctx.Memcache("article_mc")
	if err != nil {
		a.Error("MC error", err)
		missed = oids
		return
	}
	
	res, err := mcClient.GetJsonMulti(keys)
	if err != nil {
		a.Info("GetMultiMap failed err=%v", err)
		missed = oids
		return
	}
	for _, key := range keys {
		oid := key[len(articleMcBatchPrefix):]  // TODO
		resultValue, exists := res[key]
		if !exists {
			missed = append(missed, oid)
			a.Debug("key %v not found, oid=%s err=%v", key, oid, err)
		} else {
			articles[oid] = resultValue.(map[string]interface{})
		}
	}
	return
}

func (a *Article) setArticleBatchInfoToCache(articles ArticlesType, isNewID bool) (retError error) {
	mcClient, err := a.Ctx.Memcache("article_mc")
	if err != nil {
		a.Error("MC error", err)
		retError = err
		return
	}
	var key string
	for _, article := range articles {
		var id string
		isCardArticle := false
		if isNewID {
			id = article["id"].(string)
			isCardArticle = utils.IsCardArticle(id)
		} else {
			id = article["oid"].(string)
		}
		key = articleMcBatchPrefix + id

		// TODO: use set_multi
		if isCardArticle {
			err = mcClient.SetJson(key, article, articleExpireTime)
		} else {
			err = mcClient.SetArticle(key, article, articleExpireTime)
		}
		a.Debug("set mc key=%v err=%v", key, err)
		if err != nil {
			retError = err
		}
		err = mcClient.SetString(topGroupidMcPrefix + utils.GetStringValueByPath(article, "string:oid"),
			utils.GetStringValueByPath(article, "string:top_groupid"), articleExpireTime)
	}

	return
}

func (a *Article) clearArticleInfoInCache(oids []string) (retError error) {
	mcClient, err := a.Ctx.Memcache("article_mc")
	if err != nil {
		a.Error("MC error", err)
		retError = err
		return
	}
	for _, oid := range oids {
		key := articleMcPrefix + oid
		err = mcClient.Delete(key)
		if err != nil && err.Error() != "memcache: cache miss" {
			retError = err
		}
		key = articleMcBatchPrefix + oid
		err = mcClient.Delete(key)
		if err != nil && err.Error() != "memcache: cache miss" {
			retError = err
		}
	}

	return
}

// GetEffectiveOidArgs
type GetEffectiveOidArgs struct {
	CommonArgs
	Oid string `msgpack:"oid" must:"1" comment:"oid"`
}

// GetEffectiveOidResult
type GetEffectiveOidResult struct {
	Result Result
	Data   string
}

// GetEffectiveOid
func (a *Article) GetEffectiveOid(args *GetEffectiveOidArgs, reply *GetEffectiveOidResult) error {
	if !utils.CheckOid(args.Oid) {
		reply.Result = Result{Status: 1, Error: "invalid argument"}
		return nil
	}
	reply.Result = Result{Status: 0, Error: "success"}
	reply.Data = a.getEffectiveOid(args.Oid)

	return nil
}

func (a *Article) getEffectiveOid(oid string) string {
	var statuses, urlObjects, effectiveOid interface{}

	// 目前只有微博文章才去取微博内短链对象的oid
	if !strings.HasPrefix(oid, "1042015:articleMblog_") {
		return oid
	}

	effectiveOidCacheKey := "EffectiveOid:" + oid
	if info, found := a.Ctx.Cache.Get(effectiveOidCacheKey); found {
		return info.(string)
	}

	mid := strings.TrimPrefix(oid, "1042015:articleMblog_")
	ret, err := a.Ctx.WeiboApiJson("i2.api.weibo.com/2/statuses/show_batch.json",
		map[string]interface{}{"source": contexts.APPKEY2, "ids": mid})
	if err != nil {
		a.Info("get weibo info faild, ret=%v", ret)
		goto NOT_FOUND
	}
	statuses = ret["statuses"]
	if statuses == nil || len(statuses.([]interface{})) == 0 {
		goto NOT_FOUND
	}

	urlObjects = statuses.([]interface{})[0].(map[string]interface{})["url_objects"]
	if urlObjects == nil || len(urlObjects.([]interface{})) == 0 {
		goto NOT_FOUND
	}

	effectiveOid = urlObjects.([]interface{})[0].(map[string]interface{})["object_id"]
	if effectiveOid == nil {
		goto NOT_FOUND
	} else {
		eoid := effectiveOid.(string)
		a.Info("get effective oid %v for %v", eoid, oid)
		a.Ctx.Cache.Set(effectiveOidCacheKey, effectiveOid, effectiveOidExpireTime)
		a.Ctx.Cache.Set("OriginalOid:" + eoid, oid, effectiveOidExpireTime)
		return eoid
	}
	NOT_FOUND:
	a.Ctx.Cache.Set(effectiveOidCacheKey, oid, effectiveOidExpireTime)
	return oid
}

func (a *Article) getOriginalOid(oid string) string {
	if info, found := a.Ctx.Cache.Get("OriginalOid:" + oid); found {
		return info.(string)
	}
	return oid
}

func (a *Article) getArticleFromDb(oid string, atime int, isNewID bool) (result map[string]interface{}, err error) {
	months := 6
	result = make(map[string]interface{})
	t := time.Now()
	var rows []interface{}
	if atime == 0 && isNewID {
		ts, err := utils.GetTimeFromIdString(oid)
		if ts > 0 || err == nil {
			atime = int(ts)
		}
	}
	if atime > 0 {
		// 如果指定了文章发布时间，只在指定月份的文章表里查询
		t = time.Unix(int64(atime), 0)
		months = 1
	}
	selectField := "oid"
	if isNewID {
		selectField = "id"
	}
	for len(rows) == 0 && months > 0 {
		year, month, day := t.Date()
		a.Debug("query db with time:%d-%d-%d oid=%s", year, month, day, oid)
		fields := []string{"id","uid","oid","time","updtime","type","category","source","title","url","tags","images","content","flag","mid","cover","created_time","summary","cms_options"}
		if t.Unix() > 1464710400 {  // 6月份之前的表没有original_url字段
			fields = append(fields, "original_url")
		}
		if t.Unix() > 1472660000 {
			fields = append(fields, "top_groupid")
		}
		sql := fmt.Sprintf("select %s from articles_%s where %s='%s'",
			strings.Join(fields, ","), t.Format("200601"), selectField, oid)
		rows, err = a.Ctx.GetDBInstance("articles", "article_r").Query(sql)
		if rows == nil || err != nil {
			a.Error("query failed:%v", err)
			return
		}
		t = t.AddDate(0, -1, 0)
		months--
		if len(rows) == 0 {
			continue
		}
		article, ok := rows[0].(*models.Articles)
		if ok {
			result = article.Format()
		}
	}

	return
}

func (a *Article) getArticle(id string, atime int) (map[string]interface{}, /*cached*/ bool, error) {
	cachedArticles, _ := a.getArticleInfoFromCache([]string{id})
	a.Debug("getArticleInfoFromCache, num=%d", len(cachedArticles))
	if v, exists := cachedArticles[id]; exists {
		return v, true, nil
	}

	isNewID := utils.IsNewID(id)
	var result map[string]interface{}
	var err error
	isCardArticle := utils.IsCardArticle(id)
	if isCardArticle {
		results := a.getCardArticlesFromDb([]string{id})
		result = results[id]
		if len(results) == 0 || len(result) == 0 {
			return nil, false, errors.New("not found")
		}
	} else {
		result, err = a.getArticleFromDb(id, atime, isNewID)
	}
	a.Debug("Article.Fetch after get from db err=%v", err)
	if err != nil {
		a.Info("getArticleFromDb faild:%v", err)
		return result, false, err
	}

	a.Debug("setArticleInfoToCache")
	err = a.Ctx.Article().(*Article).setArticleInfoToCacheSingle(result, isNewID, isCardArticle)
	// TODO: 查不到的文章写反向缓存
	if err != nil {
		a.Info("set article cache failed:%v", err)
		return result, false, err
	}

	a.Debug("Article.Fetch after set mc")

	return result, false, err
}

// Fetch 获取单篇文章
func (a *Article) Fetch(args *ArticleFetchArgs, reply *ArticleFetchResult) error {
	t1 := time.Now().UnixNano()
	if len(args.Oid) < 5 {
		a.Error("invalid oid:%s", args.Oid)
		reply.Result = Result{Status: 1, Error: "invalid oid"}
		return nil
	}
	var oid string
	if utils.IsNewID(args.Oid) {
		oid = a.getOidByID(args.Oid)
	} else {
		oid = args.Oid
	}
	t2 := time.Now().UnixNano()
	atime := 0
	if args.Time > 0 {
		atime = args.Time
	}
	reply.Result = Result{Status: 0, Error: "success"}
	result, cached, err := a.getArticle(args.Oid, atime)
	if err != nil {
		a.Error("fetch article failed:%v", err)
		reply.Result = Result{Status: 1, Error: fmt.Sprintf("fetch article failed:%s", err.Error())}
		return nil
	}
	t3 := time.Now().UnixNano()

	result["comment_count"] = a.fetchCommentCount(oid, args.Oid)
	t4 := time.Now().UnixNano()
	if args.Needcounts == 1 {
		a.fetchCounts([]string{oid}, map[string]string{oid:args.Oid}, &ArticlesType{oid:result})
	}
	t5 := time.Now().UnixNano()
	a.filterTags(result)
	t6 := time.Now().UnixNano()
	a.Info("singlefetch time used t2:%d t3:%d t4:%d t5:%d t6:%d total:%d ms cached=%v",
		t2-t1, t3-t2, t4-t3, t5-t4, t6-t5, (t6-t1)/1e6, cached)

	reply.Data = result
	return nil
}

func (a *Article) getArticlesFromDb(missed []string, isNewID bool) (result ArticlesType, err error) {
	result = make(ArticlesType)
	oidArg := []string{}
	missedOidMap := make(map[string]bool)
	for _, tmpOid := range missed {
		oidArg = append(oidArg, " \"" + tmpOid + "\"")
		missedOidMap[tmpOid] = true
	}

	var rows []interface{}
	months := []time.Time{}
	idMonthMap := make(map[string][]string)
	selectField := "oid"
	if isNewID {
		selectField = "id"
		monthsMap := map[string]time.Time{}
		for _, id := range(missed) {
			ts, err := utils.GetTimeFromIdString(id)
			if ts > 0 || err == nil {
				t := time.Unix(ts, 0)
				dateStr := t.Format("200601")
				monthsMap[dateStr] = t
				if _, exists := idMonthMap[dateStr]; exists {
					idMonthMap[dateStr] = append(idMonthMap[dateStr], " \"" + id + "\"")
				} else {
					idMonthMap[dateStr] = []string{" \"" + id + "\""}
				}
			}
		}
		for _, t := range(monthsMap) {
			months = append(months, t)
		}
	} else {
		t := time.Now()
		for i := 0; i < 6; i++ {
			months = append(months, t)
			idMonthMap[t.Format("200601")] = oidArg
			t = t.AddDate(0, -1, 0)
		}
	}
	for m := 0; len(result) < len(missed) && m < len(months); m++ {
		t := months[m]
		year, month, day := t.Date()
		a.Debug("query db with time:%d-%d-%d oid=%s", year, month, day, missed)
		fields := []string{"id","uid","oid","time","updtime","type","category","source","title","url","tags","images","flag","mid","cover","created_time","summary","cms_options"}
		if t.Unix() > 1464710400 {  // 6月份之前的表没有original_url字段
			fields = append(fields, "original_url")
		}
		if t.Unix() > 1472660000 {
			fields = append(fields, "top_groupid")
		}
		dateStr := t.Format("200601")
		sql := fmt.Sprintf("select %s from articles_%s where %s in (%s)",
			strings.Join(fields, ","), dateStr, selectField, strings.Join(idMonthMap[dateStr], ","))
		rows, err = a.Ctx.GetDBInstance("articles", "article_r").Query(sql)
		if rows == nil || err != nil {
			a.Error("query failed:%v", err)
			return
		}
		if len(rows) == 0 {
			continue
		}

		for i := 0; i < len(rows); i++ {
			article := (rows[i]).(*models.Articles)
			if isNewID {
				result[article.Id] = article.Format()
			} else {
				result[article.Oid] = article.Format()
			}
		}
	}
	return
}

func (a *Article) getCardArticlesFromDb(ids []string) (result ArticlesType) {
	result = make(ArticlesType)
	oidArg := []string{}
	for _, id := range ids {
		oidArg = append(oidArg, " \"" + id + "\"")
	}

	var rows []interface{}
	sql := fmt.Sprintf("select * from article_card where id in (%s)", strings.Join(oidArg, ","))
	rows, err := a.Ctx.GetDBInstance("articlecard", "article_r").Query(sql)
	if rows == nil || err != nil {
		a.Error("query failed:%v", err)
		return
	}

	for i := 0; i < len(rows); i++ {
		article := (rows[i]).(*models.ArticleCard)
		result[article.Id] = article.Format()
	}
	return
}

// MultiFetch 获取文章详情
func (a *Article) MultiFetch(args *ArticleMultiFetchArgs, reply *ArticleMultiFetchResult) error {
	t1 := time.Now().UnixNano()
	oids     := []string{}
	realOids := []string{}
	oidIDMap := make(map[string]string)
	isNewID  := false
	for _, oid := range (args.Oid) {
		oids = append(oids, oid)
		if utils.IsCardArticle(oid) {
			continue
		}
		
		if utils.IsSpecial(oid) {
			continue
		}
		
		// oids是查数据的id（id或oid），realOids是查评论的id（oid）
		if utils.IsNewID(oid) {
			isNewID = true
			realOid := a.getOidByID(oid)
			if realOid != "" {
				realOids = append(realOids, realOid)
				oidIDMap[realOid] = oid
			}
		/*} else if isNewID {
			a.Info("invalid argument:%s", oid)
			reply.Result = Result{Status: 1, Error: "invalid argument, oids should in same format"}
			return nil
		}*/
		} else {
			realOids = append(realOids, oid)
			oidIDMap[oid] = oid
		}
	}
	reply.Result = Result{Status: 0, Error: "success"}
	reply.Data   = make(map[string]map[string]interface{})
	var cachedArticles ArticlesType
	var missed []string
	t2 := time.Now().UnixNano()
	cachedArticles, missed = a.getArticleBatchInfoFromCache(oids)
	t3 := time.Now().UnixNano()
	if len(cachedArticles) == len(oids) {
		// TODO: 将这个分支与后面的逻辑统一起来，不单独在此返回
		a.fetchCommentCountBatch(realOids, oidIDMap, &cachedArticles)
		t4 := time.Now().UnixNano()
		if args.Needcounts == 1 || utils.InArray(args.Fields, "counts") {
			a.fetchCounts(realOids, oidIDMap, &cachedArticles)
		}
		t5 := time.Now().UnixNano()
		if (args.Needliked == 1 || utils.InArray(args.Fields, "liked")) && args.Uid != 0 {
			a.fetchLiked(args.Uid, realOids, oidIDMap, &cachedArticles)
		}
		t6 := time.Now().UnixNano()
		for i := range(cachedArticles) {
			a.filterTags(cachedArticles[i])
		}
		t7 := time.Now().UnixNano()
		if (utils.InArray(args.Fields, "famous")) {
			a.fetchFamousActive(&cachedArticles)
		}
		t8 := time.Now().UnixNano()
		a.filterFlag(&cachedArticles)
		a.Info("multifetch time used t2:%d t3:%d t4:%d t5:%d t6:%d t7:%d t8:%d total:%d ms",
			t2-t1, t3-t2, t4-t3, t5-t4, t6-t5, t7-t6, t8-t7, (t7-t1)/1e6)
		reply.Data = cachedArticles
		return nil
	}

	var missedArticles, missedCardArticles, missedSpecials []string
	for _, id := range(missed) {
		if utils.IsCardArticle(id) {
			missedCardArticles = append(missedCardArticles, id)
		} else if utils.IsSpecial(id) {
			missedSpecials = append(missedSpecials, id)
		} else {
			missedArticles = append(missedArticles, id)
		}
	}
	t4 := time.Now().UnixNano()
	result, err := a.getArticlesFromDb(missedArticles, isNewID)
	if err != nil {
		a.Info("getArticlesFromDb faild:%v", err)
	}
	t5 := time.Now().UnixNano()

	err = a.Ctx.Article().(*Article).setArticleBatchInfoToCache(result, isNewID)
	// TODO: 查不到的文章写反向缓存
	if err != nil {
		a.Info("set article cache failed:%v", err)
	}
	t6 := time.Now().UnixNano()

	// 合并数据库读取的文章与mc读取的文章
	a.combineArticles(&result, &cachedArticles)
	if len(missedCardArticles) > 0 {
		missedCardArticlesResult := a.getCardArticlesFromDb(missedCardArticles)
		err = a.Ctx.Article().(*Article).setArticleBatchInfoToCache(missedCardArticlesResult, true)
		a.combineArticles(&result, &missedCardArticlesResult)
	}
	
	if len(missedSpecials) > 0 {
		missedSpecialsResult := a.getArticleSpecials(missedSpecials)
		err = a.Ctx.Article().(*Article).setArticleBatchInfoToCache(missedSpecialsResult, true)
		a.combineArticles(&result, &missedSpecialsResult)
	}
	
	t7 := time.Now().UnixNano()

	//a.fetchCommentCountBatch(realOids, oidIDMap, &result)
	if args.Needcounts == 1 || utils.InArray(args.Fields, "counts") {
		a.fetchCounts(realOids, oidIDMap, &result)
	}
	t8 := time.Now().UnixNano()
	if (args.Needliked == 1 || utils.InArray(args.Fields, "liked")) && args.Uid != 0 {
		a.fetchLiked(args.Uid, realOids, oidIDMap, &result)
	}
	t9 := time.Now().UnixNano()
	for i := range(result) {
		a.filterTags(result[i])
	}
	t10 := time.Now().UnixNano()
	if (utils.InArray(args.Fields, "famous")) {
		a.fetchFamousActive(&result)
	}
	t11 := time.Now().UnixNano()

	if len(result) != len(args.Oid) {
		missedOids := []string{}
		for _, oid := range (oids) {
			if _, exists := result[oid]; !exists {
				missedOids = append(missedOids, oid)
			}
		}
		a.Info("some article not return:%v", missedOids)
	}
	a.filterFlag(&result)
	reply.Data = result
	t12 := time.Now().UnixNano()
	a.Info("multifetch time used t2:%d t3:%d t4:%d t5:%d t6:%d t7:%d t8:%d t9:%d t10:%d t11=%d t12=%d total:%d ms",
		t2-t1, t3-t2, t4-t3, t5-t4, t6-t5, t7-t6, t8-t7, t9-t8, t10-t9, t11-t10, t12-t11, (t10-t1)/1e6)
	return nil
}

// 过滤标签字段，去掉标签流文章数少于5条的标签及下线标签
func (a *Article) filterTags(article map[string]interface{}) {
	oid, ok := article["oid"].(string)
	if !ok {
		return
	}
	cache := a.Ctx.GetCache("article")
	item, ok := cache.HGet(oid, "tags")
	if ok {
		cachedTags, ok := item.(string)
		if ok {
			article["tags"] = cachedTags
			return
		}
	}

	if s, exists := article["tags"]; !exists || s == "" {
		return
	} else {
		var tags, newTags []map[string]interface{}
		d := very_simple_json.NewSimpleJsonDecoder([]byte(s.(string)))
		result, err := d.Decode()
		if err == nil {
			resultArray, ok := result.([]interface{})
			if !ok {
				a.Debug("tags empty, oid=%v tags=%#v", article["oid"], result)
				return
			}
			for _, t := range(resultArray) {
				t, ok := t.(map[string]interface{})
				if ok {
					tags = append(tags, t)
				}
			}
		} else {
			a.Info("oid=%s very_simple_json decode failed, rollback to encoding/json:%v", article["oid"], err)
			err = json.Unmarshal([]byte(s.(string)), &tags)
			if err != nil {
				a.Error("filterTags failed:%v", err)
				return
			}
		}
		disabledTags := (&User{a.Type}).getDisabledTags()
		for _, tag := range(tags) {
			weight := utils.GetFloatValueByPath(tag, "float64:weight")
			objectID := utils.GetStringValueByPath(tag, "string:object_id")
			if _, exists := disabledTags[objectID]; exists {
				continue
			}
			feedCount := (&Tag{a.Type}).getFeedCount(objectID)
			if weight < 0.45 || feedCount < 5 {
				continue
			}
			newTags = append(newTags, tag)
		}
		if len(newTags) != len(tags) {
			if len(newTags) == 0 {
				article["tags"] = ""
				return
			}
			var newTagsString string
			newTagsString, err = very_simple_json.ToJsonArray(newTags, tagsJsonLayout)
			if err != nil {
				a.Info("very_simple_json.ToJson failed, rollback to encoding/json:%v", err)
				var b []byte
				b, err = json.Marshal(newTags)
				newTagsString = string(b)
			}
			if err == nil {
				article["tags"] = newTagsString
				cache.HSet(oid, "tags", newTagsString, 3 * time.Minute)
			}
		}
	}
}

func (a *Article) filterFlag (result *ArticlesType) {
	if result == nil {
		return
	}
	for oid, article := range(*result) {
		flag, ok := article["flag"]
		if ok {
			flag, ok := flag.(int)
			if ok && ((flag == 4) || (flag == 9) || (flag == 11)) {
				a.Info("remove article flag=%d oid=%s", flag, oid)
				delete(*result, oid)
			}
		}
	}
}

func (a *Article) getActiveInfo(i interface{}) (result []map[string]interface{}, count int) {
	s, ok := i.(string)
	if !ok {
		return
	}
	uidStrs := strings.Split(s, ",")
	count = len(uidStrs)
	if count == 0 {
		return
	}
	uidMap := make(map[string]bool)
	uids := []int64{}
	for _, uidStr := range(uidStrs) {
		if uidMap[uidStr] {    // uid有可能重复
			continue
		}
		uidMap[uidStr] = true
		uid, err := strconv.ParseInt(uidStr, 10, 0)
		if err != nil {
			a.Info("getActiveInfo parse uid failed:%v", err)
			continue
		}
		uids = append(uids, uid)
	}
	userService := &User{a.Type}
	
	for _, uid := range(uids) {
		info, err := userService.getUserScreenInfo(uid)
		if err != nil {
			a.Info("getUserScreenInfo failed:%v", err)
			continue
		}
		result = append(result, info)
		if len(result) > 1 {
			break
		}
	}
	return
}

func (a *Article) fetchFamousActive(result *ArticlesType) error {
	r, err := a.Ctx.Redis("r7018", 0)
	if err != nil {
		a.Error("get redis 7018 failed")
		return err
	}
	var activityUsers []map[string]interface{}
	var activityCount int
	var activityType int
	cache := a.Ctx.GetCache("article")
	for k, article := range(*result) {
		id, _ := article["id"].(string)
		oid, _ := article["oid"].(string)
		if !utils.IsTopArticle(id) {
			continue
		}
		
		item, ok := cache.HGet(oid, "famous")
		if ok {
			famous, ok := item.(map[string]interface{})
			if ok {
				if len(famous) > 0 {
					(*result)[k]["famous"] = famous
				}
				continue
			}
		}
		
		key := famousActiveKeyPrefix + oid
		ret, err := r.HGetAll(key)
		if err != nil {
			a.Error("get famous active failed:%v", err)
			continue
		}
		activityType = activityTypeStatus
		activityUsers, activityCount = a.getActiveInfo(ret["status"])
		if len(activityUsers) < 2 {
			activityType = activityTypeLike
			activityUsers, activityCount = a.getActiveInfo(ret["like"])
		}
		if len(activityUsers) > 1 {
			(*result)[k]["famous"] = map[string]interface{}{
				"type": activityType,
				"count": activityCount,
				"users": activityUsers,
			}
			cache.HSet(oid, "famous", (*result)[k]["famous"], 10 * time.Minute)
		} else {
			cache.HSet(oid, "famous", map[string]interface{}{}, 1 * time.Minute)
		}
	}
	return nil
}