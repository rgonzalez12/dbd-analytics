package steam

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/cache"
	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

// Stat represents a single player statistic with metadata
type Stat struct {
	ID          string  `json:"id"`            // schema name
	DisplayName string  `json:"display_name"`  // schema displayName
	Value       float64 `json:"value"`         // raw numeric
	Formatted   string  `json:"formatted"`     // human string (optional; derived)
	Category    string  `json:"category"`      // "killer" | "survivor" | "general"
	ValueType   string  `json:"value_type"`    // "count" | "percent" | "grade" | "level" | "duration"
	SortWeight  int     `json:"sort_weight"`   // for stable ordering in UI
	Icon        string  `json:"icon,omitempty"`
	Alias       string  `json:"alias,omitempty"`       // e.g., killer_grade
	MatchedBy   string  `json:"matched_by,omitempty"`  // rule id (for debugging)
}

// PlayerStatsResponse represents the complete stats response
type PlayerStatsResponse struct {
	Stats   []Stat                 `json:"stats"`
	Summary map[string]interface{} `json:"summary"` // totals you want (optional)
}

// Grade represents decoded grade information
type Grade struct {
	Tier string // Bronze/Silver/Gold/Iridescent
	Sub  int    // 1..4
}

// StatRule defines classification and formatting rules for stats
type StatRule struct {
	ID        string
	Category  string
	ValueType string
	Weight    int
	Alias     string
	Match     func(id, dn string) bool
}

var ruleSet = []StatRule{
	{ // killer grade
		ID: "killer_grade", Category: "killer", ValueType: "grade", Weight: 0, Alias: "killer_grade",
		Match: func(id, dn string) bool {
			s := strings.ToLower(id + "|" + dn)
			return (strings.Contains(s, "killer") && strings.Contains(s, "grade")) ||
				strings.Contains(s, "rank_killer") || strings.Contains(s, "killer_rank")
		},
	},
	{ // survivor grade
		ID: "survivor_grade", Category: "survivor", ValueType: "grade", Weight: 0, Alias: "survivor_grade",
		Match: func(id, dn string) bool {
			s := strings.ToLower(id + "|" + dn)
			return (strings.Contains(s, "survivor") && strings.Contains(s, "grade")) ||
				strings.Contains(s, "rank_camper") || strings.Contains(s, "survivor_rank")
		},
	},
	{ // highest prestige level
		ID: "highest_prestige", Category: "general", ValueType: "level", Weight: 5, Alias: "highest_prestige",
		Match: func(id, dn string) bool {
			s := strings.ToLower(id + "|" + dn)
			return strings.Contains(s, "prestige") && (strings.Contains(s, "highest") || strings.Contains(s, "max"))
		},
	},
	// Conservative percent rules (add more only when verified)
	{
		ID: "healing_progress", Category: "survivor", ValueType: "percent", Weight: 20,
		Match: func(id, dn string) bool {
			s := strings.ToLower(id + "|" + dn)
			return strings.Contains(s, "heal") && strings.Contains(s, "progress")
		},
	},
	{
		ID: "generator_progress", Category: "survivor", ValueType: "percent", Weight: 21,
		Match: func(id, dn string) bool {
			s := strings.ToLower(id + "|" + dn)
			return strings.Contains(s, "generator") && strings.Contains(s, "progress")
		},
	},
}

func findRule(id, dn string) (StatRule, bool) {
	for _, r := range ruleSet {
		if r.Match(id, dn) {
			return r, true
		}
	}
	return StatRule{}, false
}

// inferStatRule applies heuristics to categorize unknown stats (fallback heuristic)
func inferStatRule(id, dn string) StatRule {
	s := strings.ToLower(id + "|" + dn)
	switch {
	case strings.Contains(s, "killer") || strings.Contains(s, "slasher") ||
		strings.Contains(s, "hook") || strings.Contains(s, "sacrifice") || strings.Contains(s, "mori"):
		return StatRule{Category: "killer", ValueType: "count", Weight: 100}
	case strings.Contains(s, "camper") || strings.Contains(s, "survivor") ||
		strings.Contains(s, "escape") || strings.Contains(s, "heal") ||
		strings.Contains(s, "repair") || strings.Contains(s, "generator"):
		return StatRule{Category: "survivor", ValueType: "count", Weight: 100}
	default:
		return StatRule{Category: "general", ValueType: "count", Weight: 100}
	}
}

