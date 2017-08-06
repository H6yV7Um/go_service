package service

import (
	"strings"
	"fmt"
	"models"
	"time"
)


func (a *Article) getArticleSpecials(spid []string) (result ArticlesType) {
	result = make(ArticlesType)
	
	specials := a.getSpecialsBySpid(spid)
	if len(specials) == 0 {
		return
	}
	
	for _, special := range specials {
		if special["status"] == 0 || special["status"] == 1 {
			a.Info("%v is off-line", special["series_id"])
			continue
		}
		
		series_id := special["series_id"].(string)
		result[series_id] = make(map[string]interface{})
		result[series_id]["id"]           = series_id
		result[series_id]["special"]      = special
		result[series_id]["article_type"] = "special"
		
		groups   := a.getSpecialGroupsBySpid(special["id"].(int64))
		if len(groups) == 0 {
			continue
		}
		
		for i := 0; i < len(groups); i++ {
			contentsA := a.getSpecialContentsBySpid(special["id"].(int64), groups[i]["id"].(int64))
			
			if special["bigdata_id"].(string) != "" && groups[i]["type"].(int) == 1 {
				contentsB := a.getSpecialContentsByBigdata(special["bigdata_id"].(string), special["id"].(int64), groups[i]["id"].(int64))
				if len(contentsB) != 0 {
					contentsA = append(contentsA, contentsB...)
				}
			}
			
			if len(contentsA) == 0 {
				continue
			}
			
			groups[i]["contents"] = contentsA
		}
		
		result[series_id]["groups"]   = groups
	}
	
	return
}

func (a *Article) getSpecialsBySpid(spid []string) (result []map[string]interface{}) {
	result = make([]map[string]interface{}, 0)
	
	oidArg := []string{}
	for _, id := range spid {
		oidArg = append(oidArg, " \"" + id + "\"")
	}
	
	var rows []interface{}
	sql := fmt.Sprintf("select * from specials where series_id in (%s)", strings.Join(oidArg, ","))
	rows, err := a.Ctx.GetDBInstance("articlespecial", "article_r").Query(sql)
	if rows == nil || err != nil {
		a.Error("query failed:%v", err)
		return
	}
	
	if len(rows) == 0 {
		a.Info("get empty special, spid = %v", spid)
		return
	}
	
	for i := 0; i < len(rows); i++ {
		special := (rows[i]).(*models.ArticleSpecial)
		result   = append(result, special.Format())
	}
	
	return
}

// 暂不使用
func (a *Article) getSpecialBySpid(spid string) (result map[string]interface{}) {
	result = make(map[string]interface{})

	var rows []interface{}
	sql := fmt.Sprintf("select * from specials where series_id = '%s'", spid)
	rows, err := a.Ctx.GetDBInstance("articlespecial", "article_r").Query(sql)
	if rows == nil || err != nil {
		a.Error("query failed:%v", err)
		return
	}
	
	if len(rows) == 0 {
		a.Info("get empty special, spid = %d", spid)
		return
	}
	
	special := (rows[0]).(*models.ArticleSpecial)
	result   = special.Format()
	
	return
}

func (a *Article) getSpecialGroupsBySpid(spid int64) (result []map[string]interface{}) {
	result = make([]map[string]interface{}, 0)
	
	var rows []interface{}
	sql := fmt.Sprintf("select * from special_groups where spid = '%d' order by listorder desc", spid)
	rows, err := a.Ctx.GetDBInstance("articlespecialgroup", "article_r").Query(sql)
	if rows == nil || err != nil {
		a.Error("query failed:%v", err)
		return
	}
	
	if len(rows) == 0 {
		a.Info("get empty special group, spid = %d", spid)
		return
	}
	
	for i := 0; i < len(rows); i++ {
		asg := (rows[i]).(*models.ArticleSpecialGroup)
		result = append(result, asg.Format())
	}
	
	return
}

func (a *Article) getSpecialContentsBySpid(spid int64, gid int64) (result []map[string]interface{}) {
	result = make([]map[string]interface{}, 0)
	
	var rows []interface{}
	sql := fmt.Sprintf("select * from special_contents where spid = '%d' and gid = '%d' order by listorder asc", spid, gid)
	rows, err := a.Ctx.GetDBInstance("articlespecialcontent", "article_r").Query(sql)
	if rows == nil || err != nil {
		a.Error("query failed:%v", err)
		return
	}
	
	if len(rows) == 0 {
		a.Info("get empty special contents, spid = %d", spid)
		return
	}
	
	for i := 0; i < len(rows); i++ {
		asc := (rows[i]).(*models.ArticleSpecialContent)
		result = append(result, asc.Format())
	}
	
	return
}

func (a *Article) getSpecialContentsByBigdata(bigdataId string, spid int64, gid int64) (contents []map[string]interface{}){
	contents = make([]map[string]interface{}, 0)
	
	params := map[string]interface{}{
		"count": 20,
		"delete_duplicate": "true",
		"start_time": time.Now().Unix() - 86400 * 2,
		"order": 2,
		"series_id": bigdataId,
	}
	
	ret, err := a.Ctx.WeiboApiJson("i2.api.weibo.com/darwin/application/article/series_article_list.json", params)
	if err != nil {
		a.Info("special article failed to connect weibo_api_json:err:%s, bigdata_id:%v", err, bigdataId)
		return
	}
	
	if ret == nil || ret["results"] == nil {
		a.Info("special article from bigdata is empty")
		return
	}
	
	
	for _, item := range ret["results"].([]interface{}) {
		content := map[string]interface{} {
			"big": "big",
			"spid": spid,
			"oid": item.(map[string]interface{})["article_id"].(string),
			"gid": gid,
		}
		
		contents = append(contents, content)
	}
	
	return
}
