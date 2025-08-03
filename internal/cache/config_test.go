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

func TestTTLConfigFallback(t *testing.T) {
	// Test with invalid environment variables
	originalEnv := os.Getenv("CACHE_PLAYER_STATS_TTL")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("CACHE_PLAYER_STATS_TTL")
		} else {
			os.Setenv("CACHE_PLAYER_STATS_TTL", originalEnv)
		}
	}()
	
	// Set invalid duration
	os.Setenv("CACHE_PLAYER_STATS_TTL", "invalid-duration")
	
	ttlConfig := GetTTLFromEnv()
	
	// Should fall back to default
	expectedDefault := 5 * time.Minute
	if ttlConfig.PlayerStats != expectedDefault {
		t.Errorf("Expected fallback PlayerStats TTL %v, got %v", expectedDefault, ttlConfig.PlayerStats)
	}
}

func TestDefaultConfigWithTTL(t *testing.T) {
	config := DefaultConfig()
	
	// Verify TTL config is included
	if config.TTL.PlayerStats == 0 {
		t.Error("Expected PlayerStats TTL to be set in default config")
	}
	
	if config.TTL.DefaultTTL == 0 {
		t.Error("Expected DefaultTTL to be set in default config")
	}
	
	// Verify memory config uses TTL
	if config.Memory.DefaultTTL != config.TTL.DefaultTTL {
		t.Errorf("Expected Memory.DefaultTTL %v to match TTL.DefaultTTL %v", 
			config.Memory.DefaultTTL, config.TTL.DefaultTTL)
	}
}

func TestPlayerStatsConfigTTL(t *testing.T) {
	config := PlayerStatsConfig()
	
	// Verify player stats config uses configurable TTL
	if config.Memory.DefaultTTL != config.TTL.PlayerStats {
		t.Errorf("Expected Memory.DefaultTTL %v to match TTL.PlayerStats %v", 
			config.Memory.DefaultTTL, config.TTL.PlayerStats)
	}
}

func TestDevelopmentConfigTTL(t *testing.T) {
	config := DevelopmentConfig()
	
	// Verify development config has short TTLs
	expectedTTL := 30 * time.Second
	if config.TTL.PlayerStats != expectedTTL {
		t.Errorf("Expected development PlayerStats TTL %v, got %v", 
			expectedTTL, config.TTL.PlayerStats)
	}
	
	if config.TTL.DefaultTTL != expectedTTL {
		t.Errorf("Expected development DefaultTTL %v, got %v", 
			expectedTTL, config.TTL.DefaultTTL)
	}
}
