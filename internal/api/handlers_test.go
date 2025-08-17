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

// MockSteamClient implements a mock for testing
type MockSteamClient struct {
	GetPlayerSummaryFunc func(steamIDOrVanity string) (*steam.SteamPlayer, *steam.APIError)
	GetPlayerStatsFunc   func(steamIDOrVanity string) (*steam.SteamPlayerstats, *steam.APIError)
}

func (m *MockSteamClient) GetPlayerSummary(steamIDOrVanity string) (*steam.SteamPlayer, *steam.APIError) {
	if m.GetPlayerSummaryFunc != nil {
		return m.GetPlayerSummaryFunc(steamIDOrVanity)
	}
	return nil, steam.NewInternalError(nil)
}

func (m *MockSteamClient) GetPlayerStats(steamIDOrVanity string) (*steam.SteamPlayerstats, *steam.APIError) {
	if m.GetPlayerStatsFunc != nil {
		return m.GetPlayerStatsFunc(steamIDOrVanity)
	}
	return nil, steam.NewInternalError(nil)
}

// SteamClient interface for dependency injection
type SteamClient interface {
	GetPlayerSummary(steamIDOrVanity string) (*steam.SteamPlayer, *steam.APIError)
	GetPlayerStats(steamIDOrVanity string) (*steam.SteamPlayerstats, *steam.APIError)
}

// TestHandler with injectable client for testing
type TestHandler struct {
	steamClient SteamClient
}

func NewTestHandler(client SteamClient) *TestHandler {
	return &TestHandler{steamClient: client}
}

