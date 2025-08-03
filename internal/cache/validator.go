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
	MaxSerializationTime time.Duration         `json:"max_serialization_time"`
	BatchSize           int                   `json:"batch_size"`
	EnableDeepCheck     bool                  `json:"enable_deep_check"`
	MaxAgeThreshold     time.Duration         `json:"max_age_threshold"`
	CorruptionPolicy    CorruptionPolicy      `json:"corruption_policy"`
}

// CorruptionHandlingMode defines how to handle corrupted entries
type CorruptionHandlingMode int

const (
	CorruptionPurge CorruptionHandlingMode = iota // Remove corrupted entries (default)
	CorruptionQuarantine                          // Move to quarantine for analysis
	CorruptionAttemptRecover                      // Try to recover data before purging
)

// CorruptionPolicy defines corruption handling behavior
type CorruptionPolicy struct {
	Mode                CorruptionHandlingMode `json:"mode"`
	MaxQuarantineSize   int                    `json:"max_quarantine_size"`
	RecoveryAttempts    int                    `json:"recovery_attempts"`
	EnableDetailedLogs  bool                   `json:"enable_detailed_logs"`
	NotifyOnCorruption  bool                   `json:"notify_on_corruption"`
}

// QuarantinedEntry represents a corrupted entry moved to quarantine
type QuarantinedEntry struct {
	OriginalKey    string    `json:"original_key"`
	CorruptedData  []byte    `json:"corrupted_data"`
	CorruptionType string    `json:"corruption_type"`
	QuarantineTime time.Time `json:"quarantine_time"`
	RecoveryAttempts int     `json:"recovery_attempts"`
}

// DefaultValidationConfig returns production-safe validation settings
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		MaxSerializationTime: 10 * time.Millisecond, // Fail if serialization takes too long
		BatchSize:           100,                     // Process in batches to reduce lock time
		EnableDeepCheck:     false,                   // Disable expensive checks by default
		MaxAgeThreshold:     365 * 24 * time.Hour,   // 1 year max age
		CorruptionPolicy: CorruptionPolicy{
			Mode:                CorruptionPurge,
			MaxQuarantineSize:   100,
			RecoveryAttempts:    2,
			EnableDetailedLogs:  true,
			NotifyOnCorruption:  true,
		},
	}
}

// CacheValidator handles integrity checking and corruption detection
type CacheValidator struct {
	config     ValidationConfig
	quarantine map[string]QuarantinedEntry
	mu         sync.RWMutex
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

// RecoverCorruption attempts to fix corrupted entries with graceful degradation
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
	keysToQuarantine := make([]string, 0)
	
	v.mu.Lock()
	defer v.mu.Unlock()
	
	// Initialize quarantine if needed
	if v.quarantine == nil {
		v.quarantine = make(map[string]QuarantinedEntry)
	}
	
	for key, entry := range cache.data {
		corruption := v.validateEntry(key, entry)
		if corruption != "" {
			result.CorruptedEntries++
			result.CorruptionReasons[corruption]++
			
			// Handle corruption based on policy
			switch v.config.CorruptionPolicy.Mode {
			case CorruptionQuarantine:
				if len(v.quarantine) < v.config.CorruptionPolicy.MaxQuarantineSize {
					keysToQuarantine = append(keysToQuarantine, key)
					
					// Store in quarantine for analysis
					v.quarantine[key] = QuarantinedEntry{
						OriginalKey:    key,
						CorruptedData:  v.serializeEntry(entry),
						CorruptionType: corruption,
						QuarantineTime: time.Now(),
						RecoveryAttempts: 0,
					}
					
					if v.config.CorruptionPolicy.EnableDetailedLogs {
						log.Warn("Entry moved to quarantine for analysis",
							"key", key,
							"corruption_type", corruption,
							"quarantine_size", len(v.quarantine))
					}
				} else {
					// Quarantine full, fall back to purge
					keysToDelete = append(keysToDelete, key)
					log.Warn("Quarantine full, purging corrupted entry",
						"key", key,
						"corruption_type", corruption)
				}
				
			case CorruptionAttemptRecover:
				// Try to recover the entry
				if v.attemptEntryRecovery(key, entry, corruption) {
					result.RecoveredEntries++
					log.Info("Successfully recovered corrupted entry",
						"key", key,
						"corruption_type", corruption)
				} else {
					// Recovery failed, mark for deletion
					keysToDelete = append(keysToDelete, key)
					if v.config.CorruptionPolicy.EnableDetailedLogs {
						log.Warn("Recovery failed, purging corrupted entry",
							"key", key,
							"corruption_type", corruption)
					}
				}
				
			default: // CorruptionPurge
				keysToDelete = append(keysToDelete, key)
				if v.config.CorruptionPolicy.EnableDetailedLogs {
					log.Warn("Corrupted entry marked for purge",
						"key", key,
						"corruption_type", corruption,
						"dry_run", dryRun)
				}
			}
			
			// Notify if configured
			if v.config.CorruptionPolicy.NotifyOnCorruption && !dryRun {
				v.notifyCorruption(key, corruption)
			}
		}
	}
	
	// Perform recovery actions if not a dry run
	if !dryRun {
		// Remove quarantined entries from main cache
		for _, key := range keysToQuarantine {
			if entry, exists := cache.data[key]; exists {
				delete(cache.data, key)
				cache.stats.MemoryUsage -= entry.Size
			}
		}
		
		// Remove purged entries from main cache
		for _, key := range keysToDelete {
			if entry, exists := cache.data[key]; exists {
				delete(cache.data, key)
				cache.stats.MemoryUsage -= entry.Size
				result.RecoveredEntries++
			}
		}
		
		// Update cache statistics
		cache.stats.CorruptionEvents += int64(result.CorruptedEntries)
		if result.RecoveredEntries > 0 || len(keysToQuarantine) > 0 {
			cache.stats.RecoveryEvents++
		}
	}
	
	result.ValidationTime = time.Since(start)
	result.Performance.EntriesPerSecond = float64(result.TotalEntries) / result.ValidationTime.Seconds()
	
	log.Info("Cache corruption recovery completed",
		"total_entries", result.TotalEntries,
		"corrupted_entries", result.CorruptedEntries,
		"recovered_entries", result.RecoveredEntries,
		"quarantined_entries", len(keysToQuarantine),
		"dry_run", dryRun,
		"duration", result.ValidationTime)
	
	return result
}

