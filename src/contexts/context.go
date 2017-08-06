package contexts

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"libs/cache"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
	"utils"
	"errors"
	"libs/mysql"
	"models"
)

// RpcServiceStats ...
type RpcServiceStats struct {
	Count              int64
	SuccCount          int64
	QpsUpdateTime      int64
	Acc                float64
	Latency            float64
	RecentLatency      float64
	RecentLatencyCount int64
	RecentCount        int64
	Qps                float64
}

// RpcService ...
type RpcService struct {
	Rcvr  reflect.Value
	Typ   reflect.Type
	mutex sync.Mutex
	Stats map[string]*RpcServiceStats
}

// Context ...
type Context struct {
	Alias           map[string]*Alias
	Storage         map[string]*Storage
	Cfg             *Config
	RpcServiceMap   map[string]*RpcService
	Services        map[string]interface{}
	Id              int
	mutex           sync.Mutex
	Cache           *cache.Cache
	Caches          map[string]*cache.Cache
	HttpClient      *http.Client
	HttpRetryClient *http.Client
	StartTime       int64
}

const (
	REDIS = "redis"
	SSDB = "ssdb"
	MEMCACHE_NEW = "memcache"
	MYSQL_NEW = "mysql"
	APPKEY = "646811797"
	APPKEY2 = "379999222"  // FIXME: 不支持大数据接口调用
	DEBUG = true
	RPC_STAT_INTVAL = 200
	CMS_CONFIG_EXPIRE_TIME = time.Minute    // TODO: cms消息通知实时生效
)

var ctx *Context
var IpWhiteList []*net.IPNet

var (
	api_names = map[string]string{
		"user_interest_tag": "user_interest_tag.json",
		"get_user_info": "get_user_info.json",
		"get_object_info_batch": "get_object_info_batch.json",
		"hot_timeline": "objects/hot_timeline.json",
		"object_list": "comments/object/list.json",
		"users_show": "users/show.json",
		"weibo_top_dynamic": "weibo_top_dynamic.json",
	}
)

// GetContext ...
func GetContext() *Context {
	return ctx
}

func abort(msg error) {
	fmt.Println(msg)
	os.Exit(1)
}

// NewClientContext ...
func NewClientContext() *Context {
	utils.InitLogger("debug", "rpc_client.log")
	return &Context{
		Alias:         make(map[string]*Alias),
		Storage:       make(map[string]*Storage),
		Cfg:           &Config{},
		RpcServiceMap: make(map[string]*RpcService),
		Services:      make(map[string]interface{}),
		Cache:         cache.New("default", 10, 0, 60 * time.Minute),
		Caches:		   make(map[string]*cache.Cache),
		StartTime:     time.Now().Unix(),
	}
}

// NewContext ...
func NewContext(confDir string) *Context {
	envConfDir := os.Getenv("SERVICE_CONFIG_DIR")
	if envConfDir != "" {
		confDir = envConfDir
	}
	fmt.Printf("load config files from %s\n", confDir)

	cfg, err := LoadConfig(confDir)
	if err != nil {
		abort(err)
		return nil
	}

	//cfg.System.ConfigFile = cfgFile
	cfg.System.ConfigDir = confDir

	ctx = &Context{
		Alias:         make(map[string]*Alias),
		Storage:       make(map[string]*Storage),
		Cfg:           &cfg,
		RpcServiceMap: make(map[string]*RpcService),
		Services:      make(map[string]interface{}),
		Cache:         cache.New("default", 100, 0, 60 * time.Minute),
		Caches:		   make(map[string]*cache.Cache),
		StartTime:     time.Now().Unix(),
	}

	ctx.Caches["article"] = cache.New("article", 10, 0, 60 * time.Minute)
	
	ctx.registerStorage()
	ctx.registerAlias()
	ctx.registerModels()
	utils.InitLogger(cfg.System.Loglevel, cfg.System.LogFile)
	ctx.HttpClient = &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, 2 * time.Second)
				if err != nil {
					utils.WriteLog("error", "dail timeout:%v", err)
					return nil, err
				}

				c.SetDeadline(time.Now().Add(2 * time.Second))
				return c, nil
			},
			//MaxIdleConnsPerHost:   10,
			MaxIdleConnsPerHost:   0,
			DisableKeepAlives:true,
			ResponseHeaderTimeout: 1 * time.Second,
		},
		Timeout: time.Duration(2000 * time.Millisecond),
	}
	ctx.HttpRetryClient = &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, 2 * time.Second)
				if err != nil {
					utils.WriteLog("error", "dail timeout:%v", err)
					return nil, err
				}
				c.SetDeadline(time.Now().Add(2 * time.Second))
				return c, nil
			},

			//MaxIdleConnsPerHost:   10,
			MaxIdleConnsPerHost:   0,
			DisableKeepAlives:true,
			ResponseHeaderTimeout: time.Second,
		},
		Timeout: time.Duration(2000 * time.Millisecond),
	}
	return ctx
}

