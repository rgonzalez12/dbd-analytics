package cache

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerBasicOperation(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.MaxFailures = 3
	config.ResetTimeout = 100 * time.Millisecond

	cache := NewMemoryCache(DefaultConfig().Memory)
	cb := NewCircuitBreaker(config, cache)

	// Test successful operation
	result, err := cb.Execute(func() (interface{}, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}

	if result != "success" {
		t.Errorf("Expected 'success', got %v", result)
	}

	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected circuit to be closed, got %v", cb.GetState())
	}
}

func TestCircuitBreakerFailureHandling(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.MaxFailures = 2
	config.RequestVolumeThreshold = 2
	config.FailureThreshold = 0.5

	cache := NewMemoryCache(DefaultConfig().Memory)
	cb := NewCircuitBreaker(config, cache)

	// Generate failures
	for i := 0; i < 3; i++ {
		_, err := cb.Execute(func() (interface{}, error) {
			return nil, errors.New("simulated failure")
		})

		// Should return fallback data, not the original error
		if err != nil {
			t.Errorf("Expected fallback data, got error: %v", err)
		}
	}

	// Circuit should be open after threshold failures
	if cb.GetState() != CircuitOpen {
		t.Errorf("Expected circuit to be open after failures, got %v", cb.GetState())
	}
}

func TestCircuitBreakerHalfOpenRecovery(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.MaxFailures = 2
	config.ResetTimeout = 50 * time.Millisecond
	config.SuccessReset = 2
	config.RequestVolumeThreshold = 2
	config.FailureThreshold = 0.5

	cache := NewMemoryCache(DefaultConfig().Memory)
	cb := NewCircuitBreaker(config, cache)

	// Generate failures to open circuit
	for i := 0; i < 3; i++ {
		cb.Execute(func() (interface{}, error) {
			return nil, errors.New("failure")
		})
	}

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open and allow requests
	result, err := cb.Execute(func() (interface{}, error) {
		return "recovery", nil
	})

	if err != nil {
		t.Errorf("Expected success in half-open state, got error: %v", err)
	}

	if result != "recovery" {
		t.Errorf("Expected 'recovery', got %v", result)
	}

	// One more success should close the circuit
	cb.Execute(func() (interface{}, error) {
		return "success", nil
	})

	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected circuit to be closed after recovery, got %v", cb.GetState())
	}
}

func TestCircuitBreakerStaleDataFallback(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.MaxFailures = 1
	config.RequestVolumeThreshold = 1
	config.FailureThreshold = 0.5

	cache := NewMemoryCache(DefaultConfig().Memory)
	cb := NewCircuitBreaker(config, cache)

	// Put some data in the cache first
	testKey := "test:key"
	testValue := "cached_value"
	cache.Set(testKey, testValue, 1*time.Minute)

	// Generate failure to open circuit
	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure")
	})

	// Now execute with stale cache fallback
	result, err := cb.ExecuteWithStaleCache(testKey, func() (interface{}, error) {
		return nil, errors.New("still failing")
	})

	if err != nil {
		t.Errorf("Expected stale data fallback, got error: %v", err)
	}

	if result != testValue {
		t.Errorf("Expected stale cached value '%s', got %v", testValue, result)
	}
}

func TestCircuitBreakerMetrics(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	cache := NewMemoryCache(DefaultConfig().Memory)
	cb := NewCircuitBreaker(config, cache)

	// Execute some requests
	cb.Execute(func() (interface{}, error) { return "success", nil })
	cb.Execute(func() (interface{}, error) { return nil, errors.New("failure") })
	cb.Execute(func() (interface{}, error) { return "success", nil })

	metrics := cb.GetMetrics()

	if metrics.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", metrics.TotalRequests)
	}

	if metrics.SuccessfulRequests != 2 {
		t.Errorf("Expected 2 successful requests, got %d", metrics.SuccessfulRequests)
	}

	if metrics.FailedRequests != 1 {
		t.Errorf("Expected 1 failed request, got %d", metrics.FailedRequests)
	}
}

func TestCircuitBreakerDetailedStatus(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	cache := NewMemoryCache(DefaultConfig().Memory)
	cb := NewCircuitBreaker(config, cache)

	status := cb.GetDetailedStatus()

	// Check that all expected fields are present
	expectedFields := []string{
		"state", "failures", "successes", "failure_rate",
		"requests_in_window", "last_failure", "last_success",
		"config", "metrics",
	}

	for _, field := range expectedFields {
		if _, exists := status[field]; !exists {
			t.Errorf("Expected field '%s' in detailed status", field)
		}
	}

	// Initial state should be closed
	if status["state"] != "closed" {
		t.Errorf("Expected initial state 'closed', got %v", status["state"])
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.MaxFailures = 1
	config.RequestVolumeThreshold = 1
	config.FailureThreshold = 0.5

	cache := NewMemoryCache(DefaultConfig().Memory)
	cb := NewCircuitBreaker(config, cache)

	// Generate failure to open circuit
	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure")
	})

	// Verify circuit is open
	if cb.GetState() != CircuitOpen {
		t.Errorf("Expected circuit to be open, got %v", cb.GetState())
	}

	// Reset the circuit
	cb.Reset()

	// Verify circuit is closed
	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected circuit to be closed after reset, got %v", cb.GetState())
	}

	// Verify metrics are reset
	metrics := cb.GetMetrics()
	if metrics.TotalRequests != 0 {
		t.Errorf("Expected total requests to be reset to 0, got %d", metrics.TotalRequests)
	}
}

func TestCircuitBreakerSlidingWindow(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.SlidingWindowSize = 100 * time.Millisecond
	config.RequestVolumeThreshold = 2
	config.FailureThreshold = 0.5

	cache := NewMemoryCache(DefaultConfig().Memory)
	cb := NewCircuitBreaker(config, cache)

	// Add some requests
	cb.Execute(func() (interface{}, error) { return nil, errors.New("failure") })
	cb.Execute(func() (interface{}, error) { return "success", nil })

	status := cb.GetDetailedStatus()
	initialRequestCount := status["requests_in_window"].(int)

	if initialRequestCount != 2 {
		t.Errorf("Expected 2 requests in window, got %d", initialRequestCount)
	}

	// Wait for sliding window to expire
	time.Sleep(150 * time.Millisecond)

	// Add another request to trigger cleanup
	cb.Execute(func() (interface{}, error) { return "success", nil })

	status = cb.GetDetailedStatus()
	newRequestCount := status["requests_in_window"].(int)

	// Should only have the latest request
	if newRequestCount != 1 {
		t.Errorf("Expected 1 request in window after cleanup, got %d", newRequestCount)
	}
}