// Copy of GetPlayerSummary method but with injectable client
func (h *TestHandler) GetPlayerSummary(w http.ResponseWriter, r *http.Request) {
	steamID := mux.Vars(r)["steamid"]

	// Validate Steam ID format before processing
	if err := validateSteamIDOrVanity(steamID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	summary, err := h.steamClient.GetPlayerSummary(steamID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, summary)
}

// Copy of GetPlayerStats method but with injectable client
func (h *TestHandler) GetPlayerStats(w http.ResponseWriter, r *http.Request) {
	steamID := mux.Vars(r)["steamid"]

	// Validate Steam ID format before processing
	if err := validateSteamIDOrVanity(steamID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	summary, err := h.steamClient.GetPlayerSummary(steamID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	rawStats, err := h.steamClient.GetPlayerStats(steamID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	playerStats := steam.MapSteamStats(rawStats.Stats, summary.SteamID, summary.PersonaName)
	writeJSONResponse(w, playerStats)
}

func TestGetPlayerSummary(t *testing.T) {
	// Initialize logging for tests
	log.Initialize()

	tests := []struct {
		name           string
		steamID        string
		mockResponse   *steam.SteamPlayer
		mockError      *steam.APIError
		expectedStatus int
		expectError    bool
	}{
		{
			name:    "Successful response with valid Steam ID",
			steamID: "counteredspell",
			mockResponse: &steam.SteamPlayer{
				SteamID:     "counteredspell",
				PersonaName: "TestPlayer",
				Avatar:      "https://avatars.steamstatic.com/test.jpg",
				AvatarFull:  "https://avatars.steamstatic.com/test_full.jpg",
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Invalid Steam ID format",
			steamID:        "invalid@steam#id",
			mockResponse:   nil,
			mockError:      nil, // Won't be called due to validation
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Steam API timeout/network error",
			steamID:        "counteredspell",
			mockResponse:   nil,
			mockError:      steam.NewNetworkError(nil),
			expectedStatus: http.StatusBadGateway,
			expectError:    true,
		},
		{
			name:           "Steam API rate limiting",
			steamID:        "counteredspell",
			mockResponse:   nil,
			mockError:      steam.NewRateLimitError(),
			expectedStatus: http.StatusTooManyRequests,
			expectError:    true,
		},
		{
			name:           "Steam API server error (500)",
			steamID:        "counteredspell",
			mockResponse:   nil,
			mockError:      steam.NewAPIError(500, "Internal Server Error"),
			expectedStatus: http.StatusBadGateway,
			expectError:    true,
		},
		{
			name:           "Player not found",
			steamID:        "counteredspell",
			mockResponse:   nil,
			mockError:      steam.NewNotFoundError("Player"),
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockSteamClient{
				GetPlayerSummaryFunc: func(steamIDOrVanity string) (*steam.SteamPlayer, *steam.APIError) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			// Create test handler with mock client
			handler := NewTestHandler(mockClient)

			// Create request
			req := httptest.NewRequest("GET", "/api/player/"+tt.steamID+"/summary", nil)
			req = mux.SetURLVars(req, map[string]string{"steamid": tt.steamID})
			w := httptest.NewRecorder()

			// Call handler
			handler.GetPlayerSummary(w, req)

			// Verify status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Verify response content
			if tt.expectError {
				// Should return error response
				var errorResp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
					t.Fatalf("Failed to parse error response: %v", err)
				}

				if errorResp["error"] == nil {
					t.Error("Expected error field in response")
				}

				if errorResp["type"] == nil {
					t.Error("Expected type field in error response")
				}

				if errorResp["request_id"] == nil {
					t.Error("Expected request_id field in error response")
				}
			} else {
				// Should return successful player summary
				var playerResp steam.SteamPlayer
				if err := json.Unmarshal(w.Body.Bytes(), &playerResp); err != nil {
					t.Fatalf("Failed to parse success response: %v", err)
				}

				if playerResp.SteamID != tt.mockResponse.SteamID {
					t.Errorf("Expected SteamID %s, got %s", tt.mockResponse.SteamID, playerResp.SteamID)
				}

				if playerResp.PersonaName != tt.mockResponse.PersonaName {
					t.Errorf("Expected PersonaName %s, got %s", tt.mockResponse.PersonaName, playerResp.PersonaName)
				}
			}
		})
	}
}

func TestGetPlayerStats(t *testing.T) {
	// Initialize logging for tests
	log.Initialize()

	tests := []struct {
		name             string
		steamID          string
		mockSummary      *steam.SteamPlayer
		mockSummaryError *steam.APIError
		mockStats        *steam.SteamPlayerstats
		mockStatsError   *steam.APIError
		expectedStatus   int
		expectError      bool
	}{
		{
			name:    "Successful stats response",
			steamID: "counteredspell",
			mockSummary: &steam.SteamPlayer{
				SteamID:     "counteredspell",
				PersonaName: "TestPlayer",
			},
			mockSummaryError: nil,
			mockStats: &steam.SteamPlayerstats{
				SteamID:  "counteredspell",
				GameName: "Dead by Daylight",
				Stats: []steam.SteamStat{
					{Name: "DBD_DiedAsKiller", Value: 10},
					{Name: "DBD_DiedAsSurvivor", Value: 20},
				},
			},
			mockStatsError: nil,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:             "Invalid Steam ID format",
			steamID:          "invalid@steam#id",
			mockSummary:      nil,
			mockSummaryError: nil, // Won't be called due to validation
			mockStats:        nil,
			mockStatsError:   nil,
			expectedStatus:   http.StatusBadRequest,
			expectError:      true,
		},
		{
			name:             "Player summary fails",
			steamID:          "counteredspell",
			mockSummary:      nil,
			mockSummaryError: steam.NewNotFoundError("Player"),
			mockStats:        nil,
			mockStatsError:   nil,
			expectedStatus:   http.StatusNotFound,
			expectError:      true,
		},
		{
			name:    "Stats fetch fails after successful summary",
			steamID: "counteredspell",
			mockSummary: &steam.SteamPlayer{
				SteamID:     "counteredspell",
				PersonaName: "TestPlayer",
			},
			mockSummaryError: nil,
			mockStats:        nil,
			mockStatsError:   steam.NewRateLimitError(),
			expectedStatus:   http.StatusTooManyRequests,
			expectError:      true,
		},
		{
			name:    "Stats API server error",
			steamID: "counteredspell",
			mockSummary: &steam.SteamPlayer{
				SteamID:     "counteredspell",
				PersonaName: "TestPlayer",
			},
			mockSummaryError: nil,
			mockStats:        nil,
			mockStatsError:   steam.NewAPIError(503, "Service Unavailable"),
			expectedStatus:   http.StatusBadGateway,
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockSteamClient{
				GetPlayerSummaryFunc: func(steamIDOrVanity string) (*steam.SteamPlayer, *steam.APIError) {
					if tt.mockSummaryError != nil {
						return nil, tt.mockSummaryError
					}
					return tt.mockSummary, nil
				},
				GetPlayerStatsFunc: func(steamIDOrVanity string) (*steam.SteamPlayerstats, *steam.APIError) {
					if tt.mockStatsError != nil {
						return nil, tt.mockStatsError
					}
					return tt.mockStats, nil
				},
			}

			// Create test handler with mock client
			handler := NewTestHandler(mockClient)

			// Create request
			req := httptest.NewRequest("GET", "/api/player/"+tt.steamID+"/stats", nil)
			req = mux.SetURLVars(req, map[string]string{"steamid": tt.steamID})
			w := httptest.NewRecorder()

			// Call handler
			handler.GetPlayerStats(w, req)

			// Verify status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Verify response content
			if tt.expectError {
				// Should return error response
				var errorResp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
					t.Fatalf("Failed to parse error response: %v", err)
				}

				if errorResp["error"] == nil {
					t.Error("Expected error field in response")
				}

				if errorResp["type"] == nil {
					t.Error("Expected type field in error response")
				}
			} else {
				// Should return successful stats response
				var statsResp steam.DBDPlayerStats
				if err := json.Unmarshal(w.Body.Bytes(), &statsResp); err != nil {
					t.Fatalf("Failed to parse success response: %v", err)
				}

				if statsResp.SteamID != tt.mockSummary.SteamID {
					t.Errorf("Expected SteamID %s, got %s", tt.mockSummary.SteamID, statsResp.SteamID)
				}

				if statsResp.DisplayName != tt.mockSummary.PersonaName {
					t.Errorf("Expected DisplayName %s, got %s", tt.mockSummary.PersonaName, statsResp.DisplayName)
				}
			}
		})
	}
}

func TestValidateSteamIDOrVanity(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorType   steam.ErrorType
	}{
		{
			name:        "Valid Steam ID",
			input:       "counteredspell",
			expectError: false,
		},
		{
			name:        "Valid vanity URL",
			input:       "testuser",
			expectError: false,
		},
		{
			name:        "Valid vanity URL with underscore",
			input:       "test_user",
			expectError: false,
		},
		{
			name:        "Valid vanity URL with hyphen",
			input:       "test-user",
			expectError: false,
		},
		{
			name:        "Empty input",
			input:       "",
			expectError: true,
			errorType:   steam.ErrorTypeValidation,
		},
		{
			name:        "Too short Steam ID",
			input:       "123456789",
			expectError: true,
			errorType:   steam.ErrorTypeValidation,
		},
		{
			name:        "Too long Steam ID",
			input:       "765611980000000001",
			expectError: true,
			errorType:   steam.ErrorTypeValidation,
		},
		{
			name:        "Steam ID wrong prefix",
			input:       "12345678901234567",
			expectError: true,
			errorType:   steam.ErrorTypeValidation,
		},
		{
			name:        "Steam ID with letters",
			input:       "7656119800000000a",
			expectError: true,
			errorType:   steam.ErrorTypeValidation,
		},
		{
			name:        "Vanity URL too short",
			input:       "ab",
			expectError: true,
			errorType:   steam.ErrorTypeValidation,
		},
		{
			name:        "Vanity URL with special characters",
			input:       "test@user",
			expectError: true,
			errorType:   steam.ErrorTypeValidation,
		},
		{
			name:        "Vanity URL with spaces",
			input:       "test user",
			expectError: true,
			errorType:   steam.ErrorTypeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSteamIDOrVanity(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}

				if err.Type != tt.errorType {
					t.Errorf("Expected error type %s, got %s", tt.errorType, err.Type)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %s", err.Message)
				}
			}
		})
	}
}

