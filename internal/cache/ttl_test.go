package cache

import (
	"os"
	"testing"
	"time"
)

// TestTTLSourcePriority verifies that environment variables override default constants
func TestTTLSourcePriority(t *testing.T) {
	// Save original env values for cleanup
	originalPlayerStats := os.Getenv("CACHE_PLAYER_STATS_TTL")
	originalPlayerSummary := os.Getenv("CACHE_PLAYER_SUMMARY_TTL")
	originalSteamAPI := os.Getenv("CACHE_STEAM_API_TTL")
	originalDefault := os.Getenv("CACHE_DEFAULT_TTL")
	
	// Cleanup function
	defer func() {
		os.Setenv("CACHE_PLAYER_STATS_TTL", originalPlayerStats)
		os.Setenv("CACHE_PLAYER_SUMMARY_TTL", originalPlayerSummary)
		os.Setenv("CACHE_STEAM_API_TTL", originalSteamAPI)
		os.Setenv("CACHE_DEFAULT_TTL", originalDefault)
	}()

	t.Run("EnvVarsPresent_ShouldOverrideConstants", func(t *testing.T) {
		// Set environment variables
		os.Setenv("CACHE_PLAYER_STATS_TTL", "15m")
		os.Setenv("CACHE_PLAYER_SUMMARY_TTL", "30m")
		os.Setenv("CACHE_STEAM_API_TTL", "2m")
		os.Setenv("CACHE_DEFAULT_TTL", "1m")
		
		config := GetTTLFromEnv()
		
		// Verify env vars win over constants
		if config.PlayerStats != 15*time.Minute {
			t.Errorf("Expected PlayerStats TTL 15m from env, got %v", config.PlayerStats)
		}
		if config.PlayerSummary != 30*time.Minute {
			t.Errorf("Expected PlayerSummary TTL 30m from env, got %v", config.PlayerSummary)
		}
		if config.SteamAPI != 2*time.Minute {
			t.Errorf("Expected SteamAPI TTL 2m from env, got %v", config.SteamAPI)
		}
		if config.DefaultTTL != 1*time.Minute {
			t.Errorf("Expected DefaultTTL 1m from env, got %v", config.DefaultTTL)
		}
	})

	t.Run("EnvVarsMissing_ShouldUseConstants", func(t *testing.T) {
		// Clear environment variables
		os.Unsetenv("CACHE_PLAYER_STATS_TTL")
		os.Unsetenv("CACHE_PLAYER_SUMMARY_TTL")
		os.Unsetenv("CACHE_STEAM_API_TTL")
		os.Unsetenv("CACHE_DEFAULT_TTL")
		
		config := GetTTLFromEnv()
		
		// Verify constants are used as fallbacks
		if config.PlayerStats != 5*time.Minute {
			t.Errorf("Expected PlayerStats TTL 5m from constant, got %v", config.PlayerStats)
		}
		if config.PlayerSummary != 10*time.Minute {
			t.Errorf("Expected PlayerSummary TTL 10m from constant, got %v", config.PlayerSummary)
		}
		if config.SteamAPI != 3*time.Minute {
			t.Errorf("Expected SteamAPI TTL 3m from constant, got %v", config.SteamAPI)
		}
		if config.DefaultTTL != 3*time.Minute {
			t.Errorf("Expected DefaultTTL 3m from constant, got %v", config.DefaultTTL)
		}
	})

	t.Run("InvalidEnvVars_ShouldFallbackToConstants", func(t *testing.T) {
		// Set invalid environment variables
		os.Setenv("CACHE_PLAYER_STATS_TTL", "invalid")
		os.Setenv("CACHE_PLAYER_SUMMARY_TTL", "not-a-duration")
		os.Setenv("CACHE_STEAM_API_TTL", "")
		os.Setenv("CACHE_DEFAULT_TTL", "also-invalid")
		
		config := GetTTLFromEnv()
		
		// Verify fallback to constants when env vars are invalid
		if config.PlayerStats != 5*time.Minute {
			t.Errorf("Expected PlayerStats TTL 5m from fallback, got %v", config.PlayerStats)
		}
		if config.PlayerSummary != 10*time.Minute {
			t.Errorf("Expected PlayerSummary TTL 10m from fallback, got %v", config.PlayerSummary)
		}
		if config.SteamAPI != 3*time.Minute {
			t.Errorf("Expected SteamAPI TTL 3m from fallback, got %v", config.SteamAPI)
		}
		if config.DefaultTTL != 3*time.Minute {
			t.Errorf("Expected DefaultTTL 3m from fallback, got %v", config.DefaultTTL)
		}
	})

	t.Run("MixedEnvVars_ShouldHandlePartialOverrides", func(t *testing.T) {
		// Set only some environment variables
		os.Setenv("CACHE_PLAYER_STATS_TTL", "7m")
		os.Unsetenv("CACHE_PLAYER_SUMMARY_TTL")
		os.Setenv("CACHE_STEAM_API_TTL", "90s")
		os.Unsetenv("CACHE_DEFAULT_TTL")
		
		config := GetTTLFromEnv()
		
		// Verify mixed behavior: env vars override where present, constants used elsewhere
		if config.PlayerStats != 7*time.Minute {
			t.Errorf("Expected PlayerStats TTL 7m from env, got %v", config.PlayerStats)
		}
		if config.PlayerSummary != 10*time.Minute {
			t.Errorf("Expected PlayerSummary TTL 10m from constant, got %v", config.PlayerSummary)
		}
		if config.SteamAPI != 90*time.Second {
			t.Errorf("Expected SteamAPI TTL 90s from env, got %v", config.SteamAPI)
		}
		if config.DefaultTTL != 3*time.Minute {
			t.Errorf("Expected DefaultTTL 3m from constant, got %v", config.DefaultTTL)
		}
	})
}

