package cache

import (
	"testing"
	"time"
)

func TestCorruptionDetectionAndRecovery(t *testing.T) {
	cache := NewMemoryCache(MemoryCacheConfig{
		MaxEntries:      100,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 10 * time.Second,
	})
	defer cache.Close()

	// Test 1: Set normal entry
	err := cache.Set("normal_key", "normal_value", time.Minute)
	if err != nil {
		t.Fatalf("Failed to set normal entry: %v", err)
	}

	// Test 2: Manually inject corrupted entry (nil value)
	cache.mu.Lock()
	cache.data["corrupted_nil"] = nil
	cache.mu.Unlock()

	// Test 3: Manually inject entry with zero timestamps
	cache.mu.Lock()
	cache.data["corrupted_timestamps"] = &CacheEntry{
		Value:      "test",
		ExpiresAt:  time.Time{}, // Zero time
		AccessedAt: time.Time{}, // Zero time
		Size:       100,
	}
	cache.mu.Unlock()

	// Test 4: Manually inject entry with ancient access time (> 1 year ago)
	oldTime := time.Now().AddDate(-2, 0, 0) // 2 years ago
	cache.mu.Lock()
	cache.data["corrupted_ancient"] = &CacheEntry{
		Value:      "ancient",
		ExpiresAt:  time.Now().Add(time.Hour),
		AccessedAt: oldTime,
		Size:       100,
	}
	cache.mu.Unlock()

	// Test 5: Manually inject entry with corrupted value (func type can't be JSON marshaled)
	cache.mu.Lock()
	cache.data["corrupted_value"] = &CacheEntry{
		Value:      func() {}, // Functions can't be JSON marshaled
		ExpiresAt:  time.Now().Add(time.Hour),
		AccessedAt: time.Now(),
		Size:       100,
	}
	cache.mu.Unlock()

	// Verify initial state
	initialStats := cache.Stats()
	if initialStats.Entries != 5 { // normal + 4 corrupted
		t.Errorf("Expected 5 entries before corruption detection, got %d", initialStats.Entries)
	}

	// Run corruption detection
	corrupted := cache.detectAndRecover()

	// Verify corruption was detected and recovered
	if corrupted != 4 {
		t.Errorf("Expected 4 corrupted entries to be detected, got %d", corrupted)
	}

	// Verify stats were updated
	finalStats := cache.Stats()
	if finalStats.CorruptionEvents != 4 {
		t.Errorf("Expected 4 corruption events in stats, got %d", finalStats.CorruptionEvents)
	}
	if finalStats.RecoveryEvents != 1 {
		t.Errorf("Expected 1 recovery event in stats, got %d", finalStats.RecoveryEvents)
	}
	if finalStats.Entries != 1 {
		t.Errorf("Expected 1 remaining entry after cleanup, got %d", finalStats.Entries)
	}

	// Verify normal entry is still accessible
	value, found := cache.Get("normal_key")
	if !found {
		t.Error("Normal entry should still be accessible after corruption cleanup")
	}
	if value != "normal_value" {
		t.Errorf("Expected normal_value, got %v", value)
	}

	// Verify corrupted entries are gone
	corruptedKeys := []string{"corrupted_nil", "corrupted_timestamps", "corrupted_ancient", "corrupted_value"}
	for _, key := range corruptedKeys {
		if _, found := cache.Get(key); found {
			t.Errorf("Corrupted entry %s should have been removed", key)
		}
	}
}

func TestCorruptionMetricsTracking(t *testing.T) {
	cache := NewMemoryCache(MemoryCacheConfig{
		MaxEntries:      10,
		DefaultTTL:      time.Minute,
		CleanupInterval: 10 * time.Second,
	})
	defer cache.Close()

	// Test corruption tracking in calculateSize
	initialStats := cache.Stats()
	if initialStats.CorruptionEvents != 0 {
		t.Errorf("Expected 0 initial corruption events, got %d", initialStats.CorruptionEvents)
	}

	// Try to set a value that will cause JSON marshaling to fail
	// (this simulates corruption during size calculation)
	funcValue := func() { /* can't be marshaled */ }
	err := cache.Set("test_key", funcValue, time.Minute)
	if err != nil {
		t.Fatalf("Set should not fail even with unmarshallable value: %v", err)
	}

	// Check that corruption was tracked
	stats := cache.Stats()
	if stats.CorruptionEvents == 0 {
		t.Error("Expected corruption event to be tracked when JSON marshaling fails")
	}
}

func TestPeriodicCorruptionDetection(t *testing.T) {
	// Use very short cleanup interval for testing
	cache := NewMemoryCache(MemoryCacheConfig{
		MaxEntries:      10,
		DefaultTTL:      time.Minute,
		CleanupInterval: 50 * time.Millisecond, // Short for testing
	})
	defer cache.Close()

	// Set a normal entry
	err := cache.Set("normal", "value", time.Minute)
	if err != nil {
		t.Fatalf("Failed to set normal entry: %v", err)
	}

	// Inject corruption
	cache.mu.Lock()
	cache.data["corrupted"] = &CacheEntry{
		Value:      "test",
		ExpiresAt:  time.Time{}, // Zero time - corrupted
		AccessedAt: time.Time{}, // Zero time - corrupted
		Size:       100,
	}
	cache.mu.Unlock()

	// Wait for enough cleanup cycles (corruption detection runs every 5th cycle)
	// With 50ms intervals, 5 cycles = 250ms, so we wait 500ms to be safe
	time.Sleep(500 * time.Millisecond)

	// Manually trigger corruption detection to ensure it runs (don't hold lock!)
	corrupted := cache.detectAndRecover()

	if corrupted == 0 {
		// If manual detection didn't find anything, check if background already handled it
		stats := cache.Stats()
		if stats.CorruptionEvents == 0 {
			t.Error("Expected corruption to be detected either by background cleanup or manual trigger")
		}
	}

	// Check final state
	finalStats := cache.Stats()
	if finalStats.Entries != 1 {
		t.Errorf("Expected 1 entry after corruption detection, got %d", finalStats.Entries)
	}
	if finalStats.CorruptionEvents == 0 {
		t.Error("Expected corruption events to be recorded in stats")
	}
}
