package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

func GetDBDPlayerStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	steamID := vars["steamID"]

	if steamID == "" {
		http.Error(w, "Steam ID or vanity URL required", http.StatusBadRequest)
		return
	}

	// Create Steam API client
	client := steam.NewClient()

	// Fetch player summary for display name
	playerSummary, err := client.GetPlayerSummary(steamID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch player profile: %v", err), http.StatusInternalServerError)
		return
	}

	// Fetch raw Dead by Daylight stats
	rawStats, err := client.GetPlayerStats(steamID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch player stats: %v", err), http.StatusInternalServerError)
		return
	}

	// Map raw stats to organized player statistics
	playerStats := steam.MapSteamStats(rawStats.Stats, playerSummary.SteamID, playerSummary.PersonaName)

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(playerStats); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