// Release ...
func (c *Context) Release() {
	c.release()
}

// RedisClient2Hash ...
func (c *Context) RedisClient2Hash(name string, hash uint64) (*utils.Redis, error) {
	index := hash % uint64(c.storeageCount(REDIS, name))
	return c.Redis(name, int(index))
}

// RegisterRpcSerice ...
func (c *Context) RegisterRpcSerice(t interface{}) {
	sname := reflect.Indirect(reflect.ValueOf(t)).Type().Name()
	if _, present := c.RpcServiceMap[sname]; present {
		utils.WriteLog("error", "rpc: service already defined: " + sname)
		return
	}
	c.RpcServiceMap[sname] = &RpcService{
		Rcvr:  reflect.ValueOf(t),
		Typ:   reflect.TypeOf(t),
		Stats: make(map[string]*RpcServiceStats)}
	c.Services[sname] = t
}

// AddRpcMethodCount ...
func (c *Context) AddRpcMethodCount(sname string, mname string, count int) {
	if rpcService, exists := c.RpcServiceMap[sname]; exists {
		now := time.Now().UnixNano()
		rpcService.mutex.Lock()
		if _, methodExists := rpcService.Stats[mname]; !methodExists {
			// TODO：优化，初始的时候全部置0，就不用在这里判断了
			rpcService.Stats[mname] = &RpcServiceStats{}
		}
		rpcService.Stats[mname].Count += int64(count)
		rpcService.Stats[mname].RecentCount += int64(count)
		deltaT := now - rpcService.Stats[mname].QpsUpdateTime
		if rpcService.Stats[mname].Count % RPC_STAT_INTVAL == 0 || deltaT > 10 * 1e9 {
			rpcService.Stats[mname].Qps = float64(rpcService.Stats[mname].RecentCount * 1e9) /
				float64(deltaT)
			rpcService.Stats[mname].QpsUpdateTime = now
			rpcService.Stats[mname].RecentCount = 0

			rpcService.Stats[mname].Latency = float64(rpcService.Stats[mname].RecentLatency) /
				float64(rpcService.Stats[mname].RecentLatencyCount)
			rpcService.Stats[mname].RecentLatencyCount = 0
			rpcService.Stats[mname].RecentLatency = 0
		}
		rpcService.mutex.Unlock()
	}
}

// AddRpcMethodFinishStats ...
func (c *Context) AddRpcMethodFinishStats(sname string, mname string, latency float64) {
	if rpcService, exists := c.RpcServiceMap[sname]; exists {
		rpcService.mutex.Lock()
		if _, methodExists := rpcService.Stats[mname]; !methodExists {
			// TODO：优化，初始的时候全部置0，就不用在这里判断了
			rpcService.Stats[mname] = &RpcServiceStats{}
		}
		rpcService.Stats[mname].SuccCount += 1
		rpcService.Stats[mname].RecentLatencyCount += 1
		rpcService.Stats[mname].RecentLatency += latency
		rpcService.mutex.Unlock()
	}
}

// Article ...
func (c *Context) Article() interface{} {
	if _, exists := c.Services["Article"]; exists {
		return c.Services["Article"]
	}
	return nil
}

// WeiboApiRaw ...
func (c *Context) WeiboApiRaw(api string, params map[string]interface{}) (result []byte, err error) {
	var client *http.Client
	if _, exists := params["source"]; !exists {
		params["source"] = APPKEY
	}
	tauth := false
	if _, exists := params["tauth"]; exists {
		tauth = params["tauth"].(bool)
		delete(params, "tauth")
	}
	retry := false
	if _, exists := params["retry"]; exists {
		retry = params["retry"].(bool)
		delete(params, "retry")
	}
	cookie := ""
	if _, exists := params["cookie"]; exists {
		cookie = params["cookie"].(string)
		delete(params, "cookie")
	}

	if retry {
		client = c.HttpRetryClient
	} else {
		client = c.HttpClient
	}

	var req *http.Request
	reqUrl := "http://" + api
	method := "GET"
	if _, exists := params["method"]; exists {
		method = strings.ToUpper(params["method"].(string))
		delete(params, "method")
	}

	if method == "POST" {
		reqUrl += "?source=" + params["source"].(string)
		postParams := url.Values{}
		for k, v := range params {
			postParams.Set(k, v.(string))
		}
		body := ioutil.NopCloser(strings.NewReader(postParams.Encode()))
		req, err = http.NewRequest(method, reqUrl, body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	} else if method == "GET" {
		var strParams []string
		for k, v := range params {
			strParams = append(strParams, fmt.Sprintf("%v=%v", k, v))
		}

		if len(params) > 0 {
			reqUrl += "?" + strings.Join(strParams, "&")
		}
		req, err = http.NewRequest(method, reqUrl, nil)
	} else {
		err = errors.New(fmt.Sprintf("invalid request method %s", method))
		return
	}
	if err != nil {
		return
	}

	var resp *http.Response
	if tauth {
		authorization := c.TAuthApiRequest()
		if authorization == "" {
			err = errors.New("failed to get tauth token")
			return
		}
		req.Header.Add("authorization", authorization)
	}

	if cookie != "" && !tauth {
		req.AddCookie(&http.Cookie{Name:"SUB", Value:cookie})
		utils.WriteLog("debug", "cookie=%v", req.Cookies())
	}

	t1 := time.Now().UnixNano()
	resp, err = client.Do(req)
	t2 := time.Now().UnixNano()

	reqName := "unknow"
	for k, v := range (api_names) {
		if strings.Contains(api, v) {
			reqName = k
			break
		}
	}
	utils.WriteLog("info", "%dms HTTP_API %s url:%v", (t2 - t1) / 1000000, reqName, reqUrl)

	if err != nil {
		utils.WriteLog("error", "query %v failed:%v", reqUrl, err)
		if retry {
			t1 = time.Now().UnixNano()
			resp, err = client.Do(req)
			t2 = time.Now().UnixNano()
			utils.WriteLog("debug", "retry time_used=%d call url:%v", (t2 - t1) / 1000000, reqUrl)
			if err != nil {
				utils.WriteLog("error", "query %v retry failed:%v", reqUrl, err)
				return
			}
		} else {
			return
		}
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		utils.WriteLog("error", "read from http failed:%v", reqUrl, err)
		return
	}

	result = body
	return
}

// WeiboAPI ...
func (c *Context) WeiboAPI(api string, params map[string]interface{}) (result string, err error) {
	ret, err := c.WeiboApiRaw(api, params)
	if err != nil {
		return
	}
	result = string(ret)
	return
}

// WeiboApiJson ...
func (c *Context) WeiboApiJson(api string, params map[string]interface{}) (result map[string]interface{}, err error) {
	ret, err := c.WeiboApiRaw(api, params)
	if err != nil {
		return
	}
	jsonResult := make(map[string]interface{})
	err = json.Unmarshal(ret, &jsonResult)
	if err != nil {
		return
	}
	errorStr := jsonResult["error"]
	if errorStr != nil {
		err = errors.New(errorStr.(string))
	}
	result = jsonResult
	return
}

// WeiboApiJsonArray ...
func (c *Context) WeiboApiJsonArray(api string, params map[string]interface{}) (result []map[string]interface{}, err error) {
	ret, err := c.WeiboApiRaw(api, params)
	if err != nil {
		return
	}
	jsonResult := []map[string]interface{}{}
	err = json.Unmarshal(ret, &jsonResult)
	if err != nil {
		return
	}
	result = jsonResult
	return
}

// GetCmsConfig ...
func (c *Context) GetCmsConfig(key string) (result interface{}, err error) {
	cacheKey := "CMS:" + key
	if ret, found := c.Cache.Get(cacheKey); found {
		result = ret
		return
	}

	cmsRedis, err := c.Redis("cms_redis", 0)
	if err != nil {
		utils.WriteLog("error", "get cms redis failed:%v", err)
		return
	}
	result = cmsRedis.Get(key)
	c.Cache.Set(cacheKey, result, CMS_CONFIG_EXPIRE_TIME)
	return
}

// IpCheck ...
func (c *Context) IpCheck(ip string) bool {
	if !c.Cfg.System.IpCheck {
		return true
	}
	for _, ipList := range (IpWhiteList) {
		if ipList.Contains(net.ParseIP(ip)) {
			//fmt.Printf("match:%v, %v\n", ip, ipList)
			return true
		}
	}
	utils.WriteLog("error", "ip not allowed:%s", ip)
	return false
}

func (c *Context) GetCache(name string) *cache.Cache {
	ca, exists := c.Caches[name]
	if exists {
		return ca
	} else {
		utils.WriteLog("error", "got non-exists cache:%s", name)
		return c.Cache
	}
}

func (c *Context) registerModels() {
	mysql.RegisterModel(&models.Articles{})
	mysql.RegisterModel(&models.PushArticles{})
	mysql.RegisterModel(&models.Related{})
	mysql.RegisterModel(&models.Subscribe{})
	mysql.RegisterModel(&models.ToutiaoUser{})
	mysql.RegisterModel(&models.ArticleLike{})
	mysql.RegisterModel(&models.CommentLike{})
	mysql.RegisterModel(&models.ArticleCard{})
	mysql.RegisterModel(&models.ArticleSpecial{})
	mysql.RegisterModel(&models.ArticleSpecialGroup{})
	mysql.RegisterModel(&models.ArticleSpecialContent{})
}