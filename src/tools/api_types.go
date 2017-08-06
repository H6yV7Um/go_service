package tools

type common struct {
	cur_uid		int		`must:"0" comment:"当前用户id，没有传0" size:"40"`
	imei		string	`must:"0" comment:"唯一设备号" size:"40"`
	from		string	`must:"0" comment:"From值" size:"40"`
	wm			string	`must:"0" comment:"Wm值" size:"40"`
	ip			string	`must:"0" comment:"IP" size:"40"`
	time		string	`must:"0" comment:"时间戳" size:"40"`
	sign		string	`must:"0" comment:"认证码" size:"40"`
}

type ArticlesHome_timeline struct {
	common
	cnt		int		`must:"0" comment:"返回条数" size:"40"`
	load	string	`must:"0" comment:"刷新feed类型: new, more" size:"40"`
	cate_id	int		`must:"0" comment:"feed分类id (默认为0)" size:"40"`
	max_id	int		`must:"0" comment:"客户端已有feed的最大id" size:"40"`
	min_id	int		`must:"0" comment:"客户端已有feed的最小id" size:"40"`
	filter	int		`must:"0" comment:"结果是否过滤 (默认为过滤)" size:"40"`
	puicode	int		`must:"0" comment:"上一级uicode编码，值同行为日志的说明" size:"40"`
}

type ArticlesShow struct {
	common
	object_id		string	`must:"1" comment:"文章ID" size:"40"`
	activity_keys	string	`must:"0" comment:"好友评论key数组,格式参考示例" size:"40"`
	puicode	int		`must:"0" comment:"上一级uicode编码，值同行为日志的说明" size:"40"`
}

type apiType struct {
	comment		string
	api			interface{}
}

var apiTypeList map[string][]apiType

func init() {
	apiTypeList = make(map[string][]apiType)
	apiTypeList["articles"] = []apiType{
		apiType{"获取Feed", &ArticlesHome_timeline{}},
		apiType{"获取文章正文", &ArticlesShow{}},
	}
	apiTypeList["users"] = []apiType{
	}
	apiTypeList["search"] = []apiType{
	}
	apiTypeList["favor"] = []apiType{
	}
}
