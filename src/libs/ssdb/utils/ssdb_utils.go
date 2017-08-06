package main

import (
	"fmt"
	"contexts"
	"flag"
	"strconv"
	"time"
)

func main() {
	confDir := flag.String("confdir", "./conf", "config files diretory")
	start := flag.Int("start", 1, "start uid")
	flag.Parse()
	ctx := contexts.NewContext(*confDir)
	w, err := ctx.Ssdb("subscribe_ssdb_w", 0)
	if err != nil {
		fmt.Printf("failed to get ssdb instance:%v\n", err)
		return
	}
	keyStop := "WCH:_"
	total := 0
	for {
		keyStart := fmt.Sprintf("WCH:%d", *start)
		keys, err := w.Keys(keyStart, keyStop, 10)
		if err != nil {
			fmt.Printf("failed to get ssdb keys:%v\n", err)
			return
		}
		var ttl int
		for _, key := range(keys) {
			ttl, err = w.Ttl(key)
			if err != nil {
				fmt.Printf("get ttl for %s failed:%v", key, err)
				continue
			}
			removed := ""
			if ttl == -1 {
				err = w.Del(key)
				if err != nil {
					removed = fmt.Sprintf("\tremove failed:%v", err)
				} else {
					removed = "\tremoved"
				}
			}
			fmt.Printf("[%d] %s\t%d\t%s\n", total, key, ttl, removed)
			total++
		}
		newStart, err := strconv.Atoi(keys[len(keys)-1][4:])
		if err != nil {
			fmt.Printf("got error:%v (%d)\n", err, newStart)
			*start+=100
			time.Sleep(time.Millisecond)
			continue
		}
		*start = newStart
	}
	fmt.Printf("done total=%d\n", total)
}
