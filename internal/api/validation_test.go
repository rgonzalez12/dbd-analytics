package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestSteamIDValidation(t *testing.T) {
	handler := NewHandler()
	router := mux.NewRouter()
	router.HandleFunc("/api/player/{steamid}", handler.GetPlayerStatsWithAchievements).Methods("GET")

	tests := []struct {
		name           string
		steamID        string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Invalid Steam ID Format",
			steamID:        "123456789",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid Steam ID format. Must be 17 digits starting with 7656119",
		},
		{
			name:           "Invalid Steam ID Prefix",
			steamID:        "12345678901234567",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid Steam ID format. Must be 17 digits starting with 7656119",
		},
		{
			name:           "Invalid Vanity URL Format",
			steamID:        "ab",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid vanity URL format. Must be 3-32 characters, alphanumeric with underscore/hyphen only",
		},
		{
			name:           "Invalid Vanity URL Characters",
			steamID:        "test@user",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid vanity URL format. Must be 3-32 characters, alphanumeric with underscore/hyphen only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/player/" + tt.steamID

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedError != "" && w.Code == http.StatusBadRequest {
				var response StandardError
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v. Body: %s", err, w.Body.String())
				}

				if response.Message != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, response.Message)
				}

				if code, ok := response.Details["code"].(string); !ok || code != "VALIDATION_ERROR" {
					t.Errorf("Expected error code 'VALIDATION_ERROR', got '%v'", code)
				}

				// Verify request ID is present
				if requestID, ok := response.Details["request_id"].(string); !ok || requestID == "" {
					t.Errorf("Expected request ID to be present in error response details")
				}
			}
		})
	}
}
