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
	startTime      time.Time // Track cache initialization time for uptime
}

// MemoryCacheConfig holds configuration for in-memory cache
type MemoryCacheConfig struct {
	MaxEntries      int
	DefaultTTL      time.Duration
	CleanupInterval time.Duration
}

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
		startTime:     time.Now(),
	}

	go cache.cleanupWorker()

	log.Info("Memory cache initialized",
		"max_entries", config.MaxEntries,
		"default_ttl", config.DefaultTTL,
		"cleanup_interval", config.CleanupInterval)

	return cache
}

// Set stores a value with TTL
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
	mc.stats.SetsTotal++

	log.Debug("Cache entry set",
		"key", key,
		"ttl", ttl,
		"size_bytes", size,
		"total_entries", len(mc.data),
		"is_update", isUpdate,
		"sets_total", mc.stats.SetsTotal)

	return nil
}

// Get retrieves a value by key with detailed miss reason tracking
func (mc *MemoryCache) Get(key string) (interface{}, bool) {
	if key == "" {
		log.Warn("Cache operation attempted with empty key", "operation", "get")
		return nil, false
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Check shutdown state
	if mc.isShuttingDown {
		log.Debug("Cache get during shutdown", "key", key)
		return nil, false
	}

	entry, exists := mc.data[key]
	if !exists {
		mc.stats.Misses++
		mc.stats.LastMissTime = time.Now()
		log.Debug("Cache miss",
			"key", key,
			"reason", "key_not_found",
			"total_entries", len(mc.data),
			"miss_count", mc.stats.Misses)
		return nil, false
	}

	// Check if entry has expired
	if entry.IsExpired() {
		delete(mc.data, key)
		mc.stats.MemoryUsage -= entry.Size
		mc.stats.Misses++
		mc.stats.Evictions++
		mc.stats.ExpiredKeys++
		mc.stats.LastMissTime = time.Now()
		log.Debug("Cache miss",
			"key", key,
			"reason", "expired",
			"expired_at", entry.ExpiresAt,
			"age_seconds", time.Since(entry.ExpiresAt).Seconds())
		return nil, false
	}

	// Update access time for LRU tracking
	entry.UpdateAccess()
	mc.stats.Hits++
	mc.stats.LastHitTime = time.Now()

	log.Debug("Cache hit",
		"key", key,
		"age", time.Since(entry.AccessedAt),
		"total_hits", mc.stats.Hits)
	return entry.Value, true
}

func (mc *MemoryCache) Delete(key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if entry, exists := mc.data[key]; exists {
		delete(mc.data, key)
		mc.stats.MemoryUsage -= entry.Size
		mc.stats.DeletesTotal++
		log.Debug("Cache entry deleted",
			"key", key,
			"size_bytes", entry.Size,
			"deletes_total", mc.stats.DeletesTotal)
	}

	return nil
}

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

// Stats returns comprehensive cache performance metrics
func (mc *MemoryCache) Stats() CacheStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Create a copy of stats with enhanced metrics
	stats := CacheStats{
		Hits:             mc.stats.Hits,
		Misses:           mc.stats.Misses,
		Evictions:        mc.stats.Evictions,
		Entries:          len(mc.data),
		MemoryUsage:      mc.stats.MemoryUsage,
		SetsTotal:        mc.stats.SetsTotal,
		DeletesTotal:     mc.stats.DeletesTotal,
		ExpiredKeys:      mc.stats.ExpiredKeys,
		LRUEvictions:     mc.stats.LRUEvictions,
		CorruptionEvents: mc.stats.CorruptionEvents,
		RecoveryEvents:   mc.stats.RecoveryEvents,
		LastHitTime:      mc.stats.LastHitTime,
		LastMissTime:     mc.stats.LastMissTime,
		UptimeSeconds:    int64(time.Since(mc.startTime).Seconds()),
	}

	// Calculate hit rate
	totalRequests := stats.Hits + stats.Misses
	if totalRequests > 0 {
		stats.HitRate = float64(stats.Hits) / float64(totalRequests) * 100
	}

	// Calculate average key size
	if stats.Entries > 0 {
		stats.AverageKeySize = stats.MemoryUsage / int64(stats.Entries)
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
			mc.stats.ExpiredKeys++
			evicted++
		}
	}

	if evicted > 0 {
		log.Debug("Expired entries evicted",
			"count", evicted,
			"total_expired", mc.stats.ExpiredKeys)
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
		mc.stats.LRUEvictions++

		log.Debug("LRU eviction (basic policy - Redis upgrade recommended for production)",
			"key", oldestKey,
			"age", time.Since(oldestTime),
			"remaining_entries", len(mc.data),
			"memory_freed", entry.Size,
			"lru_evictions_total", mc.stats.LRUEvictions)
	}
}

// calculateSize estimates the memory size of a value in bytes
func (mc *MemoryCache) calculateSize(value interface{}) int64 {
	// Simple estimation using JSON marshaling
	// This is not perfectly accurate but gives a reasonable approximation
	data, err := json.Marshal(value)
	if err != nil {
		// Track corruption event
		mc.stats.CorruptionEvents++
		log.Error("Cache value serialization failed - potential corruption",
			"error", err.Error(),
			"corruption_events_total", mc.stats.CorruptionEvents,
			"value_type", fmt.Sprintf("%T", value))

		// Fallback to a conservative estimate
		return 1024 // 1KB default
	}

	// Add overhead for map storage, pointers, etc.
	return int64(len(data)) + 200 // ~200 bytes overhead per entry
}

