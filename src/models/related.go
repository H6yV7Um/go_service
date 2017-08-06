package models

type Related struct {
	Id				string
	Oid           	string
	Mid           	string
	Type          	int
	Img_flag      	int
	Time          	int
	Created_time  	int
	Source        	string
	Content_score 	int
	Hashcode      	uint64
	Tag           	string
	Tags          	string
	Op_tag        	int
	Titlehash     	uint64
	Title_seg     	string
	Lda_topic     	string
	Topics        	string
	Status        	uint64
	Display       	string
	Groupid       	string
	Image_hash    	string
	Source_id     	int
}

func (r *Related) Format() map[string]interface{} {
	return map[string]interface{} {
		"id": r.Id,
		"oid": r.Oid,
		"mid": r.Mid,
		"type": r.Type,
		"img_flag": r.Img_flag,
		"time": r.Time,
		"created_time": r.Created_time,
		"source": r.Source,
		"content_score": r.Content_score,
		"hashcode": r.Hashcode,
		"tag": r.Tag,
		"tags": r.Tags,
		"op_tag": r.Op_tag,
		"titlehash": r.Titlehash,
		"title_seg": r.Title_seg,
		"lda_topic": r.Lda_topic,
		"topics": r.Topics,
		"status": r.Status,
		"display": r.Display,
		"groupid": r.Groupid,
		"image_hash": r.Image_hash,
		"source_id": r.Source_id,
	}
}
