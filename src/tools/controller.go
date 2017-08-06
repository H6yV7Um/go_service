package tools

import (
	"fmt"
	"reflect"
	"strings"
	"net/http"
	"html/template"
	"os"
	"log"
	"io/ioutil"
	"contexts"
	"encoding/json"
	"service"
	"unicode"
	"strconv"
)

type toolsController struct {
	w 		 http.ResponseWriter
	r 		 *http.Request
	m   	 string
	i 		 string
	Data	 map[string]interface{}
	hijacked bool
}

type menuItem struct {
	ID string
	Name 		string
	Icon        string
	URL 		string
	Active      bool
	HasChildren bool
	Children    []*menuItem
}

var rootPath string

func init() {
	rootPath, _ = os.Getwd()
}

func makeURL(m string, i string) string {
	return fmt.Sprintf("/tools?m=%s&i=%s", m, i)
}

func (t *toolsController) init() {
	t.r.ParseForm()
	t.Data = make(map[string]interface{})
	t.hijacked = false
}

func (t *toolsController) process() {
	m, i := t.getModelAndItem()
	methodName := fmt.Sprintf("%s%s", strings.Title(m), strings.Title(i))
	method, b := reflect.TypeOf(t).MethodByName(methodName)
	if !b {
		fmt.Printf("method:%s not found!\n", methodName)
		return
	}
	t.Data["buildVersion"], t.Data["buildGithash"], t.Data["buildDate"] = service.GetBuildInfo()
	method.Func.Call([]reflect.Value{reflect.ValueOf(t)})
}

func (t *toolsController) getModelAndItem() (m string, i string) {
	m = t.r.Form.Get("m")
	if m == "" {
		m = "rpc"
	}
	i = t.r.Form.Get("i")
	if i == "" {
		i = "index"
	}
	return
}

func (t *toolsController) getModelID() string {
	m, i := t.getModelAndItem()
	return fmt.Sprintf("%s_%s", m, i)
}

func (t *toolsController) getMenuItem(name string, icon string, m string, i string) *menuItem {
	currModel, currItem := t.getModelAndItem()
	return &menuItem{ID:fmt.Sprintf("%s_%s", m, i), Name:name, Icon:icon, URL:makeURL(m, i), Active:currModel==m && currItem==i}
}

// StaticResource 提供静态文件的访问
func StaticResource(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	requestType := "unknown"
	dot := strings.LastIndex(path, ".")
	if dot > 0 {
		requestType = path[dot:]
	}
	fmt.Printf("static path=%v, type=%v\n", path, requestType)
	switch requestType {
	case ".css":
		w.Header().Set("content-type", "text/css")
	case ".js":
		w.Header().Set("content-type", "text/javascript")
	case ".html", ".htm":
		w.Header().Set("content-type", "text/html")
	case ".jpg":
		w.Header().Set("content-type", "image/jpeg")
	case ".png":
		w.Header().Set("content-type", "image/png")
	default:
		w.Header().Set("content-type", "application/octet-stream")
	}
	fin, err := os.Open(rootPath + path)
	defer fin.Close()
	if err != nil {
		log.Printf("static resource:%v\n", err)
		w.Header().Set("content-type", "text/html")
		w.Write([]byte("File not found!"))
		return
	}
	fd, _ := ioutil.ReadAll(fin)
	w.Write(fd)
}

// HTTPHandler ...
func HTTPHandler(w http.ResponseWriter, r *http.Request) {
	if !contexts.GetContext().Cfg.System.TestServer {
		if !CheckLoginBasic(w, r) {
			return
		}
	}
	c := &toolsController{w:w, r:r}
	c.init()
	m, _ := c.getModelAndItem()
	c.process()
	if c.hijacked {
		return
	}
	c.Data["menu"] = []*menuItem{
		c.getMenuItem("RPC接口测试", "fa-cubes", "rpc", "index"),
		c.getMenuItem("API接口测试", "fa-cubes", "api", "index"),
		&menuItem{ID:"user", Name:"用户工具", Icon:"fa-user", URL:"#", Active:m=="user", HasChildren:true, Children:[]*menuItem{
			c.getMenuItem("清除用户数据", "fa-eraser", "user", "clear"),
		}},
		&menuItem{ID:"article", Name:"文章工具", Icon:"fa-file", URL:"#", Active:m=="article", HasChildren:true, Children:[]*menuItem{
			c.getMenuItem("清除文章缓存", "fa-eraser", "article", "clear"),
		}},
		c.getMenuItem("降级开关", "fa-cubes", "rpc", "downgrade"),
	}

	t, err := template.ParseFiles("views/" + c.getModelID() + ".tpl",
		"views/header.tpl",
		"views/main_header.tpl",
		"views/left_side.tpl",
		"views/footer.tpl",
	)
	if err != nil {
		fmt.Printf("parse template failed:%v, modelID=%s\n", err, c.getModelID())
	}
	t.Execute(w, c.Data)
}

