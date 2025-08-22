package steam

import (
	"strings"
	"testing"
)

func TestAdeptRegexAndNormalization(t *testing.T) {
	tests := []struct {
		displayName  string
		expectedChar string
		expectedKind string
		shouldMatch  bool
	}{
		{"Adept The Onryō", "Onryō", "killer", true}, // Test macron handling
		{"Adept The Ghoul", "Ghoul", "killer", true},
		{"Adept The Dark Lord", "Dark Lord", "killer", true},
		{"Adept Ghost Face", "Ghost Face", "killer", true},
		{"Adept Dwight", "Dwight", "survivor", true},
		{"Adept Meg", "Meg", "survivor", true},
		{"Some Other Achievement", "", "", false},
		{"Not an adept", "", "", false},
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
				if char != test.expectedChar {
					t.Errorf("Expected character %q, got %q", test.expectedChar, char)
				}

				normalizedChar := strings.ToLower(char)
				kind := "survivor"

				killerNames := make(map[string]bool)
				killerNames["onryo"] = true
				killerNames["onryō"] = true
				killerNames["ghoul"] = true
				killerNames["dark lord"] = true
				killerNames["ghost face"] = true

				if killerNames[normalizedChar] {
					kind = "killer"
				} else if strings.HasPrefix(matches[1], "The ") {
					kind = "killer" // Fallback heuristic
				}

				if kind != test.expectedKind {
					t.Errorf("Expected kind %q, got %q for character %q", test.expectedKind, kind, char)
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
		{"The Onryō", "Onryō"},
		{"The Ghoul", "Ghoul"},
		{"the Dark Lord", "Dark Lord"},
		{"Ghost Face", "Ghost Face"},
		{" Dwight ", "Dwight"},
		{"  The Trapper  ", "Trapper"},
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
