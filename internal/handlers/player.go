package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/mux"
	"github.com/rgonzalez12/dbd-analytics/internal/models"
)

type SteamPlayerResponse struct {
	Response SteamResponse `json:"response"`
}

type SteamResponse struct {
	Players []SteamPlayer `json:"players"`
}

type SteamPlayer struct {
	SteamID     string `json:"steamid"`
	PersonaName string `json:"personaname"`
	Avatar      string `json:"avatar"`
	AvatarFull  string `json:"avatarfull"`
}

func GetPlayerStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	steamID := vars["steamID"]

	if steamID == "" {
		http.Error(w, "Steam ID or vanity URL required", http.StatusBadRequest)
		return
	}

	apiKey := os.Getenv("STEAM_API_KEY")
	if apiKey == "" {
		http.Error(w, "Steam API key not configured", http.StatusInternalServerError)
		return
	}

	steamPlayer, err := fetchSteamPlayer(steamID, apiKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch player data: %v", err), http.StatusInternalServerError)
		return
	}

	if steamPlayer == nil {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}

	player := models.PlayerStats{
		SteamID:              steamPlayer.SteamID,
		DisplayName:          steamPlayer.PersonaName,
		KillerPips:           123,
		SurvivorPips:         87,
		KilledCampers:        250,
		SacrificedCampers:    320,
		GeneratorPct:         73.6,
		HealPct:              51.2,
		Escapes:              64,
		SkillCheckSuccess:    421,
		BloodwebPoints:       312000,
		EscapeThroughHatch:   5,
		UnhookOrHeal:         76,
		CamperFullLoadout:    12,
		KillerPerfectGames:   7,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(player); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func fetchSteamPlayer(steamIDOrVanity, apiKey string) (*SteamPlayer, error) {
	var steamID64 string

	if len(steamIDOrVanity) == 17 && isNumeric(steamIDOrVanity) {
		steamID64 = steamIDOrVanity
	} else {
		resolvedID, err := resolveVanityURL(steamIDOrVanity, apiKey)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve vanity URL: %w", err)
		}
		if resolvedID == "" {
			return nil, fmt.Errorf("vanity URL not found: %s", steamIDOrVanity)
		}
		steamID64 = resolvedID
	}

	return fetchPlayerBySteamID64(steamID64, apiKey)
}

func isNumeric(s string) bool {
	for _, char := range s {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func resolveVanityURL(vanityURL, apiKey string) (string, error) {
	baseURL := "https://api.steampowered.com/ISteamUser/ResolveVanityURL/v0001/"
	params := url.Values{}
	params.Set("key", apiKey)
	params.Set("vanityurl", vanityURL)

	apiURL := baseURL + "?" + params.Encode()

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to make request to Steam API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("steam API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var vanityResp struct {
		Response struct {
			SteamID string `json:"steamid"`
			Success int    `json:"success"`
		} `json:"response"`
	}

	if err := json.Unmarshal(body, &vanityResp); err != nil {
		return "", fmt.Errorf("failed to parse vanity URL response: %w", err)
	}

	if vanityResp.Response.Success != 1 {
		return "", nil
	}

	return vanityResp.Response.SteamID, nil
}

func fetchPlayerBySteamID64(steamID64, apiKey string) (*SteamPlayer, error) {
	baseURL := "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/"
	params := url.Values{}
	params.Set("key", apiKey)
	params.Set("steamids", steamID64)

	apiURL := baseURL + "?" + params.Encode()

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to Steam API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var steamResp SteamPlayerResponse
	if err := json.Unmarshal(body, &steamResp); err != nil {
		return nil, fmt.Errorf("failed to parse Steam API response: %w", err)
	}

	if len(steamResp.Response.Players) == 0 {
		return nil, nil
	}

	return &steamResp.Response.Players[0], nil
}
