package steam

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"
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
	if c.apiKey == "" {
		return nil, NewValidationError("STEAM_API_KEY environment variable not set")
	}

	slog.Debug("Starting player summary request", 
		slog.String("steam_id_or_vanity", steamIDOrVanity))

	steamID64, err := c.resolveSteamID(steamIDOrVanity)
	if err != nil {
		// Wrap the error with context
		wrappedErr := &APIError{
			Type:       err.Type,
			Message:    fmt.Sprintf("GetPlayerSummary failed during Steam ID resolution: %s", err.Message),
			StatusCode: err.StatusCode,
			Retryable:  err.Retryable,
		}
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
		return nil, notFoundErr
	}

	slog.Info("Successfully retrieved player summary", 
		slog.String("steam_id", steamID64),
		slog.String("persona_name", resp.Response.Players[0].PersonaName))
	return &resp.Response.Players[0], nil
}

func (c *Client) GetPlayerStats(steamIDOrVanity string) (*SteamPlayerstats, *APIError) {
	if c.apiKey == "" {
		return nil, NewValidationError("STEAM_API_KEY environment variable not set")
	}

	slog.Debug("Starting player stats request", 
		slog.String("steam_id_or_vanity", steamIDOrVanity))

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

	slog.Info("Successfully retrieved player stats", 
		slog.String("steam_id", steamID64),
		slog.Int("stats_count", len(resp.Playerstats.Stats)))
	return &resp.Playerstats, nil
}

func (c *Client) resolveSteamID(steamIDOrVanity string) (string, *APIError) {
	if len(steamIDOrVanity) == 17 && isNumeric(steamIDOrVanity) {
		return steamIDOrVanity, nil
	}

	slog.Debug("Resolving vanity URL to Steam ID", 
		slog.String("vanity_url", steamIDOrVanity))

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

	slog.Info("Successfully resolved vanity URL", 
		slog.String("vanity_url", steamIDOrVanity), 
		slog.String("steam_id", resp.Response.SteamID))
	return resp.Response.SteamID, nil
}

func (c *Client) makeRequest(endpoint string, params url.Values, result interface{}) *APIError {
	apiURL := endpoint + "?" + params.Encode()
	start := time.Now()

	// Log outgoing Steam API request
	slog.Info("steam_api_request_start",
		slog.String("endpoint", endpoint),
		slog.String("method", "GET"),
		slog.String("url", apiURL))

	resp, err := c.client.Get(apiURL)
	requestDuration := time.Since(start)
	
	if err != nil {
		slog.Error("steam_api_request_failed",
			slog.String("error", err.Error()),
			slog.String("endpoint", endpoint),
			slog.Duration("duration", requestDuration),
			slog.String("duration_ms", fmt.Sprintf("%.2f", requestDuration.Seconds()*1000)),
			slog.String("error_type", "network_error"))
		return NewNetworkError(err)
	}
	defer resp.Body.Close()

	// Log response details
	slog.Info("steam_api_request_completed",
		slog.String("endpoint", endpoint),
		slog.Int("status_code", resp.StatusCode),
		slog.Duration("duration", requestDuration),
		slog.String("duration_ms", fmt.Sprintf("%.2f", requestDuration.Seconds()*1000)),
		slog.String("content_length", resp.Header.Get("Content-Length")))

	// Handle rate limiting with structured error
	if resp.StatusCode == http.StatusTooManyRequests {
		slog.Warn("steam_api_rate_limited",
			slog.Int("status_code", resp.StatusCode),
			slog.String("endpoint", endpoint),
			slog.Duration("duration", requestDuration))
		return NewRateLimitError()
	}

	// Handle other HTTP errors using specific retryable status codes
	if resp.StatusCode != http.StatusOK {
		slog.Error("steam_api_http_error",
			slog.Int("status_code", resp.StatusCode),
			slog.String("endpoint", endpoint),
			slog.Duration("duration", requestDuration),
			slog.String("error_type", "http_error"))
		return NewAPIError(resp.StatusCode, fmt.Sprintf("HTTP %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("steam_api_response_read_failed",
			slog.String("error", err.Error()),
			slog.String("endpoint", endpoint),
			slog.Duration("duration", requestDuration))
		return NewInternalError(fmt.Errorf("failed to read response body: %w", err))
	}

	if err := json.Unmarshal(body, result); err != nil {
		slog.Error("steam_api_json_parse_failed",
			slog.String("error", err.Error()),
			slog.String("endpoint", endpoint),
			slog.Duration("duration", requestDuration),
			slog.Int("response_size", len(body)),
			slog.String("body_preview", string(body)[:min(len(body), 200)]))
		return NewInternalError(fmt.Errorf("failed to parse JSON response: %w", err))
	}

	slog.Info("steam_api_request_success",
		slog.String("endpoint", endpoint),
		slog.Int("status_code", resp.StatusCode),
		slog.Duration("duration", requestDuration),
		slog.String("duration_ms", fmt.Sprintf("%.2f", requestDuration.Seconds()*1000)),
		slog.Int("response_size", len(body)))
	return nil
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
