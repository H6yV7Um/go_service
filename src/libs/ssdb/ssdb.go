/*
   部分代码（Client类）源自ssdb客户端代码（github.com/ssdb/gossdb），并做了改动。
*/
package ssdb

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"utils"
)

type Client struct {
	sock     *net.TCPConn
	recv_buf bytes.Buffer
}

type SsdbPool struct {
	p *Pool
}

func NewSsdbPool(host string, port string) (*SsdbPool, error) {
	return &SsdbPool{p: &Pool{
		MaxIdle:     256,
		MaxActive:   1000,
		IdleTimeout: 0,
		Dial: func() (c *Client, err error) {
			c, err = NewSsdb(host, port)
			return
		},
	}}, nil
}

func (this *SsdbPool)Stats() *utils.StorageStats {
	return &utils.StorageStats {
		ActiveConn:this.p.active,
	}
}

func (this *SsdbPool) Do(commandName string, args ...interface{}) (reply []string, err error) {
	c := this.p.Get()
	defer this.p.put(c, false)

	a := []interface{}{commandName}
	for _, v := range args {
		a = append(a, v)
	}
	return c.Do(a...)
}

func (this *SsdbPool) Ttl(key string) (result int, err error) {
	result = -1
	c := this.p.Get()
	defer this.p.put(c, false)
	args := []interface{}{"ttl", key}
	resp, err := c.Do(args...)
	if err != nil {
		return
	}
	if len(resp) == 0 || resp[0] != "ok" {
		err = fmt.Errorf("bad response")
		return
	}
	result, err = strconv.Atoi(resp[1])

	return
}

// ------------- Key Value --------------
func (this *SsdbPool) Set(key string, value string) error {
	c := this.p.Get()
	defer this.p.put(c, false)
	args := []interface{}{"set", key, value}
	_, err := c.Do(args...)
	return err
}

func (this *SsdbPool) Get(key string) (result string, err error) {
	c := this.p.Get()
	defer this.p.put(c, false)
	args := []interface{}{"get", key}
	resp, err := c.Do(args...)
	if err != nil {
		return
	}
	if len(resp) == 0 {
		err = fmt.Errorf("bad response")
		return
	}
	if resp[0] != "ok" {
		return
	}
	result = resp[1]

	return
}

func (this *SsdbPool) Del(key string) error {
	c := this.p.Get()
	defer this.p.put(c, false)
	args := []interface{}{"del", key}
	_, err := c.Do(args...)
	return err
}

func (this *SsdbPool) MultiSet(keys []string, values []string) error {
	c := this.p.Get()
	defer this.p.put(c, false)
	args := []interface{}{"multi_set"}
	if len(keys) != len(values) {
		return errors.New("key and value number not match")
	}
	for i, _ := range keys {
		args = append(args, keys[i])
		args = append(args, values[i])
	}
	_, err := c.Do(args...)
	return err
}

func (this *SsdbPool) MultiGet(keys []string) (result map[string]string, err error) {
	result = make(map[string]string)
	c := this.p.Get()
	defer this.p.put(c, false)
	args := []interface{}{"multi_get"}
	for _, value := range keys {
		args = append(args, value)
	}
	resp, err := c.Do(args...)
	if err != nil {
		return
	}
	if len(resp) == 0 {
		err = fmt.Errorf("bad response")
		return
	}
	if resp[0] != "ok" {
		return
	}
	for i := 1; i < len(resp); i += 2 {
		if i + 1 > len(resp) - 1 {
			break
		}
		result[resp[i]] = resp[i+1]
	}

	return
}

func (this *SsdbPool) Keys(start string, stop string, count int) (result []string, err error) {
	c := this.p.Get()
	defer this.p.put(c, false)
	resp, err := c.Do("keys", start, stop, count)
	if err != nil {
		return
	}
	if len(resp) == 0 {
		err = fmt.Errorf("bad response")
		return
	}
	if resp[0] != "ok" {
		return
	}
	for i := 1; i < len(resp); i++ {
		result = append(result, resp[i])
	}
	return
}

// --------------- Hashmap --------------------------

