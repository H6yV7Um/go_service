package service

import (
	"fmt"
	"strconv"
	"time"
	"utils"
	"models"
	"errors"
)

// Like ...
type Like struct {
	Type
}

// CheckLikeResult ...
type CheckLikeResult struct {
	Result Result
	Data   map[string]int
}

// LikeArgs ...
type LikeArgs struct {
	Common	CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Oid   	string 		`msgpack:"oid" must:"1" comment:"oid"`
	Uid   	uint64 		`msgpack:"uid" must:"1" comment:"uid"`
	Liked 	uint64 		`msgpack:"liked" must:"1" comment:"liked"`
}

// CheckLikeArgs ...
type CheckLikeArgs struct {
	Common	CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Oid   	[]string 	`msgpack:"oid" must:"1" comment:"oid"`
	Uid   	uint64   	`msgpack:"uid" must:"1" comment:"uid"`
	Liked 	uint64   	`msgpack:"liked" must:"1" comment:"liked"`
}

const (
	redisLikeW = "m7018"
	redisLikeR = "r7018"
	redisLikeExpireTime = 1296000
	likeTypeArticle = 1
	likeTypeComment = 2
)

// CheckLikeBatch ...
func (l *Like) CheckLikeBatch(args *CheckLikeArgs, reply *CheckLikeResult, likeType int) error {
	reply.Result = Result{Status: 0, Error: "success"}
	(*reply).Data = l.checkLikeBatch(args.Uid, args.Oid, likeType)
	return nil
}

func (l *Like) checkLikeBatch(uid uint64, oids []string, likeType int) (likedMap map[string]int) {
	likedMap = make(map[string]int)
	for _, oid := range oids {
		likedMap[oid] = 0
	}

	r, err := l.Ctx.Redis(redisLikeR)
	if err != nil {
		return
	}
	redisKey := fmt.Sprintf("al:%d", uid)
	result, err := r.HMGet(redisKey, oids)
	missed := []string{}
	if err != nil {
		missed = oids
	} else {
		for oid, liked := range(result) {
			likedStr, _ := liked.(string)
			likedMap[oid], _ = strconv.Atoi(likedStr)
		}
		for _, oid := range(oids) {
			if _, exists := likedMap[oid]; !exists {
				missed = append(missed)
			}
		}
	}
	for _, oid := range missed {
		liked, _ := l.checkLike(uid, oid, likeType)
		likedMap[oid] = liked
	}
	return
}

// CheckLike ...
func (l *Like) checkLike(uid uint64, oid string, likeType int) (liked int, err error) {
	l.Info("CheckLike oid=%v uid=%v", oid, uid)
	liked = 0
	if uid == 0 || oid == "" {
		l.Info("checklike failed, uid or oid/cid is empty")
		return 0, errors.New("checklike failed, uid or oid/cid is empty")
	}
	r, err := l.Ctx.Redis(redisLikeR)
	if err != nil {
		return 0, err
	}

	redisKey := fmt.Sprintf("al:%d", uid)
	if v, err := r.Do("HGET", redisKey, oid); v != nil && err == nil {
		l.Info("found in redis key=%v v=%v", redisKey, utils.String(v))
		return utils.Atoi(utils.String(v), 0), nil
	}

	var tablePrefix, tableField, dbModel, dbAlias string
	if likeType == likeTypeComment {
		tablePrefix = "comment_like_"
		tableField = "cid"
		dbModel = "commentlike"
		dbAlias = "commentlike_r"
	} else {
		tablePrefix = "article_like_"
		tableField = "oid"
		dbModel = "articlelike"
		dbAlias = "articlelike_r"
	}

	tableName := tablePrefix + strconv.Itoa(int(utils.Hash(strconv.FormatUint(uid, 10)) % 32))
	sql := fmt.Sprintf("SELECT * FROM %s WHERE uid='%d' AND " + tableField + "='%s'", tableName, uid, oid)

	result, err := l.Ctx.GetDBInstance(dbModel, dbAlias, uid).Query(sql)
	if result == nil || err != nil {
		l.Error("checklike query failed:%v", err)
		return
	}
	if len(result) == 0 {
		l.Info("checklike not found in db:%s", oid)
		return
	}

	var t string
	if likeType == likeTypeComment {
		likeObject := result[0].(*models.CommentLike)
		t = likeObject.Ctime
	} else {
		likeObject := result[0].(*models.ArticleLike)
		t = likeObject.Ctime
	}

	tm, _ := time.Parse("2006-01-02 15:04:05", t)
	tm2 := strconv.FormatInt(tm.Unix(), 10)
	r2, err := l.Ctx.Redis(redisLikeW)
	if err != nil {
		return 0, err
	}
	if _, err := r2.Do("HSET", redisKey, oid, tm2); err != nil {
		l.Error("Set redis(%v) failed key=%v oid=%v", redisLikeW, redisKey, oid)
		return 0, err
	}
	r2.Do("EXPIRE", redisKey, redisLikeExpireTime)
	return 1, nil
}

