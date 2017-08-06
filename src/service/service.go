package service

import (
	"contexts"
	"encoding/json"
	"fmt"
	"net/http"
	//"net/rpc"
	//"libs/yar"
	"reflect"
	"strconv"
	"strings"
	"time"
	"utils"
	"runtime"
	"sync"
)

// Type ...
type Type struct {
	Ctx 	*contexts.Context
	GoID	uint64
}

// Service ...
type Service struct {
	Type
}

// Result ...
type Result struct {
	Status int
	Error  string
}

// CommonArgs ...
type CommonArgs struct {
	IP 		   string 	`msgpack:"ip" must:"0" comment:"ip"`
	Timeout    int 		`msgpack:"timeout" must:"0" comment:"timeout(ms)"`
}

// CommonResult ...
type CommonResult struct {
	Result Result
}

// StatusResult ...
type StatusResult struct {
	Result Result
	Status      string
	Version     string
	Githash     string
	BuildDate   string
	Cache       map[string]map[string]int
	Switches    map[string]bool
	Services    map[string][]string
	ServerStart string
	ServerTime  string
	ServerUp    string
	QPS 		float64
	System      map[string]interface{}
	ErrorLogNumber 	int
}

var buildVersion = "unknown"
var buildGithash = "unknown"
var buildDate = "unknown"
var ServiceStatus = "good"

// GetBuildInfo ...
func GetBuildInfo() (ver string, gitHash string, date string) {
	ver = buildVersion
	gitHash = buildGithash
	date = buildDate
	return
}

// NewServiceType ...
func NewServiceType(c *contexts.Context) *Type {
	return &Type{Ctx:c}
}

// Debug ...
func (s *Type) Debug(format string, params ...interface{}) {
	newParams := []interface{}{s.GoID}
	newParams = append(newParams, params...)
	utils.DefaultLogger.Debugf("[%d] " + format, newParams...)
}

// Info ...
func (s *Type) Info(format string, params ...interface{}) {
	newParams := []interface{}{s.GoID}
	newParams = append(newParams, params...)
	utils.DefaultLogger.Infof("[%d] " + format, newParams...)
}

// Error ...
func (s *Type) Error(format string, params ...interface{}) {
	newParams := []interface{}{s.GoID}
	newParams = append(newParams, params...)
	utils.DefaultLogger.Errorf("[%d] " + format, newParams...)
	utils.ErrorLogNumber++
}

// Warn ...
func (s *Type) Warn(format string, params ...interface{}) {
	newParams := []interface{}{s.GoID}
	newParams = append(newParams, params...)
	utils.DefaultLogger.Warnf("[%d] " + format, newParams...)
}

// Critical ...
func (s *Type) Critical(format string, params ...interface{}) {
	newParams := []interface{}{s.GoID}
	newParams = append(newParams, params...)
	utils.DefaultLogger.Criticalf("[%d] " + format, newParams...)
	utils.ErrorLogNumber++
}

func setCommonArgs(fieldValue reflect.Value, argv map[string][]string) int64 {
	timeout := int64(0)
	for i := 0; i < fieldValue.NumField(); i++ {
		f := fieldValue.Field(i)
		fName := fieldValue.Type().Field(i).Name
		if _, exist := argv[fName]; exist {
			if fName == "Timeout" {
				intValue, _ := strconv.ParseInt(argv[fName][0], 10, 64)
				timeout = int64(intValue)
			}
			switch f.Type().Kind() {
			case reflect.Int:
				intValue, _ := strconv.ParseInt(argv[fName][0], 10, 64)
				fieldValue.Field(i).Set(reflect.ValueOf(int(intValue)))
			case reflect.Uint64:
				intValue, _ := strconv.ParseUint(argv[fName][0], 10, 64)
				fieldValue.Field(i).Set(reflect.ValueOf(intValue))
			case reflect.Int64:
				intValue, _ := strconv.ParseInt(argv[fName][0], 10, 64)
				fieldValue.Field(i).Set(reflect.ValueOf(intValue))
			case reflect.Bool:
				boolValue, _ := strconv.ParseBool(argv[fName][0])
				fieldValue.Field(i).Set(reflect.ValueOf(boolValue))
			case reflect.Slice:
				elemType := f.Type().Elem().Kind()
				arrayValue := strings.Split(argv[fName][0], ",")
				if elemType == reflect.String {
					fieldValue.Field(i).Set(reflect.ValueOf(arrayValue))
				} else if elemType == reflect.Int || elemType == reflect.Uint64 {
					intArrayValue := []int{}
					for _, v := range arrayValue {
						intArrayValue = append(intArrayValue, utils.Atoi(v))
					}
					fieldValue.Field(i).Set(reflect.ValueOf(intArrayValue))
				}
			default:
				fieldValue.Field(i).Set(reflect.ValueOf(argv[fName][0]))
			}
		}
	}
	return timeout
}

