package cache

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

// MemoryCache implements in-memory cache with TTL and LRU eviction
type MemoryCache struct {
	mu             sync.RWMutex
	data           map[string]*CacheEntry
	stats          CacheStats
	maxEntries     int
	defaultTTL     time.Duration
	cleanupTicker  *time.Ticker
	stopCleanup    chan struct{}
	shutdownOnce   sync.Once
	isShuttingDown bool
}

// MemoryCacheConfig holds configuration for in-memory cache
type MemoryCacheConfig struct {
	MaxEntries      int
	DefaultTTL      time.Duration
	CleanupInterval time.Duration
}

// NewMemoryCache creates a new in-memory cache with validated configuration
func NewMemoryCache(config MemoryCacheConfig) *MemoryCache {
	// Validate and apply defaults with bounds checking
	if config.MaxEntries <= 0 {
		config.MaxEntries = 1000
		log.Warn("Invalid MaxEntries, using default", "default", 1000)
	}
	if config.MaxEntries > 100000 {
		config.MaxEntries = 100000
		log.Warn("MaxEntries too large, capping at", "max", 100000)
	}
	if config.DefaultTTL <= 0 {
		config.DefaultTTL = 5 * time.Minute
		log.Warn("Invalid DefaultTTL, using default", "default", config.DefaultTTL)
	}
	if config.DefaultTTL > 24*time.Hour {
		config.DefaultTTL = 24 * time.Hour
		log.Warn("DefaultTTL too large, capping at", "max", config.DefaultTTL)
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 1 * time.Minute
		log.Warn("Invalid CleanupInterval, using default", "default", config.CleanupInterval)
	}
	if config.CleanupInterval < 10*time.Second {
		config.CleanupInterval = 10 * time.Second
		log.Warn("CleanupInterval too frequent, setting minimum", "min", config.CleanupInterval)
	}

	cache := &MemoryCache{
		data:          make(map[string]*CacheEntry),
		maxEntries:    config.MaxEntries,
		defaultTTL:    config.DefaultTTL,
		cleanupTicker: time.NewTicker(config.CleanupInterval),
		stopCleanup:   make(chan struct{}),
	}

	go cache.cleanupWorker()

	log.Info("Memory cache initialized",
		"max_entries", config.MaxEntries,
		"default_ttl", config.DefaultTTL,
		"cleanup_interval", config.CleanupInterval)

	return cache
}

// Set stores a value with specified TTL
func (mc *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
	if key == "" {
		return fmt.Errorf("cache key cannot be empty")
	}
	if value == nil {
		return fmt.Errorf("cache value cannot be nil")
	}
	if ttl <= 0 {
		ttl = mc.defaultTTL
	}

	mc.mu.RLock()
	if mc.isShuttingDown {
		mc.mu.RUnlock()
		return fmt.Errorf("cache is shutting down")
	}
	mc.mu.RUnlock()

	// Calculate size for memory tracking
	size := mc.calculateSize(value)
	
	entry := &CacheEntry{
		Value:      value,
		ExpiresAt:  time.Now().Add(ttl),
		AccessedAt: time.Now(),
		Size:       size,
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Double-check shutdown state under write lock
	if mc.isShuttingDown {
		return fmt.Errorf("cache is shutting down")
	}

	// Check if this is an update to existing key
	existingEntry, isUpdate := mc.data[key]
	
	// Check if we need to evict entries (only for new keys)
	if !isUpdate && len(mc.data) >= mc.maxEntries {
		mc.evictLRU()
	}

	// If updating, subtract the old size from memory usage
	if isUpdate {
		mc.stats.MemoryUsage -= existingEntry.Size
	}

	mc.data[key] = entry
	mc.stats.MemoryUsage += size

	log.Debug("Cache entry set",
		"key", key,
		"ttl", ttl,
		"size_bytes", size,
		"total_entries", len(mc.data),
		"is_update", isUpdate)

	return nil
}

// Get retrieves a value by key
func (mc *MemoryCache) Get(key string) (interface{}, bool) {
	if key == "" {
		log.Debug("Attempted to get empty cache key")
		return nil, false
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Check shutdown state
	if mc.isShuttingDown {
		return nil, false
	}

	entry, exists := mc.data[key]
	if !exists {
		mc.stats.Misses++
		log.Debug("Cache miss", "key", key, "reason", "not_found")
		return nil, false
	}

	// Check if entry has expired
	if entry.IsExpired() {
		delete(mc.data, key)
		mc.stats.MemoryUsage -= entry.Size
		mc.stats.Misses++
		mc.stats.Evictions++
		log.Debug("Cache miss (expired)", "key", key, "expired_at", entry.ExpiresAt)
		return nil, false
	}

	// Update access time for LRU tracking
	entry.UpdateAccess()
	mc.stats.Hits++

	log.Debug("Cache hit", "key", key, "age", time.Since(entry.AccessedAt))
	return entry.Value, true
}

// Delete removes a specific key from the cache
func (mc *MemoryCache) Delete(key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if entry, exists := mc.data[key]; exists {
		delete(mc.data, key)
		mc.stats.MemoryUsage -= entry.Size
		log.Debug("Cache entry deleted", "key", key)
	}

	return nil
}

// Clear removes all entries from the cache
func (mc *MemoryCache) Clear() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	entryCount := len(mc.data)
	mc.data = make(map[string]*CacheEntry)
	mc.stats.MemoryUsage = 0

	log.Info("Cache cleared", "entries_removed", entryCount)
	return nil
}

// EvictExpired removes all expired entries and returns the count
func (mc *MemoryCache) EvictExpired() int {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	return mc.evictExpiredLocked()
}

// Stats returns current cache performance metrics
func (mc *MemoryCache) Stats() CacheStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Create a copy of stats to avoid race conditions
	stats := CacheStats{
		Hits:        mc.stats.Hits,
		Misses:      mc.stats.Misses,
		Evictions:   mc.stats.Evictions,
		Entries:     len(mc.data),
		MemoryUsage: mc.stats.MemoryUsage,
	}
	
	// Calculate hit rate
	totalRequests := stats.Hits + stats.Misses
	if totalRequests > 0 {
		stats.HitRate = float64(stats.Hits) / float64(totalRequests) * 100
	}

	return stats
}