// detectAndRecover performs corruption detection and recovery
func (mc *MemoryCache) detectAndRecover() int {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	corrupted := 0
	now := time.Now()

	for key, entry := range mc.data {
		// Check for obvious corruption indicators
		if entry == nil {
			delete(mc.data, key)
			corrupted++
			continue
		}

		// Check for invalid timestamps
		if entry.ExpiresAt.IsZero() || entry.AccessedAt.IsZero() {
			delete(mc.data, key)
			mc.stats.MemoryUsage -= entry.Size
			corrupted++
			continue
		}

		// Check for impossibly old access times (> 1 year ago)
		if now.Sub(entry.AccessedAt) > 365*24*time.Hour {
			delete(mc.data, key)
			mc.stats.MemoryUsage -= entry.Size
			corrupted++
			continue
		}

		// Attempt to re-serialize value to detect corruption
		if _, err := json.Marshal(entry.Value); err != nil {
			delete(mc.data, key)
			mc.stats.MemoryUsage -= entry.Size
			corrupted++
			continue
		}
	}

	if corrupted > 0 {
		mc.stats.CorruptionEvents += int64(corrupted)
		mc.stats.RecoveryEvents++

		log.Error("Cache corruption detected and recovered",
			"corrupted_entries", corrupted,
			"corruption_events_total", mc.stats.CorruptionEvents,
			"recovery_events_total", mc.stats.RecoveryEvents,
			"remaining_entries", len(mc.data))
	}

	return corrupted
}

// cleanupWorker runs in a background goroutine to periodically clean expired entries
func (mc *MemoryCache) cleanupWorker() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Cache cleanup worker panic recovered", "panic", r)
		}
	}()

	cleanupCount := 0
	for {
		select {
		case <-mc.cleanupTicker.C:
			start := time.Now()
			evicted := mc.EvictExpired()

			// Perform corruption detection every 5th cleanup
			corrupted := 0
			if cleanupCount%5 == 0 {
				corrupted = mc.detectAndRecover()
			}

			duration := time.Since(start)
			cleanupCount++

			if evicted > 0 || corrupted > 0 {
				log.Debug("Scheduled cleanup completed",
					"evicted_entries", evicted,
					"corrupted_entries", corrupted,
					"duration", duration,
					"remaining_entries", mc.getCurrentEntryCount())
			}

			// Log comprehensive metrics every 10 cleanups for observability
			if cleanupCount%10 == 0 {
				mc.logMetrics()
			}

			// Check if cleanup is taking too long (potential performance issue)
			if duration > 100*time.Millisecond {
				log.Warn("Cache cleanup took longer than expected",
					"duration", duration,
					"evicted", evicted,
					"corrupted", corrupted,
					"performance_concern", "consider_redis_upgrade")
			}

		case <-mc.stopCleanup:
			log.Debug("Cache cleanup worker stopping")
			return
		}
	}
}

// logMetrics logs comprehensive cache metrics for observability
func (mc *MemoryCache) logMetrics() {
	stats := mc.Stats()

	log.Info("Cache performance metrics",
		"hit_rate", fmt.Sprintf("%.1f%%", stats.HitRate),
		"hits", stats.Hits,
		"misses", stats.Misses,
		"entries", stats.Entries,
		"memory_usage_mb", float64(stats.MemoryUsage)/1024/1024,
		"uptime_minutes", stats.UptimeSeconds/60,
		"sets_total", stats.SetsTotal,
		"lru_evictions", stats.LRUEvictions,
		"expired_keys", stats.ExpiredKeys,
		"avg_key_size_bytes", stats.AverageKeySize)

	// Performance warnings
	if stats.HitRate < 70 && stats.Hits+stats.Misses > 100 {
		log.Warn("Cache hit rate below recommended threshold",
			"current_hit_rate", fmt.Sprintf("%.1f%%", stats.HitRate),
			"recommended_minimum", "70%",
			"suggestion", "review_cache_ttl_or_upgrade_to_redis")
	}

	if stats.LRUEvictions > stats.ExpiredKeys*2 {
		log.Warn("High LRU eviction rate detected",
			"lru_evictions", stats.LRUEvictions,
			"expired_keys", stats.ExpiredKeys,
			"suggestion", "consider_increasing_cache_capacity_or_redis_upgrade")
	}
}

// getCurrentEntryCount safely gets the current entry count
func (mc *MemoryCache) getCurrentEntryCount() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return len(mc.data)
}

// GetStats returns a copy of the current cache statistics
func (mc *MemoryCache) GetStats() CacheStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Calculate hit rate
	totalRequests := mc.stats.Hits + mc.stats.Misses
	var hitRate float64
	if totalRequests > 0 {
		hitRate = float64(mc.stats.Hits) / float64(totalRequests)
	}

	// Return a copy to prevent race conditions
	return CacheStats{
		Hits:             mc.stats.Hits,
		Misses:           mc.stats.Misses,
		SetsTotal:        mc.stats.SetsTotal,
		DeletesTotal:     mc.stats.DeletesTotal,
		Evictions:        mc.stats.Evictions,
		ExpiredKeys:      mc.stats.ExpiredKeys,
		LRUEvictions:     mc.stats.LRUEvictions,
		MemoryUsage:      mc.stats.MemoryUsage,
		LastHitTime:      mc.stats.LastHitTime,
		LastMissTime:     mc.stats.LastMissTime,
		CorruptionEvents: mc.stats.CorruptionEvents,
		RecoveryEvents:   mc.stats.RecoveryEvents,
		Entries:          len(mc.data),
		HitRate:          hitRate,
		UptimeSeconds:    int64(time.Since(mc.startTime).Seconds()),
	}
}
