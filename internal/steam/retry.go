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
	Multiplier  float64
	Jitter      bool
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		Multiplier:  2.0,
		Jitter:      true,
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
	
	// Validate retry configuration
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 1
	}
	if config.BaseDelay <= 0 {
		config.BaseDelay = 100 * time.Millisecond
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 30 * time.Second
	}
	if config.Multiplier <= 1 {
		config.Multiplier = 2.0
	}
	
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Log retry attempt if this is not the first attempt
		if attempt > 0 && operation != "" {
			slog.Warn("Retrying operation after failure",
				slog.String("operation", operation),
				slog.Int("attempt", attempt+1),
				slog.Int("max_attempts", config.MaxAttempts),
				slog.String("last_error_type", string(lastErr.Type)),
				slog.String("last_error", lastErr.Message))
		}
		
		// Execute the function
		err, shouldStop := fn()
		
		// Success case
		if err == nil {
			if attempt > 0 && operation != "" {
				slog.Info("Operation succeeded after retry",
					slog.String("operation", operation),
					slog.Int("total_attempts", attempt+1))
			}
			return nil
		}
		
		lastErr = err
		
		// Don't retry if explicitly told to stop or error is not retryable
		if shouldStop || !err.Retryable {
			if operation != "" {
				slog.Info("Operation failed with non-retryable error",
					slog.String("operation", operation),
					slog.String("error_type", string(err.Type)),
					slog.Bool("retryable", err.Retryable))
			}
			break
		}
		
		// Don't sleep after the last attempt
		if attempt < config.MaxAttempts-1 {
			delay := calculateBackoffDelay(attempt, config)
			if operation != "" {
				slog.Debug("Waiting before retry",
					slog.String("operation", operation),
					slog.Duration("delay", delay),
					slog.Int("attempt", attempt+1))
			}
			time.Sleep(delay)
		}
	}
	
	// Log final failure
	if operation != "" {
		slog.Error("Operation failed after exhausting all retries",
			slog.String("operation", operation),
			slog.Int("total_attempts", config.MaxAttempts),
			slog.String("final_error_type", string(lastErr.Type)),
			slog.String("final_error", lastErr.Message),
			slog.Bool("retryable", lastErr.Retryable))
	}
	
	return lastErr
}

// calculateBackoffDelay implements exponential backoff with optional jitter
func calculateBackoffDelay(attempt int, config RetryConfig) time.Duration {
	// Exponential backoff: baseDelay * multiplier^attempt
	delay := float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(attempt))
	
	// Cap at maxDelay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}
	
	// Add jitter if enabled to prevent thundering herd
	if config.Jitter {
		// Jitter between 50% and 100% of calculated delay (0.5 to 1.0)
		jitterFactor := 0.5 + (rand.Float64() * 0.5)
		delay = delay * jitterFactor
	}
	
	return time.Duration(delay)
}
