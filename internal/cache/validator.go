package cache

import (
	"encoding/json"
	"sync"
	"time"
	
	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

// ValidationResult represents the result of cache validation
type ValidationResult struct {
	TotalEntries      int                    `json:"total_entries"`
	CorruptedEntries  int                    `json:"corrupted_entries"`
	RecoveredEntries  int                    `json:"recovered_entries"`
	ValidationTime    time.Duration          `json:"validation_time"`
	CorruptionReasons map[string]int         `json:"corruption_reasons"`
	Performance       ValidationPerformance  `json:"performance"`
}

// ValidationPerformance tracks performance metrics during validation
type ValidationPerformance struct {
	EntriesPerSecond     float64 `json:"entries_per_second"`
	MemoryMBValidated    float64 `json:"memory_mb_validated"`
	CPUTimePercent       float64 `json:"cpu_time_percent"`
	SerializationTimeMs  int64   `json:"serialization_time_ms"`
}

// ValidationConfig controls validation behavior and performance
type ValidationConfig struct {
	MaxSerializationTime time.Duration `json:"max_serialization_time"`
	BatchSize           int           `json:"batch_size"`
	EnableDeepCheck     bool          `json:"enable_deep_check"`
	MaxAgeThreshold     time.Duration `json:"max_age_threshold"`
}

// DefaultValidationConfig returns production-safe validation settings
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		MaxSerializationTime: 10 * time.Millisecond, // Fail if serialization takes too long
		BatchSize:           100,                     // Process in batches to reduce lock time
		EnableDeepCheck:     false,                   // Disable expensive checks by default
		MaxAgeThreshold:     365 * 24 * time.Hour,   // 1 year max age
	}
}

// CacheValidator handles integrity checking and corruption detection
type CacheValidator struct {
	config ValidationConfig
	mu     sync.RWMutex
}

// NewCacheValidator creates a new validator with the specified configuration
func NewCacheValidator(config ValidationConfig) *CacheValidator {
	return &CacheValidator{
		config: config,
	}
}

// ValidateCache performs comprehensive cache integrity validation
func (v *CacheValidator) ValidateCache(cache *MemoryCache) ValidationResult {
	start := time.Now()
	
	result := ValidationResult{
		CorruptionReasons: make(map[string]int),
		Performance:       ValidationPerformance{},
	}
	
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	
	result.TotalEntries = len(cache.data)
	totalMemory := int64(0)
	serializationStart := time.Now()
	
	for key, entry := range cache.data {
		corruption := v.validateEntry(key, entry)
		if corruption != "" {
			result.CorruptedEntries++
			result.CorruptionReasons[corruption]++
			
			log.Warn("Cache corruption detected during validation",
				"key", key,
				"corruption_type", corruption,
				"entry_age", time.Since(entry.AccessedAt))
		}
		
		totalMemory += entry.Size
	}
	
	serializationTime := time.Since(serializationStart)
	validationTime := time.Since(start)
	
	// Calculate performance metrics
	result.ValidationTime = validationTime
	result.Performance.EntriesPerSecond = float64(result.TotalEntries) / validationTime.Seconds()
	result.Performance.MemoryMBValidated = float64(totalMemory) / 1024 / 1024
	result.Performance.SerializationTimeMs = serializationTime.Milliseconds()
	
	// Performance warnings
	if validationTime > 100*time.Millisecond {
		log.Warn("Cache validation took longer than expected",
			"duration", validationTime,
			"entries", result.TotalEntries,
			"performance_concern", "consider_batch_processing")
	}
	
	if result.CorruptedEntries > 0 {
		log.Error("Cache validation completed with corruption detected",
			"total_entries", result.TotalEntries,
			"corrupted_entries", result.CorruptedEntries,
			"corruption_rate", float64(result.CorruptedEntries)/float64(result.TotalEntries)*100,
			"corruption_reasons", result.CorruptionReasons)
	} else {
		log.Info("Cache validation completed successfully",
			"total_entries", result.TotalEntries,
			"validation_time", validationTime,
			"performance", result.Performance)
	}
	
	return result
}