// Close shuts down the cache and stops background workers
func (mc *MemoryCache) Close() {
	mc.shutdownOnce.Do(func() {
		mc.mu.Lock()
		mc.isShuttingDown = true
		mc.mu.Unlock()

		// Stop the cleanup ticker
		if mc.cleanupTicker != nil {
			mc.cleanupTicker.Stop()
		}

		// Signal cleanup goroutine to stop
		close(mc.stopCleanup)

		// Give cleanup goroutine time to finish
		time.Sleep(100 * time.Millisecond)

		// Clear cache data
		mc.mu.Lock()
		entryCount := len(mc.data)
		mc.data = nil
		mc.mu.Unlock()

		log.Info("Memory cache closed", "final_entries", entryCount)
	})
}

// evictExpiredLocked removes expired entries (must be called with lock held)
func (mc *MemoryCache) evictExpiredLocked() int {
	evicted := 0
	now := time.Now()

	for key, entry := range mc.data {
		if now.After(entry.ExpiresAt) {
			delete(mc.data, key)
			mc.stats.MemoryUsage -= entry.Size
			mc.stats.Evictions++
			evicted++
		}
	}

	if evicted > 0 {
		log.Debug("Expired entries evicted", "count", evicted)
	}

	return evicted
}

// evictLRU removes the least recently used entry (must be called with lock held)
func (mc *MemoryCache) evictLRU() {
	if len(mc.data) == 0 {
		return
	}

	// Find the least recently used entry
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, entry := range mc.data {
		// Skip entries that are already expired (they'll be cleaned up separately)
		if entry.IsExpired() {
			continue
		}
		
		if first || entry.AccessedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.AccessedAt
			first = false
		}
	}

	// If no valid entry found (all expired), clean up expired entries instead
	if oldestKey == "" {
		evicted := mc.evictExpiredLocked()
		log.Debug("LRU eviction found no valid entries, cleaned expired instead", "evicted", evicted)
		return
	}

	// Remove the oldest entry
	if entry, exists := mc.data[oldestKey]; exists {
		delete(mc.data, oldestKey)
		mc.stats.MemoryUsage -= entry.Size
		mc.stats.Evictions++
		
		log.Debug("LRU eviction",
			"key", oldestKey,
			"age", time.Since(oldestTime),
			"remaining_entries", len(mc.data),
			"memory_freed", entry.Size)
	}
}

// calculateSize estimates the memory size of a value in bytes
func (mc *MemoryCache) calculateSize(value interface{}) int64 {
	// Simple estimation using JSON marshaling
	// This is not perfectly accurate but gives a reasonable approximation
	data, err := json.Marshal(value)
	if err != nil {
		// Fallback to a conservative estimate
		return 1024 // 1KB default
	}
	
	// Add overhead for map storage, pointers, etc.
	return int64(len(data)) + 200 // ~200 bytes overhead per entry
}

// cleanupWorker runs in a background goroutine to periodically clean expired entries
func (mc *MemoryCache) cleanupWorker() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Cache cleanup worker panic recovered", "panic", r)
		}
	}()

	for {
		select {
		case <-mc.cleanupTicker.C:
			// Perform cleanup
			start := time.Now()
			evicted := mc.EvictExpired()
			duration := time.Since(start)
			
			if evicted > 0 {
				log.Debug("Scheduled cleanup completed",
					"evicted_entries", evicted,
					"duration", duration,
					"remaining_entries", mc.getCurrentEntryCount())
			}

			// Check if cleanup is taking too long (potential performance issue)
			if duration > 100*time.Millisecond {
				log.Warn("Cache cleanup took longer than expected",
					"duration", duration,
					"evicted", evicted)
			}

		case <-mc.stopCleanup:
			log.Debug("Cache cleanup worker stopping")
			return
		}
	}
}

// getCurrentEntryCount safely gets the current entry count
func (mc *MemoryCache) getCurrentEntryCount() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return len(mc.data)
}
