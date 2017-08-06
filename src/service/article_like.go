package service

import "utils"


// GetCommentCounterArgs 获取计数值
type GetArticleLikeNumArgs struct {
	Common CommonArgs   `msppack:"common" must:"0" comment:"公共参数" size:"40"`
	Oid    string       `msgpack:"oid" must:"1" comment:"文章ID"`
}

// CommentCounterResult 计数值
type ArticleLikeNumResult struct {
	Result Result
	Data   int
}

// CheckLike 是否赞
func (a *Article) CheckLike(args *CheckLikeArgs, reply *CheckLikeResult) error {
	likeService := Like{a.Type}
	return likeService.CheckLikeBatch(args, reply, likeTypeArticle)
}

// Like 赞
func (a *Article) Like(args *LikeArgs, reply *CommonResult) error {
	likeService := Like{a.Type}
	err := likeService.Like(args, reply, likeTypeArticle)
	if err == nil {
		likeService.handleLikeCount(args.Oid, args.Liked, likeTypeArticle)
	}
	
	return err
}

// GetLikeNum 获取赞数
func (a *Article) GetLikeNum(args *GetArticleLikeNumArgs, reply *ArticleLikeNumResult) error {
	if !utils.CheckOid(args.Oid) {
		reply.Result = Result{Status: 1, Error: "invalid argument"}
		return nil
	}
	likeService := Like{a.Type}
	reply.Data   = likeService.GetLikeNum(args.Oid, likeTypeArticle)
	if reply.Data == -1 {
		reply.Result = Result{Status:0, Error:"operator error"}
		return nil
	}
	reply.Result = Result{Status:0, Error:"success"}
	return nil
}