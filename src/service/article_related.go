package service

import (
	"github.com/garyburd/redigo/redis"
	"utils"
	"math/rand"
	"strings"
	"time"
	"strconv"
	"errors"
)

const (
	articleRelatedPrefix = "REL:"
	topRelatedRedisPrefix  = "TAREL:"
	defaultTopRelatedCount = 20
	minTopRelatedScore     = 0.2
)

// ArticleRelatedArgs ...
type ArticleRelatedArgs struct {
	CommonArgs
	Oid          string   `msgpack:"oid"`
	Aid          string   `msgpack:"aid"`
	Uid          uint64   `msgpack:"uid"`
	Tag          string   `msgpack:"tag"`
	Count        int      `msgpack:"count"`
	WriteHist    bool     `msgpack:"write_hist"`
	ArticleInfos []string `msgpack:"articleinfos"`
}

// ArticleRelatedReplyData ...
type ArticleRelatedReplyData struct {
	Type         string   `msgpack:"type"`
	ArticleInfos []string `msgpack:"articleids"`
}

// ArticleRelatedReply ...
type ArticleRelatedReply struct {
	Code    int                      `msgpack:"code"`
	Message string                   `msgpack:"message"`
	Data    *ArticleRelatedReplyData `msgpack:"data"`
}

// ArticleRelatedResult ...
type ArticleRelatedResult struct {
	Result Result
	Data    *ArticleRelatedReplyData `msgpack:"data"`
}

// TopRelatedArgs 头条文章相关文章参数类型
type TopRelatedArgs struct {
	CommonArgs
	Oid 	string `msgpack:"oid" must:"1" comment:"oid"`
	Count 	int    `msgpack:"count" must:"0" comment:"count"`
}

// TopRelatedResult 头条文章相关文章返回类型
type TopRelatedResult struct {
	Result Result
	Data   []string
}

// Related 兼容老接口，需要将Service.ArticleRelated迁移为Article.Related
func (a *Article) Related(args *ArticleRelatedArgs, reply *ArticleRelatedResult) error {
	serviceHandler := &Service{a.Type}
	var result ArticleRelatedReply
	err := serviceHandler.ArticleRelated(args, &result)
	status := result.Code
	errMsg := result.Message
	if status == 0 {
		errMsg = "success"
	}
	reply.Result = Result{Status: status, Error: errMsg}
	reply.Data = result.Data
	return err
}

// ArticleRelated ...
func (s *Service) ArticleRelated(args *ArticleRelatedArgs, reply *ArticleRelatedReply) error {
	reply.Code = -1
	if args.Oid == "" || (args.Uid <= 0 && args.Aid == "") {
		reply.Message = "params error"
		return nil
	}

	reply.Data = new(ArticleRelatedReplyData)

	if s.Ctx.Cfg.Switch.DisableRelatedArticle {
		// 降级，不返回相关文章
		return nil
	}

	var (
		articleList []string
		count       = 3
		cacheKey    string
		hist        map[string]uint64
		hashKey     uint64
	)

	if args.Count > 0 {
		count = args.Count
	}

	if args.Uid > 0 {
		cacheKey = s.usrReadCacheKey(args.Uid)
		hashKey = uint64(args.Uid)
	} else if len(args.Aid) > 0 {
		cacheKey = s.aidReadCacheKey(args.Aid)
		hashKey = uint64(utils.CRC32(args.Aid))
	}

	relatedArticleList := s.getRelatedArticleList(args.Oid)
	if relatedArticleList != nil {
		reply.Data.Type = "related"
		hist = s.readHistory(cacheKey, hashKey)

		for _, oid := range relatedArticleList {
			if oid == "" || strings.Index(oid, ":") == -1 {
				s.Info("%v invalid item:oid=%v ", args.Oid, oid)
				continue
			}
			if hist == nil {
				articleList = append(articleList, oid)
			} else {
				if _, found := hist[oid]; !found && oid != args.Oid {
					articleList = append(articleList, oid)
				}
			}
		}
	}

	if args.WriteHist {
		s.writeHistory(cacheKey, hashKey, &[]string{args.Oid})
	}

	if articleList != nil {
		reply.Code = 0
		reply.Message = "ok"
		t := time.Now()
		rand.Seed(int64(t.Nanosecond()))
		numbers := rand.Perm(len(articleList))
		var rst []string
		for _, i := range numbers {
			if len(rst) == count {
				break
			}
			rst = append(rst, articleList[i])
		}

		reply.Data.ArticleInfos = append(reply.Data.ArticleInfos, rst...)
	}
	s.Info("ArticleRelated oid=%v related_count=%d ret_count=%d count=%v oids=%v match=%v",
		args.Oid, len(relatedArticleList), len(articleList), args.Count,
		reply.Data.ArticleInfos, len(reply.Data.ArticleInfos) == args.Count)

	return nil
}

