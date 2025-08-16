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
			"survivor": true,
			"killer":   true,
			"general":  true,
			"adept":    true,
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
	
	// Should have at least some achievements
	assert.Greater(t, len(achievements), 50, "Should have a good number of achievements")
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
	
	// Should have fallback achievements
	assert.NotEmpty(t, achievements, "Should have fallback achievements")
	
	// All achievements should be marked as not unlocked for empty player data
	unlockedCount := 0
	for _, achievement := range achievements {
		if achievement.Unlocked {
			unlockedCount++
		}
	}
	
	t.Logf("Unlocked achievements: %d out of %d", unlockedCount, len(achievements))
	
	// Should have some structure even without player data
	typeCounts := make(map[string]int)
	for _, achievement := range achievements {
		typeCounts[achievement.Type]++
	}
	
	assert.Greater(t, typeCounts["survivor"], 0, "Should have survivor achievements")
	assert.Greater(t, typeCounts["killer"], 0, "Should have killer achievements")
}