func TestErrorResponseFormatting(t *testing.T) {
	// Initialize logging for tests
	log.Initialize()

	tests := []struct {
		name           string
		inputError     *steam.APIError
		expectedStatus int
		checkFields    []string
	}{
		{
			name:           "Rate limit error",
			inputError:     steam.NewRateLimitError(),
			expectedStatus: http.StatusTooManyRequests,
			checkFields:    []string{"error", "type", "request_id", "details", "retry_after", "retryable"},
		},
		{
			name:           "Validation error",
			inputError:     steam.NewValidationError("Invalid input"),
			expectedStatus: http.StatusBadRequest,
			checkFields:    []string{"error", "type", "request_id", "details", "source"},
		},
		{
			name:           "Not found error",
			inputError:     steam.NewNotFoundError("Player"),
			expectedStatus: http.StatusNotFound,
			checkFields:    []string{"error", "type", "request_id", "details", "source"},
		},
		{
			name:           "API error",
			inputError:     steam.NewAPIError(502, "Bad Gateway"),
			expectedStatus: http.StatusBadGateway,
			checkFields:    []string{"error", "type", "request_id", "details", "source", "retry_after", "retryable"},
		},
		{
			name:           "Internal error",
			inputError:     steam.NewInternalError(nil),
			expectedStatus: http.StatusInternalServerError,
			checkFields:    []string{"error", "type", "request_id", "details", "source"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeErrorResponse(w, tt.inputError)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response format
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse error response: %v", err)
			}

			// Check required fields
			for _, field := range tt.checkFields {
				if response[field] == nil {
					t.Errorf("Expected field '%s' in response", field)
				}
			}

			// Check Content-Type header
			if w.Header().Get("Content-Type") != "application/json" {
				t.Error("Expected Content-Type: application/json")
			}

			// Check X-Request-ID header
			if w.Header().Get("X-Request-ID") == "" {
				t.Error("Expected X-Request-ID header")
			}
		})
	}
}

