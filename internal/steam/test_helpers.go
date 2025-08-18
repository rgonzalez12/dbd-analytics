package steam

import (
	"os"
	"testing"
)

// requireSteamAPIKey only skips tests in short mode
// When not in short mode, integration tests run regardless of API key availability
// This allows testing both success cases (with key) and error handling (without key)
func requireSteamAPIKey(t *testing.T) {
	t.Helper()
	
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	// Let integration tests run even without API key to test error handling
	key := os.Getenv("STEAM_API_KEY")
	if key == "" {
		t.Logf("Running integration test without STEAM_API_KEY - will test error handling behavior")
	} else {
		t.Logf("Running integration test with STEAM_API_KEY - will test live API behavior")
	}
}

// hasSteamAPIKey returns true if STEAM_API_KEY environment variable is set
func hasSteamAPIKey() bool {
	return os.Getenv("STEAM_API_KEY") != ""
}
