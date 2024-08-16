package lru

import (
	"container/list"
)

// todo: add thread-safeability

type LRUList[K comparable, V any] struct {
	cache map[K]*list.Element
	ll    *list.List
}

type Pair[K comparable, V any] struct {
	key   K
	value V
}

func NewLRUList[K comparable, V any]() *LRUList[K, V] {
	return &LRUList[K, V]{
		cache: make(map[K]*list.Element),
		ll:    list.New(),
	}
}

func (c *LRUList[K, V]) Get(key K) (V, bool) {
	if element, found := c.cache[key]; found {
		c.ll.MoveToFront(element)
		return element.Value.(*Pair[K, V]).value, true
	}
	var zeroValue V
	return zeroValue, false
}

func (c *LRUList[K, V]) Put(key K, value V) {
	if element, found := c.cache[key]; found {
		element.Value.(*Pair[K, V]).value = value
		c.ll.MoveToFront(element)
		return
	}

	pair := &Pair[K, V]{key, value}
	element := c.ll.PushFront(pair)
	c.cache[key] = element
}

func (c *LRUList[K, V]) Len() int {
	return len(c.cache)
}

func (c *LRUList[K, V]) Front() (K, V, bool) {
	element := c.ll.Front()
	if element != nil {
		pair := element.Value.(*Pair[K, V])
		return pair.key, pair.value, true
	}
	var zeroKey K
	var zeroValue V
	return zeroKey, zeroValue, false
}

func (c *LRUList[K, V]) Pop_front() (K, V, bool) {
	element := c.ll.Front()
	if element != nil {
		c.ll.Remove(element)
		pair := element.Value.(*Pair[K, V])
		delete(c.cache, pair.key)
		return pair.key, pair.value, true
	}
	var zeroKey K
	var zeroValue V
	return zeroKey, zeroValue, false
}

func (c *LRUList[K, V]) Back() (K, V, bool) {
	element := c.ll.Back()
	if element != nil {
		pair := element.Value.(*Pair[K, V])
		return pair.key, pair.value, true
	}
	var zeroKey K
	var zeroValue V
	return zeroKey, zeroValue, false
}

func (c *LRUList[K, V]) Pop_back() (K, V, bool) {
	element := c.ll.Back()
	if element != nil {
		c.ll.Remove(element)
		pair := element.Value.(*Pair[K, V])
		delete(c.cache, pair.key)
		return pair.key, pair.value, true
	}
	var zeroKey K
	var zeroValue V
	return zeroKey, zeroValue, false
}

func (c *LRUList[K, V]) Remove(key K) bool {
	if element, found := c.cache[key]; found {
		c.ll.Remove(element)
		delete(c.cache, key)
		return true
	}
	return false
}