// RunRPCService ...
func RunRPCService(ctx *contexts.Context, sname string, mname string, argv map[string][]string, r *http.Request, w http.ResponseWriter, finished *bool, mutex *sync.Mutex) string {
	goID := utils.GetGoroutineID()
	timeStart := time.Now()
	s := ctx.RpcServiceMap[sname]
	method, found := s.Typ.MethodByName(mname)
	if !found {
		utils.WriteLog("info", "RunRpcService method not found mname=%v.%v", sname, mname)
		return fmt.Sprintf("{\"Result\":{\"status\":1, \"error\":\"method not found:%v.%v\"}}", sname, mname)
	}
	args := reflect.New(method.Type.In(1).Elem())
	reply := reflect.New(method.Type.In(2).Elem())
	argsElem := args.Elem()
	argsType := argsElem.Type()
	for i := 0; i < argsElem.NumField(); i++ {
		f := argsElem.Field(i)
		fName := argsType.Field(i).Name
		if fName == "Common" {
			timeout := setCommonArgs(argsElem.Field(i), argv)
			if timeout > 0 {
				time.AfterFunc(time.Duration(timeout*int64(time.Millisecond)), func() {
					(*mutex).Lock()
					if *finished {
						(*mutex).Unlock()
						return
					}
					*finished = true
					w.Header().Set("Content-Type", "application/json;charset=utf-8")
					w.Write([]byte(fmt.Sprintf("{\"Result\":{\"Status\": -10, \"Error\": \"timeout after %d ms\"}}", timeout)))
					(*mutex).Unlock()   // 写完之后再unlock，避免handler线程返回后此处无法写
				})
			}
			continue
		}
		if _, exist := argv[fName]; exist {
			switch f.Type().Kind() {
			case reflect.Int:
				intValue, _ := strconv.ParseInt(argv[fName][0], 10, 64)
				argsElem.Field(i).Set(reflect.ValueOf(int(intValue)))
			case reflect.Uint64:
				intValue, _ := strconv.ParseUint(argv[fName][0], 10, 64)
				argsElem.Field(i).Set(reflect.ValueOf(intValue))
			case reflect.Int64:
				intValue, _ := strconv.ParseInt(argv[fName][0], 10, 64)
				argsElem.Field(i).Set(reflect.ValueOf(intValue))
			case reflect.Bool:
				boolValue, _ := strconv.ParseBool(argv[fName][0])
				argsElem.Field(i).Set(reflect.ValueOf(boolValue))
			case reflect.Slice:
				elemType := f.Type().Elem().Kind()
				arrayValue := strings.Split(argv[fName][0], ",")
				if elemType == reflect.String {
					argsElem.Field(i).Set(reflect.ValueOf(arrayValue))
				} else if elemType == reflect.Int {
					intArrayValue := []int{}
					for _, v := range arrayValue {
						i64, _ := strconv.ParseInt(v, 10, 0)
						intArrayValue = append(intArrayValue, int(i64))
					}
					argsElem.Field(i).Set(reflect.ValueOf(intArrayValue))
				} else if elemType == reflect.Int64 {
					intArrayValue := []int64{}
					for _, v := range arrayValue {
						i64, _ := strconv.ParseInt(v, 10, 0)
						intArrayValue = append(intArrayValue, i64)
					}
					argsElem.Field(i).Set(reflect.ValueOf(intArrayValue))
				} else if elemType == reflect.Uint64 {
					intArrayValue := []uint64{}
					for _, v := range arrayValue {
						i64, _ := strconv.ParseInt(v, 10, 0)
						intArrayValue = append(intArrayValue, uint64(i64))
					}
					argsElem.Field(i).Set(reflect.ValueOf(intArrayValue))
				}
			default:
				argsElem.Field(i).Set(reflect.ValueOf(argv[fName][0]))
			}
		}
	}

	v := s.Rcvr.Elem().FieldByName("GoID")
	if v.Type().Kind() == reflect.Uint64 {
		v.Set(reflect.ValueOf(goID))
	}
	method.Func.Call([]reflect.Value{s.Rcvr, args, reply})
	ctx.AddRpcMethodCount(sname, mname, 1)

	var ss []string
	for k, v := range r.Form {
		if strings.ToLower(k) == "service" {
			continue
		}
		ss = append(ss, fmt.Sprintf("%v=%v", k, v[0]))
	}
	utils.DefaultLogger.Infof("[%d] %vms %v REQUEST %v %v %v protocal=HTTP timeout=%v",
		goID,
		time.Now().Sub(timeStart).Seconds()*1000,
		strings.Split(r.RemoteAddr, ":")[0],
		sname+"."+mname,
		"success",
		strings.Join(ss, " "),
		*finished)
	if *finished {
		return ""
	}
	jsonReply, _ := json.Marshal(reply.Elem().Interface())
	return string(jsonReply)
}

