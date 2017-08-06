package es

import (
	"errors"
	"strings"
)

const (
	ARTICLE_URL = "http://i.search.toutiao.weibo.cn:9200/topweibo/article/"
	ARTICLE_QUERY_URL = "http://i.search.toutiao.weibo.cn:9200/topweibo/article/_search"
	ES_ST_URL = "http://i.search.toutiao.weibo.cn:9200/source_tag"
	ES_ST_QUERY_URL = "http://i.search.toutiao.weibo.cn:9200/source_tag/_search"
)

type QueryDsl struct {
	From   int           `json:"from"`
	Size   int           `json:"size"`
	Sort   []interface{} `json:"sort"`
	Source []string `json:"_source"`
	Query  interface{} `json:"query"`
}
// SetFields ...
func SetFields(fields []string, whitelist []string, defaults []string) ([]string, error) {
	if len(defaults) < 1 {
		return nil, errors.New("defaults fields is empty")
	}
	if len(whitelist) < 1 {
		return nil, errors.New("whitelist fields is empty")
	}

	length := len(fields)

	if length == 0 {
		return defaults, nil
	}
	var n int = 0
	for i := 0; i < length; i++ {
		if strings.EqualFold(fields[i], "_all") {
			return whitelist, nil
		}
		for j := 0; j < len(whitelist); j++ {
			if strings.EqualFold(fields[i], whitelist[j]) {
				n++
			}
		}
	}
	if n < length {
		return nil, errors.New("someone field is error")
	}
	return fields, nil

}