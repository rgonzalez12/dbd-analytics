package steam

import (
	"net/url"
	"os"
	"testing"
)

func TestDoRequestErrors(t *testing.T) {
	tests := []struct {
		name        string
		endpoint    string
		expectError bool
	}{
		{
			name:        "invalid url",
			endpoint:    "://invalid-url",
			expectError: true,
		},
		{
			name:        "404 endpoint",
			endpoint:    "https://httpbin.org/status/404",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test API key
			os.Setenv("STEAM_API_KEY", "test-key")
			defer os.Unsetenv("STEAM_API_KEY")

			client := NewClient()
			params := url.Values{
				"key": {"test-key"},
			}

			var result interface{}
			err := client.makeRequest(tt.endpoint, params, &result)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.name)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.name, err)
			}
		})
	}
}