func (this *SsdbPool) HGetAll(key string) (result map[string]string, err error) {
	result = make(map[string]string)
	c := this.p.Get()
	defer this.p.put(c, false)
	args := []interface{}{"hgetall", key}
	resp, err := c.Do(args...)
	if err != nil {
		return
	}
	if len(resp) == 0 {
		err = fmt.Errorf("bad response")
		return
	}
	if resp[0] != "ok" {
		return
	}
	for i := 1; i < len(resp); i += 2 {
		if i + 1 > len(resp) - 1 {
			break
		}
		result[resp[i]] = resp[i+1]
	}

	return
}

// --------------- Sorted Set ------------------------

func (this *SsdbPool) ZSet(name string, key string, score int64) error {
	c := this.p.Get()
	defer this.p.put(c, false)
	args := []interface{}{"zset", name, key, score}
	_, err := c.Do(args...)
	return err
}

func (this *SsdbPool) ZGet(name string, key string) (result int, err error) {
	c := this.p.Get()
	defer this.p.put(c, false)
	args := []interface{}{"zget", name, key}
	var resp []string
	resp, err = c.Do(args...)
	if err != nil {
		return
	}
	if len(resp) < 2 {
		err = fmt.Errorf("bad response")
		return
	}
	if resp[0] != "ok" {
		return
	}
	result, _ = strconv.Atoi(resp[1])
	fmt.Printf("resp=%v\n", resp)

	return
}

func (this *SsdbPool) ZMultiSet(key string, score int64, values []string) error {
	c := this.p.Get()
	defer this.p.put(c, false)
	args := []interface{}{"multi_zset", key}
	for _, value := range values {
		args = append(args, value)
		args = append(args, score)
	}
	_, err := c.Do(args...)
	return err
}

func (this *SsdbPool) ZMultiGet(key string, fields []string) (result map[string]string, err error) {
	result = make(map[string]string)
	c := this.p.Get()
	defer this.p.put(c, false)
	args := []interface{}{"multi_zget", key}
	for _, value := range fields {
		args = append(args, value)
	}
	resp, err := c.Do(args...)
	if err != nil {
		return
	}
	if len(resp) == 0 {
		err = fmt.Errorf("bad response")
		return
	}
	if resp[0] != "ok" {
		return
	}
	for i := 1; i < len(resp); i += 2 {
		if i + 1 > len(resp) - 1 {
			break
		}
		result[resp[i]] = resp[i+1]
	}

	return
}

func (this *SsdbPool) Zrrange(key string, start int, count int) (result []string, err error) {
	c := this.p.Get()
	defer this.p.put(c, false)
	resp, err := c.Do("zrrange", key, start, count)
	if err != nil {
		return
	}
	if len(resp) == 0 {
		err = fmt.Errorf("bad response")
		return
	}
	if resp[0] != "ok" {
		return
	}
	for i := 1; i < len(resp); i += 2 {
		result = append(result, resp[i])
	}
	return
}

func (this *SsdbPool) Zrscan(key string, start string, stop string, count int) (result []string, scores []int, err error) {
	c := this.p.Get()
	defer this.p.put(c, false)
	resp, err := c.Do("zrscan", key, "", start, stop, count)
	if err != nil {
		return
	}
	if len(resp) == 0 {
		err = fmt.Errorf("bad response")
		return
	}
	if resp[0] != "ok" {
		return
	}
	for i := 1; i+1 < len(resp); i += 2 {
		result = append(result, resp[i])
		n, _ := strconv.Atoi(resp[i+1])
		scores = append(scores, n)
	}
	return
}

// return total count of a zset
func (this *SsdbPool) Zcount(key string) (count int, err error) {
	c := this.p.Get()
	defer this.p.put(c, false)
	resp, err := c.Do("zcount", key, "", "")
	if err != nil {
		return
	}
	if len(resp) < 2 {
		err = fmt.Errorf("bad response")
		return
	}
	if resp[0] != "ok" {
		return
	}

	ret, _ := strconv.ParseInt(resp[1], 10, 64)
	count = int(ret)
	return
}

// -------------- List -------------------------
