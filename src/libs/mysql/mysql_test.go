package mysql

import (
	"sync"
	"testing"
	"time"
)

var (
	wg sync.WaitGroup
)

func initDb() (*MySQL, error) {
	dsn := "root:@tcp(localhost:3306)/test"
	return NewMySQL(dsn)
}

func testPoolGet(i int, p *Pool, t *testing.T) {
	defer wg.Add(-1)
	t.Logf("pool test %d start", i)
	db, err := p.Get()
	defer db.Release()
	if err != nil {
		t.Fatalf("pool get failed")
	}
	vs := map[string]interface{}{
		"id":  time.Now().Unix(),
		"ttt": time.Now().String(),
	}
	_, err = db.SetTable("t").Insert(vs)
	if err != nil {
		t.Fatalf("Insert Error:%v", err)
	}
	t.Logf("pool test %d end", i)
}

func TestPool(t *testing.T) {
	dsn := "root:@tcp(localhost:3306)/test"
	poolsize := 10
	p := NewPool(dsn, poolsize)
	for i := 0; i < 15; i = i + 1 {
		wg.Add(1)
		go testPoolGet(i, p, t)
	}
	wg.Wait()
}

func TestInsert(t *testing.T) {
	db, err := initDb()
	vs := map[string]interface{}{
		"id":  time.Now().Unix(),
		"ttt": time.Now().String(),
	}
	_, err = db.SetTable("t").Insert(vs)
	if err != nil {
		t.Fatalf("Insert Error:%v", err)
	}
}

func TestUpdate(t *testing.T) {
	db, _ := initDb()
	for i := 0; i < 101; i++ {
		w := []*ConditionOpt{
			NewConditionOpt("id", "=", 100),
		}
		vs := map[string]interface{}{
			//"id":  time.Now().Unix(),
			"ttt": time.Now().String(),
		}
		_, err := db.SetTable("t").Where(w).Update(vs)
		if err != nil {
			t.Fatalf("Update Error:%v", err)
		}
	}
}

func TestDelete(t *testing.T) {
	db, _ := initDb()
	w := []*ConditionOpt{
		NewConditionOpt("id", "=", 3),
	}
	_, err := db.SetTable("t").Where(w).Delete()
	if err != nil {
		t.Fatalf("Delete Error:%v", err)
	}
}

func TestRows(t *testing.T) {
	db, _ := initDb()
	w := []*ConditionOpt{}
	rs, err := db.SetTable("t").Where(w).Fields("ttt").Rows()
	if err != nil || len(rs) < 1 {
		t.Fatalf("Rows Error:%v, %v", err, rs)
	}
}

func TestRow(t *testing.T) {
	db, _ := initDb()
	w := []*ConditionOpt{
		NewConditionOpt("id", "=", 100),
	}
	r, e := db.SetTable("t").Where(w).Row()
	if e != nil || len(r) < 1 {
		//t.Fatalf("Row Error:%v, %v", e, r)
	}
}