func (s *Service) isVideoArticle(oid string) (bool, error) {
	isVedio := false
	
	isNewID := utils.IsNewID(oid)
	if isNewID { // 新id
		id, err := strconv.ParseInt(oid, 10, 0)
		if err != nil {
			s.Debug("article related oid is wrong, oid:%v", oid)
			return isVedio, err
		}
		idGen, err := utils.Decode(id)
		if err != nil {
			s.Debug("article related decoce is wrong, oid:%v", oid)
			return isVedio, err
		}
		if idGen.Bid == utils.ARTICLE_BID_VIDEO { //  bid 等于 9,表示文章是视频
			isVedio = true
		}
	} else {
		article := &Article{s.Type} // 创建Article对象,获取文章属性
		result, _, err := article.getArticle(oid, 0)
		if err != nil {
			s.Debug("article related fetch failed, oid:%v", oid)
			return isVedio, err
		}
		
		if _, exists := result["tags"]; !exists {
			if _, exists := result["content"]; !exists {
				s.Debug("article related no tag and content")
				return isVedio, errors.New("tag and context no exists")
			}
			switch t := result["content"].(type) {
			case string:
				isVedio = strings.Contains(t, "\"videos\"")
			case *string:
				isVedio = strings.Contains(*t, "\"videos\"")
			}
		}  else {
			switch t := result["tags"].(type) {
			case string:
				isVedio = strings.Contains(t, "1042015:tw_video")
			case *string:
				isVedio = strings.Contains(*t, "1042015:tw_video")
			}
		}
	}
	
	return isVedio, nil
	
}

func (s *Service) getRelatedArticleList(oid string) []string {
	isVedio, err := s.isVideoArticle(oid)
	if err != nil {
		s.Info("article related is vidio article failed, err:%v oid:%v", err, oid)
		return []string{}
	}
	
	if isVedio{
		s.Info("article related get from api")
		return s.getRelatedArticleListFromAPI(oid)
	}
	s.Info("article related get from redis")
	return s.getRelatedArticleListFromRedis(oid);
}

func (s *Service) getRelatedArticleListFromAPI(oid string) []string {
	params := map[string]interface{}{
		"pid":"751", "tobj":oid, "num":"20", // 默认20个,没有考虑外层count值
	}

	ret, err := s.Ctx.WeiboApiJson("i.ruf.recom.weibo.cn/get_recom", params)
	if err != nil {
		s.Info("related article failed to connect weibo_api_json:err:%s, oid=%v", err.Error(), oid)
		return []string{}
	}

	if ret == nil || ret["code"] == nil {
		s.Info("related article get weibo info empty:oid=%v", oid)
		return []string{}
	}
	
	rc, ok := ret["code"].(float64)
	if !ok || rc != 200 {
		s.Info("related article get weibo info failed:oid=%v", oid)
		return []string{}
	}
	
	results, ok := ret["results"].([]interface{})
	if ok {
		num := len(results)
		if num > 0 {
			list := []string{}
			for i := range results {
				oneID, ok := results[i].(string)
				if !ok {
					continue
				}
				kv := strings.Split(oneID, "~")
				list = append(list, kv[0])
			}
			
			return list
		}
		s.Info("related article get zero result, oid=%v", oid)
	}

	return []string{}
}

