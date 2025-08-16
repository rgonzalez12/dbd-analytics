package steam

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAchievementMappingStructure tests the new achievement mapping structure
func TestAchievementMappingStructure(t *testing.T) {
	mapper := NewAchievementMapper()
	
	// Create a test PlayerAchievements with one achievement
	playerAchievements := &PlayerAchievements{
		SteamID: "76561198000000000",
		Achievements: []SteamAchievement{
			{
				APIName:    "Adept_Dwight",
				Achieved:   1,
				UnlockTime: 1640995200, // Jan 1, 2022
			},
		},
	}

	achievements := mapper.MapPlayerAchievements(playerAchievements)
	
	// Should return achievements
	assert.NotEmpty(t, achievements)
	
	// Check the structure of mapped achievements
	for _, achievement := range achievements {
		assert.NotEmpty(t, achievement.ID, "Achievement should have an ID")
		assert.NotEmpty(t, achievement.Type, "Achievement should have a type")
		
		// Verify types are valid
		validTypes := map[string]bool{
			"adept_survivor": true,
			"adept_killer":   true,
			"general":        true,
		}
		assert.True(t, validTypes[achievement.Type], "Achievement type should be valid: %s", achievement.Type)
		
		// Log achievement details for inspection
		t.Logf("Achievement: ID=%s, Name=%s, Type=%s, Unlocked=%v, Icon=%s, Rarity=%.2f", 
			achievement.ID, achievement.Name, achievement.Type, achievement.Unlocked, achievement.Icon, achievement.Rarity)
	}
	
	// Count achievements by type
	typeCounts := make(map[string]int)
	for _, achievement := range achievements {
		typeCounts[achievement.Type]++
	}
	
	t.Logf("Achievement counts: %+v", typeCounts)
	
	// In schema-first approach without Steam API key, only processes player achievements
	// This validates the fallback behavior correctly handles single achievement
	assert.Equal(t, 1, len(achievements), "Should process exactly the player's achievements when schema unavailable")
	assert.Equal(t, "general", achievements[0].Type, "Unknown achievement should be classified as general")
	assert.True(t, achievements[0].Unlocked, "Player's achievement should be marked as unlocked")
}

// TestAchievementMappingFallback tests the hardcoded fallback behavior
func TestAchievementMappingFallback(t *testing.T) {
	mapper := NewAchievementMapper()
	
	// Test with empty player achievements to force fallback
	playerAchievements := &PlayerAchievements{
		SteamID:      "invalid",
		Achievements: []SteamAchievement{},
	}

	achievements := mapper.MapPlayerAchievements(playerAchievements)
	
	// Schema-first approach: empty player data = empty results when no schema available
	// This correctly validates the new behavior where we don't synthesize achievements
	assert.Empty(t, achievements, "Should return empty when no player achievements and no schema")
	
	t.Logf("Achievements returned for empty player data: %d", len(achievements))
	
	// Test with actual player achievements to verify fallback classification works
	playerWithAchievements := &PlayerAchievements{
		SteamID: "76561198000000000",
		Achievements: []SteamAchievement{
			{
				APIName:    "ACH_UNLOCK_DWIGHT_PERKS", // Known adept achievement
				Achieved:   1,
				UnlockTime: 1640995200,
			},
			{
				APIName:    "UNKNOWN_ACHIEVEMENT", // Unknown achievement
				Achieved:   1,
				UnlockTime: 1640995200,
			},
		},
	}
	
	achievementsWithFallback := mapper.MapPlayerAchievements(playerWithAchievements)
	assert.Len(t, achievementsWithFallback, 2, "Should process all player achievements")
	
	// Verify classification works in fallback mode
	typeCounts := make(map[string]int)
	for _, achievement := range achievementsWithFallback {
		typeCounts[achievement.Type]++
	}
	
	t.Logf("Fallback classification results: %+v", typeCounts)
	assert.Equal(t, 1, typeCounts["adept_survivor"], "Should correctly classify known adept achievement")
	assert.Equal(t, 1, typeCounts["general"], "Should classify unknown achievement as general")
}
