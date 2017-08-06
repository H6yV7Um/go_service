// Copyright 2013 <chaishushan{AT}gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// copy from：https://github.com/gyf19/yar-go

package yar

import (
	"contexts"
	"errors"
	"fmt"
	"io"
	"net"
	//"net/rpc"  // 为了定制，从go 1.6.2的net/rpc/server.go复制一份代码
	"reflect"
	"utils"
	"strings"
	"sync"
	"time"
)

var errMissingParams = errors.New("jsonrpc: request body missing params")
var errInvalidParams = errors.New("invalid params")

type YarServerCodec struct {
	r io.Reader
	w io.Writer
	c io.Closer

	// temporary work space
	req    serverRequest
	packer Packager

	mutex   sync.Mutex // protects seq, pending
	seq     uint64
	seqId   uint64
	GoID    uint64
	pending map[uint64]uint32

	t      	time.Time   // 开始处理时间
	params 	interface{} // 解码后的调用参数，用于打日志
	timeout bool        // 是否超时
	done    bool        // 是否结束
}

// NewServerCodec returns a serverCodec that communicates with the ClientCodec
// on the other end of the given conn.
func NewServerCodec(conn io.ReadWriteCloser) ServerCodec {
	return &YarServerCodec{
		r:       conn,
		w:       conn,
		c:       conn,
		t:       time.Now(),
		GoID:    0,
		pending: make(map[uint64]uint32),
		timeout: false,
		done:    false,
	}
}

func (c *YarServerCodec) ReadRequestHeader(r *Request) error {
	c.req.Reset()
	packer, err := readPack(c.r, &c.req)
	if err != nil {
		return err
	}

	c.packer = packer
	r.ServiceMethod = c.req.Method
	c.mutex.Lock()
	c.seq++
	c.pending[c.seq] = uint32(c.req.Id)
	c.req.Id = 0
	r.Seq = c.seq
	c.mutex.Unlock()
	serviceAndMethod := strings.Split(r.ServiceMethod, ".")
	if len(serviceAndMethod) == 2 {
		contexts.GetContext().AddRpcMethodCount(serviceAndMethod[0], serviceAndMethod[1], 1)
	}

	return nil
}

func (c *YarServerCodec) writeMessage(status int, msg string) {
	type ServiceResult struct {
		Status int
		Error  string
	}
	type CommonResult struct {
		Result ServiceResult
	}

	var y interface{}
	y = CommonResult{Result:ServiceResult{Status:status, Error:msg}}
	resp := serverResponse{
		Id:     int64(0),
		Error:  "",
		Result: &y,
		Output: "",
		Status: 0,
	}
	err := writePack(c.w, c.packer, 0, &resp)
	if err != nil {
		utils.DefaultLogger.Warnf("[%d] writePack failed:%s", c.GoID, err.Error())
	}
}

func (c *YarServerCodec) ReadRequestBody(x interface{}) error {
	if x == nil {
		return nil
	}
	if c.req.Params == nil {
		return errMissingParams
	}

	err := c.packer.Unmarshal(*c.req.Params, &x)
	if err != nil {
		utils.DefaultLogger.Errorf("[%d]ReadRequestBody failed:%v", c.GoID, err)
		c.writeMessage(-1, fmt.Sprintf("read params failed:%v", err))
		return errInvalidParams
	}
	argv := reflect.ValueOf(x)
	for i := 0; i < argv.Elem().NumField(); i++ {
		f := argv.Elem().Type().Field(i)
		fieldValue := argv.Elem().Field(i)
		if f.Name == "Common" {
			timeoutField := fieldValue.FieldByName("Timeout")
			if timeoutField.CanInterface() {
				timeout := timeoutField.Int()
				if timeout > 0 {
					time.AfterFunc(time.Duration(timeout*int64(time.Millisecond)), func() {
						if c.done {
							// 已经正常返回
							return
						}
						c.writeMessage(-10, fmt.Sprintf("timeout after %d ms", timeout))
						utils.DefaultLogger.Warnf("[%d] got timeout after %dms", c.GoID, timeout)
						c.timeout = true
						c.Close()
					})
				}
			} else {
				utils.WriteLogWithGoID("info", c.GoID, "ERROR: timeout not found\n")
			}
		}
	}
	c.params = x
	return err
}

func (c *YarServerCodec) WriteResponse(r *Response, x interface{}) error {
	c.mutex.Lock()
	id, ok := c.pending[r.Seq]
	if !ok {
		c.mutex.Unlock()
		return errors.New("invalid sequence number in response")
	}

	delete(c.pending, r.Seq)
	c.mutex.Unlock()

	resp := serverResponse{
		Id:     int64(id),
		Error:  "",
		Result: nil,
		Output: "",
		Status: 0,
	}

	// FIXME: 如果某个字段是空的map，php端会得到class，是否可以将map转成array？
	if r.Error == "" {
		resp.Result = &x
	} else {
		resp.Error = r.Error
	}

	msg := "success"
	Id := (uint32)(resp.Id)
	var err error
	if !c.timeout {
		err = writePack(c.w, c.packer, Id, &resp)
		c.done = true
		if err != nil {
			msg = "failed:" + err.Error()
		}
	}

	conn := c.c.(net.Conn)
	remoteAddr := conn.RemoteAddr().String()
	latency := time.Now().Sub(c.t).Seconds()*1000
	ctx := contexts.GetContext()
	timeoutMsg := "timeout=false"
	if c.timeout {
		timeoutMsg = "timeout=true"
	}
	utils.DefaultLogger.Infof("[%d] %vms %v REQUEST %v %v %v %s",
		c.GoID,
		latency,
		strings.Split(remoteAddr, ":")[0],
		r.ServiceMethod,
		msg,
		c.GetParamsLogString(),
		timeoutMsg)

	s := strings.Split(r.ServiceMethod, ".")
	if len(s) == 2 {
		ctx.AddRpcMethodFinishStats(s[0], s[1], latency)
	} else {
		utils.DefaultLogger.Errorf("[%d] invalid rpc method:%s", c.GoID, r.ServiceMethod)
	}

	if err != nil {
		return err
	}
	return nil
}

func (c *YarServerCodec) GetParamsLogString() (s string) {
	maxParamLogLength := 50
	if c.params == nil {
		return
	}
	e := reflect.ValueOf(c.params).Elem()
	t := reflect.TypeOf(c.params).Elem()
	if t.Kind() != reflect.Struct {
		return
	}
	var ss []string
	for i := 0; i < e.NumField() && i < 8; i++ {
		paramName := t.Field(i).Name
		paramValue := fmt.Sprintf("%v", e.Field(i))
		if t.Field(i).Tag.Get("disable_log") == "1" {
			ss = append(ss, fmt.Sprintf("%s=...", paramName))
			continue
		}
		if len(paramValue) > maxParamLogLength && paramName != "Oid"{
			paramValue = paramValue[0:maxParamLogLength] + "..."
		}
		ss = append(ss, fmt.Sprintf("%s=%s", paramName, paramValue))
	}
	s = strings.Join(ss, " ")
	return
}

func (s *YarServerCodec) Close() error {
	s.c.Close()     // TODO: 为何这里注释掉？
	return nil
}

