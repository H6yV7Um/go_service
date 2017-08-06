package main

import (
	"contexts"
	"flag"
	"fmt"
	"libs/yar"
	"net"
	"net/http"
	"service"
	"strings"
	_ "net/http/pprof"
	"tools"
	"runtime"
	"utils"
	"time"
	"sync"
)

// HTTPHandler 用于处理http端口（默认8090）的请求，以http方式提供rpc方法的服务
func HTTPHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.Redirect(w, r, "/tools", http.StatusMovedPermanently)
	}

	r.ParseForm()
	rpcServiceString := r.Form.Get("service")
	if rpcServiceString == "" {
		w.Write([]byte("Service empty"))
		return
	}
	s := strings.Split(rpcServiceString, ".")
	if len(s) != 2 {
		w.Write([]byte("Service invalid:" + rpcServiceString))
		return
	}

	rpcService := strings.Title(s[0])
	rpcMethod := strings.Title(s[1])
	args := make(map[string][]string)
	for k, v := range (r.Form) {
		if len(v) == 0 {
			continue
		}
		argName := strings.Title(k)
		switch argName {
		// 与golint代码规范兼容
		//case "Uid":
		//	argName = "UID"
		case "Id":
			argName = "ID"
		}
		if strings.Contains(argName, "_") {
			argName = strings.Replace(argName, "_", "", 1)
		}
		// Common参数的格式：ip=10.10.10.10,timeout=100
		if argName == "Common" {
			commonArgs := strings.Split(v[0], ",")
			for _, commonArg := range (commonArgs) {
				kv := strings.Split(commonArg, "=")
				if len(kv) != 2 {
					continue
				}
				args[strings.Title(kv[0])] = []string{strings.Title(kv[1])}
			}
			continue
		} else {
			args[argName] = []string{v[0]}
		}
	}
	//var finished AtomicBool
	var finished bool
	var mutex sync.Mutex
	rpcResult := service.RunRPCService(contexts.GetContext(), rpcService, rpcMethod, args, r, w, &finished, &mutex)

	// finished标志加锁，用于给了timeout参数超时后回写数据的同步，避免重复回写
	mutex.Lock()
	if !finished {
		finished = true
		mutex.Unlock()
		w.Header().Set("Content-Type", "application/json;charset=utf-8")
		w.Write([]byte(rpcResult))
	} else {
		mutex.Unlock()
	}
}

func startHTTPServerHelper(port int64) {
	ctx := contexts.GetContext()
	http.HandleFunc("/rpc", HTTPHandler)
	if ctx.Cfg.Switch.RunAdminWebSite {
		utils.DefaultLogger.Infof("start tools web site....")
		http.HandleFunc("/", HTTPHandler)
		http.HandleFunc("/tools", tools.HTTPHandler)
		http.HandleFunc("/static/", tools.StaticResource)
	}
	laddr := fmt.Sprintf(":%d", port)
	if ctx.Cfg.System.Host != "" && ctx.Cfg.System.Port != 0 {
		laddr = fmt.Sprintf("%s:%d", ctx.Cfg.System.Host, port)
	}
	http.ListenAndServe(laddr, nil)
}

