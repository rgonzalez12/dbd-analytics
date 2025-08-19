package steam

import (
	"testing"
)

func TestAchievementMapper(t *testing.T) {
	mapper := NewAchievementMapper()

	t.Run("MappingCoverage", func(t *testing.T) {
		coverage := mapper.ValidateMappingCoverage()

		survivorCount := coverage["survivor_adepts_mapped"].(int)
		killerCount := coverage["killer_adepts_mapped"].(int)
		totalCount := coverage["total_characters"].(int)

		// Validate expected counts based on current DBD character roster
		if survivorCount != 46 {
			t.Errorf("Expected 46 survivors mapped, got %d", survivorCount)
		}

		// 40 killers total in DBD, all should be mappable
		if killerCount != 40 {
			t.Errorf("Expected 40 killers mapped, got %d", killerCount)
		}

		if totalCount != 86 {
			t.Errorf("Expected 86 total characters, got %d", totalCount)
		}

		if totalCount != survivorCount+killerCount {
			t.Errorf("Total count mismatch: %d != %d + %d", totalCount, survivorCount, killerCount)
		}

		t.Logf("Achievement mapping: %d survivors, %d killers, %d total", survivorCount, killerCount, totalCount)
	})

	t.Run("UnknownAchievementHandling", func(t *testing.T) {
		unknownAchievements := &PlayerAchievements{
			Achievements: []SteamAchievement{
				{APIName: "ACH_UNLOCK_DWIGHT_PERKS", Achieved: 1, UnlockTime: 1234567890}, // Known adept achievement
				{APIName: "ACH_UNLOCK_MEG_PERKS", Achieved: 0, UnlockTime: 0},             // Known adept achievement
				{APIName: "UNKNOWN_ACHIEVEMENT_123", Achieved: 1, UnlockTime: 1234567890}, // Unknown achievement
			},
		}

		// When schema is unavailable (test environment), we fall back to processing only adept achievements
		mapped := mapper.MapPlayerAchievements(unknownAchievements)

		// Should map only the known adept achievements (unknown achievement filtered out)
		if len(mapped) != 2 {
			t.Errorf("Expected 2 mapped achievements (only adepts), got %d", len(mapped))
		}

		// Verify achievements have proper unlock status
		unlockedCount := 0
		for _, achievement := range mapped {
			if achievement.Unlocked {
				unlockedCount++
			}
		}

		// Should have 1 unlocked achievement (only Dwight, unknown filtered out)
		if unlockedCount != 1 {
			t.Errorf("Expected 1 unlocked achievement (Dwight only), got %d", unlockedCount)
		}

		// Verify all achievements have required fields and proper types
		for _, achievement := range mapped {
			if achievement.Name == "" {
				t.Errorf("Achievement %s has empty name", achievement.ID)
			}
			if achievement.Type == "" {
				t.Errorf("Achievement %s has empty type", achievement.ID)
			}
			// Verify only allowed types
			if achievement.Type != "general" && achievement.Type != "adept_killer" && achievement.Type != "adept_survivor" {
				t.Errorf("Achievement %s has invalid type: %s", achievement.ID, achievement.Type)
			}
		}
	})

	t.Run("TypeClassification", func(t *testing.T) {
		testAchievements := &PlayerAchievements{
			Achievements: []SteamAchievement{
				{APIName: "ACH_UNLOCK_DWIGHT_PERKS", Achieved: 1, UnlockTime: 1234567890}, // Should be adept_survivor
				{APIName: "ACH_UNLOCK_CHUCKLES_PERKS", Achieved: 0, UnlockTime: 0},        // Should be adept_killer
				{APIName: "UNKNOWN_ACHIEVEMENT", Achieved: 1, UnlockTime: 0},              // Should be general
			},
		}

		mapped := mapper.MapPlayerAchievements(testAchievements)

		// Find specific achievements and verify their types
		// In fallback mode, types are determined by AdeptAchievementMapping lookup
		for _, achievement := range mapped {
			switch achievement.ID {
			case "ACH_UNLOCK_DWIGHT_PERKS":
				// Should be adept_survivor because it's in AdeptAchievementMapping with type "survivor"
				if achievement.Type != "adept_survivor" {
					t.Errorf("ACH_UNLOCK_DWIGHT_PERKS should be adept_survivor, got %s", achievement.Type)
				}
			case "ACH_UNLOCK_CHUCKLES_PERKS":
				// Should be adept_killer because it's in AdeptAchievementMapping with type "killer"
				if achievement.Type != "adept_killer" {
					t.Errorf("ACH_UNLOCK_CHUCKLES_PERKS should be adept_killer, got %s", achievement.Type)
				}
			case "UNKNOWN_ACHIEVEMENT":
				// Should be general because it's not in AdeptAchievementMapping
				if achievement.Type != "general" {
					t.Errorf("UNKNOWN_ACHIEVEMENT should be general, got %s", achievement.Type)
				}
			}
		}
	})
}

func TestAchievementSummary(t *testing.T) {
	mapper := NewAchievementMapper()

	// Test with mixed locked/unlocked achievements using new type system
	testAchievements := []AchievementMapping{
		{ID: "ADEPT_DWIGHT", Character: "Dwight Fairfield", Type: "adept_survivor", Unlocked: true},
		{ID: "ADEPT_MEG", Character: "Meg Thomas", Type: "adept_survivor", Unlocked: false},
		{ID: "ADEPT_TRAPPER", Character: "The Trapper", Type: "adept_killer", Unlocked: true},
		{ID: "ADEPT_HILLBILLY", Character: "The Hillbilly", Type: "adept_killer", Unlocked: false},
		{ID: "GENERAL_ACHIEVEMENT", Character: "", Type: "general", Unlocked: true},
	}

	summary := mapper.GetAchievementSummary(testAchievements)

	// Verify adept counts (includes both locked and unlocked)
	adeptSurvivorCount := summary["adept_survivor_count"].(int)
	adeptKillerCount := summary["adept_killer_count"].(int)
	generalCount := summary["general_count"].(int)

	if adeptSurvivorCount != 2 {
		t.Errorf("Expected 2 adept survivors, got %d", adeptSurvivorCount)
	}

	if adeptKillerCount != 2 {
		t.Errorf("Expected 2 adept killers, got %d", adeptKillerCount)
	}

	if generalCount != 1 {
		t.Errorf("Expected 1 general achievement, got %d", generalCount)
	}

	// Verify character lists include all characters (regardless of unlock status)
	adeptSurvivors := summary["adept_survivors"].([]string)
	adeptKillers := summary["adept_killers"].([]string)

	if len(adeptSurvivors) != 2 {
		t.Errorf("Expected 2 survivors in list, got %d", len(adeptSurvivors))
	}

	if len(adeptKillers) != 2 {
		t.Errorf("Expected 2 killers in list, got %d", len(adeptKillers))
	}

	// Verify unlocked count only counts unlocked achievements
	unlockedCount := summary["unlocked_count"].(int)
	if unlockedCount != 3 {
		t.Errorf("Expected 3 unlocked achievements, got %d", unlockedCount)
	}

	// Verify completion rate
	completionRate := summary["completion_rate"].(float64)
	expectedRate := float64(3) / float64(5) * 100 // 3 unlocked out of 5 total = 60%
	if completionRate != expectedRate {
		t.Errorf("Expected completion rate %.1f%%, got %.1f%%", expectedRate, completionRate)
	}
}
