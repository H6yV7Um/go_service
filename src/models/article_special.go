package models

type ArticleSpecial struct {
	Id              int64
	Series_id       string
	Name            string
	Src             string
	Feed_src        string
	Desc            string
	Status          int
	Editor          string
	Style           int
	Type            int
	Bigdata_id      string
	Created_at      int
	Updated_at      int
}

func (a *ArticleSpecial) Format() map[string]interface{} {
	return map[string]interface{} {
		"id":         a.Id,
		"series_id":  a.Series_id,
		"name":       a.Name,
		"src":        a.Src,
		"feed_src":   a.Feed_src,
		"desc":       a.Desc,
		"status":     a.Status,
		"stype":      a.Style,
		"type":       a.Type,
		"bigdata_id": a.Bigdata_id,
		"article_type": "special",
	}
}

type ArticleSpecialGroup struct {
	Id           int64
	Name         string
	Spid         int
	Listorder    int
	Type         int
	Article_num  int
}

func (a *ArticleSpecialGroup) Format() map[string]interface{} {
	return map[string]interface{} {
		"id":          a.Id,
		"name":        a.Name,
		"listorder":   a.Listorder,
		"type":        a.Type,
		"article_num": a.Article_num,
	}
}

type ArticleSpecialContent struct {
	Id         int64
	Spid       int
	Oid        string
	Gid        int
	Listorder  int
}

func (a *ArticleSpecialContent) Format() map[string]interface{} {
	return map[string]interface{} {
		"id":          a.Id,
		"spid":        a.Spid,
		"oid":         a.Oid,
		"gid":         a.Gid,
		"listorder":   a.Listorder,
	}
}
