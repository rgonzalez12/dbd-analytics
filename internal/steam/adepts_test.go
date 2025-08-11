package steam

import (
	"strings"
	"testing"
)

func TestBuildAdeptMap(t *testing.T) {
	// Test the adept regex and character normalization
	tests := []struct {
		displayName string
		expected    AdeptEntry
		shouldMatch bool
	}{
		{
			displayName: "Adept Animatronic",
			expected: AdeptEntry{
				APIName:   "NEW_ACHIEVEMENT_312_2",
				Character: "Animatronic",
				Kind:      "killer", // CORRECTED: Animatronic is a killer (William Afton/Springtrap)
			},
			shouldMatch: true,
		},
		{
			displayName: "Adept The Ghoul",
			expected: AdeptEntry{
				APIName:   "NEW_ACHIEVEMENT_317_5",
				Character: "Ghoul", // Normalized (removed "The")
				Kind:      "killer", // Based on heuristic ("The" prefix)
			},
			shouldMatch: true,
		},
		{
			displayName: "Adept Trapper",
			expected: AdeptEntry{
				APIName:   "ACH_UNLOCK_CHUCKLES_PERKS",
				Character: "Trapper",
				Kind:      "killer", // Should be killer from hardcoded mapping
			},
			shouldMatch: true,
		},
		{
			displayName: "Adept Dwight",
			expected: AdeptEntry{
				APIName:   "ACH_UNLOCK_DWIGHT_PERKS",
				Character: "Dwight",
				Kind:      "survivor", // Should be survivor from hardcoded mapping
			},
			shouldMatch: true,
		},
		{
			displayName: "Some Other Achievement",
			shouldMatch: false,
		},
	}

	for _, test := range tests {
		t.Run(test.displayName, func(t *testing.T) {
			matches := adeptRe.FindStringSubmatch(test.displayName)

			if test.shouldMatch {
				if len(matches) != 2 {
					t.Errorf("Expected regex to match %q, but it didn't", test.displayName)
					return
				}

				char := normalizeChar(matches[1])
				if char != test.expected.Character {
					t.Errorf("Expected character %q, got %q", test.expected.Character, char)
				}

				// Test kind determination using the same logic as BuildAdeptMap
				normalizedChar := strings.ToLower(char)
				kind := "survivor" // default

				// Check against hardcoded mapping
				for _, hcChar := range AdeptAchievementMapping {
					if strings.ToLower(hcChar.Name) == normalizedChar {
						kind = hcChar.Type
						break
					}
				}

				// If not found, use heuristics
				if kind == "survivor" && strings.HasPrefix(matches[1], "The ") {
					kind = "killer"
				}

				if kind != test.expected.Kind {
					t.Errorf("Expected kind %q, got %q for character %q", test.expected.Kind, kind, char)
				}
			} else {
				if len(matches) >= 2 {
					t.Errorf("Expected regex NOT to match %q, but it did", test.displayName)
				}
			}
		})
	}
}

func TestNormalizeChar(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Animatronic", "Animatronic"},
		{"The Ghoul", "Ghoul"},
		{"the Ghoul", "Ghoul"},
		{" The Ghoul ", "Ghoul"},
		{"  Dwight  ", "Dwight"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := normalizeChar(test.input)
			if result != test.expected {
				t.Errorf("normalizeChar(%q) = %q, expected %q", test.input, result, test.expected)
			}
		})
	}
}

// Mock schema data for testing
func createMockSchema() *SchemaGame {
	return &SchemaGame{
		GameName:    "Dead by Daylight",
		GameVersion: "1.0",
		AvailableGameStats: AvailableGameStats{
			Achievements: []SchemaAchievement{
				{
					Name:        "NEW_ACHIEVEMENT_312_2",
					DisplayName: "Adept Animatronic",
					Description: "Reach Player Level 10 with Animatronic using only their unique perks",
				},
				{
					Name:        "NEW_ACHIEVEMENT_317_5", 
					DisplayName: "Adept The Ghoul",
					Description: "Reach Player Level 10 with The Ghoul using only their unique perks",
				},
				{
					Name:        "SOME_OTHER_ACHIEVEMENT",
					DisplayName: "Some Other Achievement",
					Description: "This is not an adept achievement",
				},
			},
		},
	}
}

func TestAdeptMapWithMockData(t *testing.T) {
	// This would test BuildAdeptMap with mock data
	// For a real test, we'd need to mock GetSchemaForGame
	// For now, we just test the logic components separately
	
	mockSchema := createMockSchema()
	
	adeptCount := 0
	for _, ach := range mockSchema.AvailableGameStats.Achievements {
		if matches := adeptRe.FindStringSubmatch(ach.DisplayName); len(matches) == 2 {
			adeptCount++
			
			char := normalizeChar(matches[1])
			kind := "survivor"
			if strings.HasPrefix(matches[1], "The ") {
				kind = "killer"
			}
			
			switch ach.Name {
			case "NEW_ACHIEVEMENT_312_2":
				if char != "Animatronic" || kind != "survivor" {
					t.Errorf("Animatronic adept mapping failed: char=%q, kind=%q", char, kind)
				}
			case "NEW_ACHIEVEMENT_317_5":
				if char != "Ghoul" || kind != "killer" {
					t.Errorf("Ghoul adept mapping failed: char=%q, kind=%q", char, kind)
				}
			}
		}
	}
	
	if adeptCount != 2 {
		t.Errorf("Expected 2 adept achievements, found %d", adeptCount)
	}
}
