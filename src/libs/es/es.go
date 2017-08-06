package es

import (
	"contexts"
	json"github.com/pquerna/ffjson/ffjson"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"strconv"
	"strings"
	"time"

	. "utils"
)

const (
	ES_URL       = "http://i.search.toutiao.weibo.cn:9200/topweibo/article/"
	ES_QUERY_URL = "http://i.search.toutiao.weibo.cn:9200/topweibo/article/_search"
)

type Es struct {
	ctx *contexts.Context
}
type EsQueryResult struct {
	Took      int  `json:"took"`
	Timed_out bool `json:"imed_out"`
	/*Shards struct {
	  Total int `json:"total"`
	  Successful int  `json:"successful"`
	  Failed int `json:"failed"`
	  } `json:"_shards"`*/
	Hits struct {
		Total     int     `json:"total"`
		Max_score float64 `json:"max_score"`
		EsHits    `json:"hits"`
	} `json:"hits"`
}
type EsHits []EsDoc
type EsSource struct {
	Oid           string   `json:"oid"`
	Created_time  int      `json:"created_time"`
	Time          int      `json:"time"`
	Pub_date      string   `json:"pub_date"`
	Create_date   string   `json:"create_date"`
	Tags          []string `json:"tags"`
	Flag          int      `json:"flag"`
	Last_edited   int      `json:"last_edited"`
	Category      string   `json:"category"`
	Title         string   `json:"title"`
	Content_score int      `json:"content_score"`
	Source        string   `json:"source"`
	Editor        string   `json:"editor"`
	Type          int      `json:"type"`
	Content       string   `json:"content"`
	Deleted       int      `json:"deleted"`
}
type EsDoc struct {
	Index   string   `json:"_index"`
	Types   string   `json:"_type"`
	Id      string   `json:"_id"`
	Version int      `json:"_version"`
	Found   bool     `json:"found"`
	Score   float64  `json:"_score"`
	Source  EsSource `json:"_source"`
}

type EsArgs struct {
	Oid    []string `msgpack:"oid"`
	Fields []string `msgpack:"fields"`
}

type EsMatch struct {
	Match map[string]interface{} `json:"match"`
}

type EsShould []interface{}
type EsRanges struct {
	Range map[string]EsRange `json:"range"`
}
type EsTerm struct {
	Term interface{} `json:term"`
}
type EsRange struct {
	Gt int64 `json:"gt"`
	Lt int64 `json:"lt"`
}
type EsBool struct {
	Bool interface{} `json:"bool"`
}
type EsMust struct {
	Must interface{} `json:"must"`
}

type EsQuery struct {
	//"from" : 0, "size" : 10,
	From int           `json:"from"`
	Size int           `json:"size"`
	Sort []interface{} `json:"sort"`
	/*	Sort struct {

		Time struct {
			Order string `json:"order"`
		} `json:"time"`
	} `json:"sort"`*/
	// "_source": ["account_number", "balance"]
	//Source []string `json:"_source"`
	Query struct {
		Bool struct {
			Should []interface{} `json:"should"`
			Must   []interface{} `json:"must"`
		} `json:"bool"`
	} `json:"query"`
}

func checkFields(fields []string) int {
	whitelist := [...]string{

		"a_id",
		"category",
		"content",
		"content_score",
		"create_date",
		"created_time",
		"deleted",
		"editor",
		"flag",
		"has_img",
		"has_video",
		"last_edited",
		"oid",
		"pub_date",
		"source",
		"tags",
		"time",
		"title",
		"type",
	}
	var n int = 0
	length := len(fields)
	if length == 0 {
		return 0
	}
	for i := 0; i < length; i++ {
		for j := 0; j < len(whitelist); j++ {
			if strings.EqualFold(fields[i], whitelist[j]) {
				n++
			}

		}

	}
	if n < length {
		return -1
	}
	return 1

}
func levelTransfor(level int) int64 {
	var content_score int64
	switch level {
	case 0:
		content_score = 1500
	case 1:
		content_score = 1000
	case 2:
		content_score = 500
	case 3:
		content_score = 200
	case 4:
		content_score = 100
	case 5:
		content_score = 90
	case 6:
		content_score = 40
	case 7:
		content_score = 30
	case 8:
		content_score = 20
	case 9:
		content_score = 15
	default:
		content_score = 10

	}
	return content_score
}
func (this *Es) Test(args *struct {
	S string `msgpack:"s"`
}, reply *string) error {
	log.Println("Enter Rest()")
	log.Println(args)
	*reply = "hello!" + args.S
	return nil
}

