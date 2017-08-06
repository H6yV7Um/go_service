package service

import (
	"fmt"
	"libs/es"
	"strings"
)

// Search ...
type Search struct {
	Type
}

// SearchResult ...
type SearchResult struct {
	Result Result
	Data   interface{}
}

// SearchArgs ...
type SearchArgs struct {
	Common CommonArgs `msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Oid    string     `msgpack:"oid" must:"1" comment:"oid"`
	Fields []string   `msgpack:"fields" must:"1" comment:"fields"`
}

// SearchQueryArgs ...
type SearchQueryArgs struct {
	Common     CommonArgs `msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Kw         string     `msgpack:"kw" must:"1" comment:"关键字"`
	From       int        `msgpack:"from" must:"0" comment:"起始序号" 默认0`
	Size       int        `msgpack:"size" must:"0" comment:"游标间隔" 默认20`
	Level      int        `msgpack:"level" must:"0" comment:"等级" 默认9`
	Percentage int        `msgpack:"percentage" must:"0" comment:"匹配度" 默认100`
	Fields     []string   `msgpack:"fields" must:"0" comment:"需要的字段"，根据默认返回一些基本字段，如果包含_all,则返回全部字段`
	Group      int `msgpack:"group" must:"0" comment:"是否露出"，仅对文章搜索有效`
}

// SearchQueryResult ...
type SearchQueryResult struct {
	Kw []string `msgpack:"kw"`
}
type StQuery struct {
	Kw string `msgpack:"kw"`
}

// Get ...
func (s *Search) Get(args *SearchArgs, reply *SearchResult) error {

	reply.Result = Result{Status: 0, Error: "success"}
	if args.Oid == "" {
		reply.Result = Result{Status: 500, Error: "oid is null"}
		return nil
	}
	source, err := es.Get(args.Oid)
	if err != nil {
		reply.Result = Result{Status: 501, Error: fmt.Sprintf("%s", err)}
	}
	reply.Data = source
	if len(args.Fields) == 1 {
		reply.Data = source.S(args.Fields[0])
	}
	return nil
}

// Query ...
func (s *Search) Query(args *SearchQueryArgs, reply *SearchResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	if args.Kw == "" {
		reply.Result = Result{Status: 500, Error: "Kw is null"}
		return nil
	}
	size := 20
	if args.Size > 0 {
		size = args.Size
	}
	level := 9
	if args.Level > 0 {
		level = args.Level
	}
	percentage := 10
	if args.Percentage > 0 {
		percentage = args.Percentage
	}
	Kw := args.Kw
	s.Info("search begin, kw=%v ", Kw)
	//Kw string, from, size int, level int, percentage int
	r, err := es.Query(Kw, args.From, size, level, percentage)
	if err != nil {
		s.Error("request es error, err=%v ", err)
		reply.Result = Result{Status: 502, Error: fmt.Sprintf("%s", err)}
		return nil
	}
	hits := r.Hits.EsHits
	len := len(hits)
	sli := []interface{}{}
	for i := 0; i < len; i++ {
		sli = append(sli, map[string]interface{}{
			"oid":          hits[i].Source.Oid,
			"time":         hits[i].Source.Time,
			"created_time": hits[i].Source.Created_time,
			"title":        hits[i].Source.Title,
			"pub_date":     hits[i].Source.Pub_date,
			"deleted":      hits[i].Source.Deleted,
		})
	}
	results := map[string]interface{}{
		"took":    r.Took,
		"total":   r.Hits.Total,
		"kw":      Kw,
		"results": sli,
	}

	s.Info("search end, kw=%v ,took=%v,total=%v,results=%v", Kw, results["took"], results["total"], results)
	reply.Data = results
	return nil
}

// ColumnGet ...
func (s *Search) ColumnGet(args *SearchArgs, reply *SearchResult) error {
	if args.Oid == "" {
		reply.Result = Result{Status: 500, Error: "oid is null"}
		return nil
	}

	reply.Result = Result{Status: 0, Error: "success"}
	data, err := es.ColumnGet(args.Oid, args.Fields)
	if err != nil {
		s.Error("request elasticsearch error, err=%v ", err)
		reply.Result = Result{Status: 502, Error: fmt.Sprintf("%s", err)}
		return nil
	}
	reply.Data = data.Source

	return nil

}

