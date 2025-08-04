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
	router.HandleFunc("/api/player/{steamid}/summary", handler.GetPlayerSummary).Methods("GET")
	router.HandleFunc("/api/player/{steamid}/stats", handler.GetPlayerStats).Methods("GET")

	tests := []struct {
		name           string
		steamID        string
		endpoint       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Invalid Steam ID - Too Short",
			steamID:        "123456789",
			endpoint:       "/summary",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid Steam ID format. Must be 17 digits starting with 7656119",
		},
		{
			name:           "Invalid Steam ID - Too Long",
			steamID:        "765611980000000001",
			endpoint:       "/summary",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid Steam ID format. Must be 17 digits starting with 7656119",
		},
		{
			name:           "Invalid Steam ID - Wrong Prefix",
			steamID:        "12345678901234567",
			endpoint:       "/summary",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid Steam ID format. Must be 17 digits starting with 7656119",
		},
		{
			name:           "Invalid Steam ID - Contains Letters",
			steamID:        "7656119800000000a",
			endpoint:       "/summary",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid Steam ID format. Must be 17 digits starting with 7656119",
		},
		{
			name:           "Invalid Vanity URL - Too Short",
			steamID:        "ab",
			endpoint:       "/summary",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid vanity URL format. Must be 3-32 characters, alphanumeric with underscore/hyphen only",
		},
		{
			name:           "Invalid Vanity URL - Special Characters",
			steamID:        "test@user",
			endpoint:       "/summary",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid vanity URL format. Must be 3-32 characters, alphanumeric with underscore/hyphen only",
		},
		{
			name:           "Validation Applied to Stats Endpoint",
			steamID:        "test@invalid",
			endpoint:       "/stats",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid vanity URL format. Must be 3-32 characters, alphanumeric with underscore/hyphen only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/player/" + tt.steamID + tt.endpoint

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

				if response.Error.Message != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, response.Error.Message)
				}

				if response.Error.Code != "VALIDATION_ERROR" {
					t.Errorf("Expected error code 'VALIDATION_ERROR', got '%s'", response.Error.Code)
				}

				// Verify request ID is present
				if response.Error.RequestID == "" {
					t.Errorf("Expected request ID to be present in error response")
				}
			}
		})
	}
}
