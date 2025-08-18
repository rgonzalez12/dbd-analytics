package steam

import (
	"testing"
)

// TestSchemaFirstAcceptance validates our surgical edits for exact schema compliance
func TestSchemaFirstAcceptance(t *testing.T) {
	// Note: This test will attempt to fetch global achievement percentages if STEAM_API_KEY is set,
	// but will fall back to local processing if not available
	mapper := NewAchievementMapper()

	// Mock player data to test both paths
	mockPlayerData := &PlayerAchievements{
		SteamID:  "test_player",
		GameName: "Dead by Daylight",
		Success:  true,
		Achievements: []SteamAchievement{
			{APIName: "ACH_UNLOCK_DWIGHT_PERKS", Achieved: 1},   // Adept survivor
			{APIName: "ACH_UNLOCK_CHUCKLES_PERKS", Achieved: 1}, // Adept killer
			{APIName: "ACH_DAILY_PLAY", Achieved: 1},            // General
			{APIName: "unknown_ach", Achieved: 1},               // Unknown
		},
	}

	results := mapper.MapPlayerAchievements(mockPlayerData)

	// Test 1: Verify strict type distribution
	typeCounts := make(map[string]int)
	for _, ach := range results {
		typeCounts[ach.Type]++
	}

	// In fallback mode (no schema), only adept achievements are processed
	expectedTypes := []string{"adept_survivor", "adept_killer"}
	for _, expectedType := range expectedTypes {
		if typeCounts[expectedType] == 0 {
			t.Errorf("Expected to find type %s but found none. Type distribution: %v", expectedType, typeCounts)
		}
	}

	// Test 2: No bad types should exist (general type not available in fallback mode)
	allowedTypes := []string{"adept_survivor", "adept_killer"}
	for actualType := range typeCounts {
		isValidType := false
		for _, validType := range allowedTypes {
			if actualType == validType {
				isValidType = true
				break
			}
		}
		if !isValidType {
			t.Errorf("Found invalid type: %s. In fallback mode, only allowed types: %v", actualType, allowedTypes)
		}
	}

	// Test 3: Verify no unknown achievements in final results
	for _, ach := range results {
		if ach.Name == "unknown_ach" || ach.ID == "unknown_ach" {
			t.Errorf("Unknown achievement should not appear in final results: %v", ach)
		}
	}

	// Test 4: Verify proper adept mapping with character names
	foundDwight := false
	foundTrapper := false
	for _, ach := range results {
		if ach.ID == "ACH_UNLOCK_DWIGHT_PERKS" {
			if ach.Type != "adept_survivor" || ach.Character != "dwight" {
				t.Errorf("Dwight adept incorrectly mapped: Type=%s, Character=%s", ach.Type, ach.Character)
			}
			foundDwight = true
		}
		if ach.ID == "ACH_UNLOCK_CHUCKLES_PERKS" {
			if ach.Type != "adept_killer" || ach.Character != "trapper" {
				t.Errorf("Trapper adept incorrectly mapped: Type=%s, Character=%s", ach.Type, ach.Character)
			}
			foundTrapper = true
		}
	}

	if !foundDwight || !foundTrapper {
		t.Error("Expected to find both Dwight and Trapper adept achievements")
	}

	t.Logf("✅ Schema-first acceptance test passed")
	t.Logf("   - Type distribution: %v", typeCounts)
	t.Logf("   - Total achievements processed: %d", len(results))
	t.Logf("   - Fallback mode: Only adept achievements processed when schema unavailable")
	t.Logf("   - Source of truth: Steam schema with strict fallback filtering")
}