// QuerySource ...
func (s *Search) QuerySource(args *SearchQueryArgs, reply *SearchResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	if args.Kw == "" {
		reply.Result = Result{Status: 500, Error: "Kw is null"}
		return nil
	}
	size := 20
	if args.Size > 0 {
		size = args.Size
	}
	level := 9
	if args.Level > 0 {
		level = args.Level
	}
	percentage := 10
	if args.Percentage > 0 {
		percentage = args.Percentage
	}
	Kw := args.Kw
	fields, err := es.SetFields(args.Fields, es.SourceWhitelistFields, es.SourceDefaultFields)
	if err != nil {
		reply.Result = Result{Status: 501, Error: fmt.Sprintf("%s", err)}
		return nil
	}
	s.Info("search begin, kw=%v ", Kw)
	//Kw string, from, size int, level int, percentage int
	r, err := es.QuerySource(Kw, fields, args.From, size, level, percentage)
	if err != nil {
		s.Error("request es error, err=%v ", err)
		reply.Result = Result{Status: 502, Error: fmt.Sprintf("%s", err)}
		return nil
	}

	hits := r.Hits.Hits
	len := len(hits)
	sli := []interface{}{}
	for i := 0; i < len; i++ {
		sli = append(sli,
			hits[i].Source,
		)
	}
	results := map[string]interface{}{
		"took":    r.Took,
		"total":   r.Hits.Total,
		"kw":      Kw,
		"results": sli,
	}

	s.Info("search end, kw=%v ,took=%v,total=%v,results=%v", Kw, results["took"], results["total"], results)
	reply.Data = results
	return nil
}

// QueryTag ...
func (s *Search) QueryTag(args *SearchQueryArgs, reply *SearchResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	if len(args.Kw) < 1 {
		reply.Result = Result{Status: 500, Error: "Kw is null"}
		return nil
	}
	size := 20
	if args.Size > 0 {
		size = args.Size
	}
	level := 9
	if args.Level > 0 {
		level = args.Level
	}
	percentage := 10
	if args.Percentage > 0 {
		percentage = args.Percentage
	}
	Kw := args.Kw
	fields, err := es.SetFields(args.Fields, es.TagWhitelistFields, es.TagDefaultFields)
	if err != nil {
		reply.Result = Result{Status: 501, Error: fmt.Sprintf("%s", err)}
		return nil
	}
	s.Info("search begin, kw=%v ", Kw)
	//Kw string, from, size int, level int, percentage int
	r, err := es.QueryTag(Kw, fields, args.From, size, level, percentage)
	if err != nil {
		s.Error("request es error, err=%v ", err)
		reply.Result = Result{Status: 502, Error: fmt.Sprintf("%s", err)}
		return nil
	}

	hits := r.Hits.Hits
	len := len(hits)
	sli := []interface{}{}
	for i := 0; i < len; i++ {
		sli = append(sli,
			hits[i].Source,
		)
	}
	results := map[string]interface{}{
		"took":    r.Took,
		"total":   r.Hits.Total,
		"kw":      Kw,
		"results": sli,
	}

	s.Info("search end, kw=%v ,took=%v,total=%v,results=%v", Kw, results["took"], results["total"], results)
	reply.Data = results
	return nil
}

// QueryArticle ...
func (s *Search) QueryArticle(args *SearchQueryArgs, reply *SearchResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	if len(args.Kw) < 1 {
		reply.Result = Result{Status: 500, Error: "Kw is null"}
		return nil
	}
	size := 20
	if args.Size > 0 {
		size = args.Size
	}
	level := 4
	if args.Level > 0 {
		level = args.Level
	}
	Kw := args.Kw
	percentage := 100
	if len(Kw) > 24 {
		percentage = 75
	}

	if args.Percentage > 0 {
		percentage = args.Percentage
	}

	fields, err := es.SetFields(args.Fields, es.ArticleWhitelistFields, es.ArticleDefaultFields)

	if err != nil {
		reply.Result = Result{Status: 501, Error: fmt.Sprintf("%s", err)}
		return nil
	}
	var state int = 0
	for i := 0; i < len(fields); i++ {
		if strings.EqualFold("title", fields[i]) {
			state = state | 1
			continue
		}
		if strings.EqualFold("content_score", fields[i]) {
			state = state | 2
		}

	}

	if state & 1 == 0 {
		fields = append(fields, "title")
	}
	if state & 2 == 0 {
		fields = append(fields, "content_score")
	}
	var group int = 1
	if args.Group > 1 {
		group = -1
	}
	from := 0
	if args.From > 0 {
		from = args.From - 1
	}

	if from > 0 && from < 20 {
		reply.Data = map[string]interface{}{
			"took":    args.From,
			"total":   0,
			"kw":      Kw,
			"results": []map[string]interface{}{},
		}
		return nil
	}

	s.Info("search begin, kw=%v ", Kw)
	//Kw string, from, size int, level int, percentage int
	r, err := es.QueryArticle(Kw, fields, from, size * 2, level, percentage, group)
	if err != nil {
		s.Error("request es error, err=%v ", err)
		reply.Result = Result{Status: 502, Error: fmt.Sprintf("%s", err)}
		return nil
	}

	hits := r.Hits.Hits
	length := len(hits)
	sli := []map[string]interface{}{}
	for i := 0; i < length; i++ {
		if _, ok := hits[i].Source["a_id"]; ok {
			hits[i].Source["iid"] = hits[i].Source["a_id"]
		}
		sli = append(sli,
			hits[i].Source,
		)
	}

	sli = es.RemoveRepeatArticle(sli, size)
	results := map[string]interface{}{
		"took":    r.Took,
		"total":   r.Hits.Total,
		"kw":      Kw,
		"results": sli,
	}
	s.Info("search end, kw=%v ,took=%v,total=%v,results=%v", Kw, results["took"], results["total"], results)
	reply.Data = results
	return nil
}

