// https://github.com/patrickmn/go-cache
package cache

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
	"utils"
	"container/list"
)

type Item struct {
	Object     interface{}
	Expiration int64
}

// Returns true if the item has expired.
func (item Item) Expired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

const (
	// For use with functions that take an expiration time.
	NoExpiration time.Duration = -1
	// For use with functions that take an expiration time. Equivalent to
	// passing in the same expiration duration as was given to New() or
	// NewFrom() when the cache was created (e.g. 5 minutes.)
	DefaultExpiration time.Duration = 0
	defaultMaxItems = 200000
	keysToBeDeletedOneTime = 100
)

type SingleCache struct {
	*cache
	// If this is confusing, see the comment at the bottom of New()
}

type cache struct {
	defaultExpiration time.Duration
	items             map[string]Item
	mu                sync.RWMutex
	onEvicted         func(string, interface{})
	janitor           *janitor
	maxItems		  int
	name              string
	keys              *list.List
	//muKeys            sync.RWMutex
}

type KeyType struct {
	key     string
	expire  int64
}

// Add an item to the cache, replacing any existing item. If the duration is 0
// (DefaultExpiration), the cache's default expiration time is used. If it is -1
// (NoExpiration), the item never expires.
func (c *cache) Set(k string, x interface{}, d time.Duration) {
	if len(c.items) > c.maxItems {
		// 防止cache占用内存过多
		return
	}

	// "Inlining" of set
	var e int64
	if d == DefaultExpiration {
		d = c.defaultExpiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}
	c.mu.Lock()
	c.items[k] = Item{
		Object:     x,
		Expiration: e,
	}
	// TODO: Calls to mu.Unlock are currently not deferred because defer
	// adds ~200 ns (as of go1.)
	c.mu.Unlock()

	/*c.muKeys.Lock()
	c.keys.PushBack(KeyType{k, e})
	c.muKeys.Unlock()*/
	
	//fmt.Printf("Set Cache:%s expire=%v", k, d.Seconds())
}

// Get an item from the cache. Returns the item or nil, and a bool indicating
// whether the key was found.
func (c *cache) Get(k string) (interface{}, bool) {
	c.mu.RLock()
	// "Inlining" of get and Expired
	item, found := c.items[k]
	if !found {
		c.mu.RUnlock()
		return nil, false
	}
	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			c.mu.RUnlock()
			c.Delete(k)
			return nil, false
		}
	}
	c.mu.RUnlock()
	return item.Object, true
}

func (c *cache) HGet(k string, f string) (interface{}, bool) {
	item, ok := c.Get(k)
	if !ok {
		return nil, false
	}
	m, ok := item.(map[string]interface{})
	if !ok {
		return nil, false
	}
	c.mu.RLock()
	v, ok := m[f]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return v, true
}

func (c *cache) HSet(k string, f string, v interface{}, d time.Duration) {
	var value map[string]interface{}
	item, ok := c.Get(k)
	if !ok {
		value = make(map[string]interface{})
	} else {
		value, ok = item.(map[string]interface{})
		if !ok {
			value = make(map[string]interface{})
		}
	}
	value[f] = v
	c.Set(k, value, d)
}

// Delete an item from the cache. Does nothing if the key is not in the cache.
func (c *cache) Delete(k string) {
	c.mu.Lock()
	v, evicted := c.delete(k)
	c.mu.Unlock()
	if evicted {
		c.onEvicted(k, v)
	}
}

func (c *cache) delete(k string) (interface{}, bool) {
	if c.onEvicted != nil {
		if v, found := c.items[k]; found {
			delete(c.items, k)
			return v.Object, true
		}
	}
	delete(c.items, k)
	return nil, false
}

type keyAndValue struct {
	key   string
	value interface{}
}