func calcUpTime(s int64) (days, hours, minutes, seconds, total int64) {
	total = s
	days = int64(s / (3600 * 24))
	hours = int64((s % (3600 * 24)) / 3600)
	minutes = int64((s % 3600) / 60)
	seconds = s % 60
	return
}

// Status ...
func (s *Service) Status(args *CommonArgs, reply *StatusResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	reply.Status = ServiceStatus
	reply.Version = buildVersion
	reply.Githash = buildGithash
	reply.BuildDate = buildDate
	
	reply.Cache = make(map[string]map[string]int)
	reply.Cache["default"] = make(map[string]int)
	reply.Cache["default"]["total"], reply.Cache["default"]["expired"] = s.Ctx.Cache.Status()
	for cacheName, c := range(s.Ctx.Caches) {
		reply.Cache[cacheName] = make(map[string]int)
		reply.Cache[cacheName]["total"], reply.Cache[cacheName]["expired"] = c.Status()
	}
	
	now := time.Now()
	reply.ServerStart = time.Unix(s.Ctx.StartTime, 0).Format("2006-01-02_15:04:05")
	reply.ServerTime = now.Format("2006-01-02_15:04:05")
	days, hours, minutes, seconds, total := calcUpTime(now.Unix() - s.Ctx.StartTime)
	reply.ServerUp = fmt.Sprintf("%d days %d:%d:%d, total %d seconds", days, hours, minutes, seconds, total)

	reply.Switches = make(map[string]bool)
	switchType := reflect.TypeOf(s.Ctx.Cfg.Switch)
	switchValue := reflect.ValueOf(s.Ctx.Cfg.Switch)
	for i:=0; i<switchType.NumField(); i++ {
		switchField := switchType.Field(i)
		reply.Switches[switchField.Name] = switchValue.Field(i).Interface().(bool)
	}

	reply.Services = make(map[string][]string)
	reply.QPS = 0
	for k, v := range s.Ctx.RpcServiceMap {
		methods := []string{}
		for m := 0; m < v.Typ.NumMethod(); m++ {
			method := v.Typ.Method(m)
			if method.PkgPath != "" {
				continue
			}
			stats, exists := v.Stats[method.Name]
			if !exists {
				continue
			}
			methods = append(methods, fmt.Sprintf("%s:total(%d), success(%d), qps(%.2f), latency(%.3f)",
				method.Name,stats.Count, stats.SuccCount, stats.Qps, stats.Latency))
			reply.QPS += stats.Qps
		}
		if len(methods) > 0 {
			reply.Services[k] = methods
			s.Info("%v => %v", k, methods)
		}
	}

	reply.ErrorLogNumber = utils.ErrorLogNumberIncreasement

	reply.System = make(map[string]interface{})
	reply.System["version"] = runtime.Version()
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)
	reply.System["mem_stats"] = map[string]interface{} {
		"bytes allocated and not yet freed": m.Alloc,
		"bytes allocated (even if freed)": m.TotalAlloc,
		"bytes obtained from system": m.Sys,
		"number of pointer lookups": m.Lookups,
		"number of mallocs": m.Mallocs,
		"number of frees": m.Frees,
		"gc_sys": m.GCSys,
		"next_gc": m.NextGC,
		"last_gc": m.LastGC,
		"num_gc": m.NumGC,
		"gc_cpu_fraction": m.GCCPUFraction,
		"enable_gc": m.EnableGC,
		"debug_gc": m.DebugGC,
		"heap_objects": m.HeapObjects,
	}
	reply.System["num_cgo_call"] = runtime.NumCgoCall()
	reply.System["num_cpu"] = runtime.NumCPU()
	reply.System["max_cpu"] = s.Ctx.Cfg.System.MaxCpu
	reply.System["num_goroutine"] = runtime.NumGoroutine()

	return nil
}
