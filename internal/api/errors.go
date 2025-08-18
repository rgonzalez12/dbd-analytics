package api

import (
	"encoding/json"
	"net/http"

	"github.com/rgonzalez12/dbd-analytics/internal/log"
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
		requestID = GenerateRequestID()
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
