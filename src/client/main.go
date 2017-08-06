package main

import (
	"flag"
	"fmt"
	"strings"
	"libs/yar"
	"contexts"
	"reflect"
	"strconv"
	"utils"
)

func call(client *yar.Client, ctx *contexts.Context, serviceName string, methodName string, argv map[string]string) {
	serviceMethod := serviceName + "." + methodName
	s := ctx.RpcServiceMap[serviceName]
	if s == nil {
		fmt.Printf("service %s not found\n", serviceName)
		return
	}
	method, found := s.Typ.MethodByName(methodName)
	if !found {
		fmt.Printf("method %s not found\n", methodName)
		return
	}
	args := reflect.New(method.Type.In(1).Elem())
	reply := reflect.New(method.Type.In(2).Elem()).Interface()

	argsElem := args.Elem()
	argsType := argsElem.Type()
	for i := 0; i < argsElem.NumField(); i++ {
		f := argsElem.Field(i)
		fName := argsType.Field(i).Tag.Get("msgpack")
		if _, exist := argv[fName]; exist {
			switch f.Type().Kind() {
			case reflect.Int:
				intValue, _ := strconv.ParseInt(argv[fName], 10, 64)
				argsElem.Field(i).Set(reflect.ValueOf(int(intValue)))
			case reflect.Uint64:
				intValue, _ := strconv.ParseUint(argv[fName], 10, 64)
				argsElem.Field(i).Set(reflect.ValueOf(intValue))
			case reflect.Int64:
				intValue, _ := strconv.ParseInt(argv[fName], 10, 64)
				argsElem.Field(i).Set(reflect.ValueOf(intValue))
			case reflect.Bool:
				boolValue, _ := strconv.ParseBool(argv[fName])
				argsElem.Field(i).Set(reflect.ValueOf(boolValue))
			case reflect.Slice:
				elemType := f.Type().Elem().Kind()
				arrayValue := strings.Split(argv[fName], ",")
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
				argsElem.Field(i).Set(reflect.ValueOf(argv[fName]))
			}
		}
	}
	err := client.Call(serviceMethod, args.Interface(), reply)
	if err != nil {
		fmt.Printf("call rpc failed:%v", err)
		return
	}
	utils.VarDump(reply)
}

func main() {
	server := flag.String("server", "i.rpc.toutiao.weibo.cn:8180", "server's host:port")
	flag.Usage = func() {
		fmt.Printf("rpc_client [-server host:port] Service.Method [args...]\n")
	}
	flag.Parse()

	nArgs := flag.NArg()
	if nArgs == 0 {
		flag.Usage()
		return
	}

	serviceMethod := strings.Split(flag.Arg(0), ".")
	if len(serviceMethod) != 2 {
		fmt.Printf("invalid service or method.\n")
		flag.Usage()
		return
	}
	serviceName := serviceMethod[0]
	methodName := serviceMethod[1]

	fmt.Printf("server=%s\n", *server)
	args := make(map[string]string)
	for i := 1; i < nArgs; i++ {
		fmt.Printf("    - %s\n", flag.Arg(i))
		s := strings.Split(flag.Arg(i), "=")
		if len(s) != 2 {
			fmt.Printf(">>>> invalid argument:%s\n", s)
			continue
		}
		s[0] = strings.TrimSpace(s[0])
		s[1] = strings.TrimSpace(s[1])
		args[s[0]] = s[1]
	}

	client, err := yar.Dial("tcp", *server, "msgpack")
	if err != nil {
		fmt.Printf("call rpc failed:%v\n", err)
		return
	}

	ctx := contexts.NewClientContext()
	srv := yar.NewRpcServer()
	if err := yar.RegisterService(srv, ctx); err != nil {
		fmt.Printf("Register failed: %v\n", err)
		return
	}

	call(client, ctx, serviceName, methodName, args)
	client.Close()
}