// UserLogin ...
func (t *toolsController) UserLogin () {
	t.Data["Error"] = ""
}

// UserClear ...
func (t *toolsController) UserClear () {
	fmt.Printf("hello!!!\n")
}

// ArticleClear ...
func (t *toolsController) ArticleClear () {
	fmt.Printf("hello!!!\n")
}

type argType struct {
	Name 		string
	Type 		string
	Must        string
	Comment     string
	Size        string
}
type methodType struct {
	Name 		string
	Comment		string
	Args        []*argType
}

// RpcIndex ...
func (t *toolsController) RpcIndex () {
	ctx := contexts.GetContext()
	mList := make(map[string][]*methodType)
	for k, v := range ctx.RpcServiceMap {
		for m := 0; m < v.Typ.NumMethod(); m++ {
			method := v.Typ.Method(m)
			if method.PkgPath != "" {
				continue
			}
			fmt.Printf("method:%v, %v\n", method.PkgPath, method.Name)
			// TODO: Service类不作为rpc的服务类
			methodReserved := false
			for _, n := range([]string{"Debug", "Info", "Critical", "Warn", "Error"}) {
				if strings.Compare(n, method.Name) == 0 {
					fmt.Printf("%v=>%v\n", n, method.Name)
					methodReserved = true
					continue
				}
			}
			if methodReserved {
				continue
			}
			argsElem := method.Type.In(1).Elem()
			args := []*argType{}
			for i := 0; i < argsElem.NumField(); i++ {
				f := argsElem.Field(i)
				if f.Name == "CommonArgs" {
					continue
				}
				fType := ""
				switch f.Type.Kind() {
				case reflect.Int, reflect.Uint64, reflect.Int64:
					fType = "int"
				case reflect.Bool:
					fType = "bool"
				case reflect.Slice:
					elemType := f.Type.Elem().Kind()
					if elemType == reflect.String {
						fType = "[]string"
					} else if elemType == reflect.Int || elemType == reflect.Uint64 {
						fType = "[]int"
					}
				default:
					fType = "string"
				}
				must := "可选"
				if f.Tag.Get("must") == "1" {
					must = "必选"
				}
				size := f.Tag.Get("size")
				if size == "" {
					size = "20"
				}
				args = append(args, &argType{Name:f.Tag.Get("msgpack"), Type:fType, Must:must, Size:size, Comment:f.Tag.Get("comment")})
			}
			if _, exists := mList[k]; !exists {
				mList[k] = []*methodType{}
			}
			mList[k] = append(mList[k], &methodType{Name:method.Name, Args:args})
		}
	}
	t.Data["mList"] = mList
	s, _ := json.Marshal(mList)
	t.Data["mListJson"] = template.HTML(string(s))

	/*for k, v := range(mList) {
		fmt.Printf("%v:\n", k)
		for _, m := range(v) {
			fmt.Printf("  - %v:\n", m.Name)
			for _, a := range(m.Args) {
				fmt.Printf("      %v(%v):\n", a.Name, a.Type)
			}
		}
	}*/
}

type SwitchInfo struct {
	Name 		string
	Description string
	Checked     string
	Disabled    string
}

// RpcUpdateDowngrade ...
func (t *toolsController) RpcUpdatedowngrade () {
	t.hijacked = true
	nameParam := t.r.Form.Get("name")
	checkedParam := t.r.Form.Get("checked")
	if nameParam == "" || checkedParam == "" {
		t.writeError(-1, "invalid params")
		return
	}
	v, _ := strconv.ParseBool(checkedParam)
	ctx := contexts.GetContext()
	switchType := reflect.TypeOf(ctx.Cfg.Switch)
	switchValue := reflect.ValueOf(&ctx.Cfg.Switch)
	for i:=0; i<switchType.NumField(); i++ {
		switchField := switchType.Field(i)
		name := switchField.Name
		if name == nameParam {
			switchValue.Elem().Field(i).Set(reflect.ValueOf(v))
			t.writeError(0, "success")
			return
		}
	}
	t.writeError(-2, "switch not found")
}

