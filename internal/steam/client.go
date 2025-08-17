package steam

import (
	"context"
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

func achievementTimeout() time.Duration {
	if timeoutStr := os.Getenv("ACHIEVEMENTS_TIMEOUT_SECS"); timeoutStr != "" {
		if timeoutSecs, err := strconv.Atoi(timeoutStr); err == nil && timeoutSecs > 0 {
			return time.Duration(timeoutSecs) * time.Second
		}
	}
	return 5 * time.Second // Default fallback
}

func logSteamError(level string, msg string, playerID string, err error, fields ...interface{}) {
	logger := log.SteamAPIContext(playerID, "steam_api")
	allFields := append([]interface{}{"error", err.Error()}, fields...)

	switch level {
	case "ERROR":
		logger.Error(msg, allFields...)
	case "WARN":
		logger.Warn(msg, allFields...)
	case "DEBUG":
		logger.Debug(msg, allFields...)
	default:
		logger.Info(msg, allFields...)
	}
}

func logSteamInfo(msg string, playerID string, additionalFields ...interface{}) {
	logger := log.SteamAPIContext(playerID, "steam_api")
	logger.Info(msg, additionalFields...)
}

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
	apiKey := os.Getenv("STEAM_API_KEY")
	log.Info("Creating Steam client", "api_key_exists", apiKey != "", "api_key_length", len(apiKey))

	return &Client{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: achievementTimeout(),
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

	retryErr := withRetryAndLogging(c.retryConfig, func() (*APIError, bool) {
		if err := c.makeRequest(endpoint, params, &resp); err != nil {
			wrappedErr := &APIError{
				Type:       err.Type,
				Message:    fmt.Sprintf("GetPlayerSummary API request failed: %s", err.Message),
				StatusCode: err.StatusCode,
				Retryable:  err.Retryable,
			}
			return wrappedErr, false
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

// GetUserStatsForGame gets user stats for a specific game - alias for GetPlayerStats with context
func (c *Client) GetUserStatsForGame(ctx context.Context, steamID string, appID int) (*SteamPlayerstats, *APIError) {
	// Use existing GetPlayerStats method (it already uses DBDAppID)
	return c.GetPlayerStats(steamID)
}

// GetUserStatsForGameCached retrieves user stats with caching support
func (c *Client) GetUserStatsForGameCached(ctx context.Context, steamID string, appID int, cacheManager interface{}) (*SteamPlayerstats, *APIError) {
	// Use cache if available
	if cacheManager != nil {
		cache, ok := cacheManager.(interface {
			Get(key string) (interface{}, bool)
			Set(key string, value interface{}, ttl time.Duration) error
		})
		if ok {
			cacheKey := fmt.Sprintf("user_stats_%s_%d", steamID, appID)

			if cached, found := cache.Get(cacheKey); found {
				if stats, ok := cached.(*SteamPlayerstats); ok {
					log.Debug("Using cached user stats", "steam_id", steamID, "app_id", appID,
						"cache_key", cacheKey, "stats_count", len(stats.Stats))
					return stats, nil
				} else {
					log.Warn("Invalid cached user stats type",
						"cache_key", cacheKey, "expected", "*SteamPlayerstats", "actual", fmt.Sprintf("%T", cached))
				}
			}

			// Cache miss - fetch from API
			stats, err := c.GetUserStatsForGame(ctx, steamID, appID)
			if err != nil {
				return nil, err
			}

			// Cache the result
			if cacheErr := cache.Set(cacheKey, stats, 2*time.Minute); cacheErr != nil {
				log.Warn("Failed to cache user stats", "cache_key", cacheKey, "error", cacheErr)
			} else {
				log.Debug("User stats cached successfully", "cache_key", cacheKey, "stats_count", len(stats.Stats))
			}

			return stats, nil
		}
	}

	// No cache available - direct API call
	return c.GetUserStatsForGame(ctx, steamID, appID)
}

func (c *Client) GetPlayerAchievements(steamID string, appID int) (*PlayerAchievements, *APIError) {
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
	params.Set("appid", strconv.Itoa(appID))
	params.Set("l", "english")

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
			lastErr = NewNetworkError(fmt.Errorf("error making GET request to %s: %w", apiURL, err))
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
			lastErr = NewAPIError(resp.StatusCode, fmt.Sprintf("HTTP %d from %s", resp.StatusCode, apiURL))
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
			lastErr = NewInternalError(fmt.Errorf("failed to read response body from %s: %w", apiURL, err))
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
			lastErr = NewInternalError(fmt.Errorf("failed to parse JSON response from %s: %w", apiURL, err))
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

// GetSchemaForGame retrieves the game schema including achievements and stats
func (c *Client) GetSchemaForGame(appID string) (*SchemaGame, *APIError) {
	log.Info("GetSchemaForGame called", "app_id", appID, "api_key_exists", c.apiKey != "", "api_key_length", len(c.apiKey))

	if c.apiKey == "" {
		log.Error("STEAM_API_KEY is empty in GetSchemaForGame")
		return nil, NewValidationError("STEAM_API_KEY environment variable not set")
	}

	url := fmt.Sprintf("%s/ISteamUserStats/GetSchemaForGame/v2/?key=%s&appid=%s&l=en",
		BaseURL, c.apiKey, appID)

	log.Info("Making schema request", "url", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, NewNetworkError(err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		log.Error("Network error in schema request", "error", err)
		return nil, NewNetworkError(err)
	}
	defer resp.Body.Close()

	log.Info("Schema request completed", "status_code", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		log.Error("Non-200 response from schema request", "status_code", resp.StatusCode, "url", url)
		return nil, NewAPIError(resp.StatusCode,
			fmt.Sprintf("HTTP %d from %s", resp.StatusCode, url))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("Error reading schema response body", "error", err)
		return nil, NewInternalError(err)
	}

	log.Info("Schema response read", "body_length", len(body))

	var response schemaForGameResponse
	if err := json.Unmarshal(body, &response); err != nil {
		bodyPreview := string(body)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		log.Error("Error unmarshaling schema response", "error", err, "body_preview", bodyPreview)
		return nil, NewInternalError(err)
	}

	if response.Game.AvailableGameStats.Achievements == nil {
		log.Error("Schema response has nil achievements")
	} else {
		log.Info("Schema response parsed successfully", "achievement_count", len(response.Game.AvailableGameStats.Achievements))
	}

	return &response.Game, nil
}

// FetchGlobalAchievementPercentages retrieves global achievement percentages for the specified app
func (c *Client) FetchGlobalAchievementPercentages(ctx context.Context) (map[string]float64, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("STEAM_API_KEY environment variable not set")
	}

	url := fmt.Sprintf("%s/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v0002/?gameid=%s",
		BaseURL, DBDAppID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from Steam API", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response globalAchievementPercentagesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	percentages := make(map[string]float64)
	for _, ach := range response.AchievementPercentages.Achievements {
		percentages[ach.Name] = ach.Percent
	}

	return percentages, nil
}

// GetGlobalAchievementPercentagesCached retrieves global achievement percentages with caching
func (c *Client) GetGlobalAchievementPercentagesCached(ctx context.Context, cacheManager interface{}) (map[string]float64, error) {
	// Type assertion to get the cache interface
	cache, ok := cacheManager.(interface {
		Get(string) (interface{}, bool)
		Set(string, interface{}, time.Duration) error
	})
	if !ok || cache == nil {
		// No cache available, fetch directly
		return c.FetchGlobalAchievementPercentages(ctx)
	}

	cacheKey := "global_percentages:dbd"

	// Try to get from cache first
	if cached, found := cache.Get(cacheKey); found {
		if percentages, ok := cached.(map[string]float64); ok {
			log.Debug("Global achievement percentages cache hit", "cache_key", cacheKey)
			return percentages, nil
		} else {
			log.Warn("Invalid global percentages cache entry type, removing",
				"cache_key", cacheKey, "expected", "map[string]float64", "actual", fmt.Sprintf("%T", cached))
			// Continue to fetch fresh data
		}
	}

	// Fetch fresh data
	percentages, err := c.FetchGlobalAchievementPercentages(ctx)
	if err != nil {
		return nil, err
	}

	// Cache the result for 24 hours
	if err := cache.Set(cacheKey, percentages, 24*time.Hour); err != nil {
		log.Error("Failed to cache global achievement percentages", "error", err, "cache_key", cacheKey)
		// Don't fail the request if caching fails
	} else {
		log.Debug("Global achievement percentages cached successfully", "cache_key", cacheKey, "count", len(percentages))
	}

	return percentages, nil
}
