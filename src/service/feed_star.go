package service

import (
	"errors"
	"utils"
	"strconv"
)

const (
	starFeedKeyPrefix = "feedu:famous:"
	starFeedVideoKeyPrefix = "feedu:famous_video:"
	starFeedNoVideokeyPrefix = "feedu:famous_no_video:"
	articleInfoKeyPrefix = "RD:"
	starFeedHistoryRedisNumber = 16
	starFeedHistoryKeyPrefix = "sf_his_"
)


// StarFeedGetResult ...
type StarFeedGetResult struct {
	Result Result
	Data   []string
}

// StarFeedGetArgs ...
type StarFeedGetArgs struct {
	CommonArgs
	Uid     int `msgpack:"uid" must:"1" comment:"uid"`
	FeedKey string `msgpack:"feedkey" must:"1" comment:"feedkey"`
	Num     int `msgpack:"num" must:"1" comment:"num"`
}

// StarFeedDelResult ...
type StarFeedDelResult struct {
	Result Result
}

// StarFeedDelArgs ...
type StarFeedDelArgs struct {
	CommonArgs
	FeedKey string `msgpack:"feedkey" must:"1" comment:"feedkey"`
	Oid     string `msgpack:"oid" must:"1" comment:"oid"`
}

// DelStarFeed ...
func (f *Feed)DelStarFeed(args *StarFeedDelArgs, reply *StarFeedDelResult) error {
	if err := f.delStarFeed(args); err != nil {
		reply.Result = Result{Status:1, Error:err.Error()}
		return err
	}
	reply.Result = Result{Status:0, Error:"success"}
	return nil
}

func (f *Feed)delStarFeed(args *StarFeedDelArgs) (err error) {
	wStarFeed, err := f.Ctx.Redis("star_feed_w", 0)
	if err != nil {
		f.Error("get star feed redis error:%s", err.Error())
		err = errors.New("get star feed redis error")
		return
	}

	feedKey := starFeedKeyPrefix + args.FeedKey
	err = wStarFeed.ZRem(feedKey, args.Oid)
	if err != nil {
		f.Error("del star feed error:%s", err.Error())
		err = errors.New("del star feed error")
		return
	}

	//删除7018记录
	wArtInfo, err := f.Ctx.Redis("m7018", 0)
	if err != nil {
		f.Error("get article info redis error:%s", err.Error())
		err = errors.New("get article info redis error")
		return
	}

	artKey := articleInfoKeyPrefix + args.Oid
	err = wArtInfo.HSet(artKey, "source", "")
	if err != nil {
		f.Error("set article info error:%s", err.Error())
		err = errors.New("set article info error")
		return
	}
	return
}

// GetStarFeed ...
func (f *Feed)GetStarFeed(args *StarFeedGetArgs, reply *StarFeedGetResult) error {
	data, err := f.getStarFeed(args)
	if err != nil {
		reply.Result = Result{Status: 1, Error: err.Error()}
		return err
	}
	reply.Result = Result{Status:0, Error:"success"}
	reply.Data = data
	return nil
}