// GetValidationStats returns validation statistics
func (v *CacheValidator) GetValidationStats() map[string]interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	return map[string]interface{}{
		"config":           v.config,
		"status":           "ready",
		"quarantine_size":  len(v.quarantine),
		"quarantine_limit": v.config.CorruptionPolicy.MaxQuarantineSize,
	}
}

// attemptEntryRecovery tries to recover a corrupted cache entry
func (v *CacheValidator) attemptEntryRecovery(key string, entry *CacheEntry, corruption string) bool {
	maxAttempts := v.config.CorruptionPolicy.RecoveryAttempts
	
	for attempt := 0; attempt < maxAttempts; attempt++ {
		switch corruption {
		case "expired_entry":
			// Try to extend expiration if entry was recently accessed
			if time.Since(entry.AccessedAt) < 5*time.Minute {
				entry.ExpiresAt = time.Now().Add(5 * time.Minute)
				log.Info("Extended expiration for recently accessed entry",
					"key", key, "attempt", attempt+1)
				return true
			}
			
		case "invalid_size":
			// Recalculate size
			if data, err := json.Marshal(entry.Value); err == nil {
				entry.Size = int64(len(data)) + 200
				log.Info("Recalculated entry size",
					"key", key, "new_size", entry.Size, "attempt", attempt+1)
				return true
			}
			
		case "zero_accessed_at":
			// Set accessed time to creation time or now
			if !entry.ExpiresAt.IsZero() {
				entry.AccessedAt = entry.ExpiresAt.Add(-5 * time.Minute)
			} else {
				entry.AccessedAt = time.Now()
			}
			log.Info("Recovered access time",
				"key", key, "new_accessed_at", entry.AccessedAt, "attempt", attempt+1)
			return true
			
		case "future_access_time":
			// Reset to current time
			entry.AccessedAt = time.Now()
			log.Info("Reset future access time",
				"key", key, "attempt", attempt+1)
			return true
		}
	}
	
	return false // Recovery failed
}

// serializeEntry safely serializes an entry for quarantine storage
func (v *CacheValidator) serializeEntry(entry *CacheEntry) []byte {
	if data, err := json.Marshal(entry); err == nil {
		return data
	}
	// If serialization fails, store basic info
	fallback := map[string]interface{}{
		"size":        entry.Size,
		"expires_at":  entry.ExpiresAt,
		"accessed_at": entry.AccessedAt,
		"value_type":  "serialization_failed",
	}
	if data, err := json.Marshal(fallback); err == nil {
		return data
	}
	return []byte(`{"error": "complete_serialization_failure"}`)
}

// notifyCorruption sends corruption notifications (placeholder for alerting system)
func (v *CacheValidator) notifyCorruption(key string, corruption string) {
	// In production, this could send to monitoring systems, Slack, PagerDuty, etc.
	log.Warn("Cache corruption notification",
		"key", key,
		"corruption_type", corruption,
		"timestamp", time.Now(),
		"severity", "medium")
}

// GetQuarantinedEntries returns quarantined entries for analysis
func (v *CacheValidator) GetQuarantinedEntries() map[string]QuarantinedEntry {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	result := make(map[string]QuarantinedEntry)
	for k, v := range v.quarantine {
		result[k] = v
	}
	return result
}

// ClearQuarantine removes all quarantined entries
func (v *CacheValidator) ClearQuarantine() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	count := len(v.quarantine)
	v.quarantine = make(map[string]QuarantinedEntry)
	
	log.Info("Quarantine cleared", "entries_removed", count)
	return count
}
