package cache

import (
	"testing"
	"time"
)

func TestMemoryCacheBasicOperations(t *testing.T) {
	config := MemoryCacheConfig{
		MaxEntries:      10,
		DefaultTTL:      1 * time.Second,
		CleanupInterval: 100 * time.Millisecond,
	}

	cache := NewMemoryCache(config)
	defer cache.Close()

	// Test Set and Get
	key := "test_key"
	value := "test_value"

	err := cache.Set(key, value, 1*time.Second)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	retrieved, found := cache.Get(key)
	if !found {
		t.Fatal("Expected to find cached value")
	}

	if retrieved != value {
		t.Fatalf("Expected %v, got %v", value, retrieved)
	}

	// Test cache miss
	_, found = cache.Get("non_existent_key")
	if found {
		t.Fatal("Expected cache miss for non-existent key")
	}
}

func TestMemoryCacheTTLExpiration(t *testing.T) {
	config := MemoryCacheConfig{
		MaxEntries:      10,
		DefaultTTL:      50 * time.Millisecond,
		CleanupInterval: 10 * time.Millisecond,
	}

	cache := NewMemoryCache(config)
	defer cache.Close()

	key := "expire_test"
	value := "will_expire"

	// Set with short TTL
	err := cache.Set(key, value, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Should be available immediately
	_, found := cache.Get(key)
	if !found {
		t.Fatal("Expected to find cached value immediately after set")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired now
	_, found = cache.Get(key)
	if found {
		t.Fatal("Expected cache entry to be expired")
	}
}

func TestMemoryCacheLRUEviction(t *testing.T) {
	config := MemoryCacheConfig{
		MaxEntries:      3,                // Small capacity to trigger LRU
		DefaultTTL:      10 * time.Second, // Long TTL so entries don't expire
		CleanupInterval: 1 * time.Second,
	}

	cache := NewMemoryCache(config)
	defer cache.Close()

	// Fill cache to capacity with slight delays to ensure different timestamps
	cache.Set("key1", "value1", 10*time.Second)
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	cache.Set("key2", "value2", 10*time.Second)
	time.Sleep(1 * time.Millisecond)
	cache.Set("key3", "value3", 10*time.Second)
	time.Sleep(1 * time.Millisecond)

	// Access key1 and key3 to make them recently used (key2 remains least recently used)
	_, found := cache.Get("key1")
	if !found {
		t.Fatal("Expected key1 to be accessible before eviction test")
	}
	time.Sleep(1 * time.Millisecond)

	_, found = cache.Get("key3")
	if !found {
		t.Fatal("Expected key3 to be accessible before eviction test")
	}
	time.Sleep(1 * time.Millisecond)

	// Add one more entry, should evict key2 (least recently used)
	cache.Set("key4", "value4", 10*time.Second)

	// Verify current state
	stats := cache.Stats()
	if stats.Entries != 3 {
		t.Fatalf("Expected 3 entries after eviction, got %d", stats.Entries)
	}

	// key1 should still be there (recently accessed)
	_, found = cache.Get("key1")
	if !found {
		t.Fatal("Expected key1 to still be in cache (recently accessed)")
	}

	// key2 should be evicted (least recently used - never accessed after initial set)
	_, found = cache.Get("key2")
	if found {
		t.Fatal("Expected key2 to be evicted due to LRU")
	}

	// key3 should still be there (accessed)
	_, found = cache.Get("key3")
	if !found {
		t.Fatal("Expected key3 to still be in cache")
	}

	// key4 should still be there (just added)
	_, found = cache.Get("key4")
	if !found {
		t.Fatal("Expected key4 to still be in cache")
	}
}

func TestMemoryCacheStats(t *testing.T) {
	config := MemoryCacheConfig{
		MaxEntries:      10,
		DefaultTTL:      1 * time.Second,
		CleanupInterval: 100 * time.Millisecond,
	}

	cache := NewMemoryCache(config)
	defer cache.Close()

	// Initial stats should be zero
	stats := cache.Stats()
	if stats.Hits != 0 || stats.Misses != 0 || stats.Entries != 0 {
		t.Fatal("Expected initial stats to be zero")
	}

	// Add some entries and test stats
	cache.Set("key1", "value1", 1*time.Second)
	cache.Set("key2", "value2", 1*time.Second)

	// Test cache hits
	cache.Get("key1")
	cache.Get("key1") // Second hit

	// Test cache miss
	cache.Get("non_existent")

	stats = cache.Stats()
	if stats.Hits != 2 {
		t.Fatalf("Expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Fatalf("Expected 1 miss, got %d", stats.Misses)
	}
	if stats.Entries != 2 {
		t.Fatalf("Expected 2 entries, got %d", stats.Entries)
	}

	// Check hit rate calculation
	expectedHitRate := float64(2) / float64(3) * 100 // 2 hits out of 3 total requests
	if stats.HitRate < expectedHitRate-0.1 || stats.HitRate > expectedHitRate+0.1 {
		t.Fatalf("Expected hit rate around %.2f%%, got %.2f%%", expectedHitRate, stats.HitRate)
	}
}

func TestCacheKeyGeneration(t *testing.T) {
	tests := []struct {
		prefix   string
		parts    []string
		expected string
	}{
		{"prefix", []string{}, "prefix"},
		{"prefix", []string{"part1"}, "prefix:part1"},
		{"prefix", []string{"part1", "part2"}, "prefix:part1:part2"},
		{"player_stats", []string{"example_user"}, "player_stats:example_user"},
	}

	for _, test := range tests {
		result := GenerateKey(test.prefix, test.parts...)
		if result != test.expected {
			t.Fatalf("Expected %s, got %s", test.expected, result)
		}
	}
}

func TestCacheManager(t *testing.T) {
	config := DefaultConfig()
	config.Memory.MaxEntries = 5
	config.Memory.DefaultTTL = 1 * time.Second

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.Close()

	cache := manager.GetCache()
	if cache == nil {
		t.Fatal("Expected cache to be available")
	}

	// Test basic functionality through manager
	err = cache.Set("test", "value", 1*time.Second)
	if err != nil {
		t.Fatalf("Failed to set through manager: %v", err)
	}

	value, found := cache.Get("test")
	if !found || value != "value" {
		t.Fatal("Failed to get value through manager")
	}
}
