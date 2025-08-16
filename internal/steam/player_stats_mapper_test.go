package steam

import (
	"context"
	"testing"
)

func TestDecodeGrade(t *testing.T) {
	tests := []struct {
		name         string
		input        float64
		expectedTier string
		expectedSub  int
		expectedHuman string
	}{
		{"Ash 4", 0, "Ash", 4, "Ash IV"},
		{"Bronze 1", 7, "Bronze", 1, "Bronze I"},
		{"Silver 2", 10, "Silver", 2, "Silver II"},
		{"Gold 1", 15, "Gold", 1, "Gold I"},
		{"Iridescent 1", 19, "Iridescent", 1, "Iridescent I"},
		{"Unknown grade", 999, "Unranked", 0, "Unranked"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grade, human, roman := decodeGrade(tt.input)
			
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
	case 1: return "I"
	case 2: return "II"
	case 3: return "III"
	case 4: return "IV"
	default: return ""
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		valueType string
		expected  string
	}{
		{"Count with commas", 1234567, "count", "1,234,567"},
		{"Small count", 123, "count", "123"},
		{"Percentage", 87.5, "percent", "87.5%"},
		{"Level", 100, "level", "100"},
		{"Grade", 15, "grade", "Gold I"},
		{"Duration seconds", 45, "duration", "45s"},
		{"Duration minutes", 75, "duration", "1m 15s"},
		{"Duration hours", 3665, "duration", "1h 1m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatValue(tt.value, tt.valueType)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestInferStatRule(t *testing.T) {
	tests := []struct {
		name        string
		statName    string
		displayName string
		expectedCat string
	}{
		{"Killer stat by name", "killer_hooks_total", "Total Hooks", "killer"},
		{"Survivor stat by name", "survivor_escapes", "Escapes", "survivor"},
		{"Killer stat by display", "some_stat", "Hooks Performed", "killer"},
		{"Survivor stat by display", "other_stat", "Generators Repaired", "survivor"},
		{"General stat", "bloodweb_points", "Bloodweb Points", "general"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := inferStatRule(tt.statName, tt.displayName)
			if rule.Category != tt.expectedCat {
				t.Errorf("Expected category %s, got %s", tt.expectedCat, rule.Category)
			}
		})
	}
}

func TestStatsSorting(t *testing.T) {
	stats := []Stat{
		{Category: "general", SortWeight: 100, DisplayName: "Z General"},
		{Category: "killer", SortWeight: 50, DisplayName: "B Killer"},
		{Category: "survivor", SortWeight: 10, DisplayName: "A Survivor"},
		{Category: "killer", SortWeight: 5, DisplayName: "A Killer"},
		{Category: "survivor", SortWeight: 20, DisplayName: "B Survivor"},
	}

	// Sort using the same logic as MapPlayerStats
	sortStats(stats)

	expected := []string{
		"A Killer",    // killer, weight 5
		"B Killer",    // killer, weight 50
		"A Survivor",  // survivor, weight 10
		"B Survivor",  // survivor, weight 20
		"Z General",   // general, weight 100
	}

	for i, stat := range stats {
		if stat.DisplayName != expected[i] {
			t.Errorf("Position %d: expected %s, got %s", i, expected[i], stat.DisplayName)
		}
	}
}

// Helper function to test sorting logic
func sortStats(stats []Stat) {
	// Same sorting logic as in MapPlayerStats
	for i := 0; i < len(stats)-1; i++ {
		for j := i + 1; j < len(stats); j++ {
			// Category order: killer, survivor, general
			categoryOrder := map[string]int{"killer": 0, "survivor": 1, "general": 2}
			catI, catJ := categoryOrder[stats[i].Category], categoryOrder[stats[j].Category]
			
			shouldSwap := false
			if catI != catJ {
				shouldSwap = catI > catJ
			} else if stats[i].SortWeight != stats[j].SortWeight {
				shouldSwap = stats[i].SortWeight > stats[j].SortWeight
			} else {
				shouldSwap = stats[i].DisplayName > stats[j].DisplayName
			}
			
			if shouldSwap {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
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

// Mock test for integration - would need actual schema and user data
func TestMapPlayerStatsIntegration(t *testing.T) {
	// This would be a more comprehensive integration test
	// For now, just test that the function signature works
	ctx := context.Background()
	
	// Would need mock client and cache for full test
	_, err := MapPlayerStats(ctx, "test_steam_id", nil, nil)
	
	// Should fail gracefully with nil client
	if err == nil {
		t.Error("Expected error with nil client")
	}
}
