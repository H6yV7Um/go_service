package es

import (
	json"github.com/pquerna/ffjson/ffjson"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	. "utils"
	"time"
)

const (
	ES_SOURCE_URL = "http://i.search.toutiao.weibo.cn:9200/source_idx/source/"
	ES_SOURCE_QUERY_URL = "http://i.search.toutiao.weibo.cn:9200/source_idx/source/_search"
)

type EsSourceDoc struct {
	Index   string                 `json:"_index"`
	Types   string                 `json:"_type"`
	Id      string                 `json:"_id"`
	Version int                    `json:"_version"`
	Found   bool                   `json:"found"`
	Score   float64                `json:"_score"`
	Source  map[string]interface{} `json:"_source"`
}
type EsSourceHits struct {
	Hits []EsSourceDoc `json:"hits"`
}
type EsQuerySourceResult struct {
	Took      int  `json:"took"`
	Timed_out bool `json:"imed_out"`

	Hits      struct {
			  Total     int           `json:"total"`
			  Max_score float64       `json:"max_score"`
			  Hits      []EsSourceDoc `json:"hits"`
		  } `json:"hits"`
}
type EsSourceShould []interface{}
type EsSourceMatch []interface{}
type EsSourceMust []interface{}
type EsSourceBool struct {
	EsSourceShould `json:"should"`
	EsSourceMust   `json:"must"`
}

// EsSourceQ ...
type EsSourceQ struct {
	Bool EsSourceBool `json:"bool"`
}

// EsSourceQuery ...
type EsSourceQuery struct {
	From   int           `json:"from"`
	Size   int           `json:"size"`
	Sort   []interface{} `json:"sort"`
	// "_source": ["account_number", "balance"]
	Source []string  `json:"_source"`
	Query  EsSourceQ `json:"query"`
}

// SourceWhitelistFields ...
var SourceWhitelistFields = []string{"uid", "name", "follows", "status", "time", "updtime"}

// SourceDefaultFields ...
var SourceDefaultFields = []string{"uid", "name", "follows"}



// QuerySource...
func QuerySource(Kw string, fields []string, from, size int, level int, percentage int) (EsQuerySourceResult, error) {
	var rtn EsQuerySourceResult
	match := EsSourceMatch{
		map[string]interface{}{
			"match": map[string]interface{}{

				"name": map[string]string{
					"query":                Kw,
					"minimum_should_match": fmt.Sprintf("%d%%", percentage),
				},
			},
		},
	}
	should := EsSourceShould{
		//match,
	}
	must := EsSourceMust{
		map[string]interface{}{
			"term": map[string]uint8{
				"status": 200,
			},
		},
		match,
	}
	q := EsSourceQ{
		Bool: EsSourceBool{
			should,
			must,
		},
	}
	query := EsSourceQuery{
		From: from,
		Size: size,
		Sort: []interface{}{
			"_score",
			map[string]string{
				"follows": "desc",
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
	client := TimeoutClient(time.Duration(500 * time.Millisecond), time.Duration(500 * time.Millisecond))
	req, _ := http.NewRequest("POST", ES_SOURCE_QUERY_URL, strings.NewReader(queryString))
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
