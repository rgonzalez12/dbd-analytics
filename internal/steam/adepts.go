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

func (c *Client) BuildAdeptMap() (map[string]AdeptEntry, error) {
	schema, err := c.GetSchemaForGame(DBDAppID)
	if err != nil {
		return nil, err
	}

	killerNames := make(map[string]bool)
	survivorNames := make(map[string]bool)

	for _, char := range AdeptAchievementMapping {
		normalizedName := strings.ToLower(char.Name)
		switch char.Type {
		case "killer":
			killerNames[normalizedName] = true
		default:
			survivorNames[normalizedName] = true
		}
	}

	// Add character name variations for proper type detection
	killerNames["dark lord"] = true
	killerNames["ghost face"] = true
	killerNames["good guy"] = true
	killerNames["skull merchant"] = true
	killerNames["onryo"] = true
	killerNames["onryō"] = true
	killerNames["sadako"] = true
	killerNames["the onryo"] = true

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
			switch {
			case killerNames[normalizedChar]:
				kind = "killer"
			case survivorNames[normalizedChar]:
				kind = "survivor"
			default:
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
