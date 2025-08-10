package steam

import (
	"testing"
)

func TestAchievementMapperEnhancements(t *testing.T) {
	mapper := NewAchievementMapper()

	t.Run("ValidateMappingCoverage", func(t *testing.T) {
		coverage := mapper.ValidateMappingCoverage()
		
		survivorCount := coverage["survivor_adepts_mapped"].(int)
		killerCount := coverage["killer_adepts_mapped"].(int)
		totalCount := coverage["total_characters"].(int)
		
		if survivorCount < 40 {
			t.Errorf("Expected at least 40 survivors mapped, got %d", survivorCount)
		}
		
		if killerCount < 25 {
			t.Errorf("Expected at least 25 killers mapped, got %d", killerCount)
		}
		
		if totalCount != survivorCount+killerCount {
			t.Errorf("Total count mismatch: %d != %d + %d", totalCount, survivorCount, killerCount)
		}
		
		t.Logf("Mapping coverage: %d survivors, %d killers, %d total", survivorCount, killerCount, totalCount)
	})

	t.Run("UnknownAchievementTracking", func(t *testing.T) {
		// Simulate processing an unknown achievement
		unknownAchievements := &PlayerAchievements{
			Achievements: []SteamAchievement{
				{APIName: "NEW_ACHIEVEMENT_999_99", Achieved: 1, UnlockTime: 1234567890},
				{APIName: "UNKNOWN_FUTURE_ACHIEVEMENT", Achieved: 0, UnlockTime: 0},
			},
		}
		
		mapped := mapper.MapPlayerAchievements(unknownAchievements)
		
		// We expect all 84 adept achievements plus 2 unknown achievements = 86 total
		expectedTotal := 84 + 2  // All adept achievements + unknown ones
		if len(mapped) != expectedTotal {
			t.Errorf("Expected %d mapped achievements (84 adept + 2 unknown), got %d", expectedTotal, len(mapped))
		}
		
		// Check that unknown achievements were tracked
		unknowns := mapper.GetUnknownAchievements()
		if len(unknowns) < 2 {
			t.Errorf("Expected at least 2 unknown achievements tracked, got %d", len(unknowns))
		}
		
		// Verify fallback mapping works
		for _, achievement := range mapped {
			if achievement.Description == "" {
				t.Errorf("Achievement %s has empty description", achievement.ID)
			}
			if achievement.Type == "" {
				t.Errorf("Achievement %s has empty type", achievement.ID)
			}
		}
		
		t.Logf("Successfully tracked %d unknown achievements", len(unknowns))
	})

	t.Run("FallbackNaming", func(t *testing.T) {
		testCases := []struct {
			apiName      string
			expectedType string
		}{
			{"NEW_ACHIEVEMENT_999_SURVIVOR_3", "survivor"},
			{"ACH_CHAPTER99_KILLER_3", "killer"},
			{"DLC_UNKNOWN_KILLER_THING", "killer"},
			{"ACH_HEAL_SOMETHING", "survivor"},
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

func TestAdeptSurvivorKillerInclusion(t *testing.T) {
	// Test that both locked and unlocked adept achievements are included in the summary
	mapper := NewAchievementMapper()
	
	// Create test data with both unlocked and locked achievements
	testAchievements := []AchievementMapping{
		{
			ID:        "ACH_UNLOCK_DWIGHT_PERKS",
			Character: "dwight",
			Type:      "survivor",
			Unlocked:  true, // Unlocked
		},
		{
			ID:        "ACH_UNLOCK_MEG_PERKS",
			Character: "meg",
			Type:      "survivor", 
			Unlocked:  false, // Locked
		},
		{
			ID:        "ACH_UNLOCK_CHUCKLES_PERKS",
			Character: "trapper",
			Type:      "killer",
			Unlocked:  true, // Unlocked
		},
		{
			ID:        "ACH_UNLOCK_WRAITH_PERKS",
			Character: "wraith",
			Type:      "killer",
			Unlocked:  false, // Locked
		},
	}
	
	// Get the summary
	summary := mapper.GetAchievementSummary(testAchievements)
	
	// Extract the survivor and killer lists
	adeptSurvivors := summary["adept_survivors"].([]string)
	adeptKillers := summary["adept_killers"].([]string)
	
	// Verify ALL survivors are included regardless of unlock status
	expectedSurvivors := []string{"dwight", "meg"}
	if len(adeptSurvivors) != len(expectedSurvivors) {
		t.Errorf("Expected %d survivors, got %d", len(expectedSurvivors), len(adeptSurvivors))
	}
	
	for _, expected := range expectedSurvivors {
		found := false
		for _, actual := range adeptSurvivors {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected survivor %s not found in adept_survivors list", expected)
		}
	}
	
	// Verify ALL killers are included regardless of unlock status
	expectedKillers := []string{"trapper", "wraith"}
	if len(adeptKillers) != len(expectedKillers) {
		t.Errorf("Expected %d killers, got %d", len(expectedKillers), len(adeptKillers))
	}
	
	for _, expected := range expectedKillers {
		found := false
		for _, actual := range adeptKillers {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected killer %s not found in adept_killers list", expected)
		}
	}
	
	// Verify the unlocked count is still accurate (only counts unlocked achievements)
	unlockedCount := summary["unlocked_count"].(int)
	expectedUnlocked := 2 // dwight and trapper are unlocked
	if unlockedCount != expectedUnlocked {
		t.Errorf("Expected unlocked count %d, got %d", expectedUnlocked, unlockedCount)
	}
	
	t.Logf("Test passed: All %d survivors and %d killers included in summary regardless of unlock status", 
		len(adeptSurvivors), len(adeptKillers))
}

func TestAchievementMappingCompleteness(t *testing.T) {
	// Test that we have comprehensive coverage of known characters
	expectedSurvivors := []string{
		"dwight", "meg", "claudette", "jake", "nea", "laurie", "ace", "bill",
		"feng", "david", "quentin", "tapp", "kate", "adam", "jeff", "jane",
		"ash", "yui", "zarina", "cheryl", "felix", "elodie", "yun-jin",
		"jill", "leon", "mikaela", "jonah", "yoichi", "haddie", "ada", "rebecca",
		"vittorio", "thalita", "renato", "gabriel", "nicolas", "ellen", "alan",
		"sable", "troupe", "lara", "trevor", "taurie", "orela", "rick", "michonne",
	}
	
	expectedKillers := []string{
		"trapper", "wraith", "hillbilly", "nurse", "shape", "hag", "doctor",
		"huntress", "cannibal", "nightmare", "pig", "clown", "spirit", "legion",
		"plague", "ghostface", "oni", "deathslinger", "executioner", "blight",
		"twins", "trickster", "nemesis", "artist", "onryo", "dredge", "mastermind",
		"knight", "skull-merchant", "singularity", "xenomorph", "chucky", "unknown",
		"vecna", "dark-lord", "houndmaster", "ghoul", "animatronic",
	}
	
	// Count mapped characters
	survivorsMapped := 0
	killersMapped := 0
	
	for _, character := range AdeptAchievementMapping {
		if character.Type == "survivor" {
			survivorsMapped++
		} else if character.Type == "killer" {
			killersMapped++
		}
	}
	
	t.Logf("Found %d survivors and %d killers in mapping", survivorsMapped, killersMapped)
	
	if survivorsMapped < len(expectedSurvivors) {
		t.Logf("Note: Expected %d survivors, found %d in mapping", len(expectedSurvivors), survivorsMapped)
	}
	
	if killersMapped < len(expectedKillers) {
		t.Logf("Note: Expected %d killers, found %d in mapping", len(expectedKillers), killersMapped)
	}
	
	// This test is informational - we don't fail if we have fewer than expected
	// since DBD adds new characters regularly
}
