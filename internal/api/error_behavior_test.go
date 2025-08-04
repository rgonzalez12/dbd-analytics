package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

func TestErrorDifferentiation(t *testing.T) {
	tests := []struct {
		name              string
		steamError        *steam.APIError
		expectedStatus    int
		expectedSource    string
		expectedRequestID bool
		shouldHaveRetry   bool
	}{
		{
			name: "Steam 403 Forbidden - Client Error",
			steamError: &steam.APIError{
				Type:       steam.ErrorTypeAPIError,
				Message:    "Forbidden",
				StatusCode: 403,
				Retryable:  false,
			},
			expectedStatus:    403,
			expectedSource:    "client_error",
			expectedRequestID: true,
			shouldHaveRetry:   false,
		},
		{
			name: "Steam 404 Not Found - Client Error",
			steamError: &steam.APIError{
				Type:       steam.ErrorTypeAPIError,
				Message:    "Not Found",
				StatusCode: 404,
				Retryable:  false,
			},
			expectedStatus:    404,
			expectedSource:    "client_error",
			expectedRequestID: true,
			shouldHaveRetry:   false,
		},
		{
			name: "Steam 500 Server Error - Steam API Error",
			steamError: &steam.APIError{
				Type:       steam.ErrorTypeAPIError,
				Message:    "Internal Server Error",
				StatusCode: 500,
				Retryable:  true,
			},
			expectedStatus:    502, // Mapped to Bad Gateway
			expectedSource:    "steam_api_error",
			expectedRequestID: true,
			shouldHaveRetry:   true,
		},
		{
			name: "Steam 502 Bad Gateway - Steam API Error",
			steamError: &steam.APIError{
				Type:       steam.ErrorTypeAPIError,
				Message:    "Bad Gateway",
				StatusCode: 502,
				Retryable:  true,
			},
			expectedStatus:    502,
			expectedSource:    "steam_api_error",
			expectedRequestID: true,
			shouldHaveRetry:   true,
		},
		{
			name: "Network Error - Steam API Error",
			steamError: &steam.APIError{
				Type:      steam.ErrorTypeNetwork,
				Message:   "Connection timeout",
				Retryable: true,
			},
			expectedStatus:    502,
			expectedSource:    "steam_api_error",
			expectedRequestID: true,
			shouldHaveRetry:   true,
		},
		{
			name: "Validation Error - Client Error",
			steamError: &steam.APIError{
				Type:       steam.ErrorTypeValidation,
				Message:    "Invalid Steam ID",
				StatusCode: 400,
				Retryable:  false,
			},
			expectedStatus:    400,
			expectedSource:    "client_error",
			expectedRequestID: true,
			shouldHaveRetry:   false,
		},
		{
			name: "Internal Server Error - Server Error",
			steamError: &steam.APIError{
				Type:      steam.ErrorTypeInternal,
				Message:   "Database connection failed",
				Retryable: false,
			},
			expectedStatus:    500,
			expectedSource:    "server_error",
			expectedRequestID: true,
			shouldHaveRetry:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			// Call writeErrorResponse directly
			writeErrorResponse(w, tt.steamError)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check X-Request-ID header
			requestID := w.Header().Get("X-Request-ID")
			if tt.expectedRequestID && requestID == "" {
				t.Error("Expected X-Request-ID header to be present")
			}

			// Parse response body
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			// Check request_id in response body
			if tt.expectedRequestID {
				if response["request_id"] == nil {
					t.Error("Expected 'request_id' field in response")
				}
				// Should match header
				if response["request_id"] != requestID {
					t.Error("Request ID in body should match header")
				}
			}

			// Check source field
			if response["source"] != tt.expectedSource {
				t.Errorf("Expected source '%s', got '%s'", tt.expectedSource, response["source"])
			}

			// Check retry behavior
			if tt.shouldHaveRetry {
				if response["retryable"] != true {
					t.Error("Expected 'retryable' to be true")
				}
				if response["retry_after"] == nil {
					t.Error("Expected 'retry_after' field for retryable errors")
				}
			} else {
				if response["retryable"] == true {
					t.Error("Expected 'retryable' to be false or absent")
				}
			}

			t.Logf("Response: %+v", response)
		})
	}
}

func TestHandlerErrorPropagation(t *testing.T) {
	// Test that handler errors include proper context and request IDs
	tests := []struct {
		name     string
		steamID  string
		endpoint string
	}{
		{
			name:     "Invalid Steam ID format",
			steamID:  "invalid@id",
			endpoint: "/summary",
		},
		{
			name:     "Too short Steam ID",
			steamID:  "123",
			endpoint: "/stats",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler()
			router := mux.NewRouter()
			router.HandleFunc("/api/player/{steamid}/summary", handler.GetPlayerSummary).Methods("GET")
			router.HandleFunc("/api/player/{steamid}/stats", handler.GetPlayerStats).Methods("GET")

			req := httptest.NewRequest("GET", "/api/player/"+tt.steamID+tt.endpoint, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should be a 400 error
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}

			// Should have request ID header
			requestID := w.Header().Get("X-Request-ID")
			if requestID == "" {
				t.Error("Expected X-Request-ID header")
			}

			// Parse response
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			// Should have all required fields
			requiredFields := []string{"error", "type", "request_id", "details", "source"}
			for _, field := range requiredFields {
				if response[field] == nil {
					t.Errorf("Expected '%s' field in response", field)
				}
			}

			// Should be a client error
			if response["source"] != "client_error" {
				t.Errorf("Expected source to be 'client_error', got '%s'", response["source"])
			}

			// Should have validation error type
			if response["type"] != "validation_error" {
				t.Errorf("Expected type to be 'validation_error', got '%s'", response["type"])
			}
		})
	}
}

func TestRequestIDGeneration(t *testing.T) {
	// Test that request IDs are unique
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id := generateRequestID()

		// Should not be empty
		if id == "" {
			t.Error("Request ID should not be empty")
		}

		// Should be unique
		if ids[id] {
			t.Errorf("Request ID %s was generated twice", id)
		}
		ids[id] = true

		// Should be reasonable length (hex encoded, so 16 chars for 8 bytes)
		if len(id) != 16 {
			t.Errorf("Expected request ID length of 16, got %d for ID: %s", len(id), id)
		}
	}
}
