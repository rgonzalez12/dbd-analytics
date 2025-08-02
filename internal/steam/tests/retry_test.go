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
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
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
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
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
	}

	attempts := 0
	
	err := steam.WithRetry(config, func() (*steam.APIError, bool) {
		attempts++
		return steam.NewRateLimitError(), false // Always fail with retryable error
	})

	if err == nil {
		t.Error("Expected error after exhausting all attempts")
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
		http.StatusTooManyRequests, // 429
		http.StatusBadGateway,      // 502
		http.StatusServiceUnavailable, // 503
		http.StatusGatewayTimeout,     // 504
	}

	nonRetryableCodes := []int{
		http.StatusBadRequest,          // 400
		http.StatusUnauthorized,        // 401
		http.StatusForbidden,           // 403
		http.StatusNotFound,            // 404
		http.StatusInternalServerError, // 500
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
