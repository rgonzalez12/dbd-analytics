package steam

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestEnhancedRetryLogic(t *testing.T) {
	// Save and restore original environment variable to avoid test pollution
	originalMaxRetries := os.Getenv("STEAM_MAX_RETRIES")
	defer func() {
		if originalMaxRetries != "" {
			os.Setenv("STEAM_MAX_RETRIES", originalMaxRetries)
		} else {
			os.Unsetenv("STEAM_MAX_RETRIES")
		}
	}()

	tests := []struct {
		name                  string
		maxRetries            string   // STEAM_MAX_RETRIES environment variable value
		httpResponses         []int    // HTTP status codes to return in sequence
		retryAfterHeaders     []string // Retry-After header values for each response
		rateLimitResetHeaders []string // X-RateLimit-Reset header values for each response
		expectRetries         int      // Expected number of retry attempts
		expectSuccess         bool     // Whether the final request should succeed
	}{
		{
			name:              "429 with Retry-After header should respect timing",
			maxRetries:        "3",
			httpResponses:     []int{429, 200},
			retryAfterHeaders: []string{"1", ""},
			expectRetries:     1,
			expectSuccess:     true,
		},
		{
			name:              "429 without headers should use exponential backoff",
			maxRetries:        "3",
			httpResponses:     []int{429, 200},
			retryAfterHeaders: []string{"", ""},
			expectRetries:     1,
			expectSuccess:     true,
		},
		{
			name:          "500 server error should retry",
			maxRetries:    "3",
			httpResponses: []int{500, 200},
			expectRetries: 1,
			expectSuccess: true,
		},
		{
			name:          "403 Forbidden should NOT retry",
			maxRetries:    "3",
			httpResponses: []int{403},
			expectRetries: 0,
			expectSuccess: false,
		},
		{
			name:          "404 Not Found should NOT retry",
			maxRetries:    "3",
			httpResponses: []int{404},
			expectRetries: 0,
			expectSuccess: false,
		},
		{
			name:          "Respect configured retry limit",
			maxRetries:    "2",
			httpResponses: []int{429, 429, 429},
			expectRetries: 2,
			expectSuccess: false,
		},
		{
			name:                  "X-RateLimit-Reset header should be used",
			maxRetries:            "3",
			httpResponses:         []int{429, 200},
			rateLimitResetHeaders: []string{strconv.FormatInt(time.Now().Add(1*time.Second).Unix(), 10), ""},
			expectRetries:         1,
			expectSuccess:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Configure environment variable for this test
			os.Setenv("STEAM_MAX_RETRIES", tt.maxRetries)

			// Track the number of HTTP requests made
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer func() { requestCount++ }()

				if requestCount < len(tt.httpResponses) {
					status := tt.httpResponses[requestCount]

					// Set rate limiting headers if provided for this request
					if requestCount < len(tt.retryAfterHeaders) && tt.retryAfterHeaders[requestCount] != "" {
						w.Header().Set("Retry-After", tt.retryAfterHeaders[requestCount])
					}
					if requestCount < len(tt.rateLimitResetHeaders) && tt.rateLimitResetHeaders[requestCount] != "" {
						w.Header().Set("X-RateLimit-Reset", tt.rateLimitResetHeaders[requestCount])
					}

					w.WriteHeader(status)
					if status == 200 {
						w.Write([]byte(`{"response":{"players":[{"steamid":"123","personaname":"Test"}]}}`))
					}
				} else {
					// Return success if we've exhausted the configured response sequence
					w.WriteHeader(200)
					w.Write([]byte(`{"response":{"players":[{"steamid":"123","personaname":"Test"}]}}`))
				}
			}))
			defer server.Close()

			// Create Steam client with test-optimized retry configuration
			client := &Client{
				apiKey: "test_key",
				client: &http.Client{},
				retryConfig: RetryConfig{
					MaxAttempts: func() int {
						if tt.maxRetries != "" {
							if val, err := strconv.Atoi(tt.maxRetries); err == nil && val >= 0 {
								return val
							}
						}
						return 3 // default
					}(),
					BaseDelay:  50 * time.Millisecond, // Reduced for faster test execution
					MaxDelay:   1 * time.Second,       // Reduced for faster test execution
					Multiplier: 2.0,
					Jitter:     false, // Disabled for predictable test timing
				},
			}

			start := time.Now()

			// Execute request with retry logic through makeRequest method
			endpoint := fmt.Sprintf("%s/ISteamUser/GetPlayerSummaries/v0002/", server.URL)
			params := url.Values{
				"key":      {"test_key"},
				"steamids": {"76561198000000000"},
			}
			var response playerSummaryResponse
			err := client.makeRequest(endpoint, params, &response)

			duration := time.Since(start)

			// Verify the number of retry attempts matches expectations
			actualRetries := requestCount - 1 // Subtract the initial request
			if actualRetries != tt.expectRetries {
				t.Errorf("Expected %d retries, got %d", tt.expectRetries, actualRetries)
			}

			// Verify success or failure matches expectations
			if tt.expectSuccess && err != nil {
				t.Errorf("Expected success, got error: %v", err)
			}
			if !tt.expectSuccess && err == nil {
				t.Errorf("Expected error, got success")
			}

			// Verify that retry delays actually occurred when retries were expected
			if tt.expectRetries > 0 && duration < 30*time.Millisecond {
				t.Errorf("Expected some delay for retries, but took only %v", duration)
			}

			t.Logf("Test completed: %d requests, %d retries, duration: %v", requestCount, actualRetries, duration)
		})
	}
}

