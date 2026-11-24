package cache

import (
	"container/list"
	"errors"
	"sync"
)

var ErrInvalidCapacity = errors.New("cache: capacity must be positive")

type entry[K comparable, V any] struct {
	key   K
	value V
}

type LRU[K comparable, V any] struct {
	mu       sync.Mutex
	capacity int
	entries  map[K]*list.Element
	order    *list.List
}

var _ Cache[string, int] = (*LRU[string, int])(nil)

// NewLRU creates a cache that keeps at most capacity entries.
func NewLRU[K comparable, V any](capacity int) (*LRU[K, V], error) {
	if capacity <= 0 {
		return nil, ErrInvalidCapacity
	}

	return &LRU[K, V]{
		capacity: capacity,
		entries:  make(map[K]*list.Element, capacity),
		order:    list.New(),
	}, nil
}

func (c *LRU[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.entries[key]
	if !ok {
		var zero V
		return zero, false
	}

	c.order.MoveToFront(e)
	return e.Value.(entry[K, V]).value, true
}

func (c *LRU[K, V]) Put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.entries[key]; ok {
		e.Value = entry[K, V]{key: key, value: value}
		c.order.MoveToFront(e)
		return
	}

	if c.order.Len() == c.capacity {
		oldest := c.order.Back()
		delete(c.entries, oldest.Value.(entry[K, V]).key)
		c.order.Remove(oldest)
	}

	c.entries[key] = c.order.PushFront(entry[K, V]{key: key, value: value})
}

func (c *LRU[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.entries[key]
	if !ok {
		return
	}

	delete(c.entries, key)
	c.order.Remove(e)
}

func (c *LRU[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.order.Len()
}

// Keys returns keys from most recently used to least recently used.
func (c *LRU[K, V]) Keys() []K {
	c.mu.Lock()
	defer c.mu.Unlock()

	keys := make([]K, 0, c.order.Len())
	for e := c.order.Front(); e != nil; e = e.Next() {
		keys = append(keys, e.Value.(entry[K, V]).key)
	}
	return keys
}
