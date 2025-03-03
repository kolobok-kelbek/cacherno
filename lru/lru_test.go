package lru

import (
	"testing"
)

func TestNewCache(t *testing.T) {
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

func TestCache_Add(t *testing.T) {
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

func TestCache_Get(t *testing.T) {
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

func TestCache_Remove(t *testing.T) {
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

func TestCache_LRUBehavior(t *testing.T) {
	cache, _ := NewCache[string, int](2)

	// Add initial items
	cache.Add("key1", 1)
	cache.Add("key2", 2)

	// Access key1 to make it most recently used
	cache.Get("key1")

	// Add new item, should evict key2 (least recently used)
	cache.Add("key3", 3)

	// Check if key2 was evicted
	if _, found := cache.Get("key2"); found {
		t.Error("key2 should have been evicted")
	}

	// Check if key1 and key3 are still present
	if _, found := cache.Get("key1"); !found {
		t.Error("key1 should still be present")
	}
	if _, found := cache.Get("key3"); !found {
		t.Error("key3 should be present")
	}
}

func TestCache_Concurrent(t *testing.T) {
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

	// Wait for both goroutines to finish
	<-done
	<-done
}
