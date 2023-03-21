/*
 * Auth：Will Yin
 * Date：2023/3/15 16:00

The package provides the following methods on cache:

	Set: Sets an item in the cache with an expiration time.
	SetDefault: Sets an item in the cache with the default expiration time.
	SetNoExpire: Sets an item in the cache with no expiration time.
	Replace: Replaces an item in the cache with a new one.
	Get: Gets an item from the cache.
	GetWithExpire: Gets an item from the cache with its expiration time.
	Delete: Deletes an item from the cache.
	DeleteExpired: Deletes all expired items from the cache.
	WithCallBack: Sets a callback function to be called when an item is deleted from the cache.
	Flush: Clears all items from the cache.
	ItemCount: Returns the number of items in the cache.

The janitor struct has a runJanitor method which runs a goroutine that periodically checks for expired items and deletes them.
*/

package local_cache

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

const (
	NoExpire time.Duration = -1

	DefaultExpire time.Duration = 0
)

type Object struct {
	key string
	val any
}

type Item struct {
	Obj        any
	ExpireTime int64
}

func (i *Item) Expired() bool {
	if i.ExpireTime == 0 {
		return false
	}
	return time.Now().Unix() > i.ExpireTime
}

type cache struct {
	defaultExpire time.Duration
	items         map[string]Item
	lock          sync.RWMutex
	onEvicted     func(string, any)
	*janitor
}

func newCache(d time.Duration, items map[string]Item) *cache {
	if d <= 0 {
		d = -1
	}
	return &cache{
		items:         items,
		defaultExpire: d,
	}
}

func (c *cache) Set(k string, v any, d time.Duration) {
	if d == DefaultExpire {
		d = c.defaultExpire
	}
	var e int64
	if d > 0 {
		e = time.Now().Add(d).Unix()
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.items[k] = Item{
		Obj:        v,
		ExpireTime: e,
	}
}

func (c *cache) SetDefault(k string, v any) {
	c.Set(k, v, DefaultExpire)
}

func (c *cache) SetNoExpire(k string, v any) {
	c.Set(k, v, NoExpire)
}

func (c *cache) Replace(k string, v any, d time.Duration) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if !c.exist(k) {
		return fmt.Errorf("Item %s doesn't exist", k)
	}
	// In this way, there is lock competition, and you can get the release and get it
	//c.Set(k, v, d)

	c.set(k, v, d)
	return nil
}

func (c *cache) set(k string, v any, d time.Duration) {
	if d == DefaultExpire {
		d = c.defaultExpire
	}
	var e int64
	if d > 0 {
		e = time.Now().Add(d).Unix()
	}
	c.items[k] = Item{
		Obj:        v,
		ExpireTime: e,
	}
}

func (c *cache) exist(k string) bool {
	_, ok := c.items[k]
	return ok
}

func (c *cache) Get(k string) (any, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	item, ok := c.items[k]
	if !ok {
		return nil, false
	}
	if item.ExpireTime > 0 {
		if time.Now().Unix() > item.ExpireTime {
			return nil, false
		}
	}
	return item.Obj, true
}

func (c *cache) GetWithExpire(k string) (any, time.Time, bool) {
	c.lock.Lock()
	defer c.lock.RUnlock()
	item, ok := c.items[k]
	if !ok {
		return nil, time.Time{}, false
	}
	if item.ExpireTime > 0 {
		if time.Now().Unix() > item.ExpireTime {
			return nil, time.Time{}, false
		}
		return item.Obj, time.Unix(0, item.ExpireTime), true
	}
	return item.Obj, time.Time{}, true
}

func (c *cache) Delete(k string) {
	c.lock.Lock()
	v, hasCallBack := c.delete(k)
	c.lock.Unlock()
	if hasCallBack {
		c.onEvicted(k, v)
	}
}

func (c *cache) delete(k string) (any, bool) {
	defer delete(c.items, k)
	if c.onEvicted != nil {
		val, ok := c.items[k]
		if ok {
			return val, true
		}
	}
	return nil, false
}

func (c *cache) DeleteExpired() {
	var (
		callBackObj []Object
		now         = time.Now().Unix()
	)
	c.lock.Lock()
	for key, val := range c.items {
		if val.ExpireTime > 0 && now > val.ExpireTime {
			v, hasCallBack := c.delete(key)
			if hasCallBack {
				callBackObj = append(callBackObj, Object{key: key, val: v})
			}
		}
	}
	c.lock.Unlock()
	if c.onEvicted != nil {
		for _, val := range callBackObj {
			c.onEvicted(val.key, val.val)
		}
	}
}

func (c *cache) OnEvicted(fun func(string, any)) {
	c.lock.Lock()
	c.onEvicted = fun
	c.lock.Unlock()
}

func (c *cache) Flush() {
	c.lock.Lock()
	c.items = map[string]Item{}
	c.lock.Unlock()
}

func (c *cache) ItemCount() int {
	c.lock.RLock()
	n := len(c.items)
	c.lock.RUnlock()
	return n
}

type janitor struct {
	Interval time.Duration
	stop     chan struct{}
}

func initJanitor(interval time.Duration, c *cache) {
	if interval > 0 {
		c.janitor = &janitor{
			Interval: interval,
			stop:     make(chan struct{}),
		}
		runtime.SetFinalizer(c, StopJanitor)
		go c.janitor.runJanitor(c)
	}
}

func (j *janitor) runJanitor(c *cache) {
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

func StopJanitor(c *cache) {
	c.janitor.stop <- struct{}{}
}

type Cache struct {
	*cache
}

func NewCache(defaultExpiration, cleanupInterval time.Duration) *Cache {
	items := make(map[string]Item)
	c := newCache(defaultExpiration, items)
	C := &Cache{
		c,
	}
	if cleanupInterval > 0 {
		initJanitor(cleanupInterval, c)
	}
	return C
}

func NewCacheWithItems(defaultExpiration, cleanupInterval time.Duration, items map[string]Item) *Cache {
	c := newCache(defaultExpiration, items)
	C := &Cache{
		c,
	}
	if cleanupInterval > 0 {
		initJanitor(cleanupInterval, c)
	}
	return C
}
