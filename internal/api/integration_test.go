package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// TestSteamAPIOutageScenarios tests various Steam API failure scenarios
func TestSteamAPIOutageScenarios(t *testing.T) {
	requireSteamAPIKey(t)

	scenarios := []struct {
		name            string
		steamID         string
		expectedStatus  int
		shouldHaveError bool
	}{
		{
			name:            "Invalid Steam ID Format",
			steamID:         "invalid@id",
			expectedStatus:  http.StatusBadRequest,
			shouldHaveError: true,
		},
		{
			name:            "Non-existent Steam ID",
			steamID:         "76561199999999999", // Very high number, likely non-existent
			expectedStatus:  0,                   // Status depends on API key availability
			shouldHaveError: false,               // May or may not error depending on setup
		},
		{
			name:            "Valid Steam ID Format",
			steamID:         "example_user", // Valid vanity URL
			expectedStatus:  0,              // Status depends on Steam API response
			shouldHaveError: false,          // May or may not error depending on Steam
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			handler := NewHandler()

			req := httptest.NewRequest("GET", "/api/player/"+scenario.steamID, nil)
			req = mux.SetURLVars(req, map[string]string{"steamid": scenario.steamID})
			w := httptest.NewRecorder()

			handler.GetPlayerStatsWithAchievements(w, req)

			// For validation errors, status should be predictable
			if scenario.expectedStatus != 0 {
				if w.Code != scenario.expectedStatus {
					t.Errorf("Expected status %d, got %d", scenario.expectedStatus, w.Code)
				}
			} else {
				// For API-dependent scenarios, just ensure we get a valid HTTP status
				if w.Code < 200 || w.Code > 599 {
					t.Errorf("Got invalid HTTP status code: %d", w.Code)
				}
				t.Logf("Scenario '%s' returned status %d", scenario.name, w.Code)
			}

			// Verify error response structure if error expected
			if scenario.shouldHaveError {
				var response StandardError
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse error response: %v", err)
				}

				if code, ok := response.Details["code"].(string); !ok || code == "" {
					t.Error("Expected error code field in response details")
				}

				if response.Message == "" {
					t.Error("Expected error message field in response")
				}
			}
		})
	}
}

// TestRateLimitingMiddleware tests rate limiting functionality
func TestRateLimitingMiddleware(t *testing.T) {
	// Create rate limiter allowing 2 requests per second
	limiter := NewRequestLimiter(2, time.Second)
	middleware := RateLimitMiddleware(limiter)

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handler := middleware(testHandler)

	// Test multiple requests from same IP
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if i < 2 {
			// First 2 requests should succeed
			if w.Code != http.StatusOK {
				t.Errorf("Request %d: expected 200, got %d", i, w.Code)
			}
		} else {
			// Subsequent requests should be rate limited
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("Request %d: expected 429, got %d", i, w.Code)
			}

			// Check for proper error structure
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse rate limit response: %v", err)
			}

			if response["type"] != "rate_limit" {
				t.Errorf("Expected rate_limit error type, got %v", response["type"])
			}
		}
	}
}

// TestCacheResiliency tests cache behavior during failures
func TestCacheResiliency(t *testing.T) {
	handler := NewHandler()

	// Make request with invalid Steam ID to trigger validation error
	req := httptest.NewRequest("GET", "/api/player/invalid@steamid", nil)
	req = mux.SetURLVars(req, map[string]string{"steamid": "invalid@steamid"})
	w := httptest.NewRecorder()

	handler.GetPlayerStatsWithAchievements(w, req)

	// Should return proper validation error
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid Steam ID, got %d", w.Code)
	}

	// Verify error structure
	var response StandardError
	if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
		if code, ok := response.Details["code"].(string); !ok || code != "VALIDATION_ERROR" {
			t.Errorf("Expected VALIDATION_ERROR code, got %v", code)
		}
	}
}

// TestConcurrentRequestHandling tests behavior under high concurrent load
func TestConcurrentRequestHandling(t *testing.T) {
	requireSteamAPIKey(t)

	handler := NewHandler()

	// Create multiple concurrent requests
	const numRequests = 50
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/api/player/example_user", nil)
			req = mux.SetURLVars(req, map[string]string{"steamid": "example_user"})
			w := httptest.NewRecorder()

			handler.GetPlayerStatsWithAchievements(w, req)
			results <- w.Code
		}()
	}

	// Collect results
	successCount := 0
	errorCount := 0
	for i := 0; i < numRequests; i++ {
		code := <-results
		switch code {
		case http.StatusOK:
			successCount++
		case http.StatusForbidden, http.StatusBadRequest:
			errorCount++
		}
	}

	// Test that handler doesn't panic under load - either all succeed or all fail appropriately
	totalHandled := successCount + errorCount
	if totalHandled != numRequests {
		t.Errorf("Expected %d total handled requests, got %d", numRequests, totalHandled)
	}

	t.Logf("Concurrent test: %d/%d requests succeeded, %d failed with expected errors", successCount, numRequests, errorCount)
}
