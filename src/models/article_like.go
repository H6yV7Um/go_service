package models

type ArticleLike struct {
	Uid     		int64
	Oid     		string
	Ctime    		string
}

func (a *ArticleLike) Format() map[string]interface{} {
	return map[string]interface{} {
		"uid": a.Uid,
		"oid": a.Oid,
		"ctime": a.Ctime,
	}
}

