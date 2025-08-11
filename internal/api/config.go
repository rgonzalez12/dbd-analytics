package api

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

// APIConfig holds configurable parameters for API resilience and performance
type APIConfig struct {
	// Circuit Breaker Configuration
	CBMaxFails         int `json:"cb_max_fails"`
	CBResetTimeoutSecs int `json:"cb_reset_timeout_secs"`
	CBHalfOpenRequests int `json:"cb_half_open_requests"`

	// API Timeout Configuration
	APITimeoutSecs          int `json:"api_timeout_secs"`
	OverallTimeoutSecs      int `json:"overall_timeout_secs"`
	AchievementsTimeoutSecs int `json:"achievements_timeout_secs"`

	// Retry Configuration
	MaxRetries    int `json:"max_retries"`
	BaseBackoffMs int `json:"base_backoff_ms"`
	MaxBackoffMs  int `json:"max_backoff_ms"`

	// Rate Limiting
	RateLimit  int `json:"rate_limit"`  // Requests per minute
	BurstLimit int `json:"burst_limit"` // Burst capacity

	// Computed fields for convenience
	APITimeout          time.Duration `json:"-"`
	OverallTimeout      time.Duration `json:"-"`
	AchievementsTimeout time.Duration `json:"-"`
	CBResetTimeout      time.Duration `json:"-"`
	BaseBackoff         time.Duration `json:"-"`
	MaxBackoff          time.Duration `json:"-"`
}

// DefaultAPIConfig returns sensible production defaults
func DefaultAPIConfig() APIConfig {
	config := APIConfig{
		// Circuit Breaker
		CBMaxFails:         5,
		CBResetTimeoutSecs: 60,
		CBHalfOpenRequests: 3,

		// Timeouts
		APITimeoutSecs:          10,
		OverallTimeoutSecs:      30,
		AchievementsTimeoutSecs: 5,

		// Retry - Exponential backoff with jitter
		MaxRetries:    3,    // Up to 3 retries
		BaseBackoffMs: 250,  // Start with 250ms
		MaxBackoffMs:  8000, // Cap at 8 seconds

		// Rate Limiting - Conservative for Steam API
		RateLimit:  100, // 100 requests per minute
		BurstLimit: 10,  // Allow bursts of 10
	}

	// Compute derived fields
	config.APITimeout = time.Duration(config.APITimeoutSecs) * time.Second
	config.OverallTimeout = time.Duration(config.OverallTimeoutSecs) * time.Second
	config.AchievementsTimeout = time.Duration(config.AchievementsTimeoutSecs) * time.Second
	config.CBResetTimeout = time.Duration(config.CBResetTimeoutSecs) * time.Second
	config.BaseBackoff = time.Duration(config.BaseBackoffMs) * time.Millisecond
	config.MaxBackoff = time.Duration(config.MaxBackoffMs) * time.Millisecond

	return config
}

