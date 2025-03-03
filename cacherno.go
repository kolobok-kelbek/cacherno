package cacherno

// Cache defines the basic operations that all cache implementations should support
type Cache[K comparable, V any] interface {
	Get(key K) (V, bool)
	Add(key K, value V) (evicted bool, rewritten bool)
	Remove(key K) bool
}