func (f *Feed)getStarFeed(args *StarFeedGetArgs) (data []string, err error) {
	rStarFeed, err := f.Ctx.Redis("star_feed_r", 0)
	if err != nil {
		f.Error("get star feed redis error:%s", err.Error())
		err = errors.New("get star feed redis error")
		return
	}

	sfStop := f.Ctx.Cfg.Feed.StarFeedGetMaxNum - 1

	//获取视频流
	keyVideo := starFeedVideoKeyPrefix + args.FeedKey
	resVideo, err := rStarFeed.ZRevRangeWithoutScore(keyVideo, 0, sfStop)
	if err != nil {
		f.Error("get star feed video error:%s", err.Error())
		err = errors.New("get star feed video error")
		return
	}
	//获取非视频流
	keyNoVideo := starFeedNoVideokeyPrefix + args.FeedKey
	resNoVideo, err := rStarFeed.ZRevRangeWithoutScore(keyNoVideo, 0, sfStop)
	if err != nil {
		f.Error("get star feed no video error:%s", err.Error())
		err = errors.New("get star feed no video error")
		return
	}

	rHis, wHis, hisKey, err := f.getStarFeedHisRedis(args.Uid)
	if err != nil {
		f.Error("get star feed history redis error:%s", err.Error())
		err = errors.New("get star feed history redis error")
		return
	}
	//获取历史
	hisM, err := getHistory(rHis, hisKey)
	if err != nil {
		f.Error("get star feed history error:%s", err.Error())
		err = errors.New("get star feed history error")
		return
	}

	countVideo := f.Ctx.Cfg.Feed.StarFeedVideoCount

	//过滤历史的视频流
	dataVideo := make([]string, 0)
	for k := range resVideo {
		if len(dataVideo) >= countVideo {
			break
		}
		if _, ok := hisM[resVideo[k]]; !ok {
			dataVideo = append(dataVideo, resVideo[k])
		}
	}
	//过滤历史的非视频流,按照args.Num数量来控制,以防止视频文章不满足数量时,用非视频补全
	dataNoVideo := make([]string, 0)
	for k := range resNoVideo {
		if len(dataNoVideo) >= args.Num {
			break
		}
		if _, ok := hisM[resNoVideo[k]]; !ok {
			dataNoVideo = append(dataNoVideo, resNoVideo[k])
		}
	}
	data = make([]string, 0)
	indexVideo, indexNoVideo := 0, 0
	for i := 1; i <= args.Num; i++ {
		if i % 2 == 0 {
			if indexVideo < len(dataVideo) {
				data = append(data, dataVideo[indexVideo])
				indexVideo++
			} else if indexNoVideo < len(dataNoVideo) {
				data = append(data, dataNoVideo[indexNoVideo])
				indexNoVideo++
			}
		}
		if i % 2 == 1 {
			if indexNoVideo < len(dataNoVideo) {
				data = append(data, dataNoVideo[indexNoVideo])
				indexNoVideo++
			} else if indexVideo < len(dataVideo) {
				data = append(data, dataVideo[indexVideo])
				indexVideo++
			}
		}
	}

	//写历史
	for k := range data {
		if err = wHis.LPush(hisKey, data[k]); err != nil {
			f.Error("star feed set history error:%s", err.Error())
			err = errors.New("star feed set history error")
			return
		}
	}

	//version1
	/*
		data = make([]string, args.Num)
		var count int
		for _, oid := range resfeed {
			if _, ok := hisM[oid]; !ok {
				data[count] = oid
				count++
				if err = wHis.LPush(hisKey, oid); err != nil {
					f.Error("star feed set history error:%s", err.Error())
					err = errors.New("star feed set history error")
					return
				}

			}
			if count == args.Num {
				break
			}
		}


	*/


	/*
	if count < args.Num {
		//f.Error("star feed no enough data")
		//如果数目不足,出最热的n条
		data, err = rStarFeed.ZRevRangeWithoutScore(feedKey, 0, args.Num - 1)
		if err != nil {
			f.Error("get star feed default data error:%s", err.Error())
			err = errors.New("get star feed default data error")
			return
		}
	}
	*/

	//trim history
	hisStop := f.Ctx.Cfg.Feed.StarFeedHistoryNum - 1
	err = wHis.LTrim(hisKey, 0, hisStop)
	if err != nil {
		f.Error("ltrim star feed history error:%s", err.Error())
		err = errors.New("ltrim star feed history error")
		return
	}
	//set expire
	hisExpTime := f.Ctx.Cfg.Feed.StarFeedHistoryDay * 24 * 60 * 60
	err = wHis.Expire(hisKey, hisExpTime)
	if err != nil {
		f.Error("expire star feed history error:%s", err.Error())
		err = errors.New("expire star feed history error")
		return
	}
	return
}

func (f *Feed)getStarFeedHisRedis(uid int) (r, w *utils.Redis, hisKey string, err error) {

	if uid <= 0 {
		err = errors.New("invalid uid")
		return
	}

	var index int
	index = uid % starFeedHistoryRedisNumber

	r, err = f.Ctx.Redis("star_feed_his_r", index)
	w, err = f.Ctx.Redis("star_feed_his_w", index)
	hisKey = starFeedHistoryKeyPrefix + strconv.Itoa(uid)
	return
}

func getHistory(r *utils.Redis, hisKey string) (m map[string]int, err error) {
	resI, err := r.Do("LRANGE", hisKey, 0, -1)
	if err != nil {
		return
	}

	resS, ok := resI.([]interface{})
	if !ok {
		err = errors.New("convert error")
		return
	}
	//如果该用户还没有历史,则len(resS)==0
	m = make(map[string]int)
	for k := range resS {
		if item, ok := resS[k].([]byte); ok {
			m[string(item)] = 1
		}
	}

	return
}
