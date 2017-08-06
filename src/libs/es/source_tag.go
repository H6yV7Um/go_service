package es

import (
	json"github.com/pquerna/ffjson/ffjson"
	"errors"
	"fmt"
	"io/ioutil"
	"time"
	"net/http"
	"strconv"
	"strings"
	. "utils"
)

var blockTags = map[string]bool{
	"1042015:tw_video":true,
	"1042015:tw_lweibo":true,
	"1042015:tw_twarticle":true,
	"1042015:tw_twarticle10":true,
	"1042015:tw_twarticle11":true,
	"1042015:tw_twarticle12":true,
	"1042015:tw_twarticle13":true,
	"1042015:tw_quality_video":true,
}

type EsSTQuery struct {
	From  int           `json:"from"`
	Size  int           `json:"size"`
	Sort  []interface{} `json:"sort"`
	// "_source": ["account_number", "balance"]
	Query EsArticleQ `json:"query"`
}
// QueryDistinctSt 去掉空格，过滤同名的tag标签，取文章数多的那个
func QueryDistinctSt(Kw string, from, size int, percentage int) (EsQueryArticleResult, error) {
	var rtn EsQueryArticleResult
	query := map[string]interface{}{
		"match": map[string]interface{}{
			"name": map[string]string{
				"query":                Kw,
				"minimum_should_match": fmt.Sprintf("%d%%", percentage),
			},
		},
	}
	dsl := QueryDsl{
		From: from,
		Size: size,
		Sort: []interface{}{
			map[string]string{
				"_score":"desc",
			},
			map[string]string{
				"follows": "desc",
			},
		},
		Source: []string{},
		Query:query,
	}
	js, err := json.Marshal(dsl)
	if err != nil {
		WriteLog("error", "create query json falsed, query=%v ", dsl)
		return rtn, errors.New("can't create query json")
	}
	queryString := string(js)
	WriteLog("info", "request query=%v ", queryString)
	client := TimeoutClient(time.Duration(1000 * time.Millisecond), time.Duration(1000 * time.Millisecond))
	req, _ := http.NewRequest("POST", ES_ST_QUERY_URL, strings.NewReader(queryString))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		WriteLog("error", "request es error, err=%v,response=%v ", err, req)
		return rtn, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		WriteLog("error", "error, res=%v response=%v ", query, resp)
		return rtn, errors.New("server error:" + strconv.Itoa(resp.StatusCode))
	}
	body, _ := ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &rtn); err != nil {
		return rtn, err
	}
	return rtn, nil
}
func RemoveRepeatTag(hits []map[string]interface{}, size int) []map[string]interface{} {
	var mName = map[string]int{}
	var sli []map[string]interface{}
	for i := 0; i < len(hits); i++ {
		if len(sli) == size {
			break
		}
		if strings.EqualFold(hits[i]["type"].(string), "source") {
			sli = append(sli, hits[i])
			continue
		}
		name := strings.TrimSpace(hits[i]["name"].(string))
		if n, exits := mName[name]; exits {
			if hits[i]["follows"].(float64) > sli[n]["follows"].(float64) {
				sli[n] = hits[i]
			}
			continue
		} else {
			mName[name] = len(sli)
		}
		hits[i]["name"] = name
		object_id := hits[i]["id"].(string)
		if _ , exits := blockTags[object_id ]; exits {
			continue
		}
		sli = append(sli, hits[i])
	}

	return sli

}
func QueryST(Kw string, from, size int, percentage int) (EsQueryArticleResult, error) {
	var rtn EsQueryArticleResult
	match := EsArticleMatch{
		map[string]interface{}{
			"match": map[string]interface{}{

				"name": map[string]string{
					"query":                Kw,
					"minimum_should_match": fmt.Sprintf("%d%%", percentage),
				},
			},
		},
	}
	should := EsArticleShould{
		//match,
	}
	must := EsArticleMust{
		map[string]interface{}{
			"term": map[string]uint8{
				"status": 200,
			},
		},

		match,
	}
	q := EsArticleQ{
		Bool: EsArticleBool{
			should,
			must,
		},
	}
	query := EsSTQuery{
		From: from,
		Size: size,
		Sort: []interface{}{
			"_score",
			map[string]string{
				"follows": "desc",
			},
		},
		Query: q,
	}
	js, err := json.Marshal(query)
	if err != nil {
		WriteLog("error", "parse json error, query=%v ", query)
		return rtn, errors.New("can't parse this query")
	}
	queryString := string(js)
	WriteLog("info", "request query=%v ", queryString)
	client := TimeoutClient(time.Duration(500 * time.Millisecond), time.Duration(500 * time.Millisecond))
	req, _ := http.NewRequest("POST", ES_ST_QUERY_URL, strings.NewReader(queryString))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {

		WriteLog("error", "request es error, err=%v,response=%v ", err, req)
		return rtn, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		WriteLog("error", "error, res=%v response=%v ", query, resp)
		return rtn, errors.New("server error:" + strconv.Itoa(resp.StatusCode))
	}
	body, _ := ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &rtn); err != nil {
		return rtn, err
	}
	return rtn, nil

}
