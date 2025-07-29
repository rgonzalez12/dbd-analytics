package steam

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

const (
	BaseURL = "https://api.steampowered.com"
	DBDAppID = "381210"
)

type Client struct {
	apiKey string
	client *http.Client
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
		apiKey: os.Getenv("STEAM_API_KEY"),
		client: &http.Client{},
	}
}

func (c *Client) GetPlayerSummary(steamIDOrVanity string) (*SteamPlayer, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("STEAM_API_KEY environment variable not set")
	}

	steamID64, err := c.resolveSteamID(steamIDOrVanity)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve Steam ID: %w", err)
	}

	endpoint := fmt.Sprintf("%s/ISteamUser/GetPlayerSummaries/v0002/", BaseURL)
	params := url.Values{}
	params.Set("key", c.apiKey)
	params.Set("steamids", steamID64)

	var resp playerSummaryResponse
	if err := c.makeRequest(endpoint, params, &resp); err != nil {
		return nil, fmt.Errorf("failed to fetch player summary: %w", err)
	}

	if len(resp.Response.Players) == 0 {
		return nil, fmt.Errorf("player not found")
	}

	return &resp.Response.Players[0], nil
}

func (c *Client) GetPlayerStats(steamIDOrVanity string) (*SteamPlayerstats, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("STEAM_API_KEY environment variable not set")
	}

	steamID64, err := c.resolveSteamID(steamIDOrVanity)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve Steam ID: %w", err)
	}

	endpoint := fmt.Sprintf("%s/ISteamUserStats/GetUserStatsForGame/v2/", BaseURL)
	params := url.Values{}
	params.Set("appid", DBDAppID)
	params.Set("key", c.apiKey)
	params.Set("steamid", steamID64)

	var resp playerStatsResponse
	if err := c.makeRequest(endpoint, params, &resp); err != nil {
		return nil, fmt.Errorf("failed to fetch player stats: %w", err)
	}

	return &resp.Playerstats, nil
}

func (c *Client) resolveSteamID(steamIDOrVanity string) (string, error) {
	if len(steamIDOrVanity) == 17 && isNumeric(steamIDOrVanity) {
		return steamIDOrVanity, nil
	}

	endpoint := fmt.Sprintf("%s/ISteamUser/ResolveVanityURL/v0001/", BaseURL)
	params := url.Values{}
	params.Set("key", c.apiKey)
	params.Set("vanityurl", steamIDOrVanity)

	var resp VanityURLResponse
	if err := c.makeRequest(endpoint, params, &resp); err != nil {
		return "", fmt.Errorf("failed to resolve vanity URL: %w", err)
	}

	if resp.Response.Success != 1 {
		return "", fmt.Errorf("vanity URL not found: %s", steamIDOrVanity)
	}

	return resp.Response.SteamID, nil
}

func (c *Client) makeRequest(endpoint string, params url.Values, result interface{}) error {
	apiURL := endpoint + "?" + params.Encode()

	resp, err := c.client.Get(apiURL)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to parse JSON response: %w", err)
	}

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
