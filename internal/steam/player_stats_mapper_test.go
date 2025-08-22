package steam

import (
	"context"
	"testing"
)

func TestDecodeGrade(t *testing.T) {
	tests := []struct {
		name          string
		input         float64
		fieldID       string
		expectedTier  string
		expectedSub   int
		expectedHuman string
	}{
		{"Unranked", 0, "DBD_SlasherTierIncrement", "Ash", 4, "Ash IV"},
		{"Ash IV", 16, "DBD_SlasherTierIncrement", "Ash", 4, "Ash IV"},
		{"Silver III Killer", 300, "DBD_SlasherTierIncrement", "Silver", 3, "Silver III"},
		{"Bronze IV", 73, "DBD_SlasherTierIncrement", "Bronze", 4, "Bronze IV"},
		{"Ash III Survivor", 545, "DBD_UnlockRanking", "Ash", 3, "Ash III"},
		{"Ash III Survivor (541)", 541, "DBD_UnlockRanking", "Ash", 3, "Ash III"},
		{"High grade value", 999, "DBD_SlasherTierIncrement", "Iridescent", 3, "Iridescent III"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grade, human, roman := decodeGrade(tt.input, tt.fieldID)

			if grade.Tier != tt.expectedTier {
				t.Errorf("Expected tier %s, got %s", tt.expectedTier, grade.Tier)
			}

			if grade.Sub != tt.expectedSub {
				t.Errorf("Expected sub %d, got %d", tt.expectedSub, grade.Sub)
			}

			if human != tt.expectedHuman {
				t.Errorf("Expected human %s, got %s", tt.expectedHuman, human)
			}

			if tt.expectedSub > 0 && roman != romanExpected(tt.expectedSub) {
				t.Errorf("Expected roman %s, got %s", romanExpected(tt.expectedSub), roman)
			}
		})
	}
}

func romanExpected(n int) string {
	switch n {
	case 1:
		return "I"
	case 2:
		return "II"
	case 3:
		return "III"
	case 4:
		return "IV"
	default:
		return ""
	}
}

