package cache

import "time"

// Cache defines the interface for the cache implementation
type Cache interface {
	Set(key string, value interface{}, ttl time.Duration) error
	Get(key string) (interface{}, bool)
	Delete(key string) error
	Clear() error
	EvictExpired() int
	Stats() CacheStats
}

// Metrics for cache performance
type CacheStats struct {
	Hits             int64     `json:"hits"`
	Misses           int64     `json:"misses"`
	Evictions        int64     `json:"evictions"`
	Entries          int       `json:"entries"`
	HitRate          float64   `json:"hit_rate"`
	MemoryUsage      int64     `json:"memory_usage"`
	SetsTotal        int64     `json:"sets_total"`
	DeletesTotal     int64     `json:"deletes_total"`
	ExpiredKeys      int64     `json:"expired_keys"`
	LRUEvictions     int64     `json:"lru_evictions"`
	AverageKeySize   int64     `json:"average_key_size"`
	CorruptionEvents int64     `json:"corruption_events"`
	RecoveryEvents   int64     `json:"recovery_events"`
	LastHitTime      time.Time `json:"last_hit_time"`
	LastMissTime     time.Time `json:"last_miss_time"`
	UptimeSeconds    int64     `json:"uptime_seconds"`
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

// UpdateAccess updates the last accessed timestamp for tracking
func (e *CacheEntry) UpdateAccess() {
	e.AccessedAt = time.Now()
}
