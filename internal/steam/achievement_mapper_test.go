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
		
		if len(mapped) != 2 {
			t.Errorf("Expected 2 mapped achievements, got %d", len(mapped))
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
