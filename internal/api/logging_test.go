package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

func TestStructuredLoggingValidation(t *testing.T) {
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

			router.HandleFunc("/api/player/{steamid}", handler.GetPlayerStatsWithAchievements).Methods("GET")

			req := httptest.NewRequest("GET", "/api/player/"+tt.steamID, nil)
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

func TestErrorLogging(t *testing.T) {
	log.Initialize()

	router := mux.NewRouter()

	// Add simple request ID middleware for logging context
	router.Use(RequestIDMiddleware())

	handler := NewHandler()
	router.HandleFunc("/api/player/{steamid}", handler.GetPlayerStatsWithAchievements).Methods("GET")

	// Test with invalid steam ID to trigger error logging
	req := httptest.NewRequest("GET", "/api/player/invalid@id", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 400 for invalid steam ID
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Verify error response structure
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response is not valid JSON: %s", w.Body.String())
	}

	// Check for message field (common error response format)
	if response["message"] == nil {
		t.Error("Expected 'message' field in response")
	}
}
