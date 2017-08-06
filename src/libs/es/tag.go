package es

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"net/http"

	"strconv"
	"strings"

	. "utils"
)

const (
	ES_TAG_URL       = "http://i.search.toutiao.weibo.cn:9200/tag_idx/tags/"
	ES_TAG_QUERY_URL = "http://i.search.toutiao.weibo.cn:9200/tag_idx/tags/_search"
)

type EsTagDoc struct {
	Index   string                 `json:"_index"`
	Types   string                 `json:"_type"`
	Id      string                 `json:"_id"`
	Version int                    `json:"_version"`
	Found   bool                   `json:"found"`
	Score   float64                `json:"_score"`
	Source  map[string]interface{} `json:"_source"`
}
type EsTagHits struct {
	Hits []EsTagDoc `json:"hits"`
}
type EsQueryTagResult struct {
	Took      int  `json:"took"`
	Timed_out bool `json:"imed_out"`

	Hits struct {
		Total     int        `json:"total"`
		Max_score float64    `json:"max_score"`
		Hits      []EsTagDoc `json:"hits"`
	} `json:"hits"`
}
type EsTagShould []interface{}
type EsTagMatch []interface{}
type EsTagMust []interface{}

// EsTagBool ...
type EsTagBool struct {
	EsTagShould `json:"should"`
	EsTagMust   `json:"must"`
}

// EsTagQ ...
type EsTagQ struct {
	Bool EsTagBool `json:"bool"`
}

//EsTagQuery ...
type EsTagQuery struct {
	From int           `json:"from"`
	Size int           `json:"size"`
	Sort []interface{} `json:"sort"`
	// "_source": ["account_number", "balance"]
	Source []string `json:"_source"`
	Query  EsTagQ   `json:"query"`
}

// TagWhitelistFields ...
var TagWhitelistFields = []string{"id", "name", "follows", "status", "time", "updtime", "articles"}

// TagDefaultFields ...
var TagDefaultFields = []string{"id", "name", "follows"}

// QueryTag ...
func QueryTag(Kw string, fields []string, from, size int, level int, percentage int) (EsQueryTagResult, error) {
	var rtn EsQueryTagResult
	match := EsTagMatch{
		map[string]interface{}{
			"match": map[string]interface{}{

				"name": map[string]string{
					"query":                Kw,
					"minimum_should_match": fmt.Sprintf("%d%%", percentage),
				},
			},
		},
	}
	should := EsTagShould{
	//match,
	}
	must := EsTagMust{
		map[string]interface{}{
			"term": map[string]uint8{
				"status": 200,
			},
		},
		match,
	}
	q := EsTagQ{
		Bool: EsTagBool{
			should,
			must,
		},
	}
	query := EsTagQuery{
		From: from,
		Size: size,
		Sort: []interface{}{
			"_score",
			map[string]string{
				"articles": "desc",
			},
		},
		Source: fields,
		Query:  q,
	}
	js, err := json.Marshal(query)
	if err != nil {

		WriteLog("error", "parse json error, query=%v ", query)
		return rtn, errors.New("can't parse this query")
	}
	queryString := string(js)
	WriteLog("info", "request query=%v ", queryString)
	client := &http.Client{}
	req, _ := http.NewRequest("POST", ES_TAG_QUERY_URL, strings.NewReader(queryString))
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