func TestShouldRetryError(t *testing.T) {
	tests := []struct {
		name     string
		apiError *APIError
		expected bool
	}{
		{
			name:     "Nil error should not retry",
			apiError: nil,
			expected: false,
		},
		{
			name:     "429 Too Many Requests should retry",
			apiError: &APIError{Type: ErrorTypeRateLimit, StatusCode: 429},
			expected: true,
		},
		{
			name:     "500 Internal Server Error should retry",
			apiError: &APIError{Type: ErrorTypeAPIError, StatusCode: 500},
			expected: true,
		},
		{
			name:     "502 Bad Gateway should retry",
			apiError: &APIError{Type: ErrorTypeAPIError, StatusCode: 502},
			expected: true,
		},
		{
			name:     "403 Forbidden should NOT retry",
			apiError: &APIError{Type: ErrorTypeAPIError, StatusCode: 403},
			expected: false,
		},
		{
			name:     "404 Not Found should NOT retry",
			apiError: &APIError{Type: ErrorTypeAPIError, StatusCode: 404},
			expected: false,
		},
		{
			name:     "Validation error should NOT retry",
			apiError: &APIError{Type: ErrorTypeValidation, StatusCode: 400},
			expected: false,
		},
		{
			name:     "Not found error type should NOT retry",
			apiError: &APIError{Type: ErrorTypeNotFound, StatusCode: 404},
			expected: false,
		},
		{
			name:     "Network error with retryable flag should retry",
			apiError: &APIError{Type: ErrorTypeNetwork, Retryable: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldRetryError(tt.apiError)
			if result != tt.expected {
				t.Errorf("shouldRetryError() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestExponentialBackoff(t *testing.T) {
	config := RetryConfig{
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   2 * time.Second,
		Multiplier: 2.0,
		Jitter:     false, // Disabled for predictable test results
	}

	tests := []struct {
		attempt     int
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{0, 100 * time.Millisecond, 100 * time.Millisecond},   // Base delay: 100ms
		{1, 200 * time.Millisecond, 200 * time.Millisecond},   // 100ms * 2^1 = 200ms
		{2, 400 * time.Millisecond, 400 * time.Millisecond},   // 100ms * 2^2 = 400ms
		{3, 800 * time.Millisecond, 800 * time.Millisecond},   // 100ms * 2^3 = 800ms
		{4, 1600 * time.Millisecond, 1600 * time.Millisecond}, // 100ms * 2^4 = 1.6s
		{5, 2 * time.Second, 2 * time.Second},                 // Capped at MaxDelay
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			delay := calculateBackoffDelay(tt.attempt, config)

			if delay < tt.expectedMin || delay > tt.expectedMax {
				t.Errorf("Attempt %d: expected delay between %v and %v, got %v",
					tt.attempt, tt.expectedMin, tt.expectedMax, delay)
			}
		})
	}
}

func TestRateLimitHeaderParsing(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name           string
		retryAfter     string
		rateLimitReset string
		expected       int
	}{
		{
			name:       "Valid Retry-After header",
			retryAfter: "120",
			expected:   120,
		},
		{
			name:       "Retry-After capped at maximum",
			retryAfter: "600", // 10 minutes
			expected:   300,   // Capped at 5 minutes maximum
		},
		{
			name:           "Valid X-RateLimit-Reset header",
			rateLimitReset: strconv.FormatInt(time.Now().Add(90*time.Second).Unix(), 10),
			expected:       90, // Approximately 90 seconds from now
		},
		{
			name:           "X-RateLimit-Reset in the past",
			rateLimitReset: strconv.FormatInt(time.Now().Add(-30*time.Second).Unix(), 10),
			expected:       60, // Falls back to default
		},
		{
			name:     "No headers provided",
			expected: 60, // Uses default fallback value
		},
		{
			name:       "Invalid Retry-After header",
			retryAfter: "invalid",
			expected:   60, // Falls back to default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := make(http.Header)
			if tt.retryAfter != "" {
				headers.Set("Retry-After", tt.retryAfter)
			}
			if tt.rateLimitReset != "" {
				headers.Set("X-RateLimit-Reset", tt.rateLimitReset)
			}

			result := client.parseRateLimitHeaders(headers)

			// Allow small tolerance for X-RateLimit-Reset timing calculations
			tolerance := 2
			if abs(result-tt.expected) > tolerance {
				t.Errorf("parseRateLimitHeaders() = %d, expected %d (±%d)", result, tt.expected, tolerance)
			}
		})
	}
}

func TestEnvironmentVariableConfiguration(t *testing.T) {
	// Save and restore original environment variable to avoid test pollution
	originalMaxRetries := os.Getenv("STEAM_MAX_RETRIES")
	defer func() {
		if originalMaxRetries != "" {
			os.Setenv("STEAM_MAX_RETRIES", originalMaxRetries)
		} else {
			os.Unsetenv("STEAM_MAX_RETRIES")
		}
	}()

	tests := []struct {
		name        string
		envValue    string
		expectedMax int
	}{
		{
			name:        "Default when no environment variable",
			envValue:    "",
			expectedMax: 3,
		},
		{
			name:        "Valid environment variable",
			envValue:    "5",
			expectedMax: 5,
		},
		{
			name:        "Zero retries",
			envValue:    "0",
			expectedMax: 0,
		},
		{
			name:        "Invalid environment variable falls back to default",
			envValue:    "invalid",
			expectedMax: 3,
		},
		{
			name:        "Negative value falls back to default",
			envValue:    "-1",
			expectedMax: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("STEAM_MAX_RETRIES", tt.envValue)
			} else {
				os.Unsetenv("STEAM_MAX_RETRIES")
			}

			config := DefaultRetryConfig()

			if config.MaxAttempts != tt.expectedMax {
				t.Errorf("Expected MaxAttempts = %d, got %d", tt.expectedMax, config.MaxAttempts)
			}
		})
	}
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
