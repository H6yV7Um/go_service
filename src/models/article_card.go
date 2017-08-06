package models

type ArticleCard struct {
	Id				string
	Title           string
	Bg_img          string
	Bg_img_large    string
	Guide_text      string
	Guide_scheme    string
	Groups      	string
	Created_time    int
	Updtime         int
	Editor          string
	State    		int
}

func (a *ArticleCard) Format() map[string]interface{} {
	return map[string]interface{} {
		"id": a.Id,
		"title": a.Title,
		"bg_img": a.Bg_img,
		"bg_img_large": a.Bg_img_large,
		"guide_text": a.Guide_text,
		"guide_scheme": a.Guide_scheme,
		"groups": a.Groups,
		"created_time": a.Created_time,
		"updtime": a.Updtime,
		"editor": a.Editor,
		"state": a.State,
		"article_type": "card",
	}
}
