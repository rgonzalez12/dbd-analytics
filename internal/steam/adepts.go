package steam

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/cache"
)

type AdeptEntry struct {
	APIName   string // schema 'name' (apiname)
	Character string // normalized character name
	Kind      string // "survivor" | "killer"
}

var adeptRe = regexp.MustCompile(`(?i)^Adept\s+(.+)$`)

func normalizeChar(s string) string {
	s = strings.TrimSpace(s)
	// kill leading "The " to normalize killers like "The Ghoul"
	s = strings.TrimPrefix(s, "The ")
	s = strings.TrimPrefix(s, "the ")
	return s
}

// BuildAdeptMap parses GetSchemaForGame achievements, selects those whose displayName starts with "Adept ",
// and returns map[apiname]AdeptEntry
func (c *Client) BuildAdeptMap() (map[string]AdeptEntry, error) {
	schema, err := c.GetSchemaForGame(DBDAppID)
	if err != nil {
		return nil, err
	}

	// Create lookup maps from hardcoded data for type determination
	killerNames := make(map[string]bool)
	survivorNames := make(map[string]bool)
	
	for _, char := range AdeptAchievementMapping {
		normalizedName := strings.ToLower(char.Name)
		if char.Type == "killer" {
			killerNames[normalizedName] = true
		} else {
			survivorNames[normalizedName] = true
		}
	}
	
	// Add additional name variations for characters that might appear with different display names
	// Killers with spaces in names that get normalized differently
	killerNames["dark lord"] = true     // "Adept The Dark Lord" → "Dark Lord"
	killerNames["ghost face"] = true   // "Adept Ghost Face" 
	killerNames["good guy"] = true     // "Adept Good Guy" (Chucky)
	killerNames["skull merchant"] = true // "Adept The Skull Merchant" → "Skull Merchant"
	killerNames["onryo"] = true         // "Adept The Onryo" → "Onryo" (case variations)
	killerNames["onryō"] = true         // "Adept The Onryō" → "Onryō" (with macron)
	killerNames["sadako"] = true        // Alternative name for Onryo
	killerNames["the onryo"] = true     // In case the "The" doesn't get stripped

	m := make(map[string]AdeptEntry, 128)
	for _, ach := range schema.AvailableGameStats.Achievements {
		dn := strings.TrimSpace(ach.DisplayName)
		if dn == "" {
			continue
		}
		// match "Adept X"
		if matches := adeptRe.FindStringSubmatch(dn); len(matches) == 2 {
			char := normalizeChar(matches[1])
			normalizedChar := strings.ToLower(char)
			
			// Determine type using hardcoded mapping first, then heuristics
			kind := "survivor" // default
			if killerNames[normalizedChar] {
				kind = "killer"
			} else if survivorNames[normalizedChar] {
				kind = "survivor"
			} else {
				// Fallback heuristics for unknown characters
				if strings.HasPrefix(matches[1], "The ") {
					kind = "killer"
				}
				// Additional heuristics for common killer naming patterns
				lowerChar := strings.ToLower(char)
				if strings.Contains(lowerChar, "doctor") || strings.Contains(lowerChar, "nurse") || 
				   strings.Contains(lowerChar, "spirit") || strings.Contains(lowerChar, "plague") ||
				   strings.Contains(lowerChar, "executioner") || strings.Contains(lowerChar, "blight") {
					kind = "killer"
				}
			}
			
			m[ach.Name] = AdeptEntry{APIName: ach.Name, Character: char, Kind: kind}
		}
	}
	return m, nil
}

// GetAdeptMapCached returns the adept map with caching support
func (c *Client) GetAdeptMapCached(ctx context.Context, cacheManager cache.Cache) (map[string]AdeptEntry, error) {
	key := cache.GenerateKey(cache.AdeptMapPrefix, "dbd")
	
	if cached, ok := cacheManager.Get(key); ok {
		if adeptMap, ok := cached.(map[string]AdeptEntry); ok {
			return adeptMap, nil
		}
	}
	
	m, err := c.BuildAdeptMap()
	if err != nil {
		return nil, err
	}
	
	// Cache for 24 hours
	_ = cacheManager.Set(key, m, 24*time.Hour)
	
	return m, nil
}
