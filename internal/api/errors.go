package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rgonzalez12/dbd-analytics/internal/log"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

type StandardError struct {
	Status     int                    `json:"status"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	RetryAfter *int                   `json:"retryAfter,omitempty"`
}

func writeError(w http.ResponseWriter, r *http.Request, code string, message string, statusCode int, details map[string]interface{}, retryAfter *int) {
	requestID := ""
	if id := r.Context().Value(requestIDKey); id != nil {
		if idStr, ok := id.(string); ok {
			requestID = idStr
		}
	}

	if requestID == "" {
		requestID = generateRequestID()
	}

	if details == nil {
		details = make(map[string]interface{})
	}
	details["request_id"] = requestID
	details["code"] = code

	errorResponse := StandardError{
		Status:     statusCode,
		Message:    message,
		Details:    details,
		RetryAfter: retryAfter,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(statusCode)

	log.Error("API error response",
		"request_id", requestID,
		"error_code", code,
		"status_code", statusCode,
		"method", r.Method,
		"path", r.URL.Path,
		"client_ip", r.RemoteAddr)

	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		log.Error("Failed to encode error response",
			"request_id", requestID,
			"encoding_error", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func writeValidationError(w http.ResponseWriter, r *http.Request, message string, field string) {
	details := map[string]interface{}{
		"field": field,
	}
	writeError(w, r, "VALIDATION_ERROR", message, http.StatusBadRequest, details, nil)
}

func writeSteamAPIError(w http.ResponseWriter, r *http.Request, apiErr *steam.APIError) {
	var code string
	var statusCode int
	var retryAfter *int
	details := make(map[string]interface{})

	switch apiErr.Type {
	case steam.ErrorTypeValidation:
		code = "VALIDATION_ERROR"
		statusCode = http.StatusBadRequest
		details["field"] = "steam_id"
	case steam.ErrorTypeNotFound:
		code = "NOT_FOUND"
		statusCode = http.StatusNotFound
		details["resource"] = "steam_profile"
	case steam.ErrorTypeRateLimit:
		code = "RATE_LIMITED"
		statusCode = http.StatusTooManyRequests
		if apiErr.RetryAfter > 0 {
			details["retry_after_seconds"] = apiErr.RetryAfter
			w.Header().Set("Retry-After", fmt.Sprintf("%d", apiErr.RetryAfter))
			retryAfterInt := int(apiErr.RetryAfter)
			retryAfter = &retryAfterInt
		}
	case steam.ErrorTypeAPIError:
		if apiErr.StatusCode >= 500 {
			code = "EXTERNAL_SERVICE_ERROR"
			statusCode = http.StatusBadGateway
		} else {
			code = "EXTERNAL_SERVICE_ERROR"
			statusCode = http.StatusBadGateway
		}
		details["service"] = "steam_api"
		if apiErr.StatusCode != 0 {
			details["upstream_status"] = apiErr.StatusCode
		}
	case steam.ErrorTypeNetwork:
		code = "EXTERNAL_SERVICE_UNAVAILABLE"
		statusCode = http.StatusBadGateway
		details["service"] = "steam_api"
	default:
		code = "INTERNAL_ERROR"
		statusCode = http.StatusInternalServerError
	}

	if apiErr.Retryable {
		details["retryable"] = true
	}

	writeError(w, r, code, apiErr.Message, statusCode, details, retryAfter)
}

// writeTimeoutError creates a standardized timeout error response
func writeTimeoutError(w http.ResponseWriter, r *http.Request, operation string) {
	details := map[string]interface{}{
		"operation": operation,
		"timeout":   true,
	}
	writeError(w, r, "REQUEST_TIMEOUT",
		"Request timeout during "+operation+" operation",
		http.StatusRequestTimeout, details, nil)
}
