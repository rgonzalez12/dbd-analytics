package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/rgonzalez12/dbd-analytics/internal/log"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

func TestStructuredLoggingValidation(t *testing.T) {
	// Initialize logging for testing
	log.Initialize()

	tests := []struct {
		name           string
		steamID        string
		expectedStatus int
	}{
		{
			name:           "Invalid Steam ID",
			steamID:        "invalid@id",
			expectedStatus: 400,
		},
		{
			name:           "Short Steam ID",
			steamID:        "123",
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler
			handler := NewHandler()
			router := mux.NewRouter()

			// Add the same logging middleware as main.go
			router.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					vars := mux.Vars(req)
					steamID := vars["steamid"]

					log.Info("incoming_request",
						"method", req.Method,
						"path", req.URL.Path,
						"steam_id", steamID)

					next.ServeHTTP(w, req)

					log.Info("request_completed",
						"method", req.Method,
						"path", req.URL.Path,
						"steam_id", steamID)
				})
			})

			router.HandleFunc("/api/player/{steamid}/summary", handler.GetPlayerSummary).Methods("GET")

			req := httptest.NewRequest("GET", "/api/player/"+tt.steamID+"/summary", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Verify the response is valid JSON
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Response is not valid JSON: %s", w.Body.String())
			}

			// Should have message field (flat error format)
			if response["message"] == nil {
				t.Error("Expected 'message' field in response")
			}

			t.Logf("Test case '%s' completed successfully", tt.name)
		})
	}
}

func TestLogOutputFormat(t *testing.T) {
	// Initialize logging for testing
	log.Initialize()

	log.Info("test_message",
		"test_field", "test_value",
		"test_number", 42)

	// Note: In a real test environment, we'd need to capture the output
	// For now, this tests that the calls work correctly
	t.Log("Log output format test completed")
}

func TestErrorLogging(t *testing.T) {
	// Initialize logging for testing
	log.Initialize()

	// Test error response logging
	w := httptest.NewRecorder()
	apiErr := &steam.APIError{
		Type:       steam.ErrorTypeValidation,
		Message:    "Test validation error",
		StatusCode: 400,
		Retryable:  false,
	}

	writeErrorResponse(w, apiErr)

	// Note: In a real test environment, we'd need to capture the output
	// For now, this tests that the error response generation works correctly

	if w.Code != 400 {
		t.Errorf("Expected status code 400, got %d", w.Code)
	}

	// Verify response body contains error information
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Response is not valid JSON: %s", w.Body.String())
	}

	if response["error"] != "Test validation error" {
		t.Errorf("Expected error message 'Test validation error', got %v", response["error"])
	}
}
