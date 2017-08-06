package very_simple_json

import (
	//"reflect"
	"testing"
)

func TestDecodeEncode(t *testing.T) {
	//tags := "[{\"category_id\":\"1042015:tagCategory_060\",\"category_name\":\"社会时政\",\"display_name\":\"希拉里\",\"object_id\":\"1042015:socialPoliticsPeople_100000578\",\"object_type\":\"socialPoliticsPeople\",\"weight\":1},{\"category_id\":\"root\",\"display_name\":\"美国\",\"object_id\":\"1042015:city_14000001\",\"object_type\":\"city\",\"weight\":1},{\"display_name\":\"社会时政\",\"object_id\":\"1042015:tagCategory_060\",\"object_type\":\"tagCategory\",\"weight\":1},{\"category_id\":\"1042015:tagCategory_060\",\"category_name\":\"时事\",\"display_name\":\"国际资讯\",\"object_id\":\"1042015:abilityTag_604\",\"object_type\":\"abilityTag\",\"weight\":0.899993896484375},{\"category_id\":\"1042015:tagCategory_060\",\"category_name\":\"时事\",\"display_name\":\"美国大选开锣投票\",\"object_id\":\"1042015:hotTag_4f8c2a36de05ae7971572abacc3acf71\",\"object_type\":\"hotTag\",\"weight\":0.500091552734375}]"
	//tag := "{\"category_id\":\"1042015:tagCategory_060\",\"category_name\":\"社会时政\",\"display_name\":\"希拉里\",\"object_id\":\"1042015:socialPoliticsPeople_100000578\",\"object_type\":\"socialPoliticsPeople\",\"weight\":1}"
	//text := "[{\"width\": 300, \"des_url\": \"http://ww2.sinaimg.cn/large/005M94J9jw1f9lu5tz2qzj308c08c3yo.jpg\", \"height\": 300}, {\"width\": 300, \"des_url\": \"http://ww2.sinaimg.cn/nmw690/005M94J9jw1f9lu5tz2qzj308c08c3yo.jpg\", \"height\": 300}, {\"width\": 240, \"des_url\": \"http://ww2.sinaimg.cn/crop.0.30.300.225.240/005M94J9jw1f9lu5tz2qzj308c08c3yo.jpg\", \"height\": 180}, {\"width\": 322, \"des_url\": \"http://ww4.sinaimg.cn/nmw690/005M94J9jw1f9lu5uqj17j308y0csaa9.jpg\", \"height\": 460}, {\"width\": 322, \"des_url\": \"http://ww4.sinaimg.cn/large/005M94J9jw1f9lu5uqj17j308y0csaa9.jpg\", \"height\": 460}"
	//text := `[{"display_name": "\u5218\u607a\u5a01", "weight": 1, "object_type": "moviePerson", "object_id": "1042015:moviePerson_208289", "category_id": "1042015:tagCategory_050", "category_name": "\u5a31\u4e50\u660e\u661f"}, {"display_name": "\u5185\u5730\u5267", "weight": 1, "object_type": "abilityTag", "object_id": "1042015:abilityTag_223", "category_id": "1042015:tagCategory_034", "category_name": "\u7535\u89c6\u5267"}, {"object_type": "moviePerson", "display_name": "\u738b\u9e25", "object_id": "1042015:moviePerson_bf88c2767341cc5d1e4347c1e4b2fac7", "weight": 1}, {"display_name": "\u7405\u740a\u699c", "weight": 1, "object_type": "tv", "object_id": "1042015:tv_10000000598", "category_id": "1042015:tagCategory_034", "category_name": "\u7535\u89c6\u5267"}, {"display_name": "\u6e2f\u53f0\u660e\u661f", "weight": 1, "object_type": "abilityTag", "object_id": "1042015:abilityTag_1009", "category_id": "1042015:tagCategory_050", "category_name": "\u5a31\u4e50\u660e\u661f"}, {"display_name": "\u6b4c\u624b", "weight": 1, "object_type": "abilityTag", "object_id": "1042015:abilityTag_478", "category_id": "1042015:tagCategory_050", "category_name": "\u5a31\u4e50\u660e\u661f"}, {"object_type": "tagCategory", "display_name": "\u7535\u89c6\u5267", "weight": 1, "object_id": "1042015:tagCategory_034"}, {"display_name": "\u660e\u661f\u516b\u5366", "weight": 0.9499969482421875, "object_type": "abilityTag", "object_id": "1042015:abilityTag_10845", "category_id": "1042015:tagCategory_050", "category_name": "\u5a31\u4e50\u660e\u661f"}, {"display_name": "\u4f2a\u88c5\u8005", "weight": 0.89013671875, "object_type": "tv", "object_id": "1042015:tv_861", "category_id": "1042015:tagCategory_034", "category_name": "\u7535\u89c6\u5267"}, {"display_name": "\u5076\u50cf\u5267", "weight": 0.75, "object_type": "abilityTag", "object_id": "1042015:abilityTag_667", "category_id": "1042015:tagCategory_034", "category_name": "\u7535\u89c6\u5267"}, {"display_name": "\u53e4\u88c5\u5267", "weight": 0.75, "object_type": "abilityTag", "object_id": "1042015:abilityTag_10052", "category_id": "1042015:tagCategory_034", "category_name": "\u7535\u89c6\u5267"}, {"display_name": "\u6218\u4e89\u5267", "weight": 0.649993896484375, "object_type": "abilityTag", "object_id": "1042015:abilityTag_658", "category_id": "1042015:tagCategory_034", "category_name": "\u7535\u89c6\u5267"}, {"display_name": "\u60ac\u7591\u7535\u5f71", "weight": 0.649993896484375, "object_type": "abilityTag", "object_id": "1042015:abilityTag_10007", "category_id": "1042015:tagCategory_033", "category_name": "\u7535\u5f71"}, {"object_type": "worksTag", "display_name": "\u5168\u660e\u661f\u63a2", "object_id": "1042015:worksTag_25da5c828f606414e334cc51cf1c1869", "weight": 0.54998779296875}, {"object_type": "worksTag", "display_name": "\u5468\u672b\u7236\u6bcd", "object_id": "1042015:worksTag_0fd3301d169b42974c644f805b681f64", "weight": 0.54998779296875}, {"object_type": "worksTag", "display_name": "\u9556\u884c\u5929\u4e0b\u524d\u4f20", "object_id": "1042015:worksTag_11a2aaee5419fb634ec9bbad2e09507a", "weight": 0.54998779296875}, {"display_name": "\u771f\u76f8", "weight": 0.498992919921875, "object_type": "tv", "object_id": "1042015:tv_10000000491", "category_id": "1042015:tagCategory_034", "category_name": "\u7535\u89c6\u5267"}, {"display_name": "\u5b89\u5fbd\u536b\u89c6", "weight": 0.3619232177734375, "object_type": "tvStation", "object_id": "1042015:tvStation_219", "category_id": "1042015:tagCategory_034", "category_name": "\u7535\u89c6\u5267"}, {"display_name": "\u6e56\u5357\u536b\u89c6", "weight": 0.357513427734375, "object_type": "tvStation", "object_id": "1042015:tvStation_315", "category_id": "1042015:tagCategory_034", "category_name": "\u7535\u89c6\u5267"}, {"object_type": "show", "display_name": "\u660e\u661f\u5927\u4fa6\u63a2", "object_id": "1042015:show_433f1faf0a7d11bd52f21c500e49348e", "weight": 0.35540771484375}, {"display_name": "\u4f55\u7085", "weight": 0.35394287109375, "object_type": "moviePerson", "object_id": "1042015:moviePerson_302749", "category_id": "1042015:tagCategory_050", "category_name": "\u5a31\u4e50\u660e\u661f"}, {"display_name": "\u6492\u8d1d\u5b81", "weight": 0.3534088134765625, "object_type": "moviePerson", "object_id": "1042015:moviePerson_35a857e4f72e1a049bde1a08ceaed336", "category_id": "1042015:tagCategory_050", "category_name": "\u5a31\u4e50\u660e\u661f"}, {"display_name": "\u5434\u6620\u6d01", "weight": 0.3526763916015625, "object_type": "moviePerson", "object_id": "1042015:moviePerson_303320", "category_id": "1042015:tagCategory_050", "category_name": "\u5a31\u4e50\u660e\u661f"}, {"display_name": "\u5f20\u840c", "weight": 0.345062255859375, "object_type": "moviePerson", "object_id": "1042015:moviePerson_265229", "category_id": "1042015:tagCategory_050", "category_name": "\u5a31\u4e50\u660e\u661f"}, {"object_type": "", "display_name": "\u8292\u679cTV", "object_id": "1042015:company_1286050", "weight": 0.1567840576171875}, {"object_type": "twarticle_tag", "display_name": "\u5934\u6761\u6587\u7ae0", "weight": 0, "object_id": "1042015:tw_twarticle"}, {"object_type": "twarticle_tag", "display_name": "\u5934\u6761\u6587\u7ae010", "weight": 0, "object_id": "1042015:tw_twarticle10"}]`
	text := "[]"
	d := NewSimpleJsonDecoder([]byte(text))
	result, err := d.Decode()
	t.Logf("%#v\n", result)
	if err != nil {
		t.Fatalf("parse json failed:%v\n", err)
	}
	
	/*if err != nil {
		t.Fatalf("parse json failed:%v\n", err)
	} else {
	layout := map[string]interface{} {
		"width": reflect.Int64,
		"des_url": reflect.String,
		"height": reflect.Int64,
	}
		jsonResult, err := ToJson(result, layout)
		if err != nil {
			t.Fatalf("tojson failed:%v\n", err)
		} else {
			d := NewSimpleJsonDecoder([]byte(jsonResult))
			result, err = d.Decode()
			if err != nil {
				t.Fatalf("parse json failed:%v\n", err)
			}
			jsonResult, err = ToJson(result, layout)
			if err != nil {
				t.Fatalf("tojson failed:%v\n", err)
			}
		}
	}*/
}