func (s *Service) getRelatedArticleListFromRedis(oid string) []string {
	key := articleRelatedPrefix + oid
	rds, err := s.Ctx.Redis("article_related_r")
	if err != nil {
		s.Info("failed to connect article_related_r redis:%v", err)
		return nil
	}

	list, _ := redis.Strings(rds.Do("zrange", key, 0, -1))
	if len(list) == 0 {
		articleService := &Article{s.Type}
		groupid, err := articleService.getGroupID(oid)
		if groupid == "" || err != nil {
			s.Info("failed to get groupid oid=%s :%v", oid, err)
			return nil
		}
		if groupid == oid {
			s.Info("same groupid oid=%s", oid)
			return nil
		}
		key = articleRelatedPrefix + groupid
		list, _ = redis.Strings(rds.Do("zrange", key, 0, -1))
		relatedCount := 0
		if list != nil {
			relatedCount = len(list)
		}
		s.Info("found groupid %v, count=%d err=%v", groupid, relatedCount, err)
	}
	return list
}

func (s *Service) getFeed(key string, start int, stop int, readlen int) []string {
	if key == "" {
		return nil
	}

	c, err := s.Ctx.Redis("feed")
	if err != nil {
		s.Info("failed to connect to feed redis:%s", err.Error())
		return nil
	}

	var (
		articleIds []string
	)

	result, err := redis.Strings(c.Do("ZREVRANGE", key, start, stop))
	if err != nil {
		return nil
	}
	i := 0
	for _, articleId := range result {
		if i < readlen || readlen == 0 {
			articleIds = append(articleIds, articleId)
			i = i + 1
		} else {
			break
		}
	}

	return articleIds
}


func (a *Article) getTopRelated(oid string) (related []interface{}, err error) {
	r, err := a.Ctx.Redis("article_related_r")
	if err != nil {
		a.Info("get redis failed:%v", err)
		return
	}
	key := topRelatedRedisPrefix + oid
	var result interface{}
	result, err = r.Do("ZRANGE", key, 0, 100, "withscores")
	if err != nil {
		a.Info("get related data failed:%v", err)
		return
	}
	var ok bool
	related, ok = result.([]interface{})
	if !ok || related == nil || len(related) % 2 != 0 {
		a.Info("get related data failed:bad format")
		err = errors.New("system error")
		return
	}
	return
}

// TopRelated 头条文章相关文章
func (a *Article) TopRelated(args *TopRelatedArgs, reply *TopRelatedResult) error {
	if !utils.CheckOid(args.Oid) {
		reply.Result = Result{Status: 1, Error: "invalid argument"}
		return nil
	}
	reply.Result = Result{Status: 0, Error: "success"}
	if args.Oid == "" {
		reply.Result = Result{Status: 1, Error: "failed:empty oid"}
		return errors.New("failed:empty oid")
	}
	count := defaultTopRelatedCount
	if args.Count > 0 {
		count = args.Count
	}
	related, err := a.getTopRelated(args.Oid)
	if err != nil || len(related) == 0 {
		topGroupID := a.getTopGroupID(args.Oid)
		if topGroupID != "" {
			related, err = a.getTopRelated(topGroupID)
		}
	}
	var score float64
	for i := 0; (i < len(related)) && (len(reply.Data) < count); i += 2 {
		if score, err = strconv.ParseFloat(string(related[i+1].([]byte)), 64); err != nil {
			a.Info("get related data score failed:%v", err)
			continue
		}
		if score < minTopRelatedScore {
			continue
		}
		reply.Data = append(reply.Data, string(related[i].([]byte)))
	}
	return nil
}
