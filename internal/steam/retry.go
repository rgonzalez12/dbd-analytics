package steam

import (
	"log/slog"
	"math"
	"math/rand"
	"os"
	"strconv"
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
	// Read max retries from STEAM_MAX_RETRIES environment variable, default to 3
	maxRetries := 3
	if envRetries := os.Getenv("STEAM_MAX_RETRIES"); envRetries != "" {
		if parsed, err := strconv.Atoi(envRetries); err == nil && parsed >= 0 {
			maxRetries = parsed
		}
	}

	return RetryConfig{
		MaxAttempts: maxRetries,
		BaseDelay:   500 * time.Millisecond, // Initial delay for exponential backoff
		MaxDelay:    10 * time.Second,       // Maximum delay cap to prevent excessive waiting
		Multiplier:  2.0,                    // Exponential backoff: 500ms → 1s → 2s → 4s → 8s
		Jitter:      true,                   // Add randomization to prevent thundering herd
	}
}

type RetryableFunc func() (*APIError, bool)

// shouldRetryError determines if an error is worth retrying based on its type and status code
func shouldRetryError(err *APIError) bool {
	if err == nil {
		return false
	}

	// Don't retry validation errors or not found errors
	if err.Type == ErrorTypeValidation || err.Type == ErrorTypeNotFound {
		return false
	}

	// Evaluate status codes to determine retry eligibility - only retry on 429 and 5xx errors
	if err.StatusCode > 0 {
		switch err.StatusCode {
		case 429: // Too Many Requests - always retryable
			return true
		case 403, 404: // Forbidden, Not Found - permanent failures that should not be retried
			return false
		default:
			// Only retry on 5xx server errors (500-599)
			return err.StatusCode >= 500 && err.StatusCode < 600
		}
	}

	// For other error types, use the existing Retryable field
	return err.Retryable
}

// WithRetry executes a function with exponential backoff retry logic
func WithRetry(config RetryConfig, fn RetryableFunc) *APIError {
	return withRetryAndLogging(config, fn, "")
}

// WithRetryAndLogging executes a function with enhanced retry logic and structured logging
func withRetryAndLogging(config RetryConfig, fn RetryableFunc, operation string) *APIError {
	var lastErr *APIError

	// Validate and sanitize retry configuration parameters
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
		// Log retry attempt details if this is not the initial attempt
		if attempt > 0 && operation != "" {
			slog.Warn("Retrying operation after failure",
				slog.String("operation", operation),
				slog.Int("attempt", attempt+1),
				slog.Int("max_attempts", config.MaxAttempts),
				slog.String("last_error_type", string(lastErr.Type)),
				slog.Int("last_status_code", lastErr.StatusCode),
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

		// Check if we should retry this error type
		if shouldStop || !shouldRetryError(err) {
			if operation != "" {
				slog.Info("Operation failed with non-retryable error",
					slog.String("operation", operation),
					slog.String("error_type", string(err.Type)),
					slog.Int("status_code", err.StatusCode),
					slog.Bool("retryable", shouldRetryError(err)))
			}
			break
		}

		// Don't sleep after the last attempt
		if attempt < config.MaxAttempts-1 {
			delay := calculateRetryDelay(attempt, config, err)

			if operation != "" {
				slog.Debug("Waiting before retry",
					slog.String("operation", operation),
					slog.Duration("delay", delay),
					slog.Int("attempt", attempt+1),
					slog.String("delay_source", func() string {
						if err.Type == ErrorTypeRateLimit && err.RetryAfter > 0 {
							return "retry_after_header"
						}
						return "exponential_backoff"
					}()))
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

// calculateRetryDelay determines the delay before next retry attempt
// Respects API headers (Retry-After) or uses exponential backoff
func calculateRetryDelay(attempt int, config RetryConfig, err *APIError) time.Duration {
	// If we have a rate limit error with RetryAfter, use it
	if err.Type == ErrorTypeRateLimit && err.RetryAfter > 0 {
		delay := time.Duration(err.RetryAfter) * time.Second
		// Cap the delay to max delay to prevent extremely long waits
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
		return delay
	}

	// Otherwise use exponential backoff
	return calculateBackoffDelay(attempt, config)
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