// TestTTLConstantsBackwardCompatibility verifies deprecated constants still exist
func TestTTLConstantsBackwardCompatibility(t *testing.T) {
	// Verify deprecated constants are still available for backward compatibility
	if PlayerStatsTTL != 5*time.Minute {
		t.Errorf("PlayerStatsTTL constant changed: expected 5m, got %v", PlayerStatsTTL)
	}
	if PlayerSummaryTTL != 10*time.Minute {
		t.Errorf("PlayerSummaryTTL constant changed: expected 10m, got %v", PlayerSummaryTTL)
	}
	if SteamAPITTL != 3*time.Minute {
		t.Errorf("SteamAPITTL constant changed: expected 3m, got %v", SteamAPITTL)
	}
	if DefaultTTL != 3*time.Minute {
		t.Errorf("DefaultTTL constant changed: expected 3m, got %v", DefaultTTL)
	}
}

// TestGetEnvDuration tests the core env parsing function
func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		fallback time.Duration
		expected time.Duration
	}{
		{
			name:     "ValidEnvVar",
			envKey:   "TEST_DURATION",
			envValue: "5m",
			fallback: 1*time.Minute,
			expected: 5*time.Minute,
		},
		{
			name:     "InvalidEnvVar",
			envKey:   "TEST_DURATION",
			envValue: "invalid",
			fallback: 2*time.Minute,
			expected: 2*time.Minute,
		},
		{
			name:     "EmptyEnvVar",
			envKey:   "TEST_DURATION",
			envValue: "",
			fallback: 3*time.Minute,
			expected: 3*time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			original := os.Getenv(tt.envKey)
			defer os.Setenv(tt.envKey, original)

			// Set test value
			if tt.envValue == "" {
				os.Unsetenv(tt.envKey)
			} else {
				os.Setenv(tt.envKey, tt.envValue)
			}

			result := getEnvDuration(tt.envKey, tt.fallback)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
