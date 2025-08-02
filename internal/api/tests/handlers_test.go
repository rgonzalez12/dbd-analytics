package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rgonzalez12/dbd-analytics/internal/api"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

func TestGetPlayerSummary_ValidationError(t *testing.T) {
	handler := api.NewHandler()

	// Create a request with empty steam ID in URL vars
	req := httptest.NewRequest("GET", "/player/summary", nil)
	w := httptest.NewRecorder()
	
	// Call handler directly
	handler.GetPlayerSummary(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Verify JSON error response format
	var errorResponse map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errorResponse["error"] == nil {
		t.Error("Expected 'error' field in response")
	}

	if errorResponse["type"] == nil {
		t.Error("Expected 'type' field in response")
	}

	if errorResponse["type"] != string(steam.ErrorTypeValidation) {
		t.Errorf("Expected error type %s, got %v", steam.ErrorTypeValidation, errorResponse["type"])
	}

	// Verify Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

func TestGetPlayerStats_ValidationError(t *testing.T) {
	handler := api.NewHandler()

	// Create a request with empty steam ID
	req := httptest.NewRequest("GET", "/player/stats", nil)
	w := httptest.NewRecorder()
	
	// Call handler directly
	handler.GetPlayerStats(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Verify JSON error response format
	var errorResponse map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errorResponse["error"] == nil {
		t.Error("Expected 'error' field in response")
	}

	if errorResponse["type"] != string(steam.ErrorTypeValidation) {
		t.Errorf("Expected error type %s, got %v", steam.ErrorTypeValidation, errorResponse["type"])
	}
}

func TestErrorResponseFormat(t *testing.T) {
	// Test various error types to ensure consistent JSON format
	testCases := []struct {
		name     string
		apiError *steam.APIError
		expected map[string]interface{}
	}{
		{
			name:     "RateLimit",
			apiError: steam.NewRateLimitError(),
			expected: map[string]interface{}{
				"error": "Steam API rate-limited, try again later",
				"type":  string(steam.ErrorTypeRateLimit),
			},
		},
		{
			name:     "NotFound",
			apiError: steam.NewNotFoundError("Player"),
			expected: map[string]interface{}{
				"error": "Player not found",
				"type":  string(steam.ErrorTypeNotFound),
			},
		},
		{
			name:     "Validation",
			apiError: steam.NewValidationError("Invalid input"),
			expected: map[string]interface{}{
				"error": "Invalid input",
				"type":  string(steam.ErrorTypeValidation),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			
			// Simulate the error response function behavior
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(tc.apiError.StatusCode)
			
			errorResponse := map[string]interface{}{
				"error": tc.apiError.Message,
				"type":  tc.apiError.Type,
			}
			
			if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
				t.Fatalf("Failed to encode error response: %v", err)
			}

			// Verify the response
			var actualResponse map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&actualResponse); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if actualResponse["error"] != tc.expected["error"] {
				t.Errorf("Expected error message '%s', got '%s'", tc.expected["error"], actualResponse["error"])
			}

			if actualResponse["type"] != tc.expected["type"] {
				t.Errorf("Expected error type '%s', got '%s'", tc.expected["type"], actualResponse["type"])
			}
		})
	}
}
