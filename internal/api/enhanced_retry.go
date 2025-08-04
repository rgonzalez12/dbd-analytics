package api

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/log"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

// RetryPolicy defines the advanced retry behavior
type RetryPolicy struct {
	MaxRetries     int           `json:"max_retries"`
	BaseBackoff    time.Duration `json:"base_backoff"`
	MaxBackoff     time.Duration `json:"max_backoff"`
	Multiplier     float64       `json:"multiplier"`
	JitterPercent  float64       `json:"jitter_percent"`
	CircuitBreaker bool          `json:"circuit_breaker"`

	// Per-error-type retry limits
	RateLimitRetries int `json:"rate_limit_retries"`
	NetworkRetries   int `json:"network_retries"`
	TimeoutRetries   int `json:"timeout_retries"`
}

// DefaultRetryPolicy returns production-ready retry settings
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:       5,
		BaseBackoff:      500 * time.Millisecond,
		MaxBackoff:       30 * time.Second,
		Multiplier:       2.0,
		JitterPercent:    0.25, // ±25%
		CircuitBreaker:   true,
		RateLimitRetries: 3,
		NetworkRetries:   4,
		TimeoutRetries:   2,
	}
}

// RetryContext tracks retry state and metrics
type RetryContext struct {
	Attempt        int             `json:"attempt"`
	TotalDuration  time.Duration   `json:"total_duration"`
	LastError      error           `json:"-"`
	LastErrorType  string          `json:"last_error_type"`
	ErrorCounts    map[string]int  `json:"error_counts"`
	BackoffHistory []time.Duration `json:"backoff_history"`
	StartTime      time.Time       `json:"start_time"`
}

// NewRetryContext creates a new retry context
func NewRetryContext() *RetryContext {
	return &RetryContext{
		Attempt:        0,
		ErrorCounts:    make(map[string]int),
		BackoffHistory: make([]time.Duration, 0),
		StartTime:      time.Now(),
	}
}

// EnhancedRetrier provides advanced retry capabilities with circuit breaking
type EnhancedRetrier struct {
	policy RetryPolicy
	config APIConfig
}

// NewEnhancedRetrier creates a new enhanced retrier
func NewEnhancedRetrier(policy RetryPolicy, config APIConfig) *EnhancedRetrier {
	return &EnhancedRetrier{
		policy: policy,
		config: config,
	}
}

// RetryableFunc represents a function that can be retried
type RetryableFunc func(ctx context.Context, attempt int) error

// Execute runs a function with advanced retry logic
func (r *EnhancedRetrier) Execute(ctx context.Context, operation string, fn RetryableFunc) error {
	retryCtx := NewRetryContext()

	for retryCtx.Attempt <= r.policy.MaxRetries {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled after %d attempts: %w", retryCtx.Attempt, ctx.Err())
		default:
		}

		// Calculate and apply backoff (except for first attempt)
		if retryCtx.Attempt > 0 {
			backoff := r.calculateAdvancedBackoff(retryCtx)
			retryCtx.BackoffHistory = append(retryCtx.BackoffHistory, backoff)

			log.Debug("Applying backoff before retry",
				"operation", operation,
				"attempt", retryCtx.Attempt,
				"backoff", backoff,
				"total_duration", time.Since(retryCtx.StartTime))

			select {
			case <-time.After(backoff):
				// Continue with retry
			case <-ctx.Done():
				return fmt.Errorf("operation cancelled during backoff: %w", ctx.Err())
			}
		}

		// Execute the function
		err := fn(ctx, retryCtx.Attempt)
		if err == nil {
			// Success!
			retryCtx.TotalDuration = time.Since(retryCtx.StartTime)

			if retryCtx.Attempt > 0 {
				log.Info("Operation succeeded after retry",
					"operation", operation,
					"attempts", retryCtx.Attempt+1,
					"total_duration", retryCtx.TotalDuration,
					"error_counts", retryCtx.ErrorCounts)
			}

			return nil
		}

		// Handle the error
		retryCtx.LastError = err
		retryCtx.LastErrorType = classifyError(err)
		retryCtx.ErrorCounts[retryCtx.LastErrorType]++

		// Check if we should retry based on error type and limits
		if !r.shouldRetry(retryCtx, err) {
			retryCtx.TotalDuration = time.Since(retryCtx.StartTime)

			log.Error("Operation failed after evaluation",
				"operation", operation,
				"attempts", retryCtx.Attempt+1,
				"total_duration", retryCtx.TotalDuration,
				"final_error", err,
				"error_type", retryCtx.LastErrorType,
				"error_counts", retryCtx.ErrorCounts)

			return fmt.Errorf("operation failed after %d attempts: %w", retryCtx.Attempt+1, err)
		}

		retryCtx.Attempt++
	}

	// Maximum retries exceeded
	retryCtx.TotalDuration = time.Since(retryCtx.StartTime)

	log.Error("Operation failed - maximum retries exceeded",
		"operation", operation,
		"max_retries", r.policy.MaxRetries,
		"total_duration", retryCtx.TotalDuration,
		"final_error", retryCtx.LastError,
		"error_counts", retryCtx.ErrorCounts)

	return fmt.Errorf("operation failed after %d attempts: %w", r.policy.MaxRetries+1, retryCtx.LastError)
}

