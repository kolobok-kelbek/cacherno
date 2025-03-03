package lru

import (
	"errors"
	"sync"
)

type entry[K comparable, V any] struct {
	key   K
	value V

	prev *entry[K, V]
	next *entry[K, V]
}

type Cache[K comparable, V any] struct {
	data     map[K]*entry[K, V]
	lock     sync.RWMutex
	capacity uint
	head     *entry[K, V]
	tail     *entry[K, V]
}

func NewCache[K comparable, V any](capacity uint) (*Cache[K, V], error) {
	if capacity <= 0 {
		return nil, errors.New("must provide a positive size")
	}

	return &Cache[K, V]{
		data:     make(map[K]*entry[K, V], capacity),
		lock:     sync.RWMutex{},
		capacity: capacity,
	}, nil
}

func (c *Cache[K, V]) Add(key K, value V) (evicted bool, rewritten bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if node, has := c.data[key]; has {
		node.value = value
		c.moveToFront(node)
		return false, true
	}

	node := &entry[K, V]{
		key:   key,
		value: value,
	}
	c.data[key] = node
	c.addToFront(node)

	if uint(len(c.data)) > c.capacity {
		c.removeTail()
		return true, false
	}

	return false, false
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if node, has := c.data[key]; has {
		c.moveToFront(node)
		return node.value, true
	}

	var zero V
	return zero, false
}

func (c *Cache[K, V]) Remove(key K) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if node, has := c.data[key]; has {
		c.removeNode(node)
		delete(c.data, key)
		return true
	}
	return false
}

func (c *Cache[K, V]) moveToFront(node *entry[K, V]) {
	if node == c.head {
		return
	}
	c.removeFromList(node)
	c.addToFront(node)
}

func (c *Cache[K, V]) addToFront(node *entry[K, V]) {
	if c.head == nil {
		c.head = node
		c.tail = node
		return
	}
	node.next = c.head
	node.prev = nil
	c.head.prev = node
	c.head = node
}

func (c *Cache[K, V]) removeFromList(node *entry[K, V]) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		c.head = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		c.tail = node.prev
	}
}

func (c *Cache[K, V]) removeTail() {
	if c.tail != nil {
		delete(c.data, c.tail.key)
		c.removeFromList(c.tail)
	}
}

func (c *Cache[K, V]) removeNode(node *entry[K, V]) {
	c.removeFromList(node)
}
