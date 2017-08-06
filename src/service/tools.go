package service

import (
	"utils"
	"fmt"
)

// Tools ...
type Tools struct {
	Type
}

// Base62EncodeArgs ...
type Base62EncodeArgs struct {
	Common		  CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Num           int64 		`msgpack:"num" must:"1" comment:"num" size:"40"`
}

// Base62EncodeResult ...
type Base62EncodeResult struct {
	Result Result
	Data   string
}

// Base62DecodeArgs ...
type Base62DecodeArgs struct {
	Common		  CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Str           string 		`msgpack:"str" must:"1" comment:"str" size:"40"`
}

// Base62DecodeResult ...
type Base62DecodeResult struct {
	Result Result
	Data   string
}

// Base62Encode ...
func (t *Tools) Base62Encode(args *Base62EncodeArgs, reply *Base62EncodeResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	reply.Data = string(utils.Base62Encode(args.Num))
	return nil
}

// Base62Decode ...
func (t *Tools) Base62Decode(args *Base62DecodeArgs, reply *Base62DecodeResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	num, err := utils.Base62Decode([]byte(args.Str))
	if err != nil {
		reply.Result = Result{Status: -1, Error: "failed to decode:" + err.Error()}
		reply.Data = "0"
	} else {
		reply.Data = fmt.Sprintf("%d", num)
	}

	return nil
}

// SsdbToolArgs ...
type SsdbToolArgs struct {
	Common		  CommonArgs 	`msgpack:"common" must:"0" comment:"公共参数" size:"40"`
	Resource      string 		`msgpack:"resource" must:"1" comment:"资源名" size:"40"`
	Key           string 		`msgpack:"key" must:"1" comment:"Key" size:"40"`
	Value         string 		`msgpack:"value" must:"0" comment:"Value" size:"40"`
}

// SsdbToolResult ...
type SsdbToolResult struct {
	Result Result
	Data   string
}

// SsdbTool ...
func (t *Tools) SsdbTool(args *SsdbToolArgs, reply *SsdbToolResult) error {
	reply.Result = Result{Status: 0, Error: "success"}
	dbRes := map[string]string{"subscribe": "subscribe_ssdb_r"}[args.Resource]
	if args.Key == "" {
		reply.Result = Result{Status: 1, Error: "empty key"}
		return nil
	}
	w, err := t.Ctx.Ssdb(dbRes, 0)
	if err != nil {
		t.Error("get user ssdb failed:%v", err)
		reply.Result = Result{Status: 1, Error: fmt.Sprintf("system error:%v", err)}
		return err
	}
	if args.Value != "" {
		w.Set(args.Key, args.Value)
	}
	v, err := w.Get(args.Key)
	if err != nil {
		reply.Result = Result{Status: 1, Error: fmt.Sprintf("faield to get value:%v", err)}
	} else {
		reply.Data = v
	}

	return nil
}
