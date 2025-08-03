package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rgonzalez12/dbd-analytics/internal/log"
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

// generateRequestID creates a unique request ID for error tracking and debugging
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to simple counter-based ID if cryptographic random generation fails
		return fmt.Sprintf("req_%d", len(bytes))
	}
	return hex.EncodeToString(bytes)
}

// validateSteamID validates that a Steam ID follows Steam's 64-bit ID format
func validateSteamID(steamID string) bool {
	// Steam ID must be exactly 17 digits in length
	if len(steamID) != 17 {
		return false
	}
	
	// Ensure all characters are numeric
	if _, err := strconv.ParseUint(steamID, 10, 64); err != nil {
		return false
	}
	
	// Steam IDs must start with 7656119 (Steam's standardized base)
	return steamID[:7] == "7656119"
}

// isValidVanityURL validates that a vanity URL meets Steam's naming requirements
func isValidVanityURL(vanity string) bool {
	// Steam vanity URLs must be 3-32 characters, alphanumeric plus underscore/hyphen
	if len(vanity) < 3 || len(vanity) > 32 {
		return false
	}
	
	// Use pre-compiled regex for optimal performance
	return vanityURLRegex.MatchString(vanity)
}

// validateSteamIDOrVanity validates input as either a Steam ID or vanity URL
func validateSteamIDOrVanity(input string) *steam.APIError {
	if input == "" {
		return steam.NewValidationError("Steam ID or vanity URL required")
	}
	
	// If input starts with Steam ID prefix (7656119), validate as Steam ID
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
	start := time.Now()
	steamID := mux.Vars(r)["steamid"]
	
	// Create structured logger with comprehensive request context
	requestLogger := log.WithContext(
		"steam_id", steamID,
		"client_ip", r.RemoteAddr,
		"method", r.Method,
		"path", r.URL.Path,
	)
	
	// Validate Steam ID format before making any external API calls
	if err := validateSteamIDOrVanity(steamID); err != nil {
		requestLogger.Warn("Invalid Steam ID format in GetPlayerSummary",
			"user_agent", r.UserAgent(),
			"error", err.Message,
			"validation_type", string(err.Type))
		writeErrorResponse(w, err)
		return
	}

	requestLogger.Info("Processing player summary request")

	summary, err := h.steamClient.GetPlayerSummary(steamID)
	if err != nil {
		// Log Steam API errors with comprehensive context for debugging
		requestLogger.Error("Failed to get player summary",
			"error", err.Message,
			"error_type", string(err.Type),
			"retryable", err.Retryable,
			"duration", time.Since(start))
		writeErrorResponse(w, err)
		return
	}

	requestLogger.Info("Successfully processed player summary request",
		"persona_name", summary.PersonaName,
		"duration", time.Since(start))
	writeJSONResponse(w, summary)
}

func (h *Handler) GetPlayerStats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	steamID := mux.Vars(r)["steamid"]
	
	// Create structured logger with comprehensive request context
	requestLogger := log.WithContext(
		"steam_id", steamID,
		"client_ip", r.RemoteAddr,
		"method", r.Method,
		"path", r.URL.Path,
	)
	
	// Validate Steam ID format before processing
	if err := validateSteamIDOrVanity(steamID); err != nil {
		requestLogger.Warn("Invalid Steam ID format in GetPlayerStats",
			"user_agent", r.UserAgent(),
			"error", err.Message,
			"validation_type", string(err.Type))
		writeErrorResponse(w, err)
		return
	}

	requestLogger.Info("Processing player stats request")

	summary, err := h.steamClient.GetPlayerSummary(steamID)
	if err != nil {
		requestLogger.Error("Failed to get player summary for stats request",
			"error", err.Message,
			"error_type", string(err.Type),
			"duration", time.Since(start))
		writeErrorResponse(w, err)
		return
	}

	rawStats, err := h.steamClient.GetPlayerStats(steamID)
	if err != nil {
		requestLogger.Error("Failed to get player stats",
			"persona_name", summary.PersonaName,
			"error", err.Message,
			"error_type", string(err.Type),
			"duration", time.Since(start))
		writeErrorResponse(w, err)
		return
	}

	playerStats := steam.MapSteamStats(rawStats.Stats, summary.SteamID, summary.PersonaName)
	requestLogger.Info("Successfully processed player stats request",
		"raw_stats_count", len(rawStats.Stats),
		"persona_name", summary.PersonaName,
		"duration", time.Since(start))
	writeJSONResponse(w, playerStats)
}