// gradeMapping maps raw grade values to human readable grades
// TODO: Update with actual observed values from Steam API
var gradeMapping = map[int]Grade{
	0:  {Tier: "Ash", Sub: 4},
	1:  {Tier: "Ash", Sub: 3},
	2:  {Tier: "Ash", Sub: 2},
	3:  {Tier: "Ash", Sub: 1},
	4:  {Tier: "Bronze", Sub: 4},
	5:  {Tier: "Bronze", Sub: 3},
	6:  {Tier: "Bronze", Sub: 2},
	7:  {Tier: "Bronze", Sub: 1},
	8:  {Tier: "Silver", Sub: 4},
	9:  {Tier: "Silver", Sub: 3},
	10: {Tier: "Silver", Sub: 2},
	11: {Tier: "Silver", Sub: 1},
	12: {Tier: "Gold", Sub: 4},
	13: {Tier: "Gold", Sub: 3},
	14: {Tier: "Gold", Sub: 2},
	15: {Tier: "Gold", Sub: 1},
	16: {Tier: "Iridescent", Sub: 4},
	17: {Tier: "Iridescent", Sub: 3},
	18: {Tier: "Iridescent", Sub: 2},
	19: {Tier: "Iridescent", Sub: 1},
}

// MapPlayerStats maps raw Steam stats to structured response using schema as source of truth
func MapPlayerStats(ctx context.Context, steamID string, cacheManager cache.Cache, client *Client) (*PlayerStatsResponse, error) {
	if client == nil {
		return nil, fmt.Errorf("steam client is required")
	}

	// 1) Fetch schema for stats definitions
	schema, err := client.GetSchemaForGame(DBDAppID)
	if err != nil {
		log.Error("Failed to get stats schema", "error", err, "steam_id", steamID)
		return nil, fmt.Errorf("failed to get stats schema: %w", err)
	}

	if schema == nil || schema.AvailableGameStats.Stats == nil {
		log.Warn("Schema unavailable or empty for stats", "steam_id", steamID)
		return &PlayerStatsResponse{
			Stats:   []Stat{},
			Summary: map[string]interface{}{"total_stats": 0, "source": "empty_schema"},
		}, nil
	}

	// 2) Fetch user's actual stat values
	var userStats *SteamPlayerstats
	var apiErr *APIError
	
	appID, _ := strconv.Atoi(DBDAppID) // Convert string to int
	
	if cacheManager != nil {
		userStats, apiErr = client.GetUserStatsForGameCached(ctx, steamID, appID, cacheManager)
	} else {
		userStats, apiErr = client.GetUserStatsForGame(ctx, steamID, appID)
	}
	
	if apiErr != nil {
		log.Error("Failed to get user stats", "error", apiErr, "steam_id", steamID)
		return nil, fmt.Errorf("failed to get user stats: %w", apiErr)
	}

	// 3) Build lookup map: stat name -> value
	userStatsMap := make(map[string]float64)
	if userStats != nil && userStats.Stats != nil {
		for _, stat := range userStats.Stats {
			userStatsMap[stat.Name] = stat.Value
		}
	}

	// 4) Map each schema stat to our Stat struct
	mappedStats := make([]Stat, 0, len(schema.AvailableGameStats.Stats))
	
	for _, schemaStat := range schema.AvailableGameStats.Stats {
		id := schemaStat.Name
		dn := schemaStat.DisplayName
		raw := userStatsMap[id] // 0 if missing

		rule, ok := findRule(id, dn)
		if !ok {
			rule = inferStatRule(id, dn)
		}

		// Format the value
		formatted := formatValue(raw, rule.ValueType)

		st := Stat{
			ID:          id,
			DisplayName: dn,
			Value:       raw,
			Formatted:   formatted,
			Category:    rule.Category,
			ValueType:   rule.ValueType,
			SortWeight:  rule.Weight,
			Alias:       rule.Alias,
			MatchedBy:   rule.ID,
		}

		// telemetry: looks like grade but didn't get typed as grade
		if strings.Contains(strings.ToLower(id+"|"+dn), "grade") && st.ValueType != "grade" {
			log.Debug("Stat looks like grade but rule did not match", "id", id, "name", dn, "value", raw)
		}

		mappedStats = append(mappedStats, st)
	}

	// 5) Sort stats: killer -> survivor -> general, then by weight, then by display name
	sort.Slice(mappedStats, func(i, j int) bool {
		// Category order: killer, survivor, general
		categoryOrder := map[string]int{"killer": 0, "survivor": 1, "general": 2}
		catI, catJ := categoryOrder[mappedStats[i].Category], categoryOrder[mappedStats[j].Category]
		
		if catI != catJ {
			return catI < catJ
		}
		
		// Within category: sort by weight
		if mappedStats[i].SortWeight != mappedStats[j].SortWeight {
			return mappedStats[i].SortWeight < mappedStats[j].SortWeight
		}
		
		// Finally by display name for stability
		return mappedStats[i].DisplayName < mappedStats[j].DisplayName
	})

	// 6) Build summary
	summary := buildStatsSummary(mappedStats)

	// 7) Log warning if suspiciously low count
	if len(mappedStats) < 50 {
		log.Warn("Stats count unexpectedly low - schema likely incomplete",
			"mapped_count", len(mappedStats),
			"expected_minimum", 50,
			"steam_id", steamID,
			"suggestion", "Check Steam API availability or schema completeness")
	}

	log.Info("Player stats mapping completed",
		"total_stats", len(mappedStats),
		"steam_id", steamID,
		"schema_source", "steam_api")

	return &PlayerStatsResponse{
		Stats:   mappedStats,
		Summary: summary,
	}, nil
}

