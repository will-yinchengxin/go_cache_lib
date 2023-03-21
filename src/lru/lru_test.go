package lru

import (
	"testing"
)

func TestLRU(t *testing.T) {
	lruCache := Constructor(2)

	lruCache.Put(1, 1)
	lruCache.Put(2, 2)
	t.Log(lruCache.Get(1)) // 1

	lruCache.Put(3, 3)
	t.Log(lruCache.Get(2)) // -1

	lruCache.Put(4, 4)
	t.Log(lruCache.Get(1)) // -1
	t.Log(lruCache.Get(3)) // 3
	t.Log(lruCache.Get(4)) // 4
}
