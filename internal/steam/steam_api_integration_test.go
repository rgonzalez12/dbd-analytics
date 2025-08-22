package steam_test

import (
	"os"
	"testing"

	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

func TestLiveSteamAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live API tests in short mode")
	}

	key := os.Getenv("STEAM_API_KEY")
	if key == "" {
		t.Logf("Running live API test without STEAM_API_KEY - will test error handling behavior")
	} else {
		t.Logf("Running live API test with STEAM_API_KEY - will test actual Steam API")
	}

	client := steam.NewClient()

	t.Run("PlayerSummary", func(t *testing.T) {
		summary, err := client.GetPlayerSummary("example_user") // vanity URL
		if key == "" {
			// Without API key, we expect an error
			if err == nil {
				t.Fatal("Expected error when STEAM_API_KEY is not set, but got nil")
			}
			t.Logf("Correctly received error without API key: %v", err)
		} else {
			// With API key, we expect success
			if err != nil {
				t.Fatalf("Failed to fetch player summary: %v", err)
			}
			if summary.SteamID == "" {
				t.Error("Expected non-empty SteamID in summary")
			}
		}
	})

	t.Run("PlayerStats", func(t *testing.T) {
		stats, err := client.GetPlayerStats("example_user") // vanity URL
		if key == "" {
			// Without API key, we expect an error
			if err == nil {
				t.Fatal("Expected error when STEAM_API_KEY is not set, but got nil")
			}
			t.Logf("Correctly received error without API key: %v", err)
		} else {
			// With API key, we expect success
			if err != nil {
				t.Fatalf("Failed to fetch player stats: %v", err)
			}
			if stats.SteamID == "" {
				t.Error("Expected non-empty SteamID in player stats")
			}
		}
	})
}
