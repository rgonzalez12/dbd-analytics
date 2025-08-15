package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rgonzalez12/dbd-analytics/internal/log"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

// StandardError represents the consistent JSON error response format that matches frontend ApiError type
type StandardError struct {
	Status    int                    `json:"status"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	RetryAfter *int                  `json:"retryAfter,omitempty"`
}

// writeStandardErrorResponse writes standardized JSON error responses
func writeStandardErrorResponse(w http.ResponseWriter, r *http.Request, code string, message string, statusCode int, details map[string]interface{}, retryAfter *int) {
	// Get request ID from context if available
	requestID := ""
	if id := r.Context().Value("request_id"); id != nil {
		if idStr, ok := id.(string); ok {
			requestID = idStr
		}
	}

	// If no request ID in context, generate one
	if requestID == "" {
		requestID = generateRequestID()
	}

	// Add request_id to details for debugging
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

	// Log the error with context
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
		// Fallback to plain text
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// writeValidationError writes standardized validation error responses
func writeValidationError(w http.ResponseWriter, r *http.Request, message string, field string) {
	details := map[string]interface{}{
		"field": field,
	}
	writeStandardErrorResponse(w, r, "VALIDATION_ERROR", message, http.StatusBadRequest, details, nil)
}

// writeSteamAPIError converts Steam API errors to standardized format
func writeSteamAPIError(w http.ResponseWriter, r *http.Request, apiErr *steam.APIError) {
	var code string
	var statusCode int
	var retryAfter *int
	details := make(map[string]interface{})

	// Map Steam API errors to standard error codes
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

	// Add retryability information
	if apiErr.Retryable {
		details["retryable"] = true
	}

	writeStandardErrorResponse(w, r, code, apiErr.Message, statusCode, details, retryAfter)
}

// writeTimeoutError creates a standardized timeout error response
func writeTimeoutError(w http.ResponseWriter, r *http.Request, operation string) {
	details := map[string]interface{}{
		"operation": operation,
		"timeout":   true,
	}
	writeStandardErrorResponse(w, r, "REQUEST_TIMEOUT", 
		"Request timeout during "+operation+" operation", 
		http.StatusRequestTimeout, details, nil)
}

// writeInternalError writes standardized internal error responses
func writeInternalError(w http.ResponseWriter, r *http.Request, message string) {
	writeStandardErrorResponse(w, r, "INTERNAL_ERROR", message, http.StatusInternalServerError, nil, nil)
}
