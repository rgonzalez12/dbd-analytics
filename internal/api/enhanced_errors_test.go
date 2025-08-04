package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

func TestEnhancedErrorResponses(t *testing.T) {
	tests := []struct {
		name             string
		errorType        steam.ErrorType
		statusCode       int
		expectedStatus   int
		expectRetryAfter bool
		expectDetails    bool
	}{
		{
			name:             "Rate Limit Error",
			errorType:        steam.ErrorTypeRateLimit,
			statusCode:       429,
			expectedStatus:   429,
			expectRetryAfter: true,
			expectDetails:    true,
		},
		{
			name:             "Not Found Error",
			errorType:        steam.ErrorTypeNotFound,
			statusCode:       404,
			expectedStatus:   404,
			expectRetryAfter: false,
			expectDetails:    true,
		},
		{
			name:             "API Error",
			errorType:        steam.ErrorTypeAPIError,
			statusCode:       502,
			expectedStatus:   502,
			expectRetryAfter: true,
			expectDetails:    true,
		},
		{
			name:             "Network Error",
			errorType:        steam.ErrorTypeNetwork,
			statusCode:       0, // No status code set
			expectedStatus:   502,
			expectRetryAfter: true,
			expectDetails:    true,
		},
		{
			name:             "Validation Error",
			errorType:        steam.ErrorTypeValidation,
			statusCode:       400,
			expectedStatus:   400,
			expectRetryAfter: false,
			expectDetails:    true,
		},
		{
			name:             "Internal Error",
			errorType:        steam.ErrorTypeInternal,
			statusCode:       0,
			expectedStatus:   500,
			expectRetryAfter: false,
			expectDetails:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock error
			apiErr := &steam.APIError{
				Type:       tt.errorType,
				Message:    "Test error message",
				StatusCode: tt.statusCode,
				Retryable:  tt.expectRetryAfter,
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Create a handler instance (we'll use writeErrorResponse directly)
			handler := NewHandler()
			handler.writeErrorResponseTest(w, apiErr)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Parse response body
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			// Check required fields
			if response["error"] == nil {
				t.Error("Expected 'error' field in response")
			}
			if response["type"] == nil {
				t.Error("Expected 'type' field in response")
			}

			// Check details field
			if tt.expectDetails {
				if response["details"] == nil {
					t.Error("Expected 'details' field in response")
				}
			}

			// Check retry_after field
			if tt.expectRetryAfter {
				if response["retry_after"] == nil {
					t.Error("Expected 'retry_after' field in response")
				}
				if response["retryable"] != true {
					t.Error("Expected 'retryable' to be true")
				}
			}

			t.Logf("Response: %+v", response)
		})
	}
}

// Helper method to test writeErrorResponse
func (h *Handler) writeErrorResponseTest(w http.ResponseWriter, apiErr *steam.APIError) {
	writeErrorResponse(w, apiErr)
}