// RpcDowngrade ...
func (t *toolsController) RpcDowngrade () {
	switches := []*SwitchInfo{}
	ctx := contexts.GetContext()
	switchType := reflect.TypeOf(ctx.Cfg.Switch)
	switchValue := reflect.ValueOf(ctx.Cfg.Switch)
	for i:=0; i<switchType.NumField(); i++ {
		switchField := switchType.Field(i)
		name := switchField.Name
		checked := switchValue.Field(i).Interface().(bool)
		checkedMsg := ""
		if checked {
			checkedMsg = "checked"
		}
		desc := switchField.Tag.Get("desc")
		disabled := switchField.Tag.Get("disabled")
		disabledMsg := ""
		if disabled == "1" {
			disabledMsg = "disabled"
			desc += "(不能在线修改)"
		}
		switches = append(switches, &SwitchInfo{
			Name: name,
			Description: desc,
			Checked: checkedMsg,
			Disabled: disabledMsg,
		})
	}
	
	t.Data["Switches"] = switches
}

func getApiArgType(f reflect.StructField) *argType {
	fType := ""
	switch f.Type.Kind() {
	case reflect.Int, reflect.Uint64, reflect.Int64:
		fType = "int"
	case reflect.Bool:
		fType = "bool"
	case reflect.Slice:
		elemType := f.Type.Elem().Kind()
		if elemType == reflect.String {
			fType = "[]string"
		} else if elemType == reflect.Int || elemType == reflect.Uint64 {
			fType = "[]int"
		}
	default:
		fType = "string"
	}
	must := "可选"
	if f.Tag.Get("must") == "1" {
		must = "必选"
	}
	size := f.Tag.Get("size")
	if size == "" {
		size = "20"
	}
	return &argType{Name:f.Name, Type:fType, Must:must, Size:size, Comment:f.Tag.Get("comment")}
}

func CamelCaseSplit(s string, toLower bool) []string {
	n := 0
	for i, rune := range s {
		if unicode.IsUpper(rune) {
			n++
		} else if i == 0 {
			n++
		}
	}

	a := make([]string, n)
	na := 0
	fieldStart := 0
	for i, rune := range s {
		if unicode.IsUpper(rune) {
			if i == 0 {
				continue
			}
			if toLower {
				a[na] = strings.ToLower(s[fieldStart:i])
			} else {
				a[na] = s[fieldStart:i]
			}
			na++
			fieldStart = i
		}
	}
	if fieldStart >= 0 && na < n {
		if toLower {
			a[na] = strings.ToLower(s[fieldStart:])
		} else {
			a[na] = s[fieldStart:]
		}
	}
	return a
}

// ApiIndex ...
func (t *toolsController) ApiIndex () {
	mList := make(map[string][]*methodType)
	for k, v := range apiTypeList {
		for _, api := range v {
			typ := reflect.TypeOf(api.api).Elem()
			uri := strings.Join(CamelCaseSplit(typ.Name(), true), "/")
			args := []*argType{}
			for i := 0; i < typ.NumField(); i++ {
				f := typ.Field(i)
				if f.Name == "common" {
					typCommon := reflect.TypeOf(&common{}).Elem()
					for j := 0; j < typCommon.NumField(); j++ {
						f = typCommon.Field(j)
						args = append(args, getApiArgType(f))
					}
					continue
				}
				args = append(args, getApiArgType(f))
			}
			if _, exists := mList[k]; !exists {
				mList[k] = []*methodType{}
			}
			mList[k] = append(mList[k], &methodType{Name:uri, Args:args, Comment:api.comment})
		}
	}
	t.Data["mList"] = mList
	s, _ := json.Marshal(mList)
	t.Data["mListJson"] = template.HTML(string(s))
}

func (t *toolsController) writeJson (v map[string]interface{}) {
	s, _ := json.Marshal(v)
	t.w.Header().Set("content-type", "text/json")
	t.w.Write(s)
}

func (t *toolsController) writeError (code int, msg string) {
	v := map[string]interface{}{"status": code, "error": msg}
	t.writeJson(v)
}

// ApiProxy ...
func (t *toolsController) ApiProxy () {
	t.hijacked = true
	url := t.r.Form.Get("url")
	if url == "" {
		t.writeError(1, "no url")
		return
	}
	ctx := contexts.GetContext()
	result, err := ctx.WeiboApiJson(url, map[string]interface{}{})
	if err != nil {
		t.writeError(1, fmt.Sprintf("call api faild, result=%v\n", result))
		return
	}
	t.writeJson(result)
}