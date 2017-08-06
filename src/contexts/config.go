package contexts

import (
	_ "errors"
	"gopkg.in/gcfg.v1"
	"strings"
	"net"
	"fmt"
	"path/filepath"
	"os"
	"io/ioutil"
	"errors"
)

// Config ...
type Config struct {
	System  struct {
				IsDaemon    int
				Debug       int
				Host        string
				Port        int
				HttpPort    int
				LogOutput   string
				Loglevel    string
				LogFile     string
				Pidfile     string
				RlimitCur   uint64
				RlimitMax   uint64
								 //ConfigFile  string
				ConfigDir   string
				MaxCpu      int
				TestServer  bool // 是否测试环境
				IpCheck     bool
				IpWhitelist string
				Servers     []string
			}

	Switch  struct {
				RunAdminWebSite           bool    `desc:"启动管理工具"`
				RunMonitor                bool    `desc:"是否启动进程内monitor"`
				DisableGeObjectInfoBatch  bool    `desc:"禁止调用对象信息接口"`
				DisableObjectsHotTimeline bool    `desc:"禁止调用热门评论接口"`
				DisableCommentsObjectList bool    `desc:"禁止调用最新评论接口"`
				EnableMemcacheCompression bool    `desc:"启动memcache压缩选项" disabled:"1"`
				DisableRelatedArticle     bool    `desc:"客户端不显示相关文章"`
			}

	Alias   map[string]*struct {
		Values []string
	}

	Feed    struct {
				StarFeedGetMaxNum  int      //明星流每次请求取候选的条数
				StarFeedHistoryNum int      //明星流历史保留条数
				StarFeedHistoryDay int      //明星流历史保留时间
				StarFeedVideoCount int      //明星流每次加载视频条数
				ChanFeedManualNum  int      //频道流每次请求拉取的人工最终条数
				ChanFeedMachineNum int      //频道流每次请求拉取的机器备选条数
				ChanFeedHistoryNum int      //频道流历史保留条数
				ChanFeedHistoryDay int      //频道流历史保留时间
				IdGenServers       []string //ID发生器地址列表
				IdDecServers       []string //ID解析器地址列表
			}

	Storage map[string]*struct {
		Adapter  string
		Servers  []string
		Hash     string
		Poolsize string
		Disabled bool
	}
}

/*
// LoadConfig ...
func LoadConfig(cfgFile string) (Config, error) {
	var err error
	var cfg Config
	err = gcfg.ReadFileInto(&cfg, cfgFile)
	if err != nil {
		return cfg, err
	}

	iplist := strings.Split(cfg.System.IpWhitelist, ",")
	for _, ip := range (iplist) {
		_, n, err := net.ParseCIDR(ip)
		if err != nil {
			fmt.Printf("parse ip failed for %v, err=%v\n", ip, err)
			continue
		}
		IpWhiteList = append(IpWhiteList, n)
	}

	return cfg, nil
}
*/

func LoadConfig(confDir string) (cfg Config, err error) {
	var fd *os.File
	confBytes := make([]byte, 0)
	err = filepath.Walk(confDir, func(path string, f os.FileInfo, walkerr error) (err error) {
		if walkerr != nil {
			return walkerr
		}
		if f.IsDir() {
			return
		}

		//校验文件后缀
		if !strings.HasSuffix(f.Name(), ".ini") {
			return
		}

		fd, err = os.Open(path)
		if err != nil {
			return
		}
		defer fd.Close()
		itemBytes, err := ioutil.ReadAll(fd)
		if err != nil {
			return
		}
		confBytes = append(confBytes, itemBytes...)
		confBytes = append(confBytes, []byte("\n")...) //每个文件结尾显式加换行,以保证ini文件的格式
		return nil
	})

	if err != nil {
		return
	}
	if len(confBytes) == 0 {
		err = errors.New("empty config")
		return
	}

	err = gcfg.ReadStringInto(&cfg, string(confBytes))
	if err != nil {
		return
	}

	iplist := strings.Split(cfg.System.IpWhitelist, ",")
	for _, ip := range (iplist) {
		_, n, err := net.ParseCIDR(ip)
		if err != nil {
			fmt.Printf("parse ip failed for %v, err=%v\n", ip, err)
			continue
		}
		IpWhiteList = append(IpWhiteList, n)
	}
	return
}