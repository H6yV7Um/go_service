package models

type PushArticles struct {
	Id			string
	Uid			int64
	Mid  		string
	Oid  		string
	Url  		string
	Status		int
	Time		int
	Grade		int
	Title  		string
	Abstract 	string
	Category 	string
	Labels  	string
	Source  	string
	Updtime  	string
	Content 	string
	Type		int
	Tags  		string
	Push_time 	int
	Push_rule 	string
	Push_type 	int
	Push_count 	int
	Push_ok 	int
	Push_click 	int
}

func (p *PushArticles) Format() map[string]interface{} {
	return map[string]interface{} {
		"id": p.Id,
		"uid": p.Uid,
		"mid": p.Mid,
		"oid": p.Oid,
		"url": p.Url,
		"status": p.Status,
		"time": p.Time,
		"grade": p.Grade,
		"title": p.Title,
		"abstract": p.Abstract,
		"category": p.Category,
		"labels": p.Labels,
		"source": p.Source,
		"updtime": p.Updtime,
		"content": p.Content,
		"type": p.Type,
		"tags": p.Tags,
		"push_time": p.Push_time,
		"push_rule": p.Push_rule,
		"push_type": p.Push_type,
		"push_count": p.Push_count,
		"push_ok": p.Push_ok,
		"push_click": p.Push_click,
	}
}
