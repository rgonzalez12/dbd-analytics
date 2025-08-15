package steam

import (
	"os"
	"testing"
	"time"
)

func TestAchievementsTimeoutConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		expected    time.Duration
		description string
	}{
		{
			name:        "Default_timeout_when_no_env_var",
			envValue:    "",
			expected:    5 * time.Second,
			description: "Should use default 5s when ACHIEVEMENTS_TIMEOUT_SECS not set",
		},
		{
			name:        "Valid_environment_variable",
			envValue:    "10",
			expected:    10 * time.Second,
			description: "Should use 10s when ACHIEVEMENTS_TIMEOUT_SECS=10",
		},
		{
			name:        "Invalid_environment_variable",
			envValue:    "invalid",
			expected:    5 * time.Second,
			description: "Should fallback to default when ACHIEVEMENTS_TIMEOUT_SECS=invalid",
		},
		{
			name:        "Zero_timeout",
			envValue:    "0",
			expected:    5 * time.Second,
			description: "Should fallback to default when ACHIEVEMENTS_TIMEOUT_SECS=0",
		},
		{
			name:        "Negative_timeout",
			envValue:    "-5",
			expected:    5 * time.Second,
			description: "Should fallback to default when ACHIEVEMENTS_TIMEOUT_SECS=-5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			if tt.envValue != "" {
				os.Setenv("ACHIEVEMENTS_TIMEOUT_SECS", tt.envValue)
			} else {
				os.Unsetenv("ACHIEVEMENTS_TIMEOUT_SECS")
			}

			// Test the function
						actual := achievementTimeout()

			if actual != tt.expected {
				t.Errorf("achievementTimeout() = %v, expected %v. %s",
					actual, tt.expected, tt.description)
			}

			// Cleanup
			os.Unsetenv("ACHIEVEMENTS_TIMEOUT_SECS")
		})
	}
}

func TestClientTimeoutIntegration(t *testing.T) {
	// Test that NewClient creates client with correct timeout
	os.Setenv("ACHIEVEMENTS_TIMEOUT_SECS", "3")
	defer os.Unsetenv("ACHIEVEMENTS_TIMEOUT_SECS")

	client := NewClient()

	if client.client.Timeout != 3*time.Second {
		t.Errorf("NewClient() created client with timeout %v, expected %v",
			client.client.Timeout, 3*time.Second)
	}
}
