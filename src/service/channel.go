package service

import (
	"errors"
	"utils"
)

// Channel ...
type Channel struct {
	Type
}

// ChannelGetTopFeedArgs ...
type ChannelGetTopFeedArgs struct {
	Common    CommonArgs `msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	ChannelID string     `msgpack:"cid" must:"0" comment:"频道ID"`
	Count     int        `msgpack:"count" must:"0" comment:"文章个数"`
}

// ChannelTopFeedResult ...
type ChannelTopFeedResult struct {
	Result Result
	Data   []ChannelTopFeedReplyData
}

// ChannelTopFeedReplyData ...
type ChannelTopFeedReplyData struct {
	ChannelID   string   `msppack:"channelid"`
	ChannelName string   `msgpack:"channelname"`
	Oids        []string `msgpack:"oids"`
}

const (
	ctfKeyPrefix = "feed_suggest_"
	topFeedLimit = 2000000000
)

var mapIDToName map[string]string

func init() {
	mapIDToName = make(map[string]string)
	mapIDToName["0"] = "人工推荐位"
	mapIDToName["1"] = "热点推荐位"
	mapIDToName["n_user"] = "新用户推荐位"
	mapIDToName["2"] = "娱乐"
	mapIDToName["7"] = "社会"
	mapIDToName["5"] = "科技"
	mapIDToName["3"] = "体育"
	mapIDToName["4"] = "军事"
	mapIDToName["6"] = "财经"
	mapIDToName["10021"] = "国际"
	mapIDToName["10001"] = "数码"
	mapIDToName["10"] = "汽车"
	mapIDToName["12"] = "时尚"
	mapIDToName["10014"] = "游戏"
	mapIDToName["10018"] = "电影"
	mapIDToName["9"] = "房产"
	mapIDToName["10005"] = "动漫"
	mapIDToName["10010"] = "情感"
	mapIDToName["10007"] = "美女"
	mapIDToName["13"] = "旅游"
	mapIDToName["10013"] = "历史"
	mapIDToName["10016"] = "养生"
	mapIDToName["10017"] = "摄影"
	mapIDToName["8"] = "教育"
	mapIDToName["11"] = "健康"
	mapIDToName["10004"] = "育儿"
	mapIDToName["10009"] = "健身"
	mapIDToName["10002"] = "家居"
	mapIDToName["10003"] = "神总结"
	mapIDToName["10006"] = "星座"
	mapIDToName["10008"] = "宠物"
	mapIDToName["10011"] = "搞笑"
	mapIDToName["10012"] = "美食"
	mapIDToName["10019"] = "涨知识"
	mapIDToName["10020"] = "科学"
	mapIDToName["10015"] = "视频"
	mapIDToName["toutiao"] = "今日头条"
	mapIDToName["10025"] = "奥运"
}

// GetTopFeed ...
func (c *Channel) GetTopFeed(args *ChannelGetTopFeedArgs, reply *ChannelTopFeedResult) error {
	reply.Data = make([]ChannelTopFeedReplyData, 0)
	
	rChannelTopFeed, err := c.Ctx.Redis("channel_topfeed_r", 0)
	if err != nil {
		c.Error("get ChannelTopFeed redis error:%s", err.Error())
		err = errors.New("get ChannelTopFeed redis error")
		return err
	}
	
	count := 2
	if 0 < args.Count {
		count = args.Count
	}
	
	reply.Result = Result{Status:0, Error:"success"}
	if args.ChannelID == "" {
		c.getAllChannelTopFeed(rChannelTopFeed, count, &reply.Data) // 忽略全频道置顶过程中的错误
	} else {
		err = c.getChannelTopFeed(rChannelTopFeed, args.ChannelID, count, &reply.Data)
		if err != nil {
			reply.Result = Result{Status: 1, Error: err.Error()}
		}
	}
	
	return nil
}

func (c *Channel) getChannelTopFeed(redis *utils.Redis, channelID string, count int, data *[]ChannelTopFeedReplyData) error {
	var channelName string
	if _, ok := mapIDToName[channelID]; !ok {
		c.Error("get ChannelTopFeed failed, invalid channelID:%s", channelID)
		return errors.New("invalid parameter")
	}
	channelName = mapIDToName[channelID]
	
	topFeedKey := ctfKeyPrefix + channelID
	resTopFeed, scores, err := redis.ZRevRange(topFeedKey, 0, count - 1)
	if err != nil {
		c.Error("get ChannelTopFeed feed error:%s, channel:%s", err.Error(), channelName)
		return errors.New("get ChannelTopFeed feed error")
	}
	
	if len(resTopFeed) != 0 {
		var replyData ChannelTopFeedReplyData
		replyData.ChannelID   = channelID
		replyData.ChannelName = channelName
		
		i := 0
		for i < len(resTopFeed) {
			score := scores[i]
			if score < topFeedLimit { // 置顶文章要求大于20亿
				break
			}
			replyData.Oids = append(replyData.Oids, resTopFeed[i])
			i = i + 1
		}
		
		if i > 0 {
			*data = append(*data, replyData)
		}
	}
	
	return nil
}

// Get ...
func (c *Channel) getAllChannelTopFeed(redis *utils.Redis, count int, data *[]ChannelTopFeedReplyData) {
	for channelID := range mapIDToName {
		c.getChannelTopFeed(redis, channelID, count, data)
	}
	
	return
}
