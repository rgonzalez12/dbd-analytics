package cache

import "time"

// Cache defines the interface for all cache implementations
type Cache interface {
	Set(key string, value interface{}, ttl time.Duration) error
	Get(key string) (interface{}, bool)
	Delete(key string) error
	Clear() error
	EvictExpired() int
	Stats() CacheStats
}

// CacheStats provides metrics for monitoring cache performance
type CacheStats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	Evictions   int64   `json:"evictions"`
	Entries     int     `json:"entries"`
	HitRate     float64 `json:"hit_rate"`
	MemoryUsage int64   `json:"memory_usage"`
}

// CacheEntry represents a cached item with metadata
type CacheEntry struct {
	Value      interface{} `json:"value"`
	ExpiresAt  time.Time   `json:"expires_at"`
	AccessedAt time.Time   `json:"accessed_at"`
	Size       int64       `json:"size"`
}

// IsExpired checks if the cache entry has expired
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// UpdateAccess updates the last accessed timestamp for LRU tracking
func (e *CacheEntry) UpdateAccess() {
	e.AccessedAt = time.Now()
}
