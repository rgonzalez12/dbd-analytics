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
		slog.Warn("GetPlayerSummary called without steam ID",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path))
		writeErrorResponse(w, steam.NewValidationError("Steam ID required"))
		return
	}

	slog.Info("Processing player summary request", 
		slog.String("steam_id", steamID),
		slog.String("client_ip", r.RemoteAddr))

	summary, err := h.steamClient.GetPlayerSummary(steamID)
	if err != nil {
		slog.Error("Failed to get player summary", 
			slog.String("steam_id", steamID), 
			slog.String("error", err.Message), 
			slog.String("error_type", string(err.Type)),
			slog.Bool("retryable", err.Retryable))
		writeErrorResponse(w, err)
		return
	}

	slog.Info("Successfully processed player summary request", 
		slog.String("steam_id", steamID), 
		slog.String("persona_name", summary.PersonaName))
	writeJSONResponse(w, summary)
}

func (h *Handler) GetPlayerStats(w http.ResponseWriter, r *http.Request) {
	steamID := mux.Vars(r)["steamid"]
	if steamID == "" {
		slog.Warn("GetPlayerStats called without steam ID",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path))
		writeErrorResponse(w, steam.NewValidationError("Steam ID required"))
		return
	}

	slog.Info("Processing player stats request", 
		slog.String("steam_id", steamID),
		slog.String("client_ip", r.RemoteAddr))

	summary, err := h.steamClient.GetPlayerSummary(steamID)
	if err != nil {
		slog.Error("Failed to get player summary for stats request", 
			slog.String("steam_id", steamID), 
			slog.String("error", err.Message), 
			slog.String("error_type", string(err.Type)))
		writeErrorResponse(w, err)
		return
	}

	rawStats, err := h.steamClient.GetPlayerStats(steamID)
	if err != nil {
		slog.Error("Failed to get player stats", 
			slog.String("steam_id", steamID), 
			slog.String("error", err.Message), 
			slog.String("error_type", string(err.Type)))
		writeErrorResponse(w, err)
		return
	}

	playerStats := steam.MapSteamStats(rawStats.Stats, summary.SteamID, summary.PersonaName)
	slog.Info("Successfully processed player stats request", 
		slog.String("steam_id", steamID), 
		slog.Int("raw_stats_count", len(rawStats.Stats)),
		slog.String("persona_name", summary.PersonaName))
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
		slog.Error("Failed to encode error response", 
			slog.String("error", err.Error()),
			slog.String("original_error", apiErr.Message))
		// Fallback to plain text if JSON encoding fails
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// writeJSONResponse writes a successful JSON response to the client
func writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", 
			slog.String("error", err.Error()))
		writeErrorResponse(w, steam.NewInternalError(err))
		return
	}
}
