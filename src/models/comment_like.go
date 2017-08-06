package models

type CommentLike struct {
	Uid     		int64
	Cid     		string
	Ctime    		string
}

func (c *CommentLike) Format() map[string]interface{} {
	return map[string]interface{} {
		"uid": c.Uid,
		"cid": c.Cid,
		"ctime": c.Ctime,
	}
}

