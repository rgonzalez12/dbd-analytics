package api

import (
	"encoding/json"
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
		writeErrorResponse(w, steam.NewValidationError("Steam ID required"))
		return
	}

	summary, err := h.steamClient.GetPlayerSummary(steamID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, summary)
}

func (h *Handler) GetPlayerStats(w http.ResponseWriter, r *http.Request) {
	steamID := mux.Vars(r)["steamid"]
	if steamID == "" {
		writeErrorResponse(w, steam.NewValidationError("Steam ID required"))
		return
	}

	summary, err := h.steamClient.GetPlayerSummary(steamID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	rawStats, err := h.steamClient.GetPlayerStats(steamID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	playerStats := steam.MapSteamStats(rawStats.Stats, summary.SteamID, summary.PersonaName)
	writeJSONResponse(w, playerStats)
}

// writeErrorResponse writes a structured error response to the client
func writeErrorResponse(w http.ResponseWriter, apiErr *steam.APIError) {
	statusCode := http.StatusInternalServerError
	if apiErr.StatusCode != 0 {
		statusCode = apiErr.StatusCode
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(apiErr)
}

// writeJSONResponse writes a successful JSON response to the client
func writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		writeErrorResponse(w, steam.NewInternalError(err))
		return
	}
}
