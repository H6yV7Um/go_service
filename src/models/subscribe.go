package models

type Subscribe struct {
	Uid			int64
	Imei		string
	Ctime		int
	Type		int
	Target_id	string
}

func (s *Subscribe) Format() map[string]interface{} {
	return map[string]interface{} {
		"uid": s.Uid,
		"imei": s.Imei,
		"ctime": s.Ctime,
		"type": s.Type,
		"target_id": s.Target_id,
	}
}
