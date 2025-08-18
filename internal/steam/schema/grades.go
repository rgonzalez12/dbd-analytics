package schema

import "fmt"

// Grade represents a Dead by Daylight rank/grade
type Grade struct {
	Value int    `json:"value"`
	Label string `json:"label"`
}

// Grade mapping for Dead by Daylight's 20-tier system
var grade20 = []string{
	"Ash IV", "Ash III", "Ash II", "Ash I",
	"Bronze IV", "Bronze III", "Bronze II", "Bronze I",
	"Silver IV", "Silver III", "Silver II", "Silver I",
	"Gold IV", "Gold III", "Gold II", "Gold I",
	"Iridescent IV", "Iridescent III", "Iridescent II", "Iridescent I",
}

// GradeName converts a numeric grade value to human-readable name
func GradeName(v int) string {
	switch {
	case v >= 0 && v < 20:
		return grade20[v]
	case v >= 1 && v <= 20:
		return grade20[v-1]
	case v <= 0 && v >= -19:
		return grade20[-v]
	default:
		return fmt.Sprintf("Unknown (%d)", v)
	}
}

// SurvivorGradeName extracts and formats survivor grade if present in stats
func SurvivorGradeName(stats []UIStat) *Grade {
	for _, stat := range stats {
		if stat.Key == "DBD_UnlockRanking" || stat.Key == "survivor_grade" {
			if value, ok := convertToInt(stat.Value); ok {
				return &Grade{
					Value: value,
					Label: GradeName(value),
				}
			}
		}
	}
	return nil
}

// KillerGradeName extracts and formats killer grade if present in stats
func KillerGradeName(stats []UIStat) *Grade {
	for _, stat := range stats {
		if stat.Key == "DBD_SlasherTierIncrement" || stat.Key == "killer_grade" {
			if value, ok := convertToInt(stat.Value); ok {
				return &Grade{
					Value: value,
					Label: GradeName(value),
				}
			}
		}
	}
	return nil
}

// convertToInt safely converts various numeric types to int
func convertToInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}
