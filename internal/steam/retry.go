package steam

import (
	"log/slog"
	"math"
	"math/rand"
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

// WithRetry executes a function with exponential backoff retry logic
func WithRetry(config RetryConfig, fn RetryableFunc) *APIError {
	return withRetryAndLogging(config, fn, "")
}

// WithRetryAndLogging executes a function with retry logic and structured logging
func withRetryAndLogging(config RetryConfig, fn RetryableFunc, operation string) *APIError {
	var lastErr *APIError
	
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Log retry attempt if this is not the first attempt
		if attempt > 0 && operation != "" {
			slog.Warn("Retrying operation",
				"operation", operation,
				"attempt", attempt+1,
				"max_attempts", config.MaxAttempts,
				"last_error", lastErr.Type)
		}
		
		// Execute the function
		err, shouldStop := fn()
		
		// Success case
		if err == nil {
			if attempt > 0 && operation != "" {
				slog.Info("Operation succeeded after retry",
					"operation", operation,
					"attempts", attempt+1)
			}
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
	
	// Log final failure
	if operation != "" {
		slog.Error("Operation failed after all retries",
			"operation", operation,
			"attempts", config.MaxAttempts,
			"final_error", lastErr.Type,
			"error_message", lastErr.Message)
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
	
	// Add jitter: random value between 0.5 and 1.5 of the calculated delay
	jitter := 0.5 + rand.Float64()
	delay = delay * jitter
	
	return time.Duration(delay)
}
