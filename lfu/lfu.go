package lfu

import (
	"errors"
	"sync"
)

type entry[K comparable, V any] struct {
	key       K
	value     V
	frequency uint

	// For maintaining order within frequency
	prev *entry[K, V]
	next *entry[K, V]
}

type frequencyNode struct {
	freq uint
	head any // *entry[K, V]
	tail any // *entry[K, V]
	prev *frequencyNode
	next *frequencyNode
}

type Cache[K comparable, V any] struct {
	data         map[K]*entry[K, V]
	frequencies  map[uint]*frequencyNode
	lock         sync.RWMutex
	capacity     uint
	minFrequency uint
}

func NewCache[K comparable, V any](capacity uint) (*Cache[K, V], error) {
	if capacity <= 0 {
		return nil, errors.New("must provide a positive size")
	}

	return &Cache[K, V]{
		data:         make(map[K]*entry[K, V], capacity),
		frequencies:  make(map[uint]*frequencyNode),
		capacity:     capacity,
		minFrequency: 0,
	}, nil
}

func (c *Cache[K, V]) Add(key K, value V) (evicted bool, rewritten bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if node, exists := c.data[key]; exists {
		node.value = value
		c.incrementFrequency(node)
		return false, true
	}

	if uint(len(c.data)) >= c.capacity {
		c.evictLeastFrequent()
		evicted = true
	}

	newNode := &entry[K, V]{
		key:       key,
		value:     value,
		frequency: 1,
	}
	c.data[key] = newNode
	c.addToFrequency(newNode, 1)
	c.minFrequency = 1

	return evicted, false
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if node, exists := c.data[key]; exists {
		c.incrementFrequency(node)
		return node.value, true
	}

	var zero V
	return zero, false
}

func (c *Cache[K, V]) Remove(key K) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if node, exists := c.data[key]; exists {
		c.removeFromFrequency(node)
		delete(c.data, key)
		return true
	}
	return false
}

func (c *Cache[K, V]) incrementFrequency(node *entry[K, V]) {
	oldFreq := node.frequency
	newFreq := oldFreq + 1

	c.removeFromFrequency(node)
	node.frequency = newFreq
	c.addToFrequency(node, newFreq)

	if oldFreq == c.minFrequency && c.frequencies[oldFreq] == nil {
		c.minFrequency = newFreq
	}
}

func (c *Cache[K, V]) addToFrequency(node *entry[K, V], freq uint) {
	freqNode, exists := c.frequencies[freq]
	if !exists {
		freqNode = &frequencyNode{freq: freq}
		c.frequencies[freq] = freqNode
	}

	if freqNode.head == nil {
		freqNode.head = node
		freqNode.tail = node
		return
	}

	node.next = freqNode.head.(*entry[K, V])
	freqNode.head.(*entry[K, V]).prev = node
	freqNode.head = node
}

func (c *Cache[K, V]) removeFromFrequency(node *entry[K, V]) {
	freqNode := c.frequencies[node.frequency]
	if freqNode == nil {
		return
	}

	if node.prev != nil {
		node.prev.next = node.next
	} else {
		freqNode.head = node.next
	}

	if node.next != nil {
		node.next.prev = node.prev
	} else {
		freqNode.tail = node.prev
	}

	if freqNode.head == nil {
		delete(c.frequencies, node.frequency)
	}

	node.prev = nil
	node.next = nil
}

func (c *Cache[K, V]) evictLeastFrequent() {
	if len(c.data) == 0 {
		return
	}

	freqNode := c.frequencies[c.minFrequency]
	if freqNode == nil || freqNode.tail == nil {
		return
	}

	leastFreqNode := freqNode.tail.(*entry[K, V])
	c.removeFromFrequency(leastFreqNode)
	delete(c.data, leastFreqNode.key)
}
