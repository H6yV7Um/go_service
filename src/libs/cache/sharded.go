package cache

import (
	"time"
	"os"
	"runtime"
	"math/big"
	"math"
	"crypto/rand"
	insecurerand "math/rand"
	"log"
)

const (
	defaultCleanupInterval = 60 * time.Second
)

type Cache struct {
	name    string
	seed    uint32
	m       uint32
	cs      []*cache
	janitor *shardedJanitor
}

type shardedJanitor struct {
	Interval time.Duration
	stop     chan bool
}

func (j *shardedJanitor) Run(sc *Cache) {
	j.stop = make(chan bool)
	tick := time.Tick(j.Interval)
	for {
		select {
		case <-tick:
			sc.DeleteExpired()
		case <-j.stop:
			return
		}
	}
}

func stopShardedJanitor(sc *Cache) {
	sc.janitor.stop <- true
}

func runShardedJanitor(sc *Cache, ci time.Duration) {
	log.Printf("sc runShardedJanitor\n")
	j := &shardedJanitor{
		Interval: ci,
	}
	sc.janitor = j
	go j.Run(sc)
}

func New(name string, shards int, maxItems int, defaultExpiration time.Duration) *Cache {
	max := big.NewInt(0).SetUint64(uint64(math.MaxUint32))
	rnd, err := rand.Int(rand.Reader, max)
	var seed uint32
	if err != nil {
		os.Stderr.Write([]byte("WARNING: go-cache's newShardedCache failed to read from the system CSPRNG (/dev/urandom or equivalent.) Your system's security may be compromised. Continuing with an insecure seed.\n"))
		seed = insecurerand.Uint32()
	} else {
		seed = uint32(rnd.Uint64())
	}
	if maxItems == 0 {
		maxItems = defaultMaxItems
	}
	sc := &Cache{
		name: name,
		seed: seed,
		m:    uint32(shards),
		cs:   make([]*cache, shards),
	}
	for i := 0; i < shards; i++ {
		c := &cache{
			defaultExpiration: defaultExpiration,
			items:             map[string]Item{},
		}
		c.name = name
		c.maxItems = maxItems / shards
		sc.cs[i] = c
	}
	// 随机对defaultCleanupInterval做微调，避免所有cache同时扫过期数据
	runShardedJanitor(sc, defaultCleanupInterval + time.Duration(insecurerand.Intn(10) - 5) * time.Second)
	runtime.SetFinalizer(sc, stopShardedJanitor)
	return sc
}

// djb2 with better shuffling. 5x faster than FNV with the hash.Hash overhead.
func djb33(seed uint32, k string) uint32 {
	var (
		l = uint32(len(k))
		d = 5381 + seed + l
		i = uint32(0)
	)
	// Why is all this 5x faster than a for loop?
	if l >= 4 {
		for i < l-4 {
			d = (d * 33) ^ uint32(k[i])
			d = (d * 33) ^ uint32(k[i+1])
			d = (d * 33) ^ uint32(k[i+2])
			d = (d * 33) ^ uint32(k[i+3])
			i += 4
		}
	}
	switch l - i {
	case 1:
	case 2:
		d = (d * 33) ^ uint32(k[i])
	case 3:
		d = (d * 33) ^ uint32(k[i])
		d = (d * 33) ^ uint32(k[i+1])
	case 4:
		d = (d * 33) ^ uint32(k[i])
		d = (d * 33) ^ uint32(k[i+1])
		d = (d * 33) ^ uint32(k[i+2])
	}
	return d ^ (d >> 16)
}

func (sc *Cache) bucket(k string) *cache {
	return sc.cs[djb33(sc.seed, k)%sc.m]
}

func (sc *Cache) Set(k string, x interface{}, d time.Duration) {
	sc.bucket(k).Set(k, x, d)
}

func (sc *Cache) HSet(k string, f string, v interface{}, d time.Duration) {
	sc.bucket(k).HSet(k, f, v, d)
}

func (sc *Cache) Get(k string) (interface{}, bool) {
	return sc.bucket(k).Get(k)
}

func (sc *Cache) HGet(k string, f string) (interface{}, bool) {
	return sc.bucket(k).HGet(k, f)
}

func (sc *Cache) Delete(k string) {
	sc.bucket(k).Delete(k)
}

func (sc *Cache) DeleteExpired() {
	log.Printf("start deleting expired, cache name=%s\n", sc.name)
	for _, v := range sc.cs {
		v.DeleteExpired()
	}
	log.Printf("finish deleting expired, cache name=%s\n", sc.name)
}

func (sc *Cache) Items() []map[string]Item {
	res := make([]map[string]Item, len(sc.cs))
	for i, v := range sc.cs {
		res[i] = v.Items()
	}
	return res
}

func (sc *Cache) Flush() {
	for _, v := range sc.cs {
		v.Flush()
	}
}

func (sc *Cache) Status() (total, expired int) {
	total = 0
	expired = 0
	for _, v := range sc.cs {
		t, e := v.Status()
		total += t
		expired += e
	}
	return
}