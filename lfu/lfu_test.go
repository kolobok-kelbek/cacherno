package lfu

import (
	"testing"
)

func TestNewLFUCache(t *testing.T) {
	tests := []struct {
		name     string
		capacity uint
		wantErr  bool
	}{
		{
			name:     "valid capacity",
			capacity: 5,
			wantErr:  false,
		},
		{
			name:     "zero capacity",
			capacity: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := NewCache[string, int](tt.capacity)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCache() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && cache == nil {
				t.Error("NewCache() returned nil cache without error")
			}
		})
	}
}

func TestLFUCache_Add(t *testing.T) {
	cache, _ := NewCache[string, int](2)

	tests := []struct {
		name        string
		key         string
		value       int
		wantEvicted bool
		wantRewrite bool
	}{
		{
			name:        "add first item",
			key:         "key1",
			value:       1,
			wantEvicted: false,
			wantRewrite: false,
		},
		{
			name:        "add second item",
			key:         "key2",
			value:       2,
			wantEvicted: false,
			wantRewrite: false,
		},
		{
			name:        "add third item (causes eviction)",
			key:         "key3",
			value:       3,
			wantEvicted: true,
			wantRewrite: false,
		},
		{
			name:        "rewrite existing item",
			key:         "key2",
			value:       22,
			wantEvicted: false,
			wantRewrite: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evicted, rewritten := cache.Add(tt.key, tt.value)
			if evicted != tt.wantEvicted {
				t.Errorf("Add() evicted = %v, want %v", evicted, tt.wantEvicted)
			}
			if rewritten != tt.wantRewrite {
				t.Errorf("Add() rewritten = %v, want %v", rewritten, tt.wantRewrite)
			}
		})
	}
}

func TestLFUCache_Get(t *testing.T) {
	cache, _ := NewCache[string, int](2)
	cache.Add("key1", 1)
	cache.Add("key2", 2)

	tests := []struct {
		name      string
		key       string
		wantValue int
		wantFound bool
	}{
		{
			name:      "get existing item",
			key:       "key1",
			wantValue: 1,
			wantFound: true,
		},
		{
			name:      "get non-existing item",
			key:       "key3",
			wantValue: 0,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, found := cache.Get(tt.key)
			if found != tt.wantFound {
				t.Errorf("Get() found = %v, want %v", found, tt.wantFound)
			}
			if value != tt.wantValue {
				t.Errorf("Get() value = %v, want %v", value, tt.wantValue)
			}
		})
	}
}

func TestLFUCache_Remove(t *testing.T) {
	cache, _ := NewCache[string, int](2)
	cache.Add("key1", 1)

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{
			name: "remove existing item",
			key:  "key1",
			want: true,
		},
		{
			name: "remove non-existing item",
			key:  "key2",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cache.Remove(tt.key); got != tt.want {
				t.Errorf("Remove() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLFUCache_FrequencyBehavior(t *testing.T) {
	cache, _ := NewCache[string, int](3)

	// Add initial items
	cache.Add("key1", 1)
	cache.Add("key2", 2)
	cache.Add("key3", 3)

	// Access key1 twice
	cache.Get("key1")
	cache.Get("key1")

	// Access key2 once
	cache.Get("key2")

	// Add new item, should evict key3 (least frequently used)
	cache.Add("key4", 4)

	// Check if key3 was evicted
	if _, found := cache.Get("key3"); found {
		t.Error("key3 should have been evicted (least frequently used)")
	}

	// Check if frequently used items are still present
	if _, found := cache.Get("key1"); !found {
		t.Error("key1 should still be present (most frequently used)")
	}
	if _, found := cache.Get("key2"); !found {
		t.Error("key2 should still be present (accessed once)")
	}
}

func TestLFUCache_FrequencyTiebreaker(t *testing.T) {
	cache, _ := NewCache[string, int](2)

	// Add two items
	cache.Add("key1", 1)
	cache.Add("key2", 2)

	// Access both once to have same frequency
	cache.Get("key1")
	cache.Get("key2")

	// Add new item, should evict key1 (least recently used among same frequency)
	cache.Add("key3", 3)

	// Check if key1 was evicted
	if _, found := cache.Get("key1"); found {
		t.Error("key1 should have been evicted (LRU among same frequency)")
	}

	// Check if key2 and key3 are present
	if _, found := cache.Get("key2"); !found {
		t.Error("key2 should still be present")
	}
	if _, found := cache.Get("key3"); !found {
		t.Error("key3 should be present")
	}
}

func TestLFUCache_Concurrent(t *testing.T) {
	cache, _ := NewCache[int, int](100)
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			cache.Add(i, i)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			cache.Get(i)
		}
		done <- true
	}()

	// Remover goroutine
	go func() {
		for i := 0; i < 50; i++ {
			cache.Remove(i)
		}
		done <- true
	}()

	// Wait for all goroutines to finish
	for i := 0; i < 3; i++ {
		<-done
	}
}

func TestLFUCache_FrequencyIncrement(t *testing.T) {
	cache, _ := NewCache[string, int](3)

	cache.Add("key1", 1)

	// Check frequency increments
	frequencies := []int{1, 2, 3}
	for _, freq := range frequencies {
		if _, found := cache.Get("key1"); !found {
			t.Errorf("key1 should be found after %d accesses", freq)
		}
	}
}
