package cache

import (
	"os"
	"testing"
	"time"
)

func TestConfigurableTTL(t *testing.T) {
	// Test environment variable configuration
	originalEnv := map[string]string{
		"CACHE_PLAYER_STATS_TTL":   os.Getenv("CACHE_PLAYER_STATS_TTL"),
		"CACHE_PLAYER_SUMMARY_TTL": os.Getenv("CACHE_PLAYER_SUMMARY_TTL"),
		"CACHE_STEAM_API_TTL":      os.Getenv("CACHE_STEAM_API_TTL"),
		"CACHE_DEFAULT_TTL":        os.Getenv("CACHE_DEFAULT_TTL"),
	}

	// Clean up after test
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Test with custom environment variables
	os.Setenv("CACHE_PLAYER_STATS_TTL", "2m")
	os.Setenv("CACHE_PLAYER_SUMMARY_TTL", "15m")
	os.Setenv("CACHE_STEAM_API_TTL", "90s")
	os.Setenv("CACHE_DEFAULT_TTL", "4m")

	ttlConfig := GetTTLFromEnv()

	expectedPlayerStats := 2 * time.Minute
	expectedPlayerSummary := 15 * time.Minute
	expectedSteamAPI := 90 * time.Second
	expectedDefault := 4 * time.Minute

	if ttlConfig.PlayerStats != expectedPlayerStats {
		t.Errorf("Expected PlayerStats TTL %v, got %v", expectedPlayerStats, ttlConfig.PlayerStats)
	}

	if ttlConfig.PlayerSummary != expectedPlayerSummary {
		t.Errorf("Expected PlayerSummary TTL %v, got %v", expectedPlayerSummary, ttlConfig.PlayerSummary)
	}

	if ttlConfig.SteamAPI != expectedSteamAPI {
		t.Errorf("Expected SteamAPI TTL %v, got %v", expectedSteamAPI, ttlConfig.SteamAPI)
	}

	if ttlConfig.DefaultTTL != expectedDefault {
		t.Errorf("Expected Default TTL %v, got %v", expectedDefault, ttlConfig.DefaultTTL)
	}
}

func TestTTLConfiguration(t *testing.T) {
	// Test environment variable configuration
	originalEnv := map[string]string{
		"CACHE_PLAYER_STATS_TTL":   os.Getenv("CACHE_PLAYER_STATS_TTL"),
		"CACHE_PLAYER_SUMMARY_TTL": os.Getenv("CACHE_PLAYER_SUMMARY_TTL"),
		"CACHE_STEAM_API_TTL":      os.Getenv("CACHE_STEAM_API_TTL"),
		"CACHE_DEFAULT_TTL":        os.Getenv("CACHE_DEFAULT_TTL"),
	}

	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	t.Run("CustomEnvironmentVariables", func(t *testing.T) {
		os.Setenv("CACHE_PLAYER_STATS_TTL", "2m")
		os.Setenv("CACHE_PLAYER_SUMMARY_TTL", "15m")
		os.Setenv("CACHE_STEAM_API_TTL", "90s")
		os.Setenv("CACHE_DEFAULT_TTL", "4m")

		ttlConfig := GetTTLFromEnv()

		if ttlConfig.PlayerStats != 2*time.Minute {
			t.Errorf("Expected PlayerStats TTL 2m, got %v", ttlConfig.PlayerStats)
		}
		if ttlConfig.PlayerSummary != 15*time.Minute {
			t.Errorf("Expected PlayerSummary TTL 15m, got %v", ttlConfig.PlayerSummary)
		}
		if ttlConfig.SteamAPI != 90*time.Second {
			t.Errorf("Expected SteamAPI TTL 90s, got %v", ttlConfig.SteamAPI)
		}
		if ttlConfig.DefaultTTL != 4*time.Minute {
			t.Errorf("Expected Default TTL 4m, got %v", ttlConfig.DefaultTTL)
		}
	})

	t.Run("InvalidEnvironmentVariablesFallback", func(t *testing.T) {
		os.Setenv("CACHE_PLAYER_STATS_TTL", "invalid-duration")
		os.Setenv("CACHE_DEFAULT_TTL", "not-a-duration")

		ttlConfig := GetTTLFromEnv()

		if ttlConfig.PlayerStats != 5*time.Minute {
			t.Errorf("Expected fallback PlayerStats TTL 5m, got %v", ttlConfig.PlayerStats)
		}
		if ttlConfig.DefaultTTL != 3*time.Minute {
			t.Errorf("Expected fallback Default TTL 3m, got %v", ttlConfig.DefaultTTL)
		}
	})

	t.Run("ConfigFactoryMethods", func(t *testing.T) {
		// Test default config
		defaultConfig := DefaultConfig()
		if defaultConfig.TTL.PlayerStats == 0 {
			t.Error("Expected PlayerStats TTL to be set in default config")
		}
		if defaultConfig.Memory.DefaultTTL != defaultConfig.TTL.DefaultTTL {
			t.Error("Expected Memory.DefaultTTL to match TTL.DefaultTTL")
		}

		// Test player stats config
		playerConfig := PlayerStatsConfig()
		if playerConfig.Memory.DefaultTTL != playerConfig.TTL.PlayerStats {
			t.Error("Expected Memory.DefaultTTL to match TTL.PlayerStats")
		}

		// Test development config
		devConfig := DevelopmentConfig()
		if devConfig.TTL.PlayerStats != 30*time.Second {
			t.Errorf("Expected development TTL 30s, got %v", devConfig.TTL.PlayerStats)
		}
	})
}