// writeErrorResponse writes a structured error response to the client
func writeErrorResponse(w http.ResponseWriter, apiErr *steam.APIError) {
	// Generate a unique request ID for tracing
	requestID := generateRequestID()
	
	// Determine the appropriate HTTP status code
	statusCode := determineStatusCode(apiErr)
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(statusCode)
	
	// Create enhanced error response format
	errorResponse := map[string]interface{}{
		"error":      apiErr.Message,
		"type":       string(apiErr.Type),
		"request_id": requestID,
	}
	
	// Add details for specific error types
	switch apiErr.Type {
	case steam.ErrorTypeRateLimit:
		errorResponse["details"] = "Steam API rate limit exceeded"
		// Use the actual retry_after value from the Steam API response
		retryAfter := 60 // Default fallback
		if apiErr.RetryAfter > 0 {
			retryAfter = apiErr.RetryAfter
		}
		errorResponse["retry_after"] = retryAfter
		
	case steam.ErrorTypeAPIError:
		if apiErr.StatusCode != 0 {
			errorResponse["details"] = fmt.Sprintf("Steam API returned %d %s", apiErr.StatusCode, http.StatusText(apiErr.StatusCode))
			// Differentiate client vs server errors from Steam
			if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
				errorResponse["source"] = "client_error"
			} else {
				errorResponse["source"] = "steam_api_error"
			}
		}
		if apiErr.Retryable {
			errorResponse["retry_after"] = 30 // Retry after 30 seconds for API errors
		}
		
	case steam.ErrorTypeNetwork:
		errorResponse["details"] = "Network connection to Steam API failed"
		errorResponse["source"] = "steam_api_error"
		errorResponse["retry_after"] = 30
		
	case steam.ErrorTypeNotFound:
		errorResponse["details"] = "Requested resource not found on Steam"
		errorResponse["source"] = "client_error"
		
	case steam.ErrorTypeValidation:
		errorResponse["details"] = "Invalid request parameters"
		errorResponse["source"] = "client_error"
		
	case steam.ErrorTypeInternal:
		errorResponse["details"] = "Internal server error occurred"
		errorResponse["source"] = "server_error"
	}
	
	// Add retryable flag for client guidance
	if apiErr.Retryable {
		errorResponse["retryable"] = true
	}
	
	// Log the error with request ID for tracing
	log.Error("API error response generated",
		"request_id", requestID,
		"error_type", string(apiErr.Type),
		"status_code", statusCode,
		"error_message", apiErr.Message)
	
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		log.Error("Failed to encode error response", 
			"request_id", requestID,
			"error", err.Error(),
			"original_error", apiErr.Message)
		// Fallback to plain text if JSON encoding fails
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// determineStatusCode maps API error types to appropriate HTTP status codes
func determineStatusCode(apiErr *steam.APIError) int {
	// If the error already has a status code, use it but map appropriately
	if apiErr.StatusCode != 0 {
		switch apiErr.Type {
		case steam.ErrorTypeAPIError:
			// For Steam API errors, differentiate client vs server issues
			if apiErr.StatusCode == http.StatusForbidden || apiErr.StatusCode == http.StatusNotFound {
				// Steam returned 403/404 - pass through as client error
				return apiErr.StatusCode
			} else if apiErr.StatusCode >= 500 {
				// Steam server errors - return 502 Bad Gateway
				return http.StatusBadGateway
			} else if apiErr.StatusCode == http.StatusTooManyRequests {
				// Rate limiting - pass through
				return apiErr.StatusCode
			} else {
				// Other 4xx from Steam - return 502 as it's likely Steam API issue
				return http.StatusBadGateway
			}
		default:
			return apiErr.StatusCode
		}
	}
	
	// Map error types to status codes when no status code is set
	switch apiErr.Type {
	case steam.ErrorTypeValidation:
		return http.StatusBadRequest // 400
	case steam.ErrorTypeNotFound:
		return http.StatusNotFound // 404
	case steam.ErrorTypeRateLimit:
		return http.StatusTooManyRequests // 429
	case steam.ErrorTypeAPIError, steam.ErrorTypeNetwork:
		return http.StatusBadGateway // 502 - Steam API is down/unreachable
	case steam.ErrorTypeInternal:
		return http.StatusInternalServerError // 500
	default:
		return http.StatusInternalServerError // 500 - Safe default
	}
}

// writeJSONResponse writes a successful JSON response to the client
func writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	
	// Marshal to get response size for logging
	responseBytes, err := json.Marshal(data)
	if err != nil {
		log.Error("Failed to marshal JSON response", 
			"error", err.Error())
		writeErrorResponse(w, steam.NewInternalError(err))
		return
	}
	
	// Log successful response
	log.Info("successful_response_sent",
		"status_code", http.StatusOK,
		"response_size", len(responseBytes),
		"content_type", "application/json")
	
	// Write the response
	if _, err := w.Write(responseBytes); err != nil {
		log.Error("Failed to write JSON response", 
			"error", err.Error(),
			"response_size", len(responseBytes))
		// Can't call writeErrorResponse here as headers are already sent
		return
	}
}
