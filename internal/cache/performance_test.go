package cache

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

// BenchmarkCorruptionDetection tests performance impact of corruption checks
func BenchmarkCorruptionDetection(b *testing.B) {
	cache := NewMemoryCache(MemoryCacheConfig{
		MaxEntries:      10000,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	defer cache.Close()

	// Populate cache with test data
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("test_key_%d", i)
		value := generateTestValue(i)
		cache.Set(key, value, 5*time.Minute)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.detectAndRecover()
	}
}

// BenchmarkCorruptionDetectionVsNormal compares with and without corruption detection
func BenchmarkCorruptionDetectionVsNormal(b *testing.B) {
	b.Run("WithCorruptionDetection", func(b *testing.B) {
		cache := NewMemoryCache(MemoryCacheConfig{
			MaxEntries:      5000,
			DefaultTTL:      5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
		})
		defer cache.Close()

		// Populate cache
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("test_key_%d", i)
			value := generateTestValue(i)
			cache.Set(key, value, 5*time.Minute)
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			cache.detectAndRecover()
		}
	})

	b.Run("NormalEviction", func(b *testing.B) {
		cache := NewMemoryCache(MemoryCacheConfig{
			MaxEntries:      5000,
			DefaultTTL:      5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
		})
		defer cache.Close()

		// Populate cache
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("test_key_%d", i)
			value := generateTestValue(i)
			cache.Set(key, value, 5*time.Minute)
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			cache.EvictExpired()
		}
	})
}

// BenchmarkCacheOperationsWithCorruption tests normal operations under corruption detection
func BenchmarkCacheOperationsWithCorruption(b *testing.B) {
	cache := NewMemoryCache(MemoryCacheConfig{
		MaxEntries:      5000,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 10 * time.Second, // Frequent corruption detection
	})
	defer cache.Close()

	// Pre-populate some data
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("base_key_%d", i)
		value := generateTestValue(i)
		cache.Set(key, value, 5*time.Minute)
	}

	b.ResetTimer()

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("bench_set_%d", i)
			value := generateTestValue(i)
			cache.Set(key, value, 5*time.Minute)
		}
	})

	b.Run("Get", func(b *testing.B) {
		keys := make([]string, 100)
		for i := 0; i < 100; i++ {
			keys[i] = fmt.Sprintf("base_key_%d", i)
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := keys[i%len(keys)]
			cache.Get(key)
		}
	})
}

// BenchmarkMemoryCalculation tests size calculation performance
func BenchmarkMemoryCalculation(b *testing.B) {
	cache := NewMemoryCache(MemoryCacheConfig{
		MaxEntries:      1000,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	defer cache.Close()

	values := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		values[i] = generateTestValue(i)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		value := values[i%len(values)]
		cache.calculateSize(value)
	}
}

// TestCorruptionDetectionPerformance measures corruption detection overhead
func TestCorruptionDetectionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cache := NewMemoryCache(MemoryCacheConfig{
		MaxEntries:      10000,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	defer cache.Close()

	// Populate cache with various data types
	for i := 0; i < 5000; i++ {
		key := fmt.Sprintf("perf_test_%d", i)
		value := generateComplexTestValue(i)
		cache.Set(key, value, 5*time.Minute)
	}

	// Measure corruption detection performance
	iterations := 100
	start := time.Now()

	for i := 0; i < iterations; i++ {
		cache.detectAndRecover()
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	t.Logf("Corruption detection performance:")
	t.Logf("  Entries: %d", len(cache.data))
	t.Logf("  Iterations: %d", iterations)
	t.Logf("  Total time: %v", duration)
	t.Logf("  Average per iteration: %v", avgDuration)
	t.Logf("  Entries per second: %.0f", float64(len(cache.data))/avgDuration.Seconds())

	// Performance threshold - corruption detection should process at least 10,000 entries/second
	entriesPerSecond := float64(len(cache.data)) / avgDuration.Seconds()
	if entriesPerSecond < 10000 {
		t.Errorf("Corruption detection too slow: %.0f entries/second (minimum: 10000)",
			entriesPerSecond)
	}

	// Memory usage check
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)

	t.Logf("Memory usage:")
	t.Logf("  Allocated: %d KB", m.Alloc/1024)
	t.Logf("  System: %d KB", m.Sys/1024)
	t.Logf("  GC cycles: %d", m.NumGC)
}

// BenchmarkCacheWarmVsCold tests performance difference between warm and cold cache
func BenchmarkCacheWarmVsCold(b *testing.B) {
	cache := NewMemoryCache(MemoryCacheConfig{
		MaxEntries:      5000,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	defer cache.Close()

	// Pre-populate keys for warm cache test
	warmKeys := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("warm_key_%d", i)
		warmKeys[i] = key
		cache.Set(key, generateTestValue(i), 5*time.Minute)
	}

	b.Run("ColdCache_Misses", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("cold_key_%d", i%1000)
			cache.Get(key) // Will be cache misses
		}
	})

	b.Run("WarmCache_Hits", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := warmKeys[i%len(warmKeys)]
			cache.Get(key) // Will be cache hits
		}
	})
}

// BenchmarkConcurrentReadWrite tests concurrent read/write contention
func BenchmarkConcurrentReadWrite(b *testing.B) {
	cache := NewMemoryCache(MemoryCacheConfig{
		MaxEntries:      10000,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	defer cache.Close()

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("concurrent_key_%d", i)
		cache.Set(key, generateTestValue(i), 5*time.Minute)
	}

	b.Run("ConcurrentReads", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("concurrent_key_%d", i%1000)
				cache.Get(key)
				i++
			}
		})
	})

	b.Run("ConcurrentWrites", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("write_key_%d", i)
				cache.Set(key, generateTestValue(i), 5*time.Minute)
				i++
			}
		})
	})

	b.Run("ConcurrentReadWrite", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				if i%3 == 0 {
					// 33% writes
					key := fmt.Sprintf("mixed_key_%d", i)
					cache.Set(key, generateTestValue(i), 5*time.Minute)
				} else {
					// 67% reads
					key := fmt.Sprintf("concurrent_key_%d", i%1000)
					cache.Get(key)
				}
				i++
			}
		})
	})
}
func generateTestValue(i int) interface{} {
	switch i % 4 {
	case 0:
		return fmt.Sprintf("string_value_%d", i)
	case 1:
		return i * 42
	case 2:
		return map[string]interface{}{
			"id":     i,
			"name":   fmt.Sprintf("item_%d", i),
			"active": i%2 == 0,
		}
	case 3:
		return []int{i, i + 1, i + 2, i + 3, i + 4}
	default:
		return "default"
	}
}

// generateComplexTestValue creates more complex test data
func generateComplexTestValue(i int) interface{} {
	return map[string]interface{}{
		"id":        i,
		"timestamp": time.Now().Unix(),
		"data":      generateTestValue(i),
		"metadata": map[string]interface{}{
			"version": "1.0",
			"source":  fmt.Sprintf("generator_%d", i%10),
			"tags":    []string{fmt.Sprintf("tag_%d", i%5), "test", "benchmark"},
		},
		"payload": make([]byte, 100+rand.Intn(400)), // Variable size payload
	}
}
