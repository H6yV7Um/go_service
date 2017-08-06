package ssdb

import (
	"testing"
	"runtime"
	"fmt"
)

func TestKeyValue(t *testing.T) {
	db, err := NewSsdbPool("localhost", "8888")
	if err != nil {
		t.Fatalf("New ssdb failed:%v", err)
	}
	key := "test"
	err = db.Set(key, "123")
	if err != nil {
		t.Fatalf("Ssdb set failed:%v", err)
	}
	value, err := db.Get(key)
	if err != nil {
		t.Fatalf("Ssdb get failed:%v", err)
	}
	if value != "123" {
		t.Fatalf("Ssdb get value not match:%v", value)
	}
}

func TestSortedSet(t *testing.T) {
	fmt.Printf("lalala\n")
	for i:=0; i<10; i++ {
		a, b, c, d := runtime.Caller(i)
		t.Logf("%v, %v, %v, %v", a, b, c, d)
	}
	db, err := NewSsdbPool("localhost", "8888")
	if err != nil {
		t.Fatalf("New ssdb failed:%v", err)
	}
	name := "test"
	err = db.ZSet(name, "key1", 100)
	if err != nil {
		t.Fatalf("Ssdb zset failed:%v", err)
	}
	score, err := db.ZGet(name, "key1")
	if err != nil {
		t.Fatalf("Ssdb get failed:%v", err)
	}
	if score != 100 {
		t.Fatalf("Ssdb zget score not match:%v", score)
	}
}

