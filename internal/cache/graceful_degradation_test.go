package cache

import (
	"fmt"
	"testing"
	"time"
)

func TestCorruptionPolicyQuarantine(t *testing.T) {
	config := DefaultValidationConfig()
	config.CorruptionPolicy.Mode = CorruptionQuarantine
	config.CorruptionPolicy.MaxQuarantineSize = 5
	config.CorruptionPolicy.EnableDetailedLogs = true

	validator := &CacheValidator{
		config:     config,
		quarantine: make(map[string]QuarantinedEntry),
	}

	cache := NewMemoryCache(DefaultConfig().Memory)

	// Add some corrupted entries
	cache.Set("valid", "data", time.Minute)
	cache.Set("expired", "data", time.Minute) // Set normally first
	cache.Set("invalid_size", "data", time.Minute)

	// Manually corrupt entries
	if entry, exists := cache.data["expired"]; exists {
		entry.ExpiresAt = time.Now().Add(-time.Minute) // Make it expired
	}
	if entry, exists := cache.data["invalid_size"]; exists {
		entry.Size = -100 // Invalid size
	}

	result := validator.RecoverCorruption(cache, false)

	if result.CorruptedEntries == 0 {
		t.Error("Expected to find corrupted entries")
	}

	quarantined := validator.GetQuarantinedEntries()
	if len(quarantined) == 0 {
		t.Error("Expected entries to be quarantined")
	}

	// Verify corrupted entries are not in main cache
	if _, exists := cache.Get("expired"); exists {
		t.Error("Expected expired entry to be removed from main cache")
	}

	if _, exists := cache.Get("invalid_size"); exists {
		t.Error("Expected invalid_size entry to be removed from main cache")
	}

	// Valid entry should remain
	if _, exists := cache.Get("valid"); !exists {
		t.Error("Expected valid entry to remain in cache")
	}
}

func TestCorruptionPolicyRecovery(t *testing.T) {
	config := DefaultValidationConfig()
	config.CorruptionPolicy.Mode = CorruptionAttemptRecover
	config.CorruptionPolicy.RecoveryAttempts = 2

	validator := &CacheValidator{
		config: config,
	}

	cache := NewMemoryCache(DefaultConfig().Memory)

	// Add entry with recoverable corruption (future access time)
	futureTime := time.Now().Add(10 * time.Minute)
	cache.data["future_access"] = &CacheEntry{
		Value:      "data",
		Size:       100,
		ExpiresAt:  time.Now().Add(time.Hour),
		AccessedAt: futureTime, // This will be detected as corruption
	}

	result := validator.RecoverCorruption(cache, false)

	if result.CorruptedEntries == 0 {
		t.Error("Expected to find corrupted entry")
	}

	if result.RecoveredEntries == 0 {
		t.Error("Expected to recover corrupted entry")
	}

	// Entry should still be in cache with corrected access time
	if _, exists := cache.Get("future_access"); !exists {
		t.Error("Expected recovered entry to remain in cache")
	}

	// Access time should be corrected
	if entry, exists := cache.data["future_access"]; exists {
		if entry.AccessedAt.After(time.Now().Add(time.Minute)) {
			t.Error("Expected access time to be corrected")
		}
	}
}

func TestCorruptionPolicyPurge(t *testing.T) {
	config := DefaultValidationConfig()
	config.CorruptionPolicy.Mode = CorruptionPurge
	config.CorruptionPolicy.EnableDetailedLogs = true

	validator := &CacheValidator{
		config: config,
	}

	cache := NewMemoryCache(DefaultConfig().Memory)

	// Add corrupted entry
	cache.Set("expired", "data", time.Minute) // Set normally first

	// Manually make it expired
	if entry, exists := cache.data["expired"]; exists {
		entry.ExpiresAt = time.Now().Add(-time.Minute) // Make it expired
	}

	result := validator.RecoverCorruption(cache, false)

	if result.CorruptedEntries == 0 {
		t.Error("Expected to find corrupted entry")
	}

	if result.RecoveredEntries == 0 {
		t.Error("Expected to recover (purge) corrupted entry")
	}

	// Entry should be removed from cache
	if _, exists := cache.Get("expired"); exists {
		t.Error("Expected corrupted entry to be purged from cache")
	}
}

func TestQuarantineManagement(t *testing.T) {
	config := DefaultValidationConfig()
	config.CorruptionPolicy.Mode = CorruptionQuarantine
	config.CorruptionPolicy.MaxQuarantineSize = 2

	validator := &CacheValidator{
		config:     config,
		quarantine: make(map[string]QuarantinedEntry),
	}

	cache := NewMemoryCache(DefaultConfig().Memory)

	// Add multiple corrupted entries
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("expired_%d", i)
		cache.Set(key, "data", -time.Minute) // All expired
	}

	result := validator.RecoverCorruption(cache, false)

	quarantined := validator.GetQuarantinedEntries()

	// Should only quarantine up to the limit
	if len(quarantined) > config.CorruptionPolicy.MaxQuarantineSize {
		t.Errorf("Expected quarantine size <= %d, got %d",
			config.CorruptionPolicy.MaxQuarantineSize, len(quarantined))
	}

	// Remaining corrupted entries should be purged
	totalHandled := len(quarantined) + result.RecoveredEntries
	if totalHandled != result.CorruptedEntries {
		t.Errorf("Expected all corrupted entries to be handled, got %d handled out of %d corrupted",
			totalHandled, result.CorruptedEntries)
	}

	// Test quarantine clearing
	clearedCount := validator.ClearQuarantine()
	if clearedCount != len(quarantined) {
		t.Errorf("Expected to clear %d entries, got %d", len(quarantined), clearedCount)
	}

	if len(validator.GetQuarantinedEntries()) != 0 {
		t.Error("Expected quarantine to be empty after clearing")
	}
}

