package es

import (
	//"encoding/json"
	json"github.com/pquerna/ffjson/ffjson"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
	"fmt"
	"regexp"

	. "utils"
)

const (
	ES_ARTICLE_URL = "http://i.search.toutiao.weibo.cn:9200/topweibo/article/"
	ES_ARTICLE_QUERY_URL = "http://i.search.toutiao.weibo.cn:9200/topweibo/article/_search"
)

type EsArticleDoc struct {
	Index   string                 `json:"_index"`
	Types   string                 `json:"_type"`
	Id      string                 `json:"_id"`
	Version int                    `json:"_version"`
	Found   bool                   `json:"found"`
	Score   float64                `json:"_score"`
	Source  map[string]interface{} `json:"_source"`
}
type EsArticleHits struct {
	Hits []EsArticleDoc `json:"hits"`
}
type EsQueryArticleResult struct {
	Took      int  `json:"took"`
	Timed_out bool `json:"imed_out"`

	Hits      struct {
				  Total     int            `json:"total"`
				  Max_score float64        `json:"max_score"`
				  Hits      []EsArticleDoc `json:"hits"`
			  } `json:"hits"`
}
type EsArticleShould []interface{}
type EsArticleMatch []interface{}
type EsArticleMust []interface{}
type EsArticleBool struct {
	EsArticleShould `json:"should"`
	EsArticleMust   `json:"must"`
}
type EsArticleQ struct {
	Bool EsArticleBool `json:"bool"`
}
type EsArticleQuery struct {
	From   int           `json:"from"`
	Size   int           `json:"size"`
	Sort   []interface{} `json:"sort"`
	// "_source": ["account_number", "balance"]
	Source []string   `json:"_source"`
	Query  EsArticleQ `json:"query"`
}

var ArticleWhitelistFields = []string{
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
	"group",
}
var ArticleDefaultFields = []string{"oid", "title", "created_time", "time", "a_id"}

func QueryArticle(Kw string, fields []string, from, size int, level int, percentage int, group int) (EsQueryArticleResult, error) {
	var rtn EsQueryArticleResult
	var matchType string
	if len(Kw) < 10 {
		matchType = "phrase"
	} else {
		matchType = "boolean"
	}
	match := EsArticleMatch{
		map[string]interface{}{
			"match": map[string]interface{}{

				"title": map[string]string{
					"query":                Kw,
					"minimum_should_match": fmt.Sprintf("%d%%", percentage),
					"type":matchType,

				},
			},
		},
	}
	should := EsArticleShould{
		//match,
	}
	currTimestamp := time.Now().Unix()

	must := []interface{}{
		map[string]interface{}{
			"term": map[string]uint8{
				"deleted": 0,
			},
		},
		map[string]interface{}{

			"range": map[string]interface{}{
				"time": map[string]int64{
					"gt": currTimestamp - 3600 * 24 * 90,
				},
			},
		},
		map[string]interface{}{

			"range": map[string]interface{}{
				"content_score": map[string]int64{
					"gte": levelTransfor(level),
				},
			},
		},
		map[string]interface{}{
			"terms":map[string][8]int{
				"flag":[8]int{
					0, 10, 12, 30, 31, 32, 33, 9999,
				},
			},
		},
		match,
	}
	if group == 0 || group == 1 {

		must = append(must, map[string]interface{}{
			"term":map[string]int{
				"group":group,
			},
		})
	}

	q := EsArticleQ{
		Bool: EsArticleBool{
			should,
			must,
		},
	}
	query := EsArticleQuery{
		From: from,
		Size: size,
		Sort: []interface{}{
			map[string]string{
				"hot":"desc",
			},
			map[string]string{
				"content_score":"desc",
			},

			map[string]string{
				"time": "desc",
			},
			"_score",
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
	client := TimeoutClient(time.Duration(3000 * time.Millisecond), time.Duration(3000 * time.Millisecond))
	req, _ := http.NewRequest("POST", ES_ARTICLE_QUERY_URL, strings.NewReader(queryString))
	//req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("")
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
func clearTitle(title string) string {
	r, _ := regexp.Compile("[\u4e00-\u9fa5]")
	var rtn []string
	for _, v := range title {

		if r.MatchString(string(v)) {
			rtn = append(rtn, string(v))
		}

	}
	return strings.Join(rtn, "")
}
func RemoveRepeatArticle(hits []map[string]interface{}, size int) []map[string]interface{} {
	var mTitles = map[string]int{}
	var sli []map[string]interface{}
	for i := 0; i < len(hits); i++ {
		if len(sli) == size {
			break
		}
		title := clearTitle(hits[i]["title"].(string))
		if n, exits := mTitles[title]; exits {
			if hits[i]["content_score"].(float64) > sli[n]["content_score"].(float64) {
				sli[n] = hits[i]

			}
			continue
		} else {
			mTitles[title] = len(sli)
		}
		//hits[i]["title"] = title
		sli = append(sli, hits[i])
	}

	return sli

}