// Stquery ...
func (s *Search) StQuery(args *SearchQueryArgs, reply *SearchResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	if args.Kw == "" {
		reply.Result = Result{Status: 500, Error: "Kw is null"}
		return nil
	}
	size := 20
	if args.Size > 0 {
		size = args.Size
	}
	level := 4
	percentage := 100
	Kw := args.Kw
	s.Info("search begin, kw=%v ,level=%v", Kw, level)
	r, err := es.QueryDistinctSt(Kw, args.From, 20, percentage)
	if err != nil {
		s.Error("request es error, err=%v ", err)
		reply.Result = Result{Status: 502, Error: fmt.Sprintf("%s", err)}
		return nil
	}
	hits := r.Hits.Hits
	lens := len(hits)
	sli := []map[string]interface{}{}
	for i := 0; i < lens; i++ {
		hits[i].Source["type"] = hits[i].Types
		if strings.EqualFold("tags", hits[i].Types) && s.getFeedCount(hits[i].Id) < 5 {
			continue
		}
		//hits[i].Source["aricle_num"] = s.getFeedCount(hits[i].Source["id"].(string))
		sli = append(sli,
			hits[i].Source,
		)
	}
	sli = es.RemoveRepeatTag(sli, 10)
	//articles
	//fields, err := es.SetFields(args.Fields, es.ArticleWhitelistFields, es.ArticleDefaultFields)
	fields := []string{"oid", "a_id", "title", "content_score", "created_time"}
	if err != nil {
		reply.Result = Result{Status: 501, Error: fmt.Sprintf("%s", err)}
		return nil
	}
	s.Info("articles search begin, kw=%v ", Kw)
	if len(Kw) > 24 {
		percentage = 75
	}
	atcs, err := es.QueryArticle(Kw, fields, args.From, size * 2, level, percentage, 1)
	if err != nil {
		s.Error("articles request es error, err=%v ", err)
		reply.Result = Result{Status: 502, Error: fmt.Sprintf("%s", err)}
		return nil
	}
	hits = atcs.Hits.Hits
	lens = len(hits)
	asli := []map[string]interface{}{}
	for i := 0; i < lens; i++ {
		asli = append(asli,
			hits[i].Source,
		)
	}
	asli = es.RemoveRepeatArticle(asli, size)
	results := map[string]interface{}{
		"took":        r.Took,
		"articles":    asli,
		"source_tags": sli,
	}
	s.Info("search end, kw=%v ,took=%v,total=%v,results=%v", Kw, results["took"], results["total"], results)
	reply.Data = results
	return nil
}
// 标签流长度
func (t *Search) getFeedCount(tagID string) int64 {
	if tagID == "" {
		return 0
	}
	feedTagCountCacheKey := feedTagCountCachePrefix + tagID
	if info, found := t.Ctx.Cache.Get(feedTagCountCacheKey); found {
		return info.(int64)
	}

	r, err := t.Ctx.Redis("feed")
	if err != nil {
		return 0
	}
	key := feedTagPrefix + tagID
	count, _ := r.ZCard(key)
	t.Ctx.Cache.Set(feedTagCountCacheKey, count, feedTagCountCacheExpireTime)
	t.Info("feed count for %s:%d", tagID, count)
	return count
}