func TestAttemptEntryRecovery(t *testing.T) {
	config := DefaultValidationConfig()
	validator := &CacheValidator{
		config: config,
	}

	testCases := []struct {
		name           string
		corruption     string
		setup          func() *CacheEntry
		expectRecovery bool
	}{
		{
			name:       "expired_entry_recent_access",
			corruption: "expired_entry",
			setup: func() *CacheEntry {
				return &CacheEntry{
					Value:      "data",
					Size:       100,
					ExpiresAt:  time.Now().Add(-time.Minute),     // Expired
					AccessedAt: time.Now().Add(-2 * time.Minute), // Recently accessed
				}
			},
			expectRecovery: true,
		},
		{
			name:       "invalid_size",
			corruption: "invalid_size",
			setup: func() *CacheEntry {
				return &CacheEntry{
					Value:      "test",
					Size:       -100, // Invalid
					ExpiresAt:  time.Now().Add(time.Hour),
					AccessedAt: time.Now(),
				}
			},
			expectRecovery: true,
		},
		{
			name:       "zero_accessed_at",
			corruption: "zero_accessed_at",
			setup: func() *CacheEntry {
				return &CacheEntry{
					Value:      "data",
					Size:       100,
					ExpiresAt:  time.Now().Add(time.Hour),
					AccessedAt: time.Time{}, // Zero time
				}
			},
			expectRecovery: true,
		},
		{
			name:       "future_access_time",
			corruption: "future_access_time",
			setup: func() *CacheEntry {
				return &CacheEntry{
					Value:      "data",
					Size:       100,
					ExpiresAt:  time.Now().Add(time.Hour),
					AccessedAt: time.Now().Add(10 * time.Minute), // Future
				}
			},
			expectRecovery: true,
		},
		{
			name:       "serialization_failure",
			corruption: "serialization_failure",
			setup: func() *CacheEntry {
				return &CacheEntry{
					Value:      make(chan int), // Cannot serialize
					Size:       100,
					ExpiresAt:  time.Now().Add(time.Hour),
					AccessedAt: time.Now(),
				}
			},
			expectRecovery: false, // Cannot recover serialization failures
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := tc.setup()
			recovered := validator.attemptEntryRecovery("test", entry, tc.corruption)

			if recovered != tc.expectRecovery {
				t.Errorf("Expected recovery %t for %s, got %t",
					tc.expectRecovery, tc.corruption, recovered)
			}
		})
	}
}

func TestValidationWithGracefulDegradation(t *testing.T) {
	config := DefaultValidationConfig()
	config.CorruptionPolicy.Mode = CorruptionQuarantine
	config.CorruptionPolicy.EnableDetailedLogs = false // Reduce noise in tests
	config.CorruptionPolicy.NotifyOnCorruption = true

	validator := &CacheValidator{
		config:     config,
		quarantine: make(map[string]QuarantinedEntry),
	}

	cache := NewMemoryCache(DefaultConfig().Memory)

	// Add mix of valid and corrupted entries
	cache.Set("valid1", "data1", time.Minute)
	cache.Set("valid2", "data2", time.Minute)
	cache.Set("expired1", "data", time.Minute)
	cache.Set("expired2", "data", time.Minute)

	// Manually make entries expired
	if entry, exists := cache.data["expired1"]; exists {
		entry.ExpiresAt = time.Now().Add(-time.Minute)
	}
	if entry, exists := cache.data["expired2"]; exists {
		entry.ExpiresAt = time.Now().Add(-time.Minute)
	}

	initialEntries := len(cache.data)

	result := validator.RecoverCorruption(cache, true) // Use dry run to validate without changing cache

	if result.TotalEntries != initialEntries {
		t.Errorf("Expected %d total entries, got %d", initialEntries, result.TotalEntries)
	}

	if result.CorruptedEntries == 0 {
		t.Error("Expected to find corrupted entries")
	}

	// Valid entries should remain (since it's a dry run)
	if _, exists := cache.Get("valid1"); !exists {
		t.Error("Expected valid1 to remain in cache")
	}

	if _, exists := cache.Get("valid2"); !exists {
		t.Error("Expected valid2 to remain in cache")
	}

	// For a non-dry run, also test that stats are updated
	validator.RecoverCorruption(cache, false) // Run recovery

	// Verification that stats are updated
	stats := cache.GetStats()
	if stats.CorruptionEvents == 0 {
		t.Error("Expected corruption events to be recorded in cache stats")
	}
}
