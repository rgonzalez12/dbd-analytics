package steam_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

func TestRetryLogic_SuccessAfterRetry(t *testing.T) {
	config := steam.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      false, // Disable jitter for predictable testing
	}

	attempts := 0

	err := steam.WithRetry(config, func() (*steam.APIError, bool) {
		attempts++
		if attempts < 2 {
			return steam.NewRateLimitError(), false
		}
		return nil, false
	})

	if err != nil {
		t.Errorf("Expected success after retry, got error: %v", err)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestRetryLogic_NonRetryableError(t *testing.T) {
	config := steam.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      false,
	}

	attempts := 0

	err := steam.WithRetry(config, func() (*steam.APIError, bool) {
		attempts++
		return steam.NewNotFoundError("Player"), false
	})

	if err == nil {
		t.Error("Expected error for non-retryable case")
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestRetryLogic_ExhaustAllAttempts(t *testing.T) {
	config := steam.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond, // Very small delay for test speed
		MaxDelay:    10 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      false,
	}

	attempts := 0

	err := steam.WithRetry(config, func() (*steam.APIError, bool) {
		attempts++
		return steam.NewRateLimitError(), false // Always fail with retryable error
	})

	if err == nil {
		t.Error("Expected error after exhausting all attempts")
		return
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	if err.Type != steam.ErrorTypeRateLimit {
		t.Errorf("Expected rate limit error, got %v", err.Type)
	}
}

func TestRetryLogic_SpecificStatusCodes(t *testing.T) {
	retryableCodes := []int{
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	}

	nonRetryableCodes := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
	}

	config := steam.RetryConfig{
		MaxAttempts: 2,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	// Test retryable status codes
	for _, code := range retryableCodes {
		t.Run(fmt.Sprintf("Retryable_%d", code), func(t *testing.T) {
			attempts := 0
			err := steam.WithRetry(config, func() (*steam.APIError, bool) {
				attempts++
				if attempts == 1 {
					return steam.NewAPIError(code, "test error"), false
				}
				return nil, false // Success on second attempt
			})

			if err != nil {
				t.Errorf("Expected success after retry for status %d, got error: %v", code, err)
			}

			if attempts != 2 {
				t.Errorf("Expected 2 attempts for status %d, got %d", code, attempts)
			}
		})
	}

	// Test non-retryable status codes
	for _, code := range nonRetryableCodes {
		t.Run(fmt.Sprintf("NonRetryable_%d", code), func(t *testing.T) {
			attempts := 0
			err := steam.WithRetry(config, func() (*steam.APIError, bool) {
				attempts++
				return steam.NewAPIError(code, "test error"), false
			})

			if err == nil {
				t.Errorf("Expected error for non-retryable status %d", code)
			}

			if attempts != 1 {
				t.Errorf("Expected 1 attempt for non-retryable status %d, got %d", code, attempts)
			}
		})
	}
}

func TestRetryLogic_MockHTTPServer(t *testing.T) {
	// Track the number of requests
	requestCount := 0

	// Create a mock server that returns 429 twice, then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"response":{"players":[{"steamid":"123"}]}}`))
	}))
	defer server.Close()

	// Create a client with custom retry config
	config := steam.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	// Test the retry logic with actual HTTP calls
	err := steam.WithRetry(config, func() (*steam.APIError, bool) {
		resp, httpErr := http.Get(server.URL)
		if httpErr != nil {
			return steam.NewNetworkError(httpErr), false
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			return steam.NewRateLimitError(), false
		}

		if resp.StatusCode != http.StatusOK {
			return steam.NewAPIError(resp.StatusCode, "HTTP error"), false
		}

		return nil, false
	})

	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
	}

	if requestCount != 3 {
		t.Errorf("Expected 3 HTTP requests, got %d", requestCount)
	}
}

func TestRetryLogic_MaxRetryCap(t *testing.T) {
	// Test that retry logic respects the maximum retry limit
	config := steam.RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      false,
	}

	attempts := 0

	err := steam.WithRetry(config, func() (*steam.APIError, bool) {
		attempts++
		// Always return a retryable error
		return steam.NewAPIError(http.StatusBadGateway, "upstream error"), false
	})

	if err == nil {
		t.Error("Expected error after exhausting max attempts")
		return
	}

	if attempts != 5 {
		t.Errorf("Expected exactly 5 attempts (max cap), got %d", attempts)
	}

	if err.Type != steam.ErrorTypeAPIError {
		t.Errorf("Expected API error type, got %v", err.Type)
	}
}

