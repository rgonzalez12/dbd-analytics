package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetCacheStatsJSON(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	req, err := http.NewRequest("GET", "/api/cache/stats", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.GetCacheStats(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check content type
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify expected fields
	expectedFields := []string{"cache_stats", "cache_type", "performance", "recommendations", "timestamp"}
	for _, field := range expectedFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Expected field %s in response", field)
		}
	}

	// Verify cache_stats structure
	cacheStats, ok := response["cache_stats"].(map[string]interface{})
	if !ok {
		t.Fatal("cache_stats should be an object")
	}

	expectedStatsFields := []string{"hits", "misses", "evictions", "entries", "hit_rate", "memory_usage", "corruption_events", "recovery_events"}
	for _, field := range expectedStatsFields {
		if _, exists := cacheStats[field]; !exists {
			t.Errorf("Expected field %s in cache_stats", field)
		}
	}
}

func TestEvictExpiredEntriesWithAuth(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	// Test without admin token
	req, err := http.NewRequest("POST", "/api/cache/evict", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.EvictExpiredEntries(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status code %d without admin token, got %d", http.StatusBadRequest, status)
	}

	// Create a new handler for the second test to avoid rate limiting
	handler2 := NewHandler()
	defer handler2.Close()
	
	// Test with admin token
	req2, err := http.NewRequest("POST", "/api/cache/evict", nil)
	if err != nil {
		t.Fatal(err)
	}
	req2.Header.Set("X-Admin-Token", "test-token")
	
	rr2 := httptest.NewRecorder()
	handler2.EvictExpiredEntries(rr2, req2)

	if status := rr2.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d with admin token, got %d", http.StatusOK, status)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(rr2.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify expected fields
	expectedFields := []string{"evicted_entries", "remaining_entries", "duration_ms", "timestamp", "admin_initiated"}
	for _, field := range expectedFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Expected field %s in eviction response", field)
		}
	}

	if adminInitiated, ok := response["admin_initiated"].(bool); !ok || !adminInitiated {
		t.Error("Expected admin_initiated to be true")
	}
}

func TestEvictExpiredEntriesRateLimit(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	req, err := http.NewRequest("POST", "/api/cache/evict", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Admin-Token", "test-token")

	// First request should succeed
	rr := httptest.NewRecorder()
	handler.EvictExpiredEntries(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("First eviction request should succeed, got status %d", status)
	}

	// Second immediate request should be rate limited
	rr = httptest.NewRecorder()
	handler.EvictExpiredEntries(rr, req)

	if status := rr.Code; status != http.StatusTooManyRequests {
		t.Errorf("Second immediate eviction request should be rate limited, got status %d", status)
	}
}

func TestGetMetricsPrometheusFormat(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	// Add some cache activity
	if handler.cacheManager != nil {
		cache := handler.cacheManager.GetCache()
		cache.Set("test1", "value1", time.Minute)
		cache.Set("test2", "value2", time.Minute)
		cache.Get("test1") // hit
		cache.Get("nonexistent") // miss
	}

	req, err := http.NewRequest("GET", "/metrics", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Set localhost IP to pass IP allowlisting
	req.RemoteAddr = "127.0.0.1:12345"

	rr := httptest.NewRecorder()
	handler.GetMetrics(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check content type
	expectedContentType := "text/plain; version=0.0.4"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected Content-Type %s, got %s", expectedContentType, contentType)
	}

	body := rr.Body.String()

	// Verify Prometheus format metrics are present
	expectedMetrics := []string{
		"cache_hits_total",
		"cache_misses_total", 
		"cache_evictions_total",
		"cache_lru_evictions_total",
		"cache_corruption_events_total",
		"cache_recovery_events_total",
		"cache_entries",
		"cache_memory_usage_bytes",
		"cache_hit_rate_percent",
		"cache_uptime_seconds",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("Expected metric %s not found in response", metric)
		}
	}

	// Verify HELP and TYPE comments are present
	if !strings.Contains(body, "# HELP cache_hits_total") {
		t.Error("Expected HELP comment for cache_hits_total")
	}
	if !strings.Contains(body, "# TYPE cache_hits_total counter") {
		t.Error("Expected TYPE comment for cache_hits_total")
	}

	// Verify actual metric values are numbers
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cache_") && !strings.HasPrefix(line, "#") {
			parts := strings.Fields(line)
			if len(parts) != 2 {
				t.Errorf("Invalid metric line format: %s", line)
			}
		}
	}
}

func TestCacheStatsWithCorruption(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	if handler.cacheManager == nil {
		t.Skip("Cache manager not available")
	}

	cache := handler.cacheManager.GetCache()
	
	// Trigger some corruption tracking by trying to set an unmarshallable value
	cache.Set("func_key", func(){}, time.Minute)

	req, err := http.NewRequest("GET", "/api/cache/stats", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.GetCacheStats(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	cacheStats, ok := response["cache_stats"].(map[string]interface{})
	if !ok {
		t.Fatal("cache_stats should be an object")
	}

	// Verify corruption metrics are tracked
	if _, exists := cacheStats["corruption_events"]; !exists {
		t.Error("Expected corruption_events field in cache stats")
	}
	if _, exists := cacheStats["recovery_events"]; !exists {
		t.Error("Expected recovery_events field in cache stats")
	}
}

func TestCacheStatsPerformanceAssessment(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	req, err := http.NewRequest("GET", "/api/cache/stats", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.GetCacheStats(rr, req)

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify performance assessment
	performance, ok := response["performance"].(map[string]interface{})
	if !ok {
		t.Fatal("performance should be an object")
	}

	expectedPerfFields := []string{"assessment", "memory_usage_mb", "ops_per_second", "total_requests", "uptime_hours"}
	for _, field := range expectedPerfFields {
		if _, exists := performance[field]; !exists {
			t.Errorf("Expected field %s in performance assessment", field)
		}
	}

	// Verify recommendations array exists
	if _, ok := response["recommendations"].([]interface{}); !ok {
		t.Error("Expected recommendations to be an array")
	}
}
