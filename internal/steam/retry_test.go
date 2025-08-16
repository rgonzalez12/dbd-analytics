package steam

import (
	"testing"
	"time"
)

func TestRetryLogic(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      false,
	}

	t.Run("Success after retry", func(t *testing.T) {
		attempts := 0
		err := WithRetry(config, func() (*APIError, bool) {
			attempts++
			if attempts < 2 {
				return NewRateLimitError(), false
			}
			return nil, false
		})

		if err != nil || attempts != 2 {
			t.Errorf("Expected success after 2 attempts, got error: %v, attempts: %d", err, attempts)
		}
	})

	t.Run("Non-retryable error", func(t *testing.T) {
		attempts := 0
		err := WithRetry(config, func() (*APIError, bool) {
			attempts++
			return NewNotFoundError("Player"), false
		})

		if err == nil || attempts != 1 {
			t.Errorf("Expected error after 1 attempt, got: %v, attempts: %d", err, attempts)
		}
	})

	t.Run("Exhaust all attempts", func(t *testing.T) {
		attempts := 0
		err := WithRetry(config, func() (*APIError, bool) {
			attempts++
			return NewRateLimitError(), false
		})

		if err == nil || attempts != 3 {
			t.Errorf("Expected error after 3 attempts, got: %v, attempts: %d", err, attempts)
		}
	})

	t.Run("Status code behavior", func(t *testing.T) {
		retryableCodes := []int{429, 500, 502, 503, 504}
		nonRetryableCodes := []int{400, 401, 403, 404}

		for _, code := range retryableCodes {
			apiErr := NewAPIError(code, "test")
			if !shouldRetryError(apiErr) {
				t.Errorf("Status %d should be retryable", code)
			}
		}

		for _, code := range nonRetryableCodes {
			apiErr := NewAPIError(code, "test")
			if shouldRetryError(apiErr) {
				t.Errorf("Status %d should not be retryable", code)
			}
		}
	})
}
