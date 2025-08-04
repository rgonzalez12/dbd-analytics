package steam

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

const (
	BaseURL  = "https://api.steampowered.com"
	DBDAppID = "381210"
)

// getAchievementsTimeout returns the configurable achievements fetch timeout
func getAchievementsTimeout() time.Duration {
	if timeoutStr := os.Getenv("ACHIEVEMENTS_TIMEOUT_SECS"); timeoutStr != "" {
		if timeoutSecs, err := strconv.Atoi(timeoutStr); err == nil && timeoutSecs > 0 {
			return time.Duration(timeoutSecs) * time.Second
		}
	}
	return 5 * time.Second // Default fallback
}

// logSteamError provides consistent Steam API error logging with player context
func logSteamError(level string, msg string, playerID string, err error, additionalFields ...interface{}) {
	logger := log.SteamAPIContext(playerID, "steam_api")
	
	fields := []interface{}{
		"error", err.Error(),
	}
	fields = append(fields, additionalFields...)
	
	switch level {
	case "ERROR":
		logger.Error(msg, fields...)
	case "WARN":
		logger.Warn(msg, fields...)
	case "DEBUG":
		logger.Debug(msg, fields...)
	default:
		logger.Info(msg, fields...)
	}
}

// logSteamInfo provides consistent Steam API info logging with player context
func logSteamInfo(msg string, playerID string, additionalFields ...interface{}) {
	logger := log.SteamAPIContext(playerID, "steam_api")
	logger.Info(msg, additionalFields...)
}

// logSteamPerformance provides performance-focused logging with metrics
func logSteamPerformance(operation, playerID, endpoint string, durationMs float64, additionalFields ...interface{}) {
	logger := log.PerformanceContext(operation, playerID, durationMs).With(
		"endpoint", endpoint,
		"api_provider", "steam",
	)
	
	fields := []interface{}{
		"operation_success", true,
	}
	fields = append(fields, additionalFields...)
	logger.Info("Steam API operation completed", fields...)
}

type Client struct {
	apiKey      string
	client      *http.Client
	retryConfig RetryConfig
}

type playerSummaryResponse struct {
	Response struct {
		Players []SteamPlayer `json:"players"`
	} `json:"response"`
}

type playerStatsResponse struct {
	Playerstats SteamPlayerstats `json:"playerstats"`
}

func NewClient() *Client {
	return &Client{
		apiKey:      os.Getenv("STEAM_API_KEY"),
		client:      &http.Client{
			Timeout: getAchievementsTimeout(),
		},
		retryConfig: DefaultRetryConfig(),
	}
}

func (c *Client) GetPlayerSummary(steamIDOrVanity string) (*SteamPlayer, *APIError) {
	start := time.Now()
	if c.apiKey == "" {
		return nil, NewValidationError("STEAM_API_KEY environment variable not set")
	}

	log.PlayerContext(steamIDOrVanity).Info("Starting player summary request", "steam_id_or_vanity", steamIDOrVanity)

	steamID64, err := c.resolveSteamID(steamIDOrVanity)
	if err != nil {
		// Wrap Steam ID resolution errors with additional context for debugging
		wrappedErr := &APIError{
			Type:       err.Type,
			Message:    fmt.Sprintf("GetPlayerSummary failed during Steam ID resolution: %s", err.Message),
			StatusCode: err.StatusCode,
			Retryable:  err.Retryable,
		}
		logSteamError("ERROR", "Steam ID resolution failed", steamIDOrVanity, 
			fmt.Errorf(err.Message), "duration", time.Since(start))
		return nil, wrappedErr
	}

	endpoint := fmt.Sprintf("%s/ISteamUser/GetPlayerSummaries/v0002/", BaseURL)
	logger := log.SteamAPIContext(steamIDOrVanity, endpoint)
	
	logger.Info("Executing player summary request", "resolved_steam_id", steamID64)
	
	params := url.Values{}
	params.Set("key", c.apiKey)
	params.Set("steamids", steamID64)

	var resp playerSummaryResponse
	
	// Execute API request with enhanced retry logic and structured logging
	retryErr := withRetryAndLogging(c.retryConfig, func() (*APIError, bool) {
		if err := c.makeRequest(endpoint, params, &resp); err != nil {
			// Wrap API request errors with additional context for troubleshooting
			wrappedErr := &APIError{
				Type:       err.Type,
				Message:    fmt.Sprintf("GetPlayerSummary API request failed: %s", err.Message),
				StatusCode: err.StatusCode,
				Retryable:  err.Retryable,
			}
			return wrappedErr, false // Allow retry logic to determine if request should be retried
		}
		return nil, false
	}, "GetPlayerSummary")
	
	if retryErr != nil {
		return nil, retryErr
	}

	if len(resp.Response.Players) == 0 {
		notFoundErr := NewNotFoundError("Player")
		notFoundErr.Message = fmt.Sprintf("GetPlayerSummary: player not found for Steam ID %s", steamID64)
		logSteamError("WARN", "Player not found in Steam API", steamID64,
			fmt.Errorf("player not found"), "duration", time.Since(start))
		return nil, notFoundErr
	}

	// Log successful operation with performance metrics
	durationMs := float64(time.Since(start).Nanoseconds()) / 1e6
	logSteamPerformance("GetPlayerSummary", steamID64, endpoint, durationMs,
		"persona_name", resp.Response.Players[0].PersonaName,
		"status_code", 200)
	
	return &resp.Response.Players[0], nil
}

func (c *Client) GetPlayerStats(steamIDOrVanity string) (*SteamPlayerstats, *APIError) {
	if c.apiKey == "" {
		return nil, NewValidationError("STEAM_API_KEY environment variable not set")
	}

	logSteamInfo("Starting player stats request", steamIDOrVanity, "steam_id_or_vanity", steamIDOrVanity)

	steamID64, err := c.resolveSteamID(steamIDOrVanity)
	if err != nil {
		// Wrap Steam ID resolution errors with additional context for debugging
		wrappedErr := &APIError{
			Type:       err.Type,
			Message:    fmt.Sprintf("GetPlayerStats failed during Steam ID resolution: %s", err.Message),
			StatusCode: err.StatusCode,
			Retryable:  err.Retryable,
		}
		logSteamError("ERROR", "Steam ID resolution failed for stats", steamIDOrVanity, fmt.Errorf(err.Message))
		return nil, wrappedErr
	}

	endpoint := fmt.Sprintf("%s/ISteamUserStats/GetUserStatsForGame/v2/", BaseURL)
	params := url.Values{}
	params.Set("appid", DBDAppID)
	params.Set("key", c.apiKey)
	params.Set("steamid", steamID64)

	var resp playerStatsResponse
	
	// Execute API request with enhanced retry logic and structured logging
	retryErr := withRetryAndLogging(c.retryConfig, func() (*APIError, bool) {
		if err := c.makeRequest(endpoint, params, &resp); err != nil {
			// Wrap API request errors with additional context for troubleshooting
			wrappedErr := &APIError{
				Type:       err.Type,
				Message:    fmt.Sprintf("GetPlayerStats API request failed: %s", err.Message),
				StatusCode: err.StatusCode,
				Retryable:  err.Retryable,
			}
			return wrappedErr, false
		}
		return nil, false
	}, "GetPlayerStats")
	
	if retryErr != nil {
		return nil, retryErr
	}

	logSteamInfo("Successfully retrieved player stats", steamID64,
		"stats_count", len(resp.Playerstats.Stats))
	return &resp.Playerstats, nil
}

// GetPlayerAchievements retrieves player achievements for a specific game
func (c *Client) GetPlayerAchievements(steamID, appID string) (*PlayerAchievements, *APIError) {
	start := time.Now()
	if c.apiKey == "" {
		return nil, NewValidationError("STEAM_API_KEY environment variable not set")
	}

	logSteamInfo("Starting player achievements request", steamID,
		"steam_id", steamID, "app_id", appID)

	steamID64, err := c.resolveSteamID(steamID)
	if err != nil {
		wrappedErr := &APIError{
			Type:       err.Type,
			Message:    fmt.Sprintf("GetPlayerAchievements failed during Steam ID resolution: %s", err.Message),
			StatusCode: err.StatusCode,
			Retryable:  err.Retryable,
		}
		logSteamError("ERROR", "Steam ID resolution failed for achievements", steamID,
			fmt.Errorf(err.Message), "duration", time.Since(start))
		return nil, wrappedErr
	}

	endpoint := fmt.Sprintf("%s/ISteamUserStats/GetPlayerAchievements/v0001/", BaseURL)
	params := url.Values{}
	params.Set("key", c.apiKey)
	params.Set("steamid", steamID64)
	params.Set("appid", appID)

	var resp playerAchievementsResponse
	
	retryErr := withRetryAndLogging(c.retryConfig, func() (*APIError, bool) {
		if err := c.makeRequest(endpoint, params, &resp); err != nil {
			wrappedErr := &APIError{
				Type:       err.Type,
				Message:    fmt.Sprintf("GetPlayerAchievements API request failed: %s", err.Message),
				StatusCode: err.StatusCode,
				Retryable:  err.Retryable,
			}
			return wrappedErr, false
		}
		return nil, false
	}, "GetPlayerAchievements")
	
	if retryErr != nil {
		return nil, retryErr
	}

	if !resp.Playerstats.Success {
		notFoundErr := NewNotFoundError("Player Achievements")
		notFoundErr.Message = fmt.Sprintf("GetPlayerAchievements: achievements not found for Steam ID %s", steamID64)
		logSteamError("WARN", "Player achievements not found or private", steamID64,
			fmt.Errorf("achievements not found or private"), "app_id", appID, "duration", time.Since(start))
		return nil, notFoundErr
	}

	logSteamInfo("Successfully retrieved player achievements", steamID64,
		"app_id", appID,
		"achievements_count", len(resp.Playerstats.Achievements),
		"duration", time.Since(start))
	
	return &resp.Playerstats, nil
}

func (c *Client) resolveSteamID(steamIDOrVanity string) (string, *APIError) {
	if len(steamIDOrVanity) == 17 && isNumeric(steamIDOrVanity) {
		return steamIDOrVanity, nil
	}

	logSteamInfo("Resolving vanity URL to Steam ID", steamIDOrVanity, "vanity_url", steamIDOrVanity)

	endpoint := fmt.Sprintf("%s/ISteamUser/ResolveVanityURL/v0001/", BaseURL)
	params := url.Values{}
	params.Set("key", c.apiKey)
	params.Set("vanityurl", steamIDOrVanity)

	var resp VanityURLResponse
	
	// Execute vanity URL resolution with enhanced retry logic and structured logging
	retryErr := withRetryAndLogging(c.retryConfig, func() (*APIError, bool) {
		if err := c.makeRequest(endpoint, params, &resp); err != nil {
			return err, false
		}
		return nil, false
	}, "ResolveVanityURL")
	
	if retryErr != nil {
		return "", retryErr
	}

	if resp.Response.Success != 1 {
		return "", NewNotFoundError("Vanity URL")
	}

	log.Info("Successfully resolved vanity URL", 
		"vanity_url", steamIDOrVanity, 
		"steam_id", resp.Response.SteamID)
	return resp.Response.SteamID, nil
}

// ResolveSteamID resolves a vanity URL to a Steam ID, or returns the input if it's already a Steam ID
// This is a public wrapper for the private resolveSteamID function to enable safe concurrent resolution
func (c *Client) ResolveSteamID(steamIDOrVanity string) (string, *APIError) {
	return c.resolveSteamID(steamIDOrVanity)
}

