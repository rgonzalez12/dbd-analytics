package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

var (
	// Pre-compiled regex patterns for performance
	digitOnlyRegex   = regexp.MustCompile(`^\d+$`)
	vanityURLRegex   = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

type Handler struct {
	steamClient *steam.Client
}

func NewHandler() *Handler {
	return &Handler{
		steamClient: steam.NewClient(),
	}
}

// validateSteamID validates that a Steam ID is exactly 17 digits
func validateSteamID(steamID string) bool {
	// Steam ID must be exactly 17 digits
	if len(steamID) != 17 {
		return false
	}
	
	// Must be all numeric
	if _, err := strconv.ParseUint(steamID, 10, 64); err != nil {
		return false
	}
	
	// Additional check: Steam IDs should start with 7656119 (Steam's base)
	return steamID[:7] == "7656119"
}

// isValidVanityURL validates that a vanity URL contains only allowed characters
func isValidVanityURL(vanity string) bool {
	// Vanity URLs should be 3-32 characters, alphanumeric plus underscore/hyphen
	if len(vanity) < 3 || len(vanity) > 32 {
		return false
	}
	
	// Use pre-compiled regex for performance
	return vanityURLRegex.MatchString(vanity)
}

// validateSteamIDOrVanity validates the input as either a Steam ID or vanity URL
func validateSteamIDOrVanity(input string) *steam.APIError {
	if input == "" {
		return steam.NewValidationError("Steam ID or vanity URL required")
	}
	
	// If it starts with 7656119 (Steam ID prefix), it must be a valid Steam ID
	if len(input) >= 7 && input[:7] == "7656119" {
		if !validateSteamID(input) {
			return steam.NewValidationError("Invalid Steam ID format. Must be 17 digits starting with 7656119")
		}
		return nil
	}
	
	// If it's all digits but doesn't start with Steam prefix, it's an invalid Steam ID
	if digitOnlyRegex.MatchString(input) {
		return steam.NewValidationError("Invalid Steam ID format. Must be 17 digits starting with 7656119")
	}
	
	// Otherwise validate as vanity URL
	if !isValidVanityURL(input) {
		return steam.NewValidationError("Invalid vanity URL format. Must be 3-32 characters, alphanumeric with underscore/hyphen only")
	}
	
	return nil
}

func (h *Handler) GetPlayerSummary(w http.ResponseWriter, r *http.Request) {
	steamID := mux.Vars(r)["steamid"]
	
	// Validate Steam ID format before processing
	if err := validateSteamIDOrVanity(steamID); err != nil {
		slog.Warn("Invalid Steam ID format in GetPlayerSummary",
			slog.String("steam_id", steamID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.String("error", err.Message),
			slog.String("validation_type", string(err.Type)))
		writeErrorResponse(w, err)
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
	
	// Validate Steam ID format before processing
	if err := validateSteamIDOrVanity(steamID); err != nil {
		slog.Warn("Invalid Steam ID format in GetPlayerStats",
			slog.String("steam_id", steamID),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.String("error", err.Message),
			slog.String("validation_type", string(err.Type)))
		writeErrorResponse(w, err)
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