func monitor() {
	monitorInterval := 5
	monitorKeyList := map[string]bool {
		"mysql_article_r": true,
		"mysql_articlelike_w": true,
		"mysql_commentlike_r": true,
		"mysql_toutiao_user_r": true,
		"redis_article_related_r": true,
		"redis_collection_pika_r": true,
		"redis_collection_pika_w": true,
		"redis_r7018": true,
		"redis_m7018": true,
		"redis_star_feed_r": true,
		"redis_cms_redis": true,
		"redis_comment_r": true,
		"redis_feed": true,
		"redis_star_feed_his_w": true,
		"redis_subscribed_r": true,
		"redis_subscribed_w": true,
		"redis_user_r": true,
		"redis_user_w": true,
		"redis_favor_r": true,
		"redis_channel_topfeed_r": true,
		"redis_notice_r": true,
		"ssdb_subscribe_ssdb_r": true,
		"ssdb_subscribe_ssdb_w": true,
		"memcache_article_mc": true,
	}
	errorLogQueue := &utils.LoopQueue{}
	errorLogQueue.New(60/monitorInterval)
	for {
		status := "good"
		m := &runtime.MemStats{}
		runtime.ReadMemStats(m)
		ctx := contexts.GetContext()
		connData := ctx.GetConnInfo()
		errorLogQueue.Push(utils.ErrorLogNumber)
		lastErrorLogNumber, _ := errorLogQueue.Oldest().(int)
		utils.ErrorLogNumberIncreasement = utils.ErrorLogNumber - lastErrorLogNumber
		statsMsg := []string{
			fmt.Sprintf("mem_used=%d", m.Alloc),
			fmt.Sprintf("num_goroutine=%d", runtime.NumGoroutine()),
			fmt.Sprintf("gc_cpu_fraction=%f", m.GCCPUFraction),
			fmt.Sprintf("error_log=%d", utils.ErrorLogNumber),
			fmt.Sprintf("error_log_inc=%d", utils.ErrorLogNumberIncreasement),
		}
		if m.Alloc > 200*1024*1024 || runtime.NumGoroutine() > 200 || m.GCCPUFraction > 0.2 || utils.ErrorLogNumberIncreasement > 100 {
			status = "error"
		} else if (m.Alloc > 100*1024*1024 || runtime.NumGoroutine() > 100 || m.GCCPUFraction > 0.1 ||
				utils.ErrorLogNumberIncreasement > 50) && status == "good" {
			status = "warning"
		}
		for protocol, data := range(connData) {
			for name, value := range(data) {
				key := protocol + "_" + name
				if _, exists := monitorKeyList[key]; exists {
					statsMsg = append(statsMsg, fmt.Sprintf("%s=%d", key, value))
					if value > 100 {
						status = "error"
					} else if value > 50 && status == "good" {
						status = "warning"
					}
				}
			}
		}
		statsMsg = append(statsMsg, fmt.Sprintf("status=%s", status))
		utils.MonitorLogger.Infof("MONITOR %s",strings.Join(statsMsg, " "))
		service.ServiceStatus = status
		time.Sleep(time.Second * time.Duration(monitorInterval))
	}
}

func main() {
	// 优先使用配置文件的端口和ip，然后是命令行参数，最后是默认值
	// TODO: port放到Service.Status里
	port := flag.Int64("port", 8080, "port number")
	httpPort := flag.Int64("http_port", 8090, "http test server port number(default is 0, means disable)")

	confDir := flag.String("confdir", "./conf", "config files diretory")
	flag.Parse()
	ctx := contexts.NewContext(*confDir)

	if ctx.Cfg.System.MaxCpu > 0 {
		runtime.GOMAXPROCS(ctx.Cfg.System.MaxCpu)
		utils.DefaultLogger.Infof("set max cpu number to %d", ctx.Cfg.System.MaxCpu)
	}

	laddr := fmt.Sprintf(":%d", *port)
	if ctx.Cfg.System.Port != 0 {
		if ctx.Cfg.System.Host != "" {
			laddr = fmt.Sprintf("%s:%d", ctx.Cfg.System.Host, ctx.Cfg.System.Port)
			*httpPort = int64(ctx.Cfg.System.Port + 10)
		} else {
			internalIPAddr, err := utils.GetInternalIpAddr()
			if err == nil {
				laddr = fmt.Sprintf("%s:%d", internalIPAddr, ctx.Cfg.System.Port)
				*httpPort = int64(ctx.Cfg.System.Port + 10)
			}
		}
	}
	if ctx.Cfg.System.HttpPort != 0 {
		*httpPort = int64(ctx.Cfg.System.HttpPort)
	}
	lis, err := net.Listen("tcp", laddr)
	if err != nil {
		utils.DefaultLogger.Criticalf("Listen on %v failed: %v", laddr, err)
		return
	}
	defer lis.Close()

	srv := yar.NewRpcServer()
	if err := yar.RegisterService(srv, ctx); err != nil {
		utils.DefaultLogger.Criticalf("Register: %v", err)
		return
	}

	if *httpPort != 0 {
		go startHTTPServerHelper(*httpPort)
		utils.DefaultLogger.Infof("HTTP service started on port %v", *httpPort)
	}

	if ctx.Cfg.Switch.RunMonitor {
		go monitor()
	}

	utils.DefaultLogger.Infof("RPC service started on %v", laddr)
	for {
		conn, err := lis.Accept()
		if err != nil {
			utils.DefaultLogger.Criticalf("lis.Accept() failed: %v", err)
		} else {
			conn.SetDeadline(time.Now().Add(3000 * time.Millisecond))
			go srv.ServeCodec(yar.NewServerCodec(conn))
		}
	}
	utils.DefaultLogger.Infof("Service exit")
}
