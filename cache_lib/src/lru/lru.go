package lru

import "sync"

/*
* @package src/lru/lru.go
* @author：Will Yin <826895143@qq.com>
* @copyright Copyright (C) 2023/3/21 Will

LRU（Least Recently Used）算法是一种常用的缓存淘汰算法。它基于“最近最少使用”的思想，即最近访问时间越早的数据越可能长时间不再被访问，
因此应该被淘汰。为了支持这种淘汰策略，我们需要一个数据结构来记录每个键值对的访问时间，并能快速找出最近最少使用的数据。

常见的数据结构有 哈希表、链表、双向链表 等。哈希表可以用来快速查找每个键对应的节点，但无法方便地维护访问时间。链表可以用来维护访问时间，
但无法快速查找每个键对应的节点。因此，我们可以将这两种数据结构结合起来使用，使用哈希表来快速查找每个键对应的节点，使用链表来维护访问时间。

在这个实现中，我们使用了双向链表来维护访问时间，每个节点包含了key和value字段以及前驱prev和后继next指针。
LRUCache结构体代表整个缓存，包含了容量capacity、缓存cache、头部指针head和尾部指针tail。
其中，cache是一个哈希表，用来快速查找每个键对应的节点。
*/

const (
	DefaultCapacity = 16
)

type node struct {
	key   int
	value int
	prev  *node
	next  *node
}

type LRUCache struct {
	lock      sync.RWMutex
	onEvicted func(node)
	capacity  int
	cache     map[int]*node
	head      *node
	tail      *node
}

func Constructor(capacity int) *LRUCache {
	if capacity <= 0 {
		capacity = DefaultCapacity
	}
	return &LRUCache{
		lock:     sync.RWMutex{},
		capacity: capacity,
		cache:    make(map[int]*node),
		head:     nil,
		tail:     nil,
	}
}

func ConstructorWithEvicted(onEvicted func(node), capacity int) *LRUCache {
	if capacity <= 0 {
		capacity = DefaultCapacity
	}
	return &LRUCache{
		onEvicted: onEvicted,
		lock:      sync.RWMutex{},
		capacity:  capacity,
		cache:     make(map[int]*node),
		head:      nil,
		tail:      nil,
	}
}

// Get 获取元素
func (this *LRUCache) Get(key int) int {
	this.lock.RLock()
	getNode, ok := this.cache[key]
	this.lock.RUnlock()
	if ok {
		this.remove(getNode)
		this.addToHead(getNode)
		return getNode.value
	}
	return -1
}

// Put 添加元素
func (this *LRUCache) Put(key int, value int) {
	if nodeNew, ok := this.cache[key]; ok {
		// 如果key已存在，更新其值并移到头部
		nodeNew.value = value
		this.remove(nodeNew)
		this.addToHead(nodeNew)
	} else {
		// 如果key不存在，创建新节点并添加到头部
		nodeNew = &node{
			key:   key,
			value: value,
			prev:  nil,
			next:  nil,
		}
		// 如果容量已满，删除尾部节点
		if len(this.cache) == this.capacity {
			delete(this.cache, this.tail.key)
			this.remove(this.tail)
		}
		this.addToHead(nodeNew)
		this.cache[key] = nodeNew
	}
	return
}

func (this *LRUCache) Len() int {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return len(this.cache)
}

// remove 移除元素
func (this *LRUCache) remove(node *node) {
	this.lock.Lock()
	defer this.lock.Unlock()
	if node.prev == nil { // 如果节点是头部节点，更新头部指针
		this.head = node.next
	} else {
		node.prev.next = node.next
	}
	if node.next == nil { // 如果节点是尾部节点，更新尾部指针
		this.tail = node.prev
	} else {
		node.next.prev = node.prev
	}
	return
}

// addToHead 将元素添加至头部
func (this *LRUCache) addToHead(node *node) {
	this.lock.Lock()
	defer this.lock.Unlock()
	node.prev = nil
	node.next = this.head
	if this.head != nil {
		this.head.prev = node
	}
	this.head = node
	if this.tail == nil {
		this.tail = node
	}
	return
}
