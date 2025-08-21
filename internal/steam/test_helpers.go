package steam

import (
	"os"
	"testing"
)

// requireSteamAPIKey skips tests only in short mode
func requireSteamAPIKey(t *testing.T) {
	t.Helper()
	
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	key := os.Getenv("STEAM_API_KEY")
	if key == "" {
		t.Logf("Running integration test without STEAM_API_KEY - will test error handling behavior")
	} else {
		t.Logf("Running integration test with STEAM_API_KEY - will test live API behavior")
	}
}