// decodeGrade converts raw grade value to human readable format
func decodeGrade(v float64) (Grade, string, string) {
	gradeCode := int(v)
	
	if grade, exists := gradeMapping[gradeCode]; exists {
		human := fmt.Sprintf("%s %s", grade.Tier, roman(grade.Sub))
		return grade, human, roman(grade.Sub)
	}
	
	// Unknown grade code - log for investigation
	log.Debug("Unknown grade code detected", 
		"grade_code", gradeCode, 
		"suggestion", "Consider adding to gradeMapping")
	
	// Return safe default
	defaultGrade := Grade{Tier: "Unranked", Sub: 0}
	return defaultGrade, "Unranked", ""
}

// roman converts 1-4 to Roman numerals I-IV
func roman(n int) string {
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

// formatValue formats a raw value according to its type
func formatValue(v float64, valueType string) string {
	switch valueType {
	case "percent":
		return fmt.Sprintf("%.1f%%", v)
	case "grade":
		_, human, _ := decodeGrade(v)
		return human
	case "level":
		return strconv.Itoa(int(v))
	case "duration":
		// If we ever get seconds, format as duration
		return formatDuration(int64(v))
	default: // "count"
		return formatInt(int(v))
	}
}

// formatInt formats an integer with commas for readability
func formatInt(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	
	str := strconv.Itoa(n)
	var result strings.Builder
	
	for i, char := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteRune(char)
	}
	
	return result.String()
}

// formatDuration formats seconds into human readable duration
func formatDuration(seconds int64) string {
	duration := time.Duration(seconds) * time.Second
	
	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm %ds", int(duration.Minutes()), int(duration.Seconds())%60)
	} else {
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}

// buildStatsSummary creates aggregate statistics
func buildStatsSummary(stats []Stat) map[string]interface{} {
	summary := map[string]interface{}{
		"total_stats":    len(stats),
		"killer_count":   0,
		"survivor_count": 0,
		"general_count":  0,
		"grade_stats":    []Stat{},
		"prestige_level": 0,
	}

	var gradeStats []Stat
	var maxPrestige float64

	for _, stat := range stats {
		switch stat.Category {
		case "killer":
			summary["killer_count"] = summary["killer_count"].(int) + 1
		case "survivor":
			summary["survivor_count"] = summary["survivor_count"].(int) + 1
		case "general":
			summary["general_count"] = summary["general_count"].(int) + 1
		}

		if stat.ValueType == "grade" {
			gradeStats = append(gradeStats, stat)
		}

		if stat.Alias == "highest_prestige" && stat.Value > maxPrestige {
			maxPrestige = stat.Value
		}
	}

	summary["grade_stats"] = gradeStats
	summary["prestige_level"] = int(maxPrestige)

	return summary
}
