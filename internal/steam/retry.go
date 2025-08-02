package steam

import (
	"math"
	"time"
)

type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Second,
		MaxDelay:    10 * time.Second,
	}
}

type RetryableFunc func() (*APIError, bool)

func WithRetry(config RetryConfig, fn RetryableFunc) *APIError {
	var lastErr *APIError
	
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Execute the function
		err, shouldStop := fn()
		
		// Success case
		if err == nil {
			return nil
		}
		
		lastErr = err
		
		// Don't retry if explicitly told to stop or error is not retryable
		if shouldStop || !err.Retryable {
			break
		}
		
		// Don't sleep after the last attempt
		if attempt < config.MaxAttempts-1 {
			delay := calculateBackoffDelay(attempt, config)
			time.Sleep(delay)
		}
	}
	
	return lastErr
}

// calculateBackoffDelay implements exponential backoff with jitter
func calculateBackoffDelay(attempt int, config RetryConfig) time.Duration {
	// Exponential backoff: baseDelay * 2^attempt
	delay := float64(config.BaseDelay) * math.Pow(2, float64(attempt))
	
	// Cap at maxDelay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}
	
	return time.Duration(delay)
}
