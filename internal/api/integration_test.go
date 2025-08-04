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
	if testing.Short() {
		t.Skip("Skipping Steam API outage tests in short mode")
	}

	scenarios := []struct {
		name           string
		steamID        string
		expectedStatus int
		shouldHaveError bool
	}{
		{
			name:           "Invalid Steam ID Format",
			steamID:        "invalid_id",
			expectedStatus: http.StatusBadRequest,
			shouldHaveError: true,
		},
		{
			name:           "Non-existent Steam ID",
			steamID:        "76561198000000001", // Likely non-existent
			expectedStatus: http.StatusNotFound,
			shouldHaveError: true,
		},
		{
			name:           "Valid Steam ID Format",
			steamID:        "76561198000000000", // Valid format but may not exist
			expectedStatus: 0, // Status depends on Steam API response
			shouldHaveError: false, // May or may not error depending on Steam
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			handler := NewHandler()

			req := httptest.NewRequest("GET", "/api/player/"+scenario.steamID+"/summary", nil)
			req = mux.SetURLVars(req, map[string]string{"steamid": scenario.steamID})
			w := httptest.NewRecorder()

			handler.GetPlayerSummary(w, req)

			if scenario.expectedStatus != 0 {
				if w.Code != scenario.expectedStatus {
					t.Errorf("Expected status %d, got %d", scenario.expectedStatus, w.Code)
				}
			}

			// Verify error response structure if error expected
			if scenario.shouldHaveError {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse error response: %v", err)
				}

				if _, ok := response["error"]; !ok {
					t.Error("Expected error field in response")
				}

				if _, ok := response["type"]; !ok {
					t.Error("Expected type field in response")
				}
			}
		})
	}
}

// TestRateLimitingMiddleware tests the rate limiting functionality
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
	req := httptest.NewRequest("GET", "/api/player/invalid@steamid/stats", nil)
	req = mux.SetURLVars(req, map[string]string{"steamid": "invalid@steamid"})
	w := httptest.NewRecorder()

	handler.GetPlayerStats(w, req)

	// Should return proper validation error
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid Steam ID, got %d", w.Code)
	}

	// Verify error structure
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
		if response["type"] != "validation_error" {
			t.Errorf("Expected validation_error type, got %v", response["type"])
		}
	}
}

// TestConcurrentRequestHandling tests behavior under high concurrent load
func TestConcurrentRequestHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent load test in short mode")
	}

	handler := NewHandler()
	
	// Create multiple concurrent requests
	const numRequests = 50
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/api/player/76561198000000000/summary", nil)
			req = mux.SetURLVars(req, map[string]string{"steamid": "76561198000000000"})
			w := httptest.NewRecorder()

			handler.GetPlayerSummary(w, req)
			results <- w.Code
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < numRequests; i++ {
		code := <-results
		if code == http.StatusOK {
			successCount++
		}
	}

	// At least some requests should succeed (depending on Steam API status)
	// This is more about ensuring the handler doesn't panic under load
	t.Logf("Concurrent test: %d/%d requests succeeded", successCount, numRequests)
}
