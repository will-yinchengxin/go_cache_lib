package local_cache

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	ce := NewCache(time.Second*2, time.Second*4)
	ce.cache.OnEvicted(func(s string, a any) {
		t.Log("delete", s)
	})

	ce.Set("name", "will", time.Second*2)
	ce.Delete("name")

	ce.Set("age", 13, DefaultExpire)
	time.Sleep(time.Second * 4)
	t.Log(ce.Get("age"))

	ce.Set("sex", "man", DefaultExpire)
	time.Sleep(time.Second * 5)
	t.Log(ce.Get("sex"))
	t.Log(ce.items)
}

func TestCahceWithOutJanitor(t *testing.T) {
	ce := NewCache(time.Second*2, 0)
	ce.cache.OnEvicted(func(s string, a any) {
		t.Log("delete", s)
	})
	ce.Set("sex", "man", DefaultExpire)
	time.Sleep(time.Second * 5)
	t.Log(ce.Get("sex"))
	t.Log(ce.items)
}