// LoadAPIConfigFromEnv loads configuration from environment variables with fallbacks
func LoadAPIConfigFromEnv() APIConfig {
	config := DefaultAPIConfig()

	// Load from environment with fallbacks
	config.CBMaxFails = getEnvInt("CB_MAX_FAILS", config.CBMaxFails)
	config.CBResetTimeoutSecs = getEnvInt("CB_RESET_TIMEOUT_SECS", config.CBResetTimeoutSecs)
	config.CBHalfOpenRequests = getEnvInt("CB_HALF_OPEN_REQUESTS", config.CBHalfOpenRequests)

	config.APITimeoutSecs = getEnvInt("API_TIMEOUT_SECS", config.APITimeoutSecs)
	config.OverallTimeoutSecs = getEnvInt("OVERALL_TIMEOUT_SECS", config.OverallTimeoutSecs)
	config.AchievementsTimeoutSecs = getEnvInt("ACHIEVEMENTS_TIMEOUT_SECS", config.AchievementsTimeoutSecs)

	config.MaxRetries = getEnvInt("MAX_RETRIES", config.MaxRetries)
	config.BaseBackoffMs = getEnvInt("BASE_BACKOFF_MS", config.BaseBackoffMs)
	config.MaxBackoffMs = getEnvInt("MAX_BACKOFF_MS", config.MaxBackoffMs)

	config.RateLimit = getEnvInt("RATE_LIMIT_PER_MIN", config.RateLimit)
	config.BurstLimit = getEnvInt("BURST_LIMIT", config.BurstLimit)

	// Apply validation and fix invalid values
	if config.CBMaxFails <= 0 {
		config.CBMaxFails = 5
	}
	if config.CBResetTimeoutSecs <= 0 {
		config.CBResetTimeoutSecs = 60
	}
	if config.APITimeoutSecs <= 0 {
		config.APITimeoutSecs = 10
	}
	if config.AchievementsTimeoutSecs <= 0 {
		config.AchievementsTimeoutSecs = 5
	}
	if config.MaxRetries < 0 {
		config.MaxRetries = 3
	}
	if config.BaseBackoffMs <= 0 {
		config.BaseBackoffMs = 250
	}
	if config.MaxBackoffMs <= config.BaseBackoffMs {
		config.MaxBackoffMs = config.BaseBackoffMs * 10
	}
	if config.RateLimit <= 0 {
		config.RateLimit = 100
	}

	// Compute derived fields
	config.APITimeout = time.Duration(config.APITimeoutSecs) * time.Second
	config.OverallTimeout = time.Duration(config.OverallTimeoutSecs) * time.Second
	config.AchievementsTimeout = time.Duration(config.AchievementsTimeoutSecs) * time.Second
	config.CBResetTimeout = time.Duration(config.CBResetTimeoutSecs) * time.Second
	config.BaseBackoff = time.Duration(config.BaseBackoffMs) * time.Millisecond
	config.MaxBackoff = time.Duration(config.MaxBackoffMs) * time.Millisecond

	// Log configuration for debugging
	log.Info("API configuration loaded",
		"cb_max_fails", config.CBMaxFails,
		"cb_reset_timeout", config.CBResetTimeout,
		"api_timeout", config.APITimeout,
		"achievements_timeout", config.AchievementsTimeout,
		"max_retries", config.MaxRetries,
		"base_backoff", config.BaseBackoff,
		"rate_limit", config.RateLimit,
		"source", "environment_with_defaults")

	return config
}

// getEnvInt safely parses an integer from environment variable with fallback
func getEnvInt(envKey string, fallback int) int {
	if value := os.Getenv(envKey); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			log.Debug("Configuration loaded from environment",
				"env_key", envKey,
				"value", parsed)
			return parsed
		}
		log.Warn("Invalid integer in environment variable, using fallback",
			"env_key", envKey,
			"value", value,
			"fallback", fallback)
	}
	return fallback
}

// Validate performs basic validation on configuration values
func (c *APIConfig) Validate() error {
	if c.CBMaxFails <= 0 {
		return fmt.Errorf("CB_MAX_FAILS must be positive, got %d", c.CBMaxFails)
	}
	if c.APITimeoutSecs <= 0 {
		return fmt.Errorf("API_TIMEOUT_SECS must be positive, got %d", c.APITimeoutSecs)
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("MAX_RETRIES must be non-negative, got %d", c.MaxRetries)
	}
	if c.BaseBackoffMs <= 0 {
		return fmt.Errorf("BASE_BACKOFF_MS must be positive, got %d", c.BaseBackoffMs)
	}
	if c.MaxBackoffMs < c.BaseBackoffMs {
		return fmt.Errorf("MAX_BACKOFF_MS (%d) must be >= BASE_BACKOFF_MS (%d)", c.MaxBackoffMs, c.BaseBackoffMs)
	}
	return nil
}
