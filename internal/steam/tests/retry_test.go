package steam_test

import (
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
		t.Error("Expected non-retryable error to fail immediately")
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
	}
}
