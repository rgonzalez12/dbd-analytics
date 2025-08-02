package api

import (
	"encoding/json"
	"log/slog"
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
		slog.Warn("GetPlayerSummary called without steam ID")
		writeErrorResponse(w, steam.NewValidationError("Steam ID required"))
		return
	}

	slog.Info("Getting player summary", "steam_id", steamID)

	summary, err := h.steamClient.GetPlayerSummary(steamID)
	if err != nil {
		slog.Error("Failed to get player summary", "steam_id", steamID, "error", err.Message, "error_type", err.Type)
		writeErrorResponse(w, err)
		return
	}

	slog.Info("Successfully retrieved player summary", "steam_id", steamID, "persona_name", summary.PersonaName)
	writeJSONResponse(w, summary)
}

func (h *Handler) GetPlayerStats(w http.ResponseWriter, r *http.Request) {
	steamID := mux.Vars(r)["steamid"]
	if steamID == "" {
		slog.Warn("GetPlayerStats called without steam ID")
		writeErrorResponse(w, steam.NewValidationError("Steam ID required"))
		return
	}

	slog.Info("Getting player stats", "steam_id", steamID)

	summary, err := h.steamClient.GetPlayerSummary(steamID)
	if err != nil {
		slog.Error("Failed to get player summary for stats", "steam_id", steamID, "error", err.Message, "error_type", err.Type)
		writeErrorResponse(w, err)
		return
	}

	rawStats, err := h.steamClient.GetPlayerStats(steamID)
	if err != nil {
		slog.Error("Failed to get player stats", "steam_id", steamID, "error", err.Message, "error_type", err.Type)
		writeErrorResponse(w, err)
		return
	}

	playerStats := steam.MapSteamStats(rawStats.Stats, summary.SteamID, summary.PersonaName)
	slog.Info("Successfully retrieved and mapped player stats", "steam_id", steamID, "stats_count", len(rawStats.Stats))
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
	
	// Create consistent error response format
	errorResponse := map[string]interface{}{
		"error": apiErr.Message,
		"type":  apiErr.Type,
	}
	
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		slog.Error("Failed to encode error response", "error", err)
		// Fallback to plain text if JSON encoding fails
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// writeJSONResponse writes a successful JSON response to the client
func writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
		writeErrorResponse(w, steam.NewInternalError(err))
		return
	}
}