func TestRateLimitRetryAfterValue(t *testing.T) {
	// Initialize logging for tests
	log.Initialize()

	tests := []struct {
		name               string
		retryAfterValue    int
		expectedRetryAfter int
	}{
		{
			name:               "Custom retry after value",
			retryAfterValue:    120,
			expectedRetryAfter: 120,
		},
		{
			name:               "Default retry after when zero",
			retryAfterValue:    0,
			expectedRetryAfter: 60, // Default fallback
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			// Create rate limit error with specific retry after value
			var rateLimitError *steam.APIError
			if tt.retryAfterValue > 0 {
				rateLimitError = steam.NewRateLimitErrorWithRetryAfter(tt.retryAfterValue)
			} else {
				rateLimitError = steam.NewRateLimitError()
			}

			writeErrorResponse(w, rateLimitError)

			// Check status code
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("Expected status 429, got %d", w.Code)
			}

			// Parse response
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			// Check retry_after value
			retryAfter, ok := response["retry_after"]
			if !ok {
				t.Error("Expected 'retry_after' field in response")
			}

			// Convert to int for comparison
			retryAfterInt, ok := retryAfter.(float64) // JSON numbers are float64
			if !ok {
				t.Errorf("Expected retry_after to be a number, got %T", retryAfter)
			}

			if int(retryAfterInt) != tt.expectedRetryAfter {
				t.Errorf("Expected retry_after %d, got %d", tt.expectedRetryAfter, int(retryAfterInt))
			}

			t.Logf("Test '%s': retry_after correctly set to %d seconds", tt.name, int(retryAfterInt))
		})
	}
}