func TestRetryLogic_HTTP429Failures(t *testing.T) {
	// Test specific HTTP 429 (Too Many Requests) failure scenarios
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// Always return 429 to test retry behavior
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer server.Close()

	config := steam.RetryConfig{
		MaxAttempts: 4,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    20 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      false,
	}

	err := steam.WithRetry(config, func() (*steam.APIError, bool) {
		resp, httpErr := http.Get(server.URL)
		if httpErr != nil {
			return steam.NewNetworkError(httpErr), false
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			return steam.NewRateLimitError(), false
		}

		return nil, false
	})

	if err == nil {
		t.Error("Expected rate limit error after all retries")
		return
	}

	if requestCount != 4 {
		t.Errorf("Expected 4 HTTP requests, got %d", requestCount)
	}

	if err.Type != steam.ErrorTypeRateLimit {
		t.Errorf("Expected rate limit error type, got %v", err.Type)
	}
}

func TestRetryLogic_HTTP500Failures(t *testing.T) {
	// Test HTTP 500 retry behavior
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer server.Close()

	config := steam.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      false,
	}

	err := steam.WithRetry(config, func() (*steam.APIError, bool) {
		resp, httpErr := http.Get(server.URL)
		if httpErr != nil {
			return steam.NewNetworkError(httpErr), false
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return steam.NewAPIError(resp.StatusCode, "HTTP error"), false
		}

		return nil, false
	})

	if err == nil {
		t.Error("Expected API error for 500 status")
		return
	}

	// Should make all retry attempts since 500 errors are now retryable
	if requestCount != 3 { // MaxAttempts = 3 total attempts
		t.Errorf("Expected 3 HTTP requests (MaxAttempts) for 500), got %d", requestCount)
	}

	if err.Type != steam.ErrorTypeAPIError {
		t.Errorf("Expected API error type, got %v", err.Type)
	}
}

func TestRetryLogic_ExponentialBackoffTiming(t *testing.T) {
	// Test that exponential backoff timing works correctly
	config := steam.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond, // Match our enhanced implementation
		MaxDelay:    2 * time.Second,
		Multiplier:  2.0,
		Jitter:      false, // Disable jitter for predictable timing
	}

	attempts := 0
	var attemptTimes []time.Time

	start := time.Now()

	err := steam.WithRetry(config, func() (*steam.APIError, bool) {
		attempts++
		attemptTimes = append(attemptTimes, time.Now())

		// Fail on first two attempts, succeed on third (use 500 error to force exponential backoff)
		if attempts < 3 {
			return steam.NewAPIError(500, "test error"), false
		}
		return nil, false
	})

	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
	}

	if len(attemptTimes) != 3 {
		t.Errorf("Expected 3 attempts, got %d", len(attemptTimes))
	}

	// Check timing between attempts
	if len(attemptTimes) >= 2 {
		// First retry delay should be around baseDelay (100ms)
		firstDelay := attemptTimes[1].Sub(attemptTimes[0])
		if firstDelay < 90*time.Millisecond || firstDelay > 150*time.Millisecond {
			t.Errorf("First retry delay should be ~100ms, got %v", firstDelay)
		}
	}

	if len(attemptTimes) >= 3 {
		// Second retry delay should be around baseDelay * multiplier (200ms)
		secondDelay := attemptTimes[2].Sub(attemptTimes[1])
		if secondDelay < 180*time.Millisecond || secondDelay > 250*time.Millisecond {
			t.Errorf("Second retry delay should be ~200ms, got %v", secondDelay)
		}
	}

	totalTime := time.Since(start)
	// Total time should be at least the sum of delays (~300ms) but not too much more
	if totalTime < 250*time.Millisecond || totalTime > 500*time.Millisecond {
		t.Errorf("Total retry time should be ~300ms, got %v", totalTime)
	}
}

func TestRetryLogic_ConfigValidation(t *testing.T) {
	// Test that invalid configurations are corrected
	invalidConfig := steam.RetryConfig{
		MaxAttempts: 0,   // Invalid
		BaseDelay:   0,   // Invalid
		MaxDelay:    0,   // Invalid
		Multiplier:  0.5, // Invalid (should be > 1)
		Jitter:      true,
	}

	attempts := 0

	err := steam.WithRetry(invalidConfig, func() (*steam.APIError, bool) {
		attempts++
		if attempts == 1 {
			return nil, false // Succeed on first attempt
		}
		return steam.NewRateLimitError(), false
	})

	if err != nil {
		t.Errorf("Expected success with corrected config, got error: %v", err)
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt with corrected config, got %d", attempts)
	}
}