// Delete all expired items from the cache.
func (c *cache) DeleteExpired() {
	var expiredKeys []string
	now := time.Now().UnixNano()
	//total := len(c.items)
	for k, v := range c.items {
		if v.Expiration > 0 && now > v.Expiration {
			expiredKeys = append(expiredKeys, k)
		}
	}

	var lockedTime int64 = 0
	var evictedItems []*keyAndValue
	for i := 0; i < len(expiredKeys); {
		t1 := time.Now().UnixNano()
		stop := utils.Min(i + keysToBeDeletedOneTime, len(expiredKeys))
		c.mu.Lock()
		for ; i < stop; i++ {
			k := expiredKeys[i]
			ov, evicted := c.delete(k)
			if evicted {
				evictedItems = append(evictedItems, &keyAndValue{key:k, value:ov})
			}
		}
		c.mu.Unlock()
		lockedTime += time.Now().UnixNano() - t1
	}

	for _, v := range evictedItems {
		c.onEvicted(v.key, v.value)
	}
	//log.Printf("delete %s expired, time_used=%d lock_time=%d deleted=%d total=%d\n",
	//	c.name, time.Now().UnixNano()-now, lockedTime, len(expiredKeys), total)
}

// not used
func (c *cache) DeleteExpiredNew() {
	//var expiredKeys []string
	var evictedItems []*keyAndValue
	var next *list.Element
	now := time.Now().UnixNano()
	total := c.keys.Len()
	
	for i := 0; i < total; i += keysToBeDeletedOneTime {
		c.mu.Lock()
		e := c.keys.Front()
		for e != nil{
			next = e.Next()
			item := e.Value.(KeyType)
			if item.expire > 0 && now > item.expire {
				k := item.key
				ov, evicted := c.delete(k)
				if evicted {
					evictedItems = append(evictedItems,	&keyAndValue{key:k, value:ov})
				}
				c.keys.Remove(e)
			} else {
				c.keys.MoveToBack(e)
			}
			e = next
		}
		c.mu.Unlock()
	}
	for _, v := range evictedItems {
		c.onEvicted(v.key, v.value)
	}
	//log.Printf("cache delete expired, time_used=%d deleted=%d total=%d\n",
	//	time.Now().UnixNano()-now, len(expiredKeys), total)
}

func (c *cache) Status() (total, expired int) {
	total = len(c.items)
	expired = 0
	now := time.Now().UnixNano()
	for _, v := range c.items {
		if v.Expiration > 0 && now > v.Expiration {
			expired += 1
		}
	}
	return
}

// Sets an (optional) function that is called with the key and value when an
// item is evicted from the cache. (Including when it is deleted manually, but
// not when it is overwritten.) Set to nil to disable.
func (c *cache) OnEvicted(f func(string, interface{})) {
	c.mu.Lock()
	c.onEvicted = f
	c.mu.Unlock()
}

// Write the cache's items (using Gob) to an io.Writer.
//
// NOTE: This method is deprecated in favor of c.Items() and NewFrom() (see the
// documentation for NewFrom().)
func (c *cache) Save(w io.Writer) (err error) {
	enc := gob.NewEncoder(w)
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("Error registering item types with Gob library")
		}
	}()
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, v := range c.items {
		gob.Register(v.Object)
	}
	err = enc.Encode(&c.items)
	return
}

// Save the cache's items to the given filename, creating the file if it
// doesn't exist, and overwriting it if it does.
//
// NOTE: This method is deprecated in favor of c.Items() and NewFrom() (see the
// documentation for NewFrom().)
func (c *cache) SaveFile(fname string) error {
	fp, err := os.Create(fname)
	if err != nil {
		return err
	}
	err = c.Save(fp)
	if err != nil {
		fp.Close()
		return err
	}
	return fp.Close()
}

// Add (Gob-serialized) cache items from an io.Reader, excluding any items with
// keys that already exist (and haven't expired) in the current cache.
//
// NOTE: This method is deprecated in favor of c.Items() and NewFrom() (see the
// documentation for NewFrom().)
func (c *cache) Load(r io.Reader) error {
	dec := gob.NewDecoder(r)
	items := map[string]Item{}
	err := dec.Decode(&items)
	if err == nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		for k, v := range items {
			ov, found := c.items[k]
			if !found || ov.Expired() {
				c.items[k] = v
			}
		}
	}
	return err
}

// Load and add cache items from the given filename, excluding any items with
// keys that already exist in the current cache.
//
// NOTE: This method is deprecated in favor of c.Items() and NewFrom() (see the
// documentation for NewFrom().)
func (c *cache) LoadFile(fname string) error {
	fp, err := os.Open(fname)
	if err != nil {
		return err
	}
	err = c.Load(fp)
	if err != nil {
		fp.Close()
		return err
	}
	return fp.Close()
}

