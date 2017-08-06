package service

import "errors"

const (
	tagFeedTimeKeyPrefix = "feedu_t:"
	tagFeedHotKeyPrefix = "feedu:"
)

type TagFeedGetArgs struct {
	CommonArgs
	TagID  string `msgpack:"tag_id" must:"1" comment:"tag_id"`
	Type   int `msgpack:"type" must:"1" comment:"type"`
	Offset int `msgpack:"offset" must:"1" comment:"offset"`
	Num    int `msgpack:"num" must:"1" comment:"num"`
}

type TagFeedGetResult struct {
	Result
	Data []string
}

type TagFeedCountArgs struct {
	CommonArgs
	TagID string `msgpack:"tag_id" must:"1" comment:"tag_id"`
	Type  int `msgpack:"type" must:"1" comment:"type"`
}

type TagFeedCountResult struct {
	Result
	Data int64
}

func (f *Feed)GetTagFeed(args *TagFeedGetArgs, reply *TagFeedGetResult) error {
	data, err := f.getTagFeed(args)
	if err != nil {
		reply.Result = Result{Status:1, Error:err.Error()}
		return err
	}
	reply.Result = Result{Status:0, Error:"success"}
	reply.Data = data
	return nil
}

func (f *Feed)CountTagFeed(args *TagFeedCountArgs, reply *TagFeedCountResult) error {
	data, err := f.countTagFeed(args)
	if err != nil {
		reply.Result = Result{Status:1, Error:err.Error()}
		return err
	}
	reply.Result = Result{Status:0, Error:"success"}
	reply.Data = data
	return nil
}

func (f *Feed)getTagFeed(args *TagFeedGetArgs) (data []string, err error) {
	rTagFeed, err := f.Ctx.Redis("star_feed_r", 0)
	if err != nil {
		f.Error("get tag feed redis error:%s", err.Error())
		err = errors.New("get tag feed redis error")
		return
	}
	var feedKeyPre string
	if args.Type == 0 {
		feedKeyPre = tagFeedTimeKeyPrefix
	} else if args.Type == 1 {
		feedKeyPre = tagFeedHotKeyPrefix
	}

	feedKey := feedKeyPre + args.TagID
	start := args.Offset
	stop := args.Offset + args.Num - 1

	data, err = rTagFeed.ZRevRangeWithoutScore(feedKey, start, stop)
	if err != nil {
		f.Error("get tag feed error:%s", err.Error())
		err = errors.New("get tag feed error")
		return
	}
	return
}

func (f *Feed)countTagFeed(args *TagFeedCountArgs) (data int64, err error) {
	rTagFeed, err := f.Ctx.Redis("star_feed_r", 0)
	if err != nil {
		f.Error("get tag feed redis error:%s", err.Error())
		err = errors.New("get tag feed redis error")
		return
	}
	var feedKeyPre string
	if args.Type == 0 {
		feedKeyPre = tagFeedTimeKeyPrefix
	} else if args.Type == 1 {
		feedKeyPre = tagFeedHotKeyPrefix
	}

	feedKey := feedKeyPre + args.TagID
	data, err = rTagFeed.ZCard(feedKey)
	if err != nil {
		f.Error("count tag feed error:%s", err.Error())
		err = errors.New("count tag feed error")
		return
	}
	return
}