// S ...
func (ths *EsSource) S(s string) string {
	switch s {
	case "Pub_date":
		return ths.Pub_date
	case "Oid":
		return ths.Oid
	case "Time":
		return strconv.Itoa(ths.Time)
	default:
		return ""
	}

}

// Get ...
func Get(oid string) (EsSource, error) {
	var source EsSource
	url := ES_URL + oid
	response, err := http.Get(url)
	if err != nil {
		return source, errors.New("http err")
	}
	//defer log.Println(err)
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return source, errors.New("server error:" + strconv.Itoa(response.StatusCode))
	}
	body, _ := ioutil.ReadAll(response.Body)
	ob, err := decodeJSON(string(body))
	if err != nil {
		//log.Println(string(body))
		return source, err
		//return  errors.New("json decode error")
	}
	log.Println(ob.Found)
	if ob.Found == false {
		return source, errors.New("not found")
	}
	source = ob.Source
	/*  log.Println(reflect.TypeOf(source))*/
	log.Println(source)
	return source, nil

}

// Query ...
func Query(Kw string, from, size int, level int, percentage int) (EsQueryResult, error) {
	var rtn EsQueryResult
	var title EsMatch

	title.Match = map[string]interface{}{
		"title": map[string]interface{}{
			"query": Kw,
			//"boost":                9,
			"minimum_should_match": fmt.Sprintf("%d%%", percentage),
		},
	}
	var content EsMatch
	content.Match = map[string]interface{}{
		"content": map[string]interface{}{
			"query": Kw,
			//"boost": 0.5,
			"minimum_should_match": fmt.Sprintf("%d%%", percentage),
		},
	}
	var query EsQuery
	query.From = from
	query.Size = size
	query.Query.Bool.Should = []interface{}{title, content}
	currTimestamp := time.Now().Unix()
	sortTime := map[string]string{
		"time": "desc",
	}
	sort := make([]interface{}, 2)
	sort[0] = "_score"
	sort[1] = sortTime
	query.Sort = sort
	timeRanges := map[string]EsRange{
		"time": EsRange{
			Gt: currTimestamp - 3600*24*30,
			Lt: currTimestamp,
		},
	}
	scroreRanges := map[string]EsRange{
		"content_score": EsRange{
			Gt: levelTransfor(level),
			Lt: 1500,
		},
	}
	musts := make([]interface{}, 4)
	musts[0] = EsRanges{
		Range: timeRanges,
	}
	musts[1] = EsRanges{
		Range: scroreRanges,
	}
	musts[2] = EsBool{
		Bool: map[string]interface{}{
			"must": title,
		},
	}
	musts[3] = EsBool{
		Bool: map[string]interface{}{
			"must": map[string]interface{}{
				"term": map[string]int{
					"deleted": 0,
				},
			},
		},
	}
	/*	musts[3] = EsTerm{
		Term: map[string]int{
			"deleted": 0,
		},
	}*/
	query.Query.Bool.Must = musts

	bs, err := json.Marshal(query)
	if err != nil {

		WriteLog("error", "parse json error, query=%v ", query)
		return rtn, errors.New("can't parse ths query")
	}
	queryString := string(bs)
	WriteLog("error", "query %v ", queryString)
	client := &http.Client{}
	req, _ := http.NewRequest("POST", ES_QUERY_URL, strings.NewReader(queryString))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {

		WriteLog("error", "request es error, err=%v,response=%v ", err, req)
		return rtn, err
	}
	defer resp.Body.Close()
	var bodystr string
	if resp.StatusCode != 200 {
		WriteLog("error", "error, res=%v response=%v ", query, resp)
		return rtn, errors.New("server error:" + strconv.Itoa(resp.StatusCode))
	}
	body, _ := ioutil.ReadAll(resp.Body)
	bodystr = string(body)
	ob, err := decodeEsResultsJSON(bodystr)
	if err != nil {
		WriteLog("error", "parse response json error, response.body=%v ", query, bodystr)
		return rtn, err
	}
	return ob, nil
}

/**
 * 格式化返回数据，
 * 1.如果只传一个字段，则返回该字段
 * 2.如果返回多字段，则返回一个dict
 */
func (ths *EsSource) esFormate(fields []string) error {

	return nil

}
func decodeJSON(strJSON string) (EsDoc, error) {
	var esData EsDoc
	if err := json.Unmarshal([]byte(strJSON), &esData); err != nil {
		return EsDoc{}, err
	}
	return esData, nil
}

// DecodeEsResultsJson...
func decodeEsResultsJSON(strJSON string) (EsQueryResult, error) {
	var esData EsQueryResult
	if err := json.Unmarshal([]byte(strJSON), &esData); err != nil {
		return esData, err
	}
	return esData, nil
}