// calculateAdvancedBackoff computes backoff with multiple strategies
func (r *EnhancedRetrier) calculateAdvancedBackoff(retryCtx *RetryContext) time.Duration {
	// Base exponential backoff
	backoff := time.Duration(float64(r.policy.BaseBackoff) * math.Pow(r.policy.Multiplier, float64(retryCtx.Attempt-1)))

	// Apply error-type specific adjustments
	switch retryCtx.LastErrorType {
	case "rate_limited":
		// Aggressive backoff for rate limiting
		backoff = time.Duration(float64(backoff) * 2.0)
	case "timeout":
		// Moderate backoff for timeouts
		backoff = time.Duration(float64(backoff) * 1.5)
	case "network_error":
		// Quick recovery for network issues
		backoff = time.Duration(float64(backoff) * 1.2)
	}

	// Cap at maximum
	if backoff > r.policy.MaxBackoff {
		backoff = r.policy.MaxBackoff
	}

	// Apply jitter to prevent thundering herd
	if r.policy.JitterPercent > 0 {
		jitterRange := time.Duration(float64(backoff) * r.policy.JitterPercent)
		jitter := time.Duration(rand.Int63n(int64(jitterRange*2))) - jitterRange
		backoff += jitter
	}

	// Ensure minimum backoff
	if backoff < r.policy.BaseBackoff {
		backoff = r.policy.BaseBackoff
	}

	return backoff
}

// shouldRetry determines if we should retry based on error type and limits
func (r *EnhancedRetrier) shouldRetry(retryCtx *RetryContext, err error) bool {
	// Check overall attempt limit
	if retryCtx.Attempt >= r.policy.MaxRetries {
		return false
	}

	// Check error-type specific limits
	errorType := retryCtx.LastErrorType
	switch errorType {
	case "rate_limited":
		return retryCtx.ErrorCounts[errorType] <= r.policy.RateLimitRetries
	case "network_error":
		return retryCtx.ErrorCounts[errorType] <= r.policy.NetworkRetries
	case "timeout":
		return retryCtx.ErrorCounts[errorType] <= r.policy.TimeoutRetries
	case "private_profile", "validation_error", "not_found":
		// Never retry these
		return false
	case "steam_api_down":
		// Always retry these (up to global limit)
		return true
	default:
		// Unknown errors - be conservative but allow some retries
		return retryCtx.ErrorCounts[errorType] <= 2
	}
}

// GetRetryMetrics returns current retry metrics
func (retryCtx *RetryContext) GetRetryMetrics() map[string]interface{} {
	return map[string]interface{}{
		"attempt":         retryCtx.Attempt,
		"total_duration":  retryCtx.TotalDuration.String(),
		"error_counts":    retryCtx.ErrorCounts,
		"backoff_history": retryCtx.BackoffHistory,
		"last_error_type": retryCtx.LastErrorType,
	}
}

// SteamAPIRetrier provides Steam-specific retry wrapper
type SteamAPIRetrier struct {
	retrier *EnhancedRetrier
	client  SteamClientInterface
}

// NewSteamAPIRetrier creates a Steam API retrier
func NewSteamAPIRetrier(policy RetryPolicy, config APIConfig, client SteamClientInterface) *SteamAPIRetrier {
	return &SteamAPIRetrier{
		retrier: NewEnhancedRetrier(policy, config),
		client:  client,
	}
}

// GetPlayerStatsWithRetry fetches player stats with enhanced retry
func (s *SteamAPIRetrier) GetPlayerStatsWithRetry(ctx context.Context, steamID string) (*steam.SteamPlayerstats, error) {
	var result *steam.SteamPlayerstats

	err := s.retrier.Execute(ctx, "GetPlayerStats", func(ctx context.Context, attempt int) error {
		stats, apiErr := s.client.GetPlayerStats(steamID)
		if apiErr != nil {
			return apiErr
		}
		result = stats
		return nil
	})

	return result, err
}

// GetPlayerAchievementsWithRetry fetches achievements with enhanced retry
func (s *SteamAPIRetrier) GetPlayerAchievementsWithRetry(ctx context.Context, steamID string) (*steam.PlayerAchievements, error) {
	var result *steam.PlayerAchievements

	err := s.retrier.Execute(ctx, "GetPlayerAchievements", func(ctx context.Context, attempt int) error {
		achievements, apiErr := s.client.GetPlayerAchievements(steamID, "381210")
		if apiErr != nil {
			return apiErr
		}
		result = achievements
		return nil
	})

	return result, err
}