// Like ...
func (l *Like) Like(args *LikeArgs, reply *CommonResult, likeType int) error {
	reply.Result = Result{Status: 0, Error: "success"}
	if args.Uid == 0 || args.Oid == "" {
		l.Info("Like failed, uid or oid/cid is empty")
		reply.Result = Result{Status: 1, Error: "empty params"}
		return nil
	}

	r, err := l.Ctx.Redis(redisLikeW)
	if err != nil {
		l.Error("Like get redis failed:%v", err)
		reply.Result = Result{Status: 1, Error: "system error"}
		return nil
	}

	redisKey := fmt.Sprintf("al:%d", args.Uid)
	if args.Liked == 1 {
		_, err = r.Do("HSET", redisKey, args.Oid, args.Liked)
		r.Do("EXPIRE", redisKey, redisLikeExpireTime)
	} else {
		_, err = r.Do("HDEL", redisKey, args.Oid)
	}
	if err != nil {
		l.Error("Update redis(%v) failed key=%v oid=%v liked=%v", redisLikeW, redisKey, args.Oid, args.Liked)
		reply.Result = Result{Status: 1, Error: "system error"}
		return nil
	}
	
	var tablePrefix, tableField, dbModel, dbAlias string
	if likeType == likeTypeComment {
		tablePrefix = "comment_like_"
		tableField = "cid"
		dbModel = "commentlike"
		dbAlias = "commentlike_w"
	} else {
		tablePrefix = "article_like_"
		tableField = "oid"
		dbModel = "articlelike"
		dbAlias = "articlelike_w"
	}
	tableName := tablePrefix + strconv.Itoa(int(utils.Hash(strconv.FormatUint(args.Uid, 10)) % 32))

	var sql string
	if args.Liked == 1 {
		sql = fmt.Sprintf("INSERT into %s SET uid='%d', " + tableField + "='%s' ON DUPLICATE KEY UPDATE " + tableField + "=" + tableField, tableName, args.Uid, args.Oid)
	} else {
		sql = fmt.Sprintf("DELETE from %s WHERE uid='%d' AND " + tableField + "='%s'", tableName, args.Uid, args.Oid)
	}
	
	result, err := l.Ctx.GetDBInstance(dbModel, dbAlias, args.Uid).Query(sql)
	l.Debug("like result=%v err=%v", result, err)
	if result == nil || err != nil {
		l.Error("like update db failed:%v", err)
		reply.Result = Result{Status: 1, Error: fmt.Sprintf("system error:%v", err)}
		return nil
	}

	return nil
}

func (l *Like) handleLikeCount(id string, liked uint64, likeType int) {
	if likeType == likeTypeArticle {
		l.handleArticleLikeCount(id, liked)
	} else {
		l.handleCommentLikeCount(id, liked)
	}
	
	return
}

func (l *Like) handleArticleLikeCount(oid string, liked uint64) {
	w, err := l.Ctx.Ssdb("subscribe_ssdb_w", 0)
	if err != nil {
		l.Error("increaseLikeCount get subscribe_ssdb_w failed:%v", err)
		return
	}
	
	key   := "ac_" + oid
	field := "l"
	var value int
	if liked == 1 {
		value = 1
	} else {
		value = -1
	}
	
	_, err = w.Do("hincr", key, field, value) // 不支持大写
	if err != nil {
		l.Error("handleLikeCount hincr failed, key:%v, field:%v, value:%v", key, field, value)
		return
	}
	
	key = articleCountsPrefix + oid
	l.Ctx.Cache.Delete(key)
	
	return
}

func (l *Like) handleCommentLikeCount(cid string, liked uint64) {
	counterService := Counter{l.Type}
	contentType := ContentTypeLike
	counterType := CounterTypeComment
	if liked == 1 {
		counterService.Increase(cid, counterType, contentType)
	} else {
		counterService.Decrease(cid, counterType, contentType)
	}
	
	return
}

func (l *Like) GetLikeNum(id string, likeType int) int {
	var ret int
	if likeType == likeTypeArticle {
		ret = l.getArticleLikeNum(id)
	} else {
		ret = l.getCommonLikeNum(id)
	}
	
	return ret
}

func (l *Like) getArticleLikeNum(oid string) int {
	r, err := l.Ctx.Ssdb("subscribe_ssdb_r", 0)
	if err != nil {
		l.Error("increaseLikeCount get subscribe_ssdb_w failed:%v", err)
		return -1
	}
	
	key      := "ac_" + oid
	field    := "l"
	ret, err := r.Do("hget", key, field)
	if err != nil {
		l.Error("get article likeNum failed, key:%v, field:%v", key, field)
		return -1
	}
	
	data := 0
	if len(ret) == 2 && (ret[0] == "ok" || ret[0] == "OK" || ret[0] == "Ok")   {
		data, _ = strconv.Atoi(ret[1])
		if data < 0 {
			l.recoverArticleLikeNum(oid)
			data = 0
		}
	}
	
	return data
}

func (l *Like) recoverArticleLikeNum(oid string) {
	w, err := l.Ctx.Ssdb("subscribe_ssdb_w", 0)
	if err != nil {
		l.Error("increaseLikeCount get subscribe_ssdb_w failed:%v", err)
		return
	}
	
	key   := "ac_" + oid
	field := "l"
	_, err = w.Do("hset", key, field, 0)
	if err != nil {
		l.Error("recover articleLikeNum failed, key:%v, field:%v", key, field)
	}
	
	return
}

func (l *Like) getCommonLikeNum(cid string) int {
	counterService := Counter{l.Type}
	data := counterService.Get(cid, CounterTypeComment, ContentTypeLike)
	return data
}