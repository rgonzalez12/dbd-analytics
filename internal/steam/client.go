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
		client:      &http.Client{},
		retryConfig: DefaultRetryConfig(),
	}
}

func (c *Client) GetPlayerSummary(steamIDOrVanity string) (*SteamPlayer, *APIError) {
	start := time.Now()
	if c.apiKey == "" {
		return nil, NewValidationError("STEAM_API_KEY environment variable not set")
	}

	log.Debug("Starting player summary request", 
		"steam_id_or_vanity", steamIDOrVanity)

	steamID64, err := c.resolveSteamID(steamIDOrVanity)
	if err != nil {
		// Wrap the error with context
		wrappedErr := &APIError{
			Type:       err.Type,
			Message:    fmt.Sprintf("GetPlayerSummary failed during Steam ID resolution: %s", err.Message),
			StatusCode: err.StatusCode,
			Retryable:  err.Retryable,
		}
		log.Error("Steam ID resolution failed", 
			"steam_id_or_vanity", steamIDOrVanity,
			"error", err.Message,
			"duration", time.Since(start))
		return nil, wrappedErr
	}

	endpoint := fmt.Sprintf("%s/ISteamUser/GetPlayerSummaries/v0002/", BaseURL)
	params := url.Values{}
	params.Set("key", c.apiKey)
	params.Set("steamids", steamID64)

	var resp playerSummaryResponse
	
	// Retry logic for Steam API calls with structured logging
	retryErr := withRetryAndLogging(c.retryConfig, func() (*APIError, bool) {
		if err := c.makeRequest(endpoint, params, &resp); err != nil {
			// Wrap API request errors with context
			wrappedErr := &APIError{
				Type:       err.Type,
				Message:    fmt.Sprintf("GetPlayerSummary API request failed: %s", err.Message),
				StatusCode: err.StatusCode,
				Retryable:  err.Retryable,
			}
			return wrappedErr, false // Don't stop retrying unless it's not retryable
		}
		return nil, false
	}, "GetPlayerSummary")
	
	if retryErr != nil {
		return nil, retryErr
	}

	if len(resp.Response.Players) == 0 {
		notFoundErr := NewNotFoundError("Player")
		notFoundErr.Message = fmt.Sprintf("GetPlayerSummary: player not found for Steam ID %s", steamID64)
		log.Warn("Player not found in Steam API", 
			"steam_id", steamID64,
			"duration", time.Since(start))
		return nil, notFoundErr
	}

	log.Info("Successfully retrieved player summary", 
		"steam_id", steamID64,
		"persona_name", resp.Response.Players[0].PersonaName,
		"duration", time.Since(start))
	return &resp.Response.Players[0], nil
}

func (c *Client) GetPlayerStats(steamIDOrVanity string) (*SteamPlayerstats, *APIError) {
	if c.apiKey == "" {
		return nil, NewValidationError("STEAM_API_KEY environment variable not set")
	}

	log.Debug("Starting player stats request", 
		"steam_id_or_vanity", steamIDOrVanity)

	steamID64, err := c.resolveSteamID(steamIDOrVanity)
	if err != nil {
		// Wrap the error with context
		wrappedErr := &APIError{
			Type:       err.Type,
			Message:    fmt.Sprintf("GetPlayerStats failed during Steam ID resolution: %s", err.Message),
			StatusCode: err.StatusCode,
			Retryable:  err.Retryable,
		}
		return nil, wrappedErr
	}

	endpoint := fmt.Sprintf("%s/ISteamUserStats/GetUserStatsForGame/v2/", BaseURL)
	params := url.Values{}
	params.Set("appid", DBDAppID)
	params.Set("key", c.apiKey)
	params.Set("steamid", steamID64)

	var resp playerStatsResponse
	
	// Retry logic for Steam API calls with structured logging
	retryErr := withRetryAndLogging(c.retryConfig, func() (*APIError, bool) {
		if err := c.makeRequest(endpoint, params, &resp); err != nil {
			// Wrap API request errors with context
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

	log.Info("Successfully retrieved player stats", 
		"steam_id", steamID64,
		"stats_count", len(resp.Playerstats.Stats))
	return &resp.Playerstats, nil
}

func (c *Client) resolveSteamID(steamIDOrVanity string) (string, *APIError) {
	if len(steamIDOrVanity) == 17 && isNumeric(steamIDOrVanity) {
		return steamIDOrVanity, nil
	}

	log.Debug("Resolving vanity URL to Steam ID", 
		"vanity_url", steamIDOrVanity)

	endpoint := fmt.Sprintf("%s/ISteamUser/ResolveVanityURL/v0001/", BaseURL)
	params := url.Values{}
	params.Set("key", c.apiKey)
	params.Set("vanityurl", steamIDOrVanity)

	var resp VanityURLResponse
	
	// Retry logic for vanity URL with structured logging
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

func (c *Client) makeRequest(endpoint string, params url.Values, result interface{}) *APIError {
	apiURL := endpoint + "?" + params.Encode()
	start := time.Now()

	// Log outgoing Steam API request
	log.Info("steam_api_request_start",
		"endpoint", endpoint,
		"method", "GET",
		"url", apiURL)

	resp, err := c.client.Get(apiURL)
	requestDuration := time.Since(start)
	
	if err != nil {
		log.Error("steam_api_request_failed",
			"error", err.Error(),
			"endpoint", endpoint,
			"duration", requestDuration,
			"duration_ms", fmt.Sprintf("%.2f", requestDuration.Seconds()*1000),
			"error_type", "network_error")
		return NewNetworkError(err)
	}
	defer resp.Body.Close()

	// Log response details
	log.Info("steam_api_request_completed",
		"endpoint", endpoint,
		"status_code", resp.StatusCode,
		"duration", requestDuration,
		"duration_ms", fmt.Sprintf("%.2f", requestDuration.Seconds()*1000),
		"content_length", resp.Header.Get("Content-Length"))

	// Handle rate limiting with Retry-After header parsing
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := c.parseRetryAfterHeader(resp.Header.Get("Retry-After"))
		log.Warn("steam_api_rate_limited",
			"status_code", resp.StatusCode,
			"endpoint", endpoint,
			"duration", requestDuration,
			"retry_after", retryAfter)
		return NewRateLimitErrorWithRetryAfter(retryAfter)
	}

	// Handle other HTTP errors using specific retryable status codes
	if resp.StatusCode != http.StatusOK {
		log.Error("steam_api_http_error",
			"status_code", resp.StatusCode,
			"endpoint", endpoint,
			"duration", requestDuration,
			"error_type", "http_error")
		return NewAPIError(resp.StatusCode, fmt.Sprintf("HTTP %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("steam_api_response_read_failed",
			"error", err.Error(),
			"endpoint", endpoint,
			"duration", requestDuration)
		return NewInternalError(fmt.Errorf("failed to read response body: %w", err))
	}

	if err := json.Unmarshal(body, result); err != nil {
		log.Error("steam_api_json_parse_failed",
			"error", err.Error(),
			"endpoint", endpoint,
			"duration", requestDuration,
			"response_size", len(body),
			"body_preview", string(body)[:min(len(body), 200)])
		return NewInternalError(fmt.Errorf("failed to parse JSON response: %w", err))
	}

	log.Info("steam_api_request_success",
		"endpoint", endpoint,
		"status_code", resp.StatusCode,
		"duration", requestDuration,
		"duration_ms", fmt.Sprintf("%.2f", requestDuration.Seconds()*1000),
		"response_size", len(body))
	return nil
}

// parseRetryAfterHeader parses the Retry-After header value
// Returns retry time in seconds, defaulting to 60 if parsing fails
func (c *Client) parseRetryAfterHeader(retryAfterValue string) int {
	if retryAfterValue == "" {
		return 60 // Default to 60 seconds if no header present
	}
	
	// Try to parse as seconds (integer)
	if seconds, err := strconv.Atoi(retryAfterValue); err == nil && seconds > 0 {
		// Cap at reasonable maximum (5 minutes)
		if seconds > 300 {
			return 300
		}
		return seconds
	}
	
	// If parsing fails, default to 60 seconds
	log.Debug("Failed to parse Retry-After header, using default",
		"retry_after_value", retryAfterValue,
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
