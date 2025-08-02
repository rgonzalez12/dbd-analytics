package steam

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
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

	slog.Debug("Getting player summary", "steam_id_or_vanity", steamIDOrVanity)

	steamID64, err := c.resolveSteamID(steamIDOrVanity)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/ISteamUser/GetPlayerSummaries/v0002/", BaseURL)
	params := url.Values{}
	params.Set("key", c.apiKey)
	params.Set("steamids", steamID64)

	var resp playerSummaryResponse
	
	// Retry logic for Steam API calls with structured logging
	retryErr := withRetryAndLogging(c.retryConfig, func() (*APIError, bool) {
		if err := c.makeRequest(endpoint, params, &resp); err != nil {
			return err, false // Don't stop retrying unless it's not retryable
		}
		return nil, false
	}, "GetPlayerSummary")
	
	if retryErr != nil {
		return nil, retryErr
	}

	if len(resp.Response.Players) == 0 {
		return nil, NewNotFoundError("Player")
	}

	slog.Debug("Successfully retrieved player summary", "steam_id", steamID64)
	return &resp.Response.Players[0], nil
}

func (c *Client) GetPlayerStats(steamIDOrVanity string) (*SteamPlayerstats, *APIError) {
	if c.apiKey == "" {
		return nil, NewValidationError("STEAM_API_KEY environment variable not set")
	}

	slog.Debug("Getting player stats", "steam_id_or_vanity", steamIDOrVanity)

	steamID64, err := c.resolveSteamID(steamIDOrVanity)
	if err != nil {
		return nil, err
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
			return err, false
		}
		return nil, false
	}, "GetPlayerStats")
	
	if retryErr != nil {
		return nil, retryErr
	}

	slog.Debug("Successfully retrieved player stats", "steam_id", steamID64)
	return &resp.Playerstats, nil
}

func (c *Client) resolveSteamID(steamIDOrVanity string) (string, *APIError) {
	if len(steamIDOrVanity) == 17 && isNumeric(steamIDOrVanity) {
		return steamIDOrVanity, nil
	}

	slog.Debug("Resolving vanity URL", "vanity_url", steamIDOrVanity)

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

	slog.Debug("Successfully resolved vanity URL", "vanity_url", steamIDOrVanity, "steam_id", resp.Response.SteamID)
	return resp.Response.SteamID, nil
}

func (c *Client) makeRequest(endpoint string, params url.Values, result interface{}) *APIError {
	apiURL := endpoint + "?" + params.Encode()

	slog.Debug("Making Steam API request", "url", apiURL)

	resp, err := c.client.Get(apiURL)
	if err != nil {
		slog.Error("Network error during Steam API request", "error", err, "url", apiURL)
		return NewNetworkError(err)
	}
	defer resp.Body.Close()

	// Handle rate limiting with structured error
	if resp.StatusCode == http.StatusTooManyRequests {
		slog.Warn("Steam API rate limit hit", "status_code", resp.StatusCode)
		return NewRateLimitError()
	}

	// Handle other HTTP errors using specific retryable status codes
	if resp.StatusCode != http.StatusOK {
		slog.Error("Steam API HTTP error", "status_code", resp.StatusCode, "url", apiURL)
		return NewAPIError(resp.StatusCode, fmt.Sprintf("HTTP %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Failed to read response body", "error", err)
		return NewInternalError(fmt.Errorf("failed to read response body: %w", err))
	}

	if err := json.Unmarshal(body, result); err != nil {
		slog.Error("Failed to parse JSON response", "error", err, "body", string(body))
		return NewInternalError(fmt.Errorf("failed to parse JSON response: %w", err))
	}

	slog.Debug("Steam API request successful", "status_code", resp.StatusCode)
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
