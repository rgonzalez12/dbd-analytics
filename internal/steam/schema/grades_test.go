package schema

import "testing"

func TestGradeName(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		// Standard 0-19 range
		{"Ash IV (0)", 0, "Ash IV"},
		{"Ash III (1)", 1, "Ash III"},
		{"Ash I (3)", 3, "Ash I"},
		{"Bronze IV (4)", 4, "Bronze IV"},
		{"Bronze I (7)", 7, "Bronze I"},
		{"Silver IV (8)", 8, "Silver IV"},
		{"Silver I (11)", 11, "Silver I"},
		{"Gold IV (12)", 12, "Gold IV"},
		{"Gold I (15)", 15, "Gold I"},
		{"Iridescent IV (16)", 16, "Iridescent IV"},
		{"Iridescent I (19)", 19, "Iridescent I"},

		// 1-20 range (offset by 1)
		{"Ash III (1-20)", 1, "Ash III"},
		{"Ash II (2-20)", 2, "Ash II"},
		{"Iridescent I (20)", 20, "Iridescent I"},

		// Negative values (inverted)
		{"Ash IV (-0)", 0, "Ash IV"},
		{"Ash III (-1)", -1, "Ash III"},
		{"Ash I (-3)", -3, "Ash I"},
		{"Bronze IV (-4)", -4, "Bronze IV"},
		{"Iridescent I (-19)", -19, "Iridescent I"},

		// Out of bounds
		{"Unknown positive", 25, "Unknown (25)"},
		{"Unknown negative", -25, "Unknown (-25)"},
		{"Unknown large", 999, "Unknown (999)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GradeName(tt.input)
			if result != tt.expected {
				t.Errorf("GradeName(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSurvivorGradeName(t *testing.T) {
	tests := []struct {
		name     string
		stats    []UIStat
		expected *Grade
	}{
		{
			name: "Valid survivor grade",
			stats: []UIStat{
				{Key: "DBD_UnlockRanking", Value: 5},
				{Key: "other_stat", Value: 100},
			},
			expected: &Grade{Value: 5, Label: "Bronze III"},
		},
		{
			name: "Alternative survivor grade key",
			stats: []UIStat{
				{Key: "survivor_grade", Value: 12},
			},
			expected: &Grade{Value: 12, Label: "Gold IV"},
		},
		{
			name: "No survivor grade",
			stats: []UIStat{
				{Key: "other_stat", Value: 100},
			},
			expected: nil,
		},
		{
			name: "Invalid value type",
			stats: []UIStat{
				{Key: "DBD_UnlockRanking", Value: "invalid"},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SurvivorGradeName(tt.stats)
			if (result == nil) != (tt.expected == nil) {
				t.Errorf("SurvivorGradeName() = %v, want %v", result, tt.expected)
				return
			}
			if result != nil && tt.expected != nil {
				if result.Value != tt.expected.Value || result.Label != tt.expected.Label {
					t.Errorf("SurvivorGradeName() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestKillerGradeName(t *testing.T) {
	tests := []struct {
		name     string
		stats    []UIStat
		expected *Grade
	}{
		{
			name: "Valid killer grade",
			stats: []UIStat{
				{Key: "DBD_SlasherTierIncrement", Value: 8},
				{Key: "other_stat", Value: 100},
			},
			expected: &Grade{Value: 8, Label: "Silver IV"},
		},
		{
			name: "Alternative killer grade key",
			stats: []UIStat{
				{Key: "killer_grade", Value: 19},
			},
			expected: &Grade{Value: 19, Label: "Iridescent I"},
		},
		{
			name: "No killer grade",
			stats: []UIStat{
				{Key: "other_stat", Value: 100},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := KillerGradeName(tt.stats)
			if (result == nil) != (tt.expected == nil) {
				t.Errorf("KillerGradeName() = %v, want %v", result, tt.expected)
				return
			}
			if result != nil && tt.expected != nil {
				if result.Value != tt.expected.Value || result.Label != tt.expected.Label {
					t.Errorf("KillerGradeName() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestConvertToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int
		ok       bool
	}{
		{"int", 42, 42, true},
		{"int32", int32(42), 42, true},
		{"int64", int64(42), 42, true},
		{"float32", float32(42.7), 42, true},
		{"float64", float64(42.9), 42, true},
		{"string", "42", 0, false},
		{"nil", nil, 0, false},
		{"bool", true, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := convertToInt(tt.input)
			if ok != tt.ok {
				t.Errorf("convertToInt(%v) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if ok && result != tt.expected {
				t.Errorf("convertToInt(%v) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}