// validateEntry checks a single cache entry for corruption
func (v *CacheValidator) validateEntry(key string, entry *CacheEntry) string {
	if entry == nil {
		return "nil_entry"
	}
	
	// Check for invalid timestamps
	if entry.ExpiresAt.IsZero() {
		return "zero_expires_at"
	}
	
	if entry.AccessedAt.IsZero() {
		return "zero_accessed_at"
	}
	
	// Check for impossibly old access times
	if time.Since(entry.AccessedAt) > v.config.MaxAgeThreshold {
		return "ancient_access_time"
	}
	
	// Check for future access times (clock skew or corruption)
	if entry.AccessedAt.After(time.Now().Add(1 * time.Minute)) {
		return "future_access_time"
	}
	
	// Check if entry has already expired (shouldn't be in cache)
	if entry.IsExpired() {
		return "expired_entry"
	}
	
	// Check for negative or zero size
	if entry.Size <= 0 {
		return "invalid_size"
	}
	
	// Serialization check (most expensive)
	serializationStart := time.Now()
	_, err := json.Marshal(entry.Value)
	serializationTime := time.Since(serializationStart)
	
	if err != nil {
		return "serialization_failure"
	}
	
	// Check if serialization is taking too long (potential corruption or complex object)
	if serializationTime > v.config.MaxSerializationTime {
		return "slow_serialization"
	}
	
	// Deep validation (expensive, disabled by default)
	if v.config.EnableDeepCheck {
		if corruption := v.deepValidateEntry(key, entry); corruption != "" {
			return corruption
		}
	}
	
	return "" // No corruption detected
}

// deepValidateEntry performs expensive deep validation checks
func (v *CacheValidator) deepValidateEntry(key string, entry *CacheEntry) string {
	// Check if key matches expected patterns
	if len(key) == 0 {
		return "empty_key"
	}
	
	// Check for suspicious key patterns
	if len(key) > 1000 { // Arbitrarily long keys might indicate corruption
		return "oversized_key"
	}
	
	// Validate value consistency by re-serializing and comparing size
	data, err := json.Marshal(entry.Value)
	if err != nil {
		return "deep_serialization_failure"
	}
	
	calculatedSize := int64(len(data)) + 200 // Same calculation as in memory.go
	sizeDiff := float64(entry.Size-calculatedSize) / float64(calculatedSize)
	
	// If size differs by more than 50%, something might be wrong
	if sizeDiff > 0.5 || sizeDiff < -0.5 {
		return "size_mismatch"
	}
	
	return ""
}

// RecoverCorruption attempts to fix corrupted entries in place
func (v *CacheValidator) RecoverCorruption(cache *MemoryCache, dryRun bool) ValidationResult {
	start := time.Now()
	
	result := ValidationResult{
		CorruptionReasons: make(map[string]int),
		Performance:       ValidationPerformance{},
	}
	
	if !dryRun {
		cache.mu.Lock()
		defer cache.mu.Unlock()
	} else {
		cache.mu.RLock()
		defer cache.mu.RUnlock()
	}
	
	result.TotalEntries = len(cache.data)
	keysToDelete := make([]string, 0)
	
	for key, entry := range cache.data {
		corruption := v.validateEntry(key, entry)
		if corruption != "" {
			result.CorruptedEntries++
			result.CorruptionReasons[corruption]++
			
			// Mark for deletion (can't delete while iterating)
			keysToDelete = append(keysToDelete, key)
			
			log.Warn("Corrupted entry marked for recovery",
				"key", key,
				"corruption_type", corruption,
				"dry_run", dryRun)
		}
	}
	
	// Perform recovery (deletion) if not a dry run
	if !dryRun {
		for _, key := range keysToDelete {
			if entry, exists := cache.data[key]; exists {
				delete(cache.data, key)
				cache.stats.MemoryUsage -= entry.Size
				result.RecoveredEntries++
			}
		}
		
		// Update cache statistics
		cache.stats.CorruptionEvents += int64(result.CorruptedEntries)
		if result.RecoveredEntries > 0 {
			cache.stats.RecoveryEvents++
		}
	}
	
	result.ValidationTime = time.Since(start)
	result.Performance.EntriesPerSecond = float64(result.TotalEntries) / result.ValidationTime.Seconds()
	
	log.Info("Cache corruption recovery completed",
		"total_entries", result.TotalEntries,
		"corrupted_entries", result.CorruptedEntries,
		"recovered_entries", result.RecoveredEntries,
		"dry_run", dryRun,
		"duration", result.ValidationTime)
	
	return result
}

// GetValidationStats returns validation statistics
func (v *CacheValidator) GetValidationStats() map[string]interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	return map[string]interface{}{
		"config": v.config,
		"status": "ready",
	}
}
