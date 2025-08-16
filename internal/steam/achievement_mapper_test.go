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
		
		// Note: 40 killers total in DBD, all should be mappable even though Demogorgon has no achievement
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
				{APIName: "ACH_UNLOCK_DWIGHT_PERKS", Achieved: 1, UnlockTime: 1234567890}, // Known achievement
				{APIName: "ACH_UNLOCK_MEG_PERKS", Achieved: 0, UnlockTime: 0},             // Known achievement
			},
		}
		
		mapped := mapper.MapPlayerAchievements(unknownAchievements)
		
		// With schema-first approach, we get all 86 hardcoded achievements (since no schema available in test)
		// regardless of what's in the player's achievement list
		expectedTotal := 86
		if len(mapped) != expectedTotal {
			t.Errorf("Expected %d mapped achievements, got %d", expectedTotal, len(mapped))
		}
		
		// Verify achievements have proper unlock status
		unlockedCount := 0
		for _, achievement := range mapped {
			if achievement.Unlocked {
				unlockedCount++
			}
		}
		
		// Should have 1 unlocked achievement (Dwight)
		if unlockedCount != 1 {
			t.Errorf("Expected 1 unlocked achievement, got %d", unlockedCount)
		}
		
		// Verify all achievements have required fields
		for _, achievement := range mapped {
			if achievement.Name == "" {
				t.Errorf("Achievement %s has empty name", achievement.ID)
			}
			if achievement.Type == "" {
				t.Errorf("Achievement %s has empty type", achievement.ID)
			}
		}
	})

	t.Run("FallbackTypeDetection", func(t *testing.T) {
		testCases := []struct {
			apiName      string
			expectedType string
		}{
			{"ACH_CHAPTER99_SURVIVOR_3", "survivor"},
			{"ACH_CHAPTER99_KILLER_3", "killer"},
			{"NEW_ACHIEVEMENT_999_SURVIVOR_3", "survivor"},
			{"DLC_UNKNOWN_KILLER_THING", "killer"},
			{"RANDOM_ACHIEVEMENT", "general"},
		}
		
		for _, tc := range testCases {
			mapping := mapper.getAchievementMapping(tc.apiName)
			if mapping.Type != tc.expectedType {
				t.Errorf("Achievement %s: expected type %s, got %s", tc.apiName, tc.expectedType, mapping.Type)
			}
			
			if mapping.DisplayName == "" {
				t.Errorf("Achievement %s has empty display name", tc.apiName)
			}
		}
	})
}

func TestAchievementSummary(t *testing.T) {
	mapper := NewAchievementMapper()
	
	// Test with mixed locked/unlocked achievements
	testAchievements := []AchievementMapping{
		{ID: "ACH_UNLOCK_DWIGHT_PERKS", Character: "dwight", Type: "survivor", Unlocked: true},
		{ID: "ACH_UNLOCK_MEG_PERKS", Character: "meg", Type: "survivor", Unlocked: false},
		{ID: "ACH_UNLOCK_CHUCKLES_PERKS", Character: "trapper", Type: "killer", Unlocked: true},
		{ID: "ACH_UNLOCKHILLBILY_PERKS", Character: "hillbilly", Type: "killer", Unlocked: false},
	}
	
	summary := mapper.GetAchievementSummary(testAchievements)
	
	// Verify all characters are included regardless of unlock status
	adeptSurvivors := summary["adept_survivors"].([]string)
	adeptKillers := summary["adept_killers"].([]string)
	
	if len(adeptSurvivors) != 2 {
		t.Errorf("Expected 2 survivors, got %d", len(adeptSurvivors))
	}
	
	if len(adeptKillers) != 2 {
		t.Errorf("Expected 2 killers, got %d", len(adeptKillers))
	}
	
	// Verify unlocked count only counts unlocked achievements
	unlockedCount := summary["unlocked_count"].(int)
	if unlockedCount != 2 {
		t.Errorf("Expected 2 unlocked achievements, got %d", unlockedCount)
	}
}