func TestDetermineValueType(t *testing.T) {
	tests := []struct {
		name        string
		statID      string
		displayName string
		value       float64
		expected    string
	}{
		{"Percentage float", "DBD_GeneratorPct_float", "Generators Repaired", 100.5, "float"},
		{"Heal percentage", "DBD_HealPct_float", "Survivors Healed", 50.0, "float"},
		{"Killer grade", "DBD_SlasherTierIncrement", "Killer Grade", 300, "grade"},
		{"Survivor grade", "DBD_UnlockRanking", "Survivor Grade", 541, "grade"},
		{"Killer bloodpoints", "DBD_SlasherSkulls", "Killer Bloodpoints", 98000, "count"},
		{"Survivor bloodpoints", "DBD_CamperSkulls", "Survivor Bloodpoints", 125000, "count"},
		{"Prestige level", "DBD_BloodwebMaxPrestigeLevel", "Highest Prestige", 82, "level"},
		{"Max level", "DBD_BloodwebPerkMaxLevel", "Max Perk Level", 3, "level"},
		{"Time played", "DBD_TimePlayed", "Time Played", 3600, "duration"},
		{"Session time", "DBD_SessionTime", "Session Time", 1800, "duration"},
		{"Regular count", "DBD_SacrificedCampers", "Survivors Sacrificed", 1048, "count"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineValueType(tt.statID, tt.displayName, tt.value)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCategorizeStats(t *testing.T) {
	tests := []struct {
		name        string
		statID      string
		displayName string
		expected    string
	}{
		{"Killer stat by ID", "DBD_SlasherTierIncrement", "Killer Grade", "killer"},
		{"Survivor stat by ID", "DBD_UnlockRanking", "Survivor Grade", "survivor"},
		{"Killer stat by display", "DBD_SomeKillerStat", "Hooks Performed", "killer"},
		{"Survivor stat by display", "DBD_SomeSurvivorStat", "Generators Repaired", "survivor"},
		{"General stat", "DBD_BloodwebPoints", "Bloodpoints Earned", "general"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category := categorizeStats(tt.statID, tt.displayName)
			if category != tt.expected {
				t.Errorf("Expected category %s, got %s", tt.expected, category)
			}
		})
	}
}

func TestFallbackDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		statID   string
		expected string
	}{
		{"DBD prefix removal", "DBD_Generator_Repaired", "Generator Repaired"},
		{"Camper to Survivor", "DBD_Camper_Escapes", "Survivor Escapes"},
		{"Slasher to Killer", "DBD_Slasher_Hooks", "Killer Hooks"}, 
		{"Complex replacement", "DBD_Camper_Slasher_Interaction", "Survivor Killer Interaction"},
		{"No DBD prefix", "Some_Other_Stat", "Some Other Stat"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fallbackDisplayName(tt.statID)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		valueType string
		fieldID   string
		expected  string
	}{
		{"Count with commas", 1234567, "count", "DBD_SacrificedCampers", "1,234,567"},
		{"Small count", 123, "count", "DBD_Escapes", "123"},
		{"Float value", 87.5, "float", "DBD_GeneratorPct_float", "87.5"},
		{"Level", 100, "level", "DBD_BloodwebMaxLevel", "100"},
		{"Grade - Killer", 300, "grade", "DBD_SlasherTierIncrement", "Silver III"},
		{"Grade - Survivor", 541, "grade", "DBD_UnlockRanking", "Ash III"},
		{"Duration seconds", 45, "duration", "DBD_TimePlayed", "45s"},
		{"Duration minutes", 75, "duration", "DBD_SessionTime", "1m 15s"},
		{"Duration hours", 3665, "duration", "DBD_TotalPlayTime", "1h 1m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatValue(tt.value, tt.valueType, tt.fieldID)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestFormatInt(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{123, "123"},
		{1234, "1,234"},
		{12345, "12,345"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
		{0, "0"},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.input)), func(t *testing.T) {
			result := formatInt(tt.input)
			if result != tt.expected {
				t.Errorf("formatInt(%d) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int64
		expected string
	}{
		{"Seconds only", 45, "45s"},
		{"Minutes and seconds", 75, "1m 15s"},
		{"Hours and minutes", 3665, "1h 1m"},
		{"Hours only", 3600, "1h 0m"},      // Always shows minutes
		{"Minutes only", 300, "5m 0s"},     // Always shows seconds
		{"Zero", 0, "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.seconds)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestAliasMap(t *testing.T) {
	// Test key aliases exist
	requiredAliases := []string{
		"DBD_SlasherTierIncrement",
		"DBD_UnlockRanking", 
		"DBD_SlasherSkulls",
		"DBD_CamperSkulls",
		"DBD_BloodwebMaxPrestigeLevel",
		"DBD_GeneratorPct_float",
		"DBD_HealPct_float",
	}

	for _, alias := range requiredAliases {
		if _, exists := aliases[alias]; !exists {
			t.Errorf("Required alias %s not found in aliases map", alias)
		}
	}

	// Test specific alias mappings
	if aliases["DBD_SlasherTierIncrement"] != "Killer Grade" {
		t.Errorf("Expected 'Killer Grade', got %s", aliases["DBD_SlasherTierIncrement"])
	}

	if aliases["DBD_UnlockRanking"] != "Survivor Grade" {
		t.Errorf("Expected 'Survivor Grade', got %s", aliases["DBD_UnlockRanking"])
	}
}

// Mock test for integration - would need actual schema and user data
func TestMapPlayerStatsIntegration(t *testing.T) {
	// More comprehensive integration test
	// Test function signature works
	ctx := context.Background()

	// Would need mock client and cache for full test
	_, err := MapPlayerStats(ctx, "test_steam_id", nil, nil)

	// Should fail gracefully with nil client
	if err == nil {
		t.Error("Expected error with nil client")
	}
}
