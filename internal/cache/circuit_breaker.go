package cache

import (
	"errors"
	"sync"
	"time"
	
	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

// CircuitState represents the current state of the circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota // Normal operation
	CircuitOpen                       // Failing, blocking requests  
	CircuitHalfOpen                   // Testing if service recovered
)

// CircuitBreakerConfig defines circuit breaker behavior
type CircuitBreakerConfig struct {
	MaxFailures     int           `json:"max_failures"`      // Failures before opening
	ResetTimeout    time.Duration `json:"reset_timeout"`     // Time before trying half-open
	SuccessReset    int           `json:"success_reset"`     // Successes needed to close
	FailureThreshold float64      `json:"failure_threshold"` // Failure rate threshold
	RequestVolumeThreshold int    `json:"request_volume_threshold"` // Min requests for evaluation
	SlidingWindowSize time.Duration `json:"sliding_window_size"` // Time window for metrics
}

// DefaultCircuitBreakerConfig returns production-safe circuit breaker settings
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:            5,
		ResetTimeout:           30 * time.Second,
		SuccessReset:           3,
		FailureThreshold:       0.5, // 50% failure rate
		RequestVolumeThreshold: 10,  // Need at least 10 requests
		SlidingWindowSize:      60 * time.Second,
	}
}

// CircuitBreakerMetrics tracks circuit breaker performance
type CircuitBreakerMetrics struct {
	TotalRequests    int64 `json:"total_requests"`
	SuccessfulRequests int64 `json:"successful_requests"`
	FailedRequests   int64 `json:"failed_requests"`
	CircuitOpenCount int64 `json:"circuit_open_count"`
	LastFailure      time.Time `json:"last_failure"`
	LastSuccess      time.Time `json:"last_success"`
}

// RequestResult represents the outcome of a request
type RequestResult struct {
	Success   bool
	Timestamp time.Time
	Error     error
}

// CircuitBreaker implements the circuit breaker pattern for cache fallback
type CircuitBreaker struct {
	config           CircuitBreakerConfig
	state            CircuitState
	failures         int
	successes        int
	lastFailureTime  time.Time
	lastSuccessTime  time.Time
	requestHistory   []RequestResult
	metrics          CircuitBreakerMetrics
	fallbackCache    Cache // Fallback cache for stale data
	mu               sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker with fallback cache
func NewCircuitBreaker(config CircuitBreakerConfig, fallbackCache Cache) *CircuitBreaker {
	return &CircuitBreaker{
		config:        config,
		state:         CircuitClosed,
		fallbackCache: fallbackCache,
		requestHistory: make([]RequestResult, 0),
	}
}

// Execute runs a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() (interface{}, error)) (interface{}, error) {
	return cb.executeWithOptions(fn, true) // Use generic fallback
}

// ExecuteWithStaleCache executes with cache-first fallback strategy
func (cb *CircuitBreaker) ExecuteWithStaleCache(key string, fn func() (interface{}, error)) (interface{}, error) {
	result, err := cb.executeWithOptions(fn, false) // Don't use generic fallback
	if err != nil {
		// Enhanced observability: log circuit breaker activity
		log.Warn("Circuit breaker triggered for key", 
			"key", key,
			"error", err,
			"circuit_state", cb.getStateString(),
			"failure_count", cb.failures,
			"last_failure", cb.lastFailureTime)
			
		// Try to get stale data from fallback cache
		if staleData, exists := cb.getStaleData(key); exists {
			log.Info("Serving stale data from fallback cache",
				"key", key,
				"circuit_state", cb.getStateString())
			return staleData, nil
		}
		
		log.Warn("No stale data available for key", 
			"key", key,
			"circuit_state", cb.getStateString())
	}
	return result, err
}