func (c *Client) makeRequest(endpoint string, params url.Values, result interface{}) *APIError {
	var lastErr *APIError
	
	for attempt := 0; attempt <= c.retryConfig.MaxAttempts; attempt++ {
		// If this is a retry attempt, wait before trying again
		if attempt > 0 {
			delay := c.calculateRetryDelay(lastErr, attempt-1)
			
			log.Info("steam_api_retry_attempt",
				"attempt", attempt,
				"max_attempts", c.retryConfig.MaxAttempts,
				"delay_seconds", delay.Seconds(),
				"endpoint", endpoint)
			
			time.Sleep(delay)
		}
		
		apiURL := endpoint + "?" + params.Encode()
		start := time.Now()

		// Log outgoing Steam API request
		log.Info("steam_api_request_start",
			"endpoint", endpoint,
			"method", "GET",
			"url", apiURL,
			"attempt", attempt+1)

		resp, err := c.client.Get(apiURL)
		requestDuration := time.Since(start)
		
		if err != nil {
			log.Error("steam_api_request_failed",
				"error", err.Error(),
				"endpoint", endpoint,
				"duration", requestDuration,
				"duration_ms", fmt.Sprintf("%.2f", requestDuration.Seconds()*1000),
				"error_type", "network_error",
				"attempt", attempt+1)
			lastErr = NewNetworkError(err)
			if !shouldRetryError(lastErr) || attempt >= c.retryConfig.MaxAttempts {
				return lastErr
			}
			continue
		}
		defer resp.Body.Close()

		// Log response details
		log.Info("steam_api_request_completed",
			"endpoint", endpoint,
			"status_code", resp.StatusCode,
			"duration", requestDuration,
			"duration_ms", fmt.Sprintf("%.2f", requestDuration.Seconds()*1000),
			"content_length", resp.Header.Get("Content-Length"),
			"attempt", attempt+1)

		// Handle rate limiting with enhanced header parsing
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := c.parseRateLimitHeaders(resp.Header)
			log.Warn("steam_api_rate_limited",
				"status_code", resp.StatusCode,
				"endpoint", endpoint,
				"duration", requestDuration,
				"retry_after_seconds", retryAfter,
				"retry_after_header", resp.Header.Get("Retry-After"),
				"rate_limit_reset_header", resp.Header.Get("X-RateLimit-Reset"),
				"attempt", attempt+1)
			lastErr = NewRateLimitErrorWithRetryAfter(retryAfter)
			if !shouldRetryError(lastErr) || attempt >= c.retryConfig.MaxAttempts {
				return lastErr
			}
			continue
		}

		// Handle other HTTP errors using specific retryable status codes
		if resp.StatusCode != http.StatusOK {
			log.Error("steam_api_http_error",
				"status_code", resp.StatusCode,
				"endpoint", endpoint,
				"duration", requestDuration,
				"error_type", "http_error",
				"attempt", attempt+1)
			lastErr = NewAPIError(resp.StatusCode, fmt.Sprintf("HTTP %d", resp.StatusCode))
			if !shouldRetryError(lastErr) || attempt >= c.retryConfig.MaxAttempts {
				return lastErr
			}
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error("steam_api_response_read_failed",
				"error", err.Error(),
				"endpoint", endpoint,
				"duration", requestDuration,
				"attempt", attempt+1)
			lastErr = NewInternalError(fmt.Errorf("failed to read response body: %w", err))
			if !shouldRetryError(lastErr) || attempt >= c.retryConfig.MaxAttempts {
				return lastErr
			}
			continue
		}

		if err := json.Unmarshal(body, result); err != nil {
			log.Error("steam_api_json_parse_failed",
				"error", err.Error(),
				"endpoint", endpoint,
				"duration", requestDuration,
				"response_size", len(body),
				"body_preview", string(body)[:min(len(body), 200)],
				"attempt", attempt+1)
			lastErr = NewInternalError(fmt.Errorf("failed to parse JSON response: %w", err))
			if !shouldRetryError(lastErr) || attempt >= c.retryConfig.MaxAttempts {
				return lastErr
			}
			continue
		}

		log.Info("steam_api_request_success",
			"endpoint", endpoint,
			"status_code", resp.StatusCode,
			"duration", requestDuration,
			"duration_ms", fmt.Sprintf("%.2f", requestDuration.Seconds()*1000),
			"attempt", attempt+1)

		return nil // Success!
	}
	
	return lastErr
}

// calculateRetryDelay calculates the delay before the next retry attempt
func (c *Client) calculateRetryDelay(lastErr *APIError, attempt int) time.Duration {
	// If we have a rate limit error, check if we have rate limit headers
	if lastErr != nil && lastErr.Type == ErrorTypeRateLimit && lastErr.StatusCode == 429 {
		// If RetryAfter is set to a reasonable value (not the default 60s fallback), use it
		if lastErr.RetryAfter > 0 && lastErr.RetryAfter <= 10 {
			return time.Duration(lastErr.RetryAfter) * time.Second
		}
	}
	
	// Otherwise use exponential backoff (including when rate limit has no useful headers)
	return calculateBackoffDelay(attempt, c.retryConfig)
}

// parseRateLimitHeaders parses rate limit headers and returns retry time in seconds
// Checks Retry-After and X-RateLimit-Reset headers, defaulting to 60 if not found
func (c *Client) parseRateLimitHeaders(headers http.Header) int {
	// First check Retry-After header (preferred)
	if retryAfter := headers.Get("Retry-After"); retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
			// Cap at reasonable maximum (5 minutes)
			if seconds > 300 {
				return 300
			}
			return seconds
		}
	}
	
	// Check X-RateLimit-Reset header (Unix timestamp)
	if resetTime := headers.Get("X-RateLimit-Reset"); resetTime != "" {
		if timestamp, err := strconv.ParseInt(resetTime, 10, 64); err == nil {
			resetAt := time.Unix(timestamp, 0)
			secondsUntilReset := int(time.Until(resetAt).Seconds())
			
			// Only use if it's positive and reasonable (within 5 minutes)
			if secondsUntilReset > 0 && secondsUntilReset <= 300 {
				return secondsUntilReset
			}
		}
	}
	
	// Default to 60 seconds if no valid headers found
	log.Debug("No valid rate limit headers found, using default",
		"retry_after", headers.Get("Retry-After"),
		"rate_limit_reset", headers.Get("X-RateLimit-Reset"),
		"default_seconds", 60)
	return 60
}

func isNumeric(s string) bool {
	for _, char := range s {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
