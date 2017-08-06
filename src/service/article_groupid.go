package service

import (
	"time"
	"fmt"
	"utils"
	"models"
)

const (
	groupidMcPrefix        = "GID:"
	topGroupidMcPrefix     = "TOPGID:"
	groupidMcExpireTime    = 24 * time.Hour
)

// GetGroupIDArgs 获取文章分组id参数类型
type GetGroupIDArgs struct {
	CommonArgs
	Oid string `msgpack:"oid" must:"1" comment:"oid"`
}

// GetGroupIDResult 获取文章分组id返回类型
type GetGroupIDResult struct {
	Result Result
	Data   string
}

func (a *Article) getGroupID(oid string) (string, error) {
	cache := a.Ctx.GetCache("article")
	item, ok := cache.HGet(oid, "gid")
	gid, ok := item.(string)
	if ok {
		return gid, nil
	}
	//mcClient, err := a.Ctx.MC("article")
	mcClient, err := a.Ctx.Memcache("article_mc")
	if err != nil {
		a.Error("MC error", err)
		return "", err
	}
	
	//res := mcClient.Get(groupidMcPrefix + oid)
	res, _ := mcClient.GetJson(groupidMcPrefix + oid)
	gid, ok = res.(string)
	if ok && gid != "" {
		cache.HSet(oid, "gid", gid, 0)
		return gid, nil
	}
	
	months := 6
	t := time.Now()
	for months > 0 {
		year, month, day := t.Date()
		a.Debug("query db with time:%d-%d-%d oid=%s", year, month, day, oid)
		sql := fmt.Sprintf("select groupid from related_%s where `oid`='%s'", t.Format("200601"), oid)
		rows, err := a.Ctx.GetDBInstance("related", "article_r").Query(sql)
		if rows == nil || err != nil {
			a.Error("query related table failed:%v", err)
			return "", err
		}
		t = t.AddDate(0, -1, 0)
		months--
		if len(rows) == 0 {
			continue
		}
		article, ok := rows[0].(*models.Related)
		if ok {
			gid = article.Groupid
			break
		}
	}
	if gid != "" {
		//mcClient.Set(groupidMcPrefix + oid, gid, int64(groupidMcExpireTime))
		mcClient.SetString(groupidMcPrefix + oid, gid, int(groupidMcExpireTime))
		cache.HSet(oid, "gid", gid, 0)
		return gid, nil
	}
	
	cache.HSet(oid, "gid", gid, 10 * time.Minute)
	return "", nil
}

// GetGroupID 获取文章分组id
func (a *Article) GetGroupID(args *GetGroupIDArgs, reply *GetGroupIDResult) error {
	if !utils.CheckOid(args.Oid) {
		reply.Result = Result{Status: 1, Error: "invalid argument"}
		return nil
	}
	reply.Result = Result{Status: 0, Error: "success"}
	reply.Data, _ = a.getGroupID(args.Oid)
	
	return nil
}

func (a *Article) getTopGroupID(oid string) (gid string) {
	cache := a.Ctx.GetCache("article")
	item, ok := cache.HGet(oid, "tgid")
	gid, ok = item.(string)
	if ok {
		return
	}
	
	//mcClient, err := a.Ctx.MC("article")
	mcClient, err := a.Ctx.Memcache("article_mc")
	if err != nil {
		a.Error("MC error", err)
		return
	}
	
	//res := mcClient.Get(topGroupidMcPrefix + oid)
	gid = mcClient.GetString(topGroupidMcPrefix + oid)
	cache.HSet(oid, "tgid", gid, 0)
	
	return
}

// GetGroupID 获取头条文章分组id
func (a *Article) GetTopGroupID(args *GetGroupIDArgs, reply *GetGroupIDResult) error {
	if !utils.CheckOid(args.Oid) {
		reply.Result = Result{Status: 1, Error: "invalid argument"}
		return nil
	}
	reply.Result = Result{Status: 0, Error: "success"}
	reply.Data = a.getTopGroupID(args.Oid)
	
	return nil
}