// executeWithOptions is the internal execution method
func (cb *CircuitBreaker) executeWithOptions(fn func() (interface{}, error), useGenericFallback bool) (interface{}, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	// Clean old requests from sliding window
	cb.cleanOldRequests()
	
	// Only check if circuit should be opened if we're in closed state
	if cb.state == CircuitClosed && cb.shouldOpenCircuit() {
		cb.openCircuit()
	}
	
	switch state := cb.state; state {
	case CircuitOpen:
		// Circuit is open, check if we should try half-open with jitter to prevent thundering herd
		timeoutWithJitter := addJitter(cb.config.ResetTimeout, 0.2) // 20% jitter
		if time.Since(cb.lastFailureTime) > timeoutWithJitter {
			cb.state = CircuitHalfOpen
			cb.successes = 0
			log.Info("Circuit breaker entering half-open state with jitter",
				"base_timeout", cb.config.ResetTimeout,
				"actual_timeout", timeoutWithJitter)
		} else {
			// Circuit still open, return fallback data only if using generic fallback
			if useGenericFallback {
				return cb.getFallbackData()
			} else {
				return nil, errors.New("circuit breaker open")
			}
		}
		
	case CircuitHalfOpen:
		// Allow limited requests to test if service recovered
		
	case CircuitClosed:
		// Normal operation
		
	default:
		// Should not happen, but handle gracefully
		log.Warn("Circuit breaker in unknown state, treating as closed")
	}
	
	// Execute the function
	result, err := fn()
	cb.recordRequest(err == nil)
	
	if err != nil {
		cb.handleFailure(err)
		
		// Check if circuit should be opened after recording the failure
		if cb.state == CircuitClosed && cb.shouldOpenCircuit() {
			cb.openCircuit()
		}
		
		// Return fallback data on failure only if using generic fallback
		if useGenericFallback {
			if fallback, fallbackErr := cb.getFallbackData(); fallbackErr == nil {
				log.Warn("Using fallback data due to upstream failure",
					"original_error", err,
					"circuit_state", cb.getStateString())
				return fallback, nil
			}
		}
		return nil, err
	}
	
	cb.handleSuccess()
	return result, nil
}

// recordRequest adds a request result to the sliding window
func (cb *CircuitBreaker) recordRequest(success bool) {
	now := time.Now()
	result := RequestResult{
		Success:   success,
		Timestamp: now,
	}
	
	cb.requestHistory = append(cb.requestHistory, result)
	cb.metrics.TotalRequests++
	
	if success {
		cb.metrics.SuccessfulRequests++
		cb.metrics.LastSuccess = now
	} else {
		cb.metrics.FailedRequests++
		cb.metrics.LastFailure = now
	}
}

// cleanOldRequests removes requests outside the sliding window
func (cb *CircuitBreaker) cleanOldRequests() {
	cutoff := time.Now().Add(-cb.config.SlidingWindowSize)
	newHistory := make([]RequestResult, 0, len(cb.requestHistory))
	
	for _, req := range cb.requestHistory {
		if req.Timestamp.After(cutoff) {
			newHistory = append(newHistory, req)
		}
	}
	
	cb.requestHistory = newHistory
}

// shouldOpenCircuit determines if circuit should be opened
func (cb *CircuitBreaker) shouldOpenCircuit() bool {
	if len(cb.requestHistory) < cb.config.RequestVolumeThreshold {
		return false // Not enough data
	}
	
	failures := 0
	for _, req := range cb.requestHistory {
		if !req.Success {
			failures++
		}
	}
	
	failureRate := float64(failures) / float64(len(cb.requestHistory))
	return failureRate >= cb.config.FailureThreshold
}

// openCircuit transitions to open state
func (cb *CircuitBreaker) openCircuit() {
	if cb.state != CircuitOpen {
		cb.state = CircuitOpen
		cb.metrics.CircuitOpenCount++
		cb.lastFailureTime = time.Now()
		
		log.Warn("Circuit breaker opened due to high failure rate",
			"failure_rate", cb.getFailureRate(),
			"failures", cb.failures,
			"total_requests", len(cb.requestHistory))
	}
}

