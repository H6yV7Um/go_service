package models

import (
	"strings"
	"utils"
)

const (
	articleModelVersion = 20161108
)

type Articles struct {
	Id				string
	Uid     		int64
	Oid     		string
	Time    		int64
	Updtime         int64
	Type            int
	Grade           int
	Card_url        string
	Category        string
	Source          string
	Title           string
	Url             string
	Original_url    string
	Tags            string
	Images          string
	Content         string
	Flag            int
	Mid             string
	Hash            string
	Cover           string
	Lda_topic       string
	Content_score   int
	Editor          string
	Last_edited     int
	Feeds           string
	Cms_options     string
	Created_time    int64
	Groupid         int
	Summary         string
	Gid         	int64
	Display_id      int64
	Top_groupid     string
}

func (a *Articles) Format() map[string]interface{} {
	articleType := "article"
	if a.Type == 14 {
		// 直播
		articleType = "live"
	} else if a.Type == 15 {
		// 图集
		articleType = "slide"
	} else if strings.Contains(a.Tags, "1042015:tw_video") {
		// 视频
		articleType = "video"
	} else if utils.IsCardArticle(a.Id) {
		// 组card
		articleType = "card"
	}
	// 专题
	// 广告
	return map[string]interface{} {
		"id": a.Id,
		"uid": a.Uid,
		"oid": a.Oid,
		"time": a.Time,
		"updtime": a.Updtime,
		"type": a.Type,
		"category": a.Category,
		"source": a.Source,
		"title": a.Title,
		"url": a.Url,
		"original_url": a.Original_url,
		"tags": a.Tags,
		"images": a.Images,
		"content": a.Content,
		"flag": a.Flag,
		"mid": a.Mid,
		"cover": a.Cover,
		"created_time": a.Created_time,
		"abstract": a.Summary,
		"cms_options": a.Cms_options,
		"gid": a.Gid,
		"display_id": a.Display_id,
		"top_groupid": a.Top_groupid,
		"article_type": articleType,
	}
}