// Returns the items in the cache. This may include items that have expired,
// but have not yet been cleaned up. If this is significant, the Expiration
// fields of the items should be checked. Note that explicit synchronization
// is needed to use a cache and its corresponding Items() return value at
// the same time, as the map is shared.
func (c *cache) Items() map[string]Item {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.items
}

// Returns the number of items in the cache. This may include items that have
// expired, but have not yet been cleaned up. Equivalent to len(c.Items()).
func (c *cache) ItemCount() int {
	c.mu.RLock()
	n := len(c.items)
	c.mu.RUnlock()
	return n
}

// Delete all items from the cache.
func (c *cache) Flush() {
	c.mu.Lock()
	c.items = map[string]Item{}
	c.mu.Unlock()
}

type janitor struct {
	Interval time.Duration
	stop     chan bool
}

func (j *janitor) Run(c *cache) {
	log.Printf("run janitor in cache...\n")
	j.stop = make(chan bool)
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			c.DeleteExpired()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}

func stopJanitor(c *SingleCache) {
	c.janitor.stop <- true
}

func runJanitor(c *cache, ci time.Duration) {
	j := &janitor{
		Interval: ci,
	}
	c.janitor = j
	go j.Run(c)
}

func newCache(de time.Duration, m map[string]Item) *cache {
	if de == 0 {
		de = -1
	}
	c := &cache{
		defaultExpiration: de,
		items:             m,
	}
	return c
}

func newCacheWithJanitor(de time.Duration, ci time.Duration, m map[string]Item) *SingleCache {
	log.Printf("new cache...\n")
	c := newCache(de, m)
	c.keys = list.New()
	
	// This trick ensures that the janitor goroutine (which--granted it
	// was enabled--is running DeleteExpired on c forever) does not keep
	// the returned C object from being garbage collected. When it is
	// garbage collected, the finalizer stops the janitor goroutine, after
	// which c can be collected.
	C := &SingleCache{c}
	if ci > 0 {
		runJanitor(c, ci)
		runtime.SetFinalizer(C, stopJanitor)
	}
	return C
}

// Return a new cache with a given default expiration duration and cleanup
// interval. If the expiration duration is less than one (or NoExpiration),
// the items in the cache never expire (by default), and must be deleted
// manually. If the cleanup interval is less than one, expired items are not
// deleted from the cache before calling c.DeleteExpired().
func NewSingleCache(name string, maxItems int, defaultExpiration time.Duration, cleanupInterval time.Duration) *SingleCache {
	items := make(map[string]Item)
	cache := newCacheWithJanitor(defaultExpiration, cleanupInterval, items)
	cache.name = name
	if maxItems == 0 {
		maxItems = defaultMaxItems
	}
	cache.maxItems = maxItems
	return cache
}

// Return a new cache with a given default expiration duration and cleanup
// interval. If the expiration duration is less than one (or NoExpiration),
// the items in the cache never expire (by default), and must be deleted
// manually. If the cleanup interval is less than one, expired items are not
// deleted from the cache before calling c.DeleteExpired().
//
// NewFrom() also accepts an items map which will serve as the underlying map
// for the cache. This is useful for starting from a deserialized cache
// (serialized using e.g. gob.Encode() on c.Items()), or passing in e.g.
// make(map[string]Item, 500) to improve startup performance when the cache
// is expected to reach a certain minimum size.
//
// Only the cache's methods synchronize access to this map, so it is not
// recommended to keep any references to the map around after creating a cache.
// If need be, the map can be accessed at a later point using c.Items() (subject
// to the same caveat.)
//
// Note regarding serialization: When using e.g. gob, make sure to
// gob.Register() the individual types stored in the cache before encoding a
// map retrieved with c.Items(), and to register those same types before
// decoding a blob containing an items map.
func NewFrom(defaultExpiration, cleanupInterval time.Duration, items map[string]Item) *SingleCache {
	return newCacheWithJanitor(defaultExpiration, cleanupInterval, items)
}
