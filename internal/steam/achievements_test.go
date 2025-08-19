package steam

import (
	"testing"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/models"
)

func TestProcessAchievements(t *testing.T) {
	tests := []struct {
		name            string
		achievements    []SteamAchievement
		expectSurvivors map[string]bool
		expectKillers   map[string]bool
	}{
		{
			name:         "empty achievements",
			achievements: []SteamAchievement{},
			expectSurvivors: map[string]bool{
				"dwight": false,
				"meg":    false,
			},
			expectKillers: map[string]bool{
				"trapper": false,
				"wraith":  false,
			},
		},
		{
			name: "some unlocked achievements",
			achievements: []SteamAchievement{
				{APIName: "ACH_UNLOCK_DWIGHT_PERKS", Achieved: 1},   // dwight
				{APIName: "ACH_UNLOCK_CHUCKLES_PERKS", Achieved: 1}, // trapper
				{APIName: "ACH_UNLOCK_MEG_PERKS", Achieved: 0},      // meg (not unlocked)
			},
			expectSurvivors: map[string]bool{
				"dwight": true,
				"meg":    false,
			},
			expectKillers: map[string]bool{
				"trapper": true,
				"wraith":  false,
			},
		},
		{
			name: "unknown achievements ignored",
			achievements: []SteamAchievement{
				{APIName: "UNKNOWN_ACHIEVEMENT", Achieved: 1},
				{APIName: "ACH_UNLOCK_DWIGHT_PERKS", Achieved: 1}, // dwight
			},
			expectSurvivors: map[string]bool{
				"dwight": true,
				"meg":    false,
			},
			expectKillers: map[string]bool{
				"trapper": false,
				"wraith":  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessAchievements(tt.achievements)

			// Check that result is not nil
			if result == nil {
				t.Fatal("ProcessAchievements returned nil")
			}

			// Check survivor achievements
			for character, expected := range tt.expectSurvivors {
				if actual, exists := result.AdeptSurvivors[character]; !exists {
					t.Errorf("Expected survivor %s to exist in result", character)
				} else if actual != expected {
					t.Errorf("Survivor %s: expected %v, got %v", character, expected, actual)
				}
			}

			// Check killer achievements
			for character, expected := range tt.expectKillers {
				if actual, exists := result.AdeptKillers[character]; !exists {
					t.Errorf("Expected killer %s to exist in result", character)
				} else if actual != expected {
					t.Errorf("Killer %s: expected %v, got %v", character, expected, actual)
				}
			}

			// Check that LastUpdated is recent
			if time.Since(result.LastUpdated) > time.Minute {
				t.Errorf("LastUpdated timestamp is too old: %v", result.LastUpdated)
			}
		})
	}
}

func TestAdeptAchievementMapping(t *testing.T) {
	survivorCount := 0
	killerCount := 0

	for _, character := range AdeptAchievementMapping {
		switch character.Type {
		case "survivor":
			survivorCount++
		case "killer":
			killerCount++
		default:
			t.Errorf("Invalid character type: %s", character.Type)
		}
	}

	// Verify we have the correct counts for current DBD roster
	if survivorCount != 46 {
		t.Errorf("Expected 46 survivors, got %d", survivorCount)
	}
	// 40 killers total, but Demogorgon doesn't have an adept achievement
	if killerCount != 40 {
		t.Errorf("Expected 40 killers in mapping, got %d", killerCount)
	}

	// Check that specific known characters exist with correct types
	testCharacters := []struct {
		name     string
		charType string
	}{
		{"dwight", "survivor"},
		{"trapper", "killer"},
		{"onryo", "killer"},       // Verify Onryo classification fix
		{"animatronic", "killer"}, // Verify Animatronic classification fix
	}

	for _, tc := range testCharacters {
		found := false
		for _, character := range AdeptAchievementMapping {
			if character.Name == tc.name && character.Type == tc.charType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find %s as %s in achievements mapping", tc.name, tc.charType)
		}
	}
}

func TestPlayerStatsWithAchievements(t *testing.T) {
	// Test that the enhanced model works correctly
	stats := models.PlayerStats{
		SteamID:     "counteredspell",
		DisplayName: "TestPlayer",
		KillerPips:  100,
	}

	achievements := &models.AchievementData{
		AdeptSurvivors: map[string]bool{
			"dwight": true,
			"meg":    false,
		},
		AdeptKillers: map[string]bool{
			"trapper": true,
			"wraith":  false,
		},
		LastUpdated: time.Now(),
	}

	response := models.PlayerStatsWithAchievements{
		PlayerStats:  stats,
		Achievements: achievements,
		DataSources: models.DataSourceStatus{
			Stats: models.DataSourceInfo{
				Success:   true,
				Source:    "api",
				FetchedAt: time.Now(),
			},
			Achievements: models.DataSourceInfo{
				Success:   true,
				Source:    "cache",
				FetchedAt: time.Now(),
			},
		},
	}

	// Verify structure
	if response.SteamID != stats.SteamID {
		t.Errorf("Expected SteamID %s, got %s", stats.SteamID, response.SteamID)
	}
	if response.Achievements == nil {
		t.Error("Expected achievements to be present")
	}
	if !response.DataSources.Stats.Success {
		t.Error("Expected stats to be successful")
	}
	if !response.DataSources.Achievements.Success {
		t.Error("Expected achievements to be successful")
	}
}