// handleFailure processes a failed request
func (cb *CircuitBreaker) handleFailure(err error) {
	cb.failures++
	cb.lastFailureTime = time.Now()
	
	if cb.state == CircuitHalfOpen {
		// Failure in half-open state, go back to open
		cb.state = CircuitOpen
		cb.successes = 0
		log.Warn("Circuit breaker returned to open state after half-open failure", 
			"error", err)
	}
}

// handleSuccess processes a successful request
func (cb *CircuitBreaker) handleSuccess() {
	cb.lastSuccessTime = time.Now()
	
	if cb.state == CircuitHalfOpen {
		cb.successes++
		if cb.successes >= cb.config.SuccessReset {
			// Enough successes, close the circuit
			cb.state = CircuitClosed
			cb.failures = 0
			cb.successes = 0
			log.Info("Circuit breaker recovered and closed",
				"recovery_successes", cb.config.SuccessReset,
				"total_failures_cleared", cb.failures,
				"downtime_duration", time.Since(cb.lastFailureTime),
				"recovery_time", time.Now())
		}
	} else if cb.state == CircuitClosed {
		// Reset failure count on success
		cb.failures = 0
	}
}

// getFallbackData returns cached fallback data
func (cb *CircuitBreaker) getFallbackData() (interface{}, error) {
	if cb.fallbackCache == nil {
		return nil, errors.New("circuit breaker open and no fallback cache available")
	}
	
	// This is a simplified fallback - in practice you'd want to:
	// - Return the most recent cached data
	// - Extend TTL on stale but usable data
	// - Return default/empty data structure
	
	return map[string]interface{}{
		"status": "fallback",
		"message": "Service temporarily unavailable, using cached data",
		"timestamp": time.Now(),
	}, nil
}

// getStaleData attempts to retrieve stale data from fallback cache
func (cb *CircuitBreaker) getStaleData(key string) (interface{}, bool) {
	if cb.fallbackCache == nil {
		return nil, false
	}
	
	// Try to get data even if expired
	if memCache, ok := cb.fallbackCache.(*MemoryCache); ok {
		memCache.mu.RLock()
		defer memCache.mu.RUnlock()
		
		if entry, exists := memCache.data[key]; exists {
			// Return stale data regardless of expiration
			entry.AccessedAt = time.Now() // Update access time
			return entry.Value, true
		}
	}
	
	return nil, false
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// getStateString returns human-readable state
func (cb *CircuitBreaker) getStateString() string {
	switch state := cb.state; state {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// getFailureRate calculates current failure rate
func (cb *CircuitBreaker) getFailureRate() float64 {
	if len(cb.requestHistory) == 0 {
		return 0.0
	}
	
	failures := 0
	for _, req := range cb.requestHistory {
		if !req.Success {
			failures++
		}
	}
	
	return float64(failures) / float64(len(cb.requestHistory))
}

// GetMetrics returns circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() CircuitBreakerMetrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	metrics := cb.metrics
	return metrics
}

// GetDetailedStatus returns comprehensive status information
func (cb *CircuitBreaker) GetDetailedStatus() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	return map[string]interface{}{
		"state":           cb.getStateString(),
		"failures":        cb.failures,
		"successes":       cb.successes,
		"failure_rate":    cb.getFailureRate(),
		"requests_in_window": len(cb.requestHistory),
		"last_failure":    cb.lastFailureTime,
		"last_success":    cb.lastSuccessTime,
		"config":          cb.config,
		"metrics":         cb.metrics,
	}
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.state = CircuitClosed
	cb.failures = 0
	cb.successes = 0
	cb.requestHistory = make([]RequestResult, 0)
	
	// Reset metrics
	cb.metrics = CircuitBreakerMetrics{}
	
	log.Info("Circuit breaker manually reset to closed state")
}
