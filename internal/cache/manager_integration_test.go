package cache

import (
	"errors"
	"testing"
	"time"
)

func TestManagerWithCircuitBreaker(t *testing.T) {
	config := DefaultConfig()
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()
	
	// Verify circuit breaker is created
	cb := manager.GetCircuitBreaker()
	if cb == nil {
		t.Fatal("Expected circuit breaker to be created")
	}
	
	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected initial circuit state to be closed, got %v", cb.GetState())
	}
}

func TestManagerExecuteWithFallback(t *testing.T) {
	config := DefaultConfig()
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()
	
	// First, put some data in cache for fallback
	testKey := "test:fallback"
	testValue := "cached_data"
	manager.GetCache().Set(testKey, testValue, time.Minute)
	
	// Test successful execution
	result, err := manager.ExecuteWithFallback("test:success", func() (interface{}, error) {
		return "fresh_data", nil
	})
	
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	
	if result != "fresh_data" {
		t.Errorf("Expected 'fresh_data', got %v", result)
	}
	
	// Test failed execution with fallback
	result, err = manager.ExecuteWithFallback(testKey, func() (interface{}, error) {
		return nil, errors.New("simulated API failure")
	})
	
	if err != nil {
		t.Errorf("Expected fallback data, got error: %v", err)
	}
	
	if result != testValue {
		t.Errorf("Expected fallback value '%s', got %v", testValue, result)
	}
}

func TestManagerGetCacheStatus(t *testing.T) {
	config := DefaultConfig()
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()
	
	status := manager.GetCacheStatus()
	
	// Check expected fields
	expectedFields := []string{"cache_type", "config", "circuit_breaker", "cache_stats"}
	for _, field := range expectedFields {
		if _, exists := status[field]; !exists {
			t.Errorf("Expected field '%s' in cache status", field)
		}
	}
	
	// Verify cache type
	if status["cache_type"] != MemoryCacheType {
		t.Errorf("Expected cache type %v, got %v", MemoryCacheType, status["cache_type"])
	}
	
	// Verify circuit breaker status is included
	cbStatus, ok := status["circuit_breaker"].(map[string]interface{})
	if !ok {
		t.Error("Expected circuit_breaker to be a map")
	} else {
		if cbStatus["state"] != "closed" {
			t.Errorf("Expected circuit breaker state 'closed', got %v", cbStatus["state"])
		}
	}
}

func TestManagerCircuitBreakerIntegration(t *testing.T) {
	config := DefaultConfig()
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()
	
	cb := manager.GetCircuitBreaker()
	
	// Configure circuit breaker for quick testing
	cb.config.MaxFailures = 2
	cb.config.RequestVolumeThreshold = 2
	cb.config.FailureThreshold = 0.5
	
	// Generate failures to open circuit
	for i := 0; i < 3; i++ {
		manager.ExecuteWithFallback("test", func() (interface{}, error) {
			return nil, errors.New("failure")
		})
	}
	
	// Verify circuit is open
	if cb.GetState() != CircuitOpen {
		t.Errorf("Expected circuit to be open after failures, got %v", cb.GetState())
	}
	
	// Check that status reflects the open circuit
	status := manager.GetCacheStatus()
	cbStatus := status["circuit_breaker"].(map[string]interface{})
	if cbStatus["state"] != "open" {
		t.Errorf("Expected circuit breaker status to show 'open', got %v", cbStatus["state"])
	}
}

func TestManagerWithTTLConfiguration(t *testing.T) {
	// Test that manager properly uses TTL configuration
	config := DefaultConfig()
	
	// Verify TTL config is present
	if config.TTL.PlayerStats == 0 {
		t.Error("Expected PlayerStats TTL to be configured")
	}
	
	if config.TTL.DefaultTTL == 0 {
		t.Error("Expected DefaultTTL to be configured")
	}
	
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()
	
	// Verify cache uses the configured TTL
	cache := manager.GetCache()
	if memCache, ok := cache.(*MemoryCache); ok {
		if memCache.defaultTTL != config.TTL.DefaultTTL {
			t.Errorf("Expected cache DefaultTTL %v to match config TTL %v",
				memCache.defaultTTL, config.TTL.DefaultTTL)
		}
	}
}

func TestManagerClose(t *testing.T) {
	config := DefaultConfig()
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	
	// Verify manager can be closed without error
	err = manager.Close()
	if err != nil {
		t.Errorf("Expected clean close, got error: %v", err)
	}
}

func TestCircuitBreakerWithRealCache(t *testing.T) {
	config := DefaultConfig()
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()
	
	// Add some test data to cache
	cache := manager.GetCache()
	cache.Set("player:123", map[string]interface{}{
		"name": "TestPlayer",
		"rank": "Gold",
	}, time.Minute)
	
	// Test successful execution
	result, err := manager.ExecuteWithFallback("api:player:123", func() (interface{}, error) {
		return map[string]interface{}{
			"name": "TestPlayer",
			"rank": "Platinum", // Updated data
		}, nil
	})
	
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}
	
	if resultMap["rank"] != "Platinum" {
		t.Errorf("Expected fresh data 'Platinum', got %v", resultMap["rank"])
	}
	
	// Test fallback to stale cache data
	result, err = manager.ExecuteWithFallback("player:123", func() (interface{}, error) {
		return nil, errors.New("API temporarily unavailable")
	})
	
	if err != nil {
		t.Errorf("Expected stale data fallback, got error: %v", err)
	}
	
	staleMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected stale result to be a map")
	}
	
	if staleMap["rank"] != "Gold" {
		t.Errorf("Expected stale data 'Gold', got %v", staleMap["rank"])
	}
}
