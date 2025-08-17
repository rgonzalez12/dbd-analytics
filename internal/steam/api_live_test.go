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
		t.Skip("Skipping live API tests: STEAM_API_KEY not set")
	}

	client := steam.NewClient()

	t.Run("PlayerSummary", func(t *testing.T) {
		summary, err := client.GetPlayerSummary("counteredspell") // vanity URL
		if err != nil {
			t.Fatalf("Failed to fetch player summary: %v", err)
		}

		if summary.SteamID == "" {
			t.Error("Expected non-empty SteamID in summary")
		}
	})

	t.Run("PlayerStats", func(t *testing.T) {
		stats, err := client.GetPlayerStats("counteredspell") // vanity URL
		if err != nil {
			t.Fatalf("Failed to fetch player stats: %v", err)
		}

		if stats.SteamID == "" {
			t.Error("Expected non-empty SteamID in player stats")
		}
	})
}
