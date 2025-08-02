package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

type Handler struct {
	steamClient *steam.Client
}

func NewHandler() *Handler {
	return &Handler{
		steamClient: steam.NewClient(),
	}
}

func (h *Handler) GetPlayerSummary(w http.ResponseWriter, r *http.Request) {
	steamID := mux.Vars(r)["steamid"]
	if steamID == "" {
		http.Error(w, "Steam ID required", http.StatusBadRequest)
		return
	}

	summary, err := h.steamClient.GetPlayerSummary(steamID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch player summary: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summary); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetPlayerStats(w http.ResponseWriter, r *http.Request) {
	steamID := mux.Vars(r)["steamid"]
	if steamID == "" {
		http.Error(w, "Steam ID required", http.StatusBadRequest)
		return
	}

	summary, err := h.steamClient.GetPlayerSummary(steamID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch player profile: %v", err), http.StatusInternalServerError)
		return
	}

	rawStats, err := h.steamClient.GetPlayerStats(steamID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch player stats: %v", err), http.StatusInternalServerError)
		return
	}

	playerStats := steam.MapSteamStats(rawStats.Stats, summary.SteamID, summary.PersonaName)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(playerStats); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
