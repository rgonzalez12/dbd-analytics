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
	{ // Survivor grade field - DBD_UnlockRanking appears to be survivor/overall grade
		ID: "DBD_UnlockRanking", Category: "survivor", ValueType: "grade", Weight: 0, Alias: "survivor_grade",
		Match: func(id, dn string) bool {
			return id == "DBD_UnlockRanking"
		},
	},
	{ // Killer grade field - DBD_SlasherTierIncrement is killer-specific progression
		ID: "DBD_SlasherTierIncrement", Category: "killer", ValueType: "grade", Weight: 0, Alias: "killer_grade",
		Match: func(id, dn string) bool {
			return id == "DBD_SlasherTierIncrement"
		},
	},
	{ // highest prestige level
		ID: "highest_prestige", Category: "general", ValueType: "level", Weight: 5, Alias: "highest_prestige",
		Match: func(id, dn string) bool {
			s := strings.ToLower(id + "|" + dn)
			return strings.Contains(s, "prestige") && (strings.Contains(s, "highest") || strings.Contains(s, "max")) ||
				id == "DBD_BloodwebMaxPrestigeLevel" // explicit match for known field
		},
	},
	// Conservative percent rules removed for now - add back only when verified by actual payloads
}

// displayNameOverrides provides human-readable names for common stats
var displayNameOverrides = map[string]string{
	"DBD_BloodwebMaxPrestigeLevel":      "Highest Prestige Level",
	"DBD_BloodwebPrestige3MaxLevel":     "Legacy Prestige (Old System)",
	"DBD_DailyRitualsRerolled":          "Daily Rituals Rerolled",
	"DBD_CamperMaxLevel":                "Survivor Max Level",
	"DBD_SlasherMaxLevel":               "Killer Max Level",
	"DBD_Chapter9Slasher_Stat1":         "Plague Vomit Hits",
	"DBD_Chapter9Camper_Stat1":          "Cleansing Pool Uses",
	"DBD_Chapter9Camper_Stat2":          "Infection Cleanses",
	"DBD_Stat_Camper_DB_TotalDistanceRun": "Total Distance Run",
	"DBD_Stat_Slasher_TotalHooks":       "Total Hooks",
	"DBD_RankResetLastTime":             "Last Rank Reset",
	"DBD_Stat_Camper_DB_Unhooks":        "Unhooks Performed",
	"DBD_Stat_Camper_DB_HealOthers":     "Others Healed",
	"DBD_Stat_Camper_DB_Escapes":        "Successful Escapes",
	"DBD_Stat_Slasher_DB_KillKillers":   "Kills as Killer",
	"DBD_Stat_Slasher_DB_TimesKilled":   "Times Killed Others",
	"DBD_UnlockRanking":                 "Survivor Grade",
	"DBD_SlasherTierIncrement":          "Killer Grade",
	"DBD_Bloodwebpoints":                "Bloodpoints",
	"DBD_Bloodwebmaxlevel":              "Max Bloodweb Level",
	"DBD_Bloodwebperkmaxlevel":          "Max Perk Level",
	"DBD_Chainsawhit":                   "Chainsaw Hits",
	"DBD_Skillchecksuccess":             "Skill Checks Succeeded",
	"DBD_Uncloakattack":                 "Uncloak Attacks",
	"DBD_Trappickup":                    "Trap Pickups",
	"DBD_Burnoffering_Ultrarare":        "Ultra Rare Offerings Used",
	"DBD_Maxbloodwebpointsonecategory":  "Max Bloodpoints Single Category",
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
// Based on user feedback: DBD_SlasherTierIncrement=16 corresponds to Ash IV (lowest rank)
// This means the mapping is inverted from what we initially assumed
var gradeMapping = map[int]Grade{
	// User confirmed: 16 = Ash IV (lowest killer rank)
	16: {Tier: "Ash", Sub: 4},
	17: {Tier: "Ash", Sub: 3},
	18: {Tier: "Ash", Sub: 2},
	19: {Tier: "Ash", Sub: 1},
	20: {Tier: "Bronze", Sub: 4},
	21: {Tier: "Bronze", Sub: 3},
	22: {Tier: "Bronze", Sub: 2},
	23: {Tier: "Bronze", Sub: 1},
	24: {Tier: "Silver", Sub: 4},
	25: {Tier: "Silver", Sub: 3},
	26: {Tier: "Silver", Sub: 2},
	27: {Tier: "Silver", Sub: 1},
	28: {Tier: "Gold", Sub: 4},
	29: {Tier: "Gold", Sub: 3},
	30: {Tier: "Gold", Sub: 2},
	31: {Tier: "Gold", Sub: 1},
	32: {Tier: "Iridescent", Sub: 4},
	33: {Tier: "Iridescent", Sub: 3},
	34: {Tier: "Iridescent", Sub: 2},
	35: {Tier: "Iridescent", Sub: 1},
}

// survivorGradeMapping maps DBD_UnlockRanking values to survivor grades
// Based on confirmed user data from in-game screenshots
var survivorGradeMapping = map[int]Grade{
	// CONFIRMED data points from user screenshots:
	4226: {Tier: "Gold", Sub: 1},     // Previous grade - Gold I
	4227: {Tier: "Gold", Sub: 1},     // Gold I (slight variation)
	4228: {Tier: "Iridescent", Sub: 4}, // CURRENT: User promoted to Iridescent IV
	
	// TODO: Collect more data points from different players to build complete mapping
	// The survivor grade system appears to use much higher numbers than killer grades
}

// MapPlayerStats maps raw Steam stats to structured response using schema as source of truth
func MapPlayerStats(ctx context.Context, steamID string, cacheManager cache.Cache, client *Client) (*PlayerStatsResponse, error) {
	if client == nil {
		return nil, fmt.Errorf("steam client is required")
	}

	// 1) Fetch schema for stats definitions (don't bail if empty)
	schema, err := client.GetSchemaForGame(DBDAppID)
	if err != nil {
		log.Error("Failed to get stats schema", "error", err, "steam_id", steamID)
		return nil, fmt.Errorf("failed to get stats schema: %w", err)
	}

	// 2) Fetch user's actual stat values with safe DBDAppID parsing
	var userStats *SteamPlayerstats
	var apiErr *APIError
	
	appID, parseErr := strconv.Atoi(DBDAppID)
	if parseErr != nil || appID == 0 {
		log.Error("Invalid DBDAppID; defaulting", "DBDAppID", DBDAppID, "err", parseErr)
		appID = 381210
	}
	
	if cacheManager != nil {
		userStats, apiErr = client.GetUserStatsForGameCached(ctx, steamID, appID, cacheManager)
	} else {
		userStats, apiErr = client.GetUserStatsForGame(ctx, steamID, appID)
	}
	
	if apiErr != nil {
		log.Error("Failed to get user stats", "error", apiErr, "steam_id", steamID)
		return nil, fmt.Errorf("failed to get user stats: %w", apiErr)
	}

	// 3) Build schema lookup map
	schemaByID := map[string]string{}
	if schema != nil && schema.AvailableGameStats.Stats != nil {
		for _, ss := range schema.AvailableGameStats.Stats {
			schemaByID[ss.Name] = ss.DisplayName
		}
	}

	// 4) Build user stats lookup map
	userByID := map[string]float64{}
	if userStats != nil && userStats.Stats != nil {
		for _, us := range userStats.Stats {
			userByID[us.Name] = us.Value
		}
	}

	// Don't bail on empty schema - proceed with union logic
	if schema == nil || schema.AvailableGameStats.Stats == nil {
		log.Warn("Schema unavailable/empty; mapping from user stats only", "steam_id", steamID)
		// proceed with union logic; schemaByID will be empty, userByID will drive output
	}

	// 5) Build union keyset
	keys := make([]string, 0, len(schemaByID)+len(userByID))
	seen := map[string]struct{}{}
	for k := range schemaByID {
		keys = append(keys, k)
		seen[k] = struct{}{}
	}
	for k := range userByID {
		if _, ok := seen[k]; !ok {
			keys = append(keys, k)
		}
	}

	// Helper to humanize IDs when schema display name is missing
	humanizeID := func(id string) string {
		s := strings.TrimSpace(strings.ReplaceAll(id, "_", " "))
		if s == "" {
			return id
		}
		parts := strings.Fields(strings.ToLower(s))
		for i := range parts {
			p := parts[i]
			if len(p) > 0 {
				parts[i] = strings.ToUpper(p[:1]) + p[1:]
			}
		}
		return strings.Join(parts, " ")
	}

	// 6) Map each stat in the union
	mapped := make([]Stat, 0, len(keys))
	
	// DEBUG: Log all stat names to find separate killer/survivor grades
	log.Info("DEBUG: All available stats", "total_count", len(keys))
	
	// Look specifically for tier/increment fields that might be separate grades
	var killerTierField, survivorTierField string
	var killerTierValue, survivorTierValue float64
	
	for i, id := range keys {
		value := userByID[id]
		idLower := strings.ToLower(id)
		
		// Look for specific tier increment fields
		if strings.Contains(idLower, "slashertierincrement") {
			killerTierField = id
			killerTierValue = value
			log.Info("DEBUG: Found killer tier field", "field", id, "value", value)
		}
		if strings.Contains(idLower, "campertierincrement") || strings.Contains(idLower, "survivortierincrement") || id == "DBD_UnlockRanking" {
			survivorTierField = id
			survivorTierValue = value
			log.Info("DEBUG: Found survivor tier field", "field", id, "value", value)
		}
		
		// Look for grade/rank fields OR killer/survivor specific fields
		if strings.Contains(idLower, "rank") || strings.Contains(idLower, "grade") || 
		   strings.Contains(idLower, "slasher") || strings.Contains(idLower, "camper") ||
		   strings.Contains(idLower, "killer") || strings.Contains(idLower, "survivor") {
			log.Info("DEBUG: Potential grade/role field", "index", i, "id", id, "value", value)
		}
	}
	
	// Log findings about tier fields
	log.Info("DEBUG: Tier field analysis", "killer_field", killerTierField, "killer_value", killerTierValue, "survivor_field", survivorTierField, "survivor_value", survivorTierValue)
	
	for _, id := range keys {
		dn := schemaByID[id]
		if dn == "" {
			dn = humanizeID(id)
		}
		
		// Use display name overrides for better names
		if override, exists := displayNameOverrides[id]; exists {
			dn = override
		}
		
		raw := userByID[id] // 0 if missing

		rule, ok := findRule(id, dn)
		if !ok {
			rule = inferStatRule(id, dn)
		}

		st := Stat{
			ID:          id,
			DisplayName: dn,
			Value:       raw,
			Formatted:   formatValue(raw, rule.ValueType),
			Category:    rule.Category,
			ValueType:   rule.ValueType,
			SortWeight:  rule.Weight,
			Alias:       rule.Alias,
			MatchedBy:   rule.ID,
		}

		// Skip the legacy prestige stat if we have the newer one
		if id == "DBD_BloodwebPrestige3MaxLevel" {
			if _, hasMaxPrestige := userByID["DBD_BloodwebMaxPrestigeLevel"]; hasMaxPrestige {
				continue // Skip legacy stat when newer one exists
			}
		}

		// Optional: drop pure unknown zeroes to reduce noise
		// if st.Value == 0 && st.Alias == "" && schemaByID[id] == "" { continue }

		// Diagnostic: "looks like grade" but not typed as grade
		if strings.Contains(strings.ToLower(id+"|"+dn), "grade") && st.ValueType != "grade" {
			log.Debug("Looks like grade but not typed as grade", "id", id, "name", dn, "value", raw)
		}

		mapped = append(mapped, st)
	}

	// 7) Sort stats: killer -> survivor -> general, then by weight, then by display name
	sort.Slice(mapped, func(i, j int) bool {
		// Category order: killer, survivor, general
		categoryOrder := map[string]int{"killer": 0, "survivor": 1, "general": 2}
		catI, catJ := categoryOrder[mapped[i].Category], categoryOrder[mapped[j].Category]
		
		if catI != catJ {
			return catI < catJ
		}
		
		// Within category: sort by weight
		if mapped[i].SortWeight != mapped[j].SortWeight {
			return mapped[i].SortWeight < mapped[j].SortWeight
		}
		
		// Finally by display name for stability
		return mapped[i].DisplayName < mapped[j].DisplayName
	})

	// 8) Build summary
	summary := buildStatsSummary(mapped)

	// 9) Enhanced diagnostics for low stats count
	if len(mapped) < 50 {
		sampleSchema, i := make([]string, 0, 10), 0
		for k := range schemaByID {
			sampleSchema = append(sampleSchema, k)
			i++
			if i == 10 {
				break
			}
		}
		sampleUser, j := make([]string, 0, 10), 0
		for k := range userByID {
			sampleUser = append(sampleUser, k)
			j++
			if j == 10 {
				break
			}
		}
		log.Warn("Low stats mapped; check mismatches",
			"mapped_count", len(mapped),
			"schema_keys", len(schemaByID), "user_keys", len(userByID),
			"schema_sample", sampleSchema, "user_sample", sampleUser,
		)
	}

	log.Info("Player stats mapping completed",
		"total_stats", len(mapped),
		"steam_id", steamID,
		"schema_source", "union_mapping")

	return &PlayerStatsResponse{
		Stats:   mapped,
		Summary: summary,
	}, nil
}

// decodeGrade converts raw grade value to human readable format
func decodeGrade(v float64) (Grade, string, string) {
	gradeCode := int(v)
	
	// Check if this is a killer grade (DBD_SlasherTierIncrement) - typically 16-35 range
	if grade, exists := gradeMapping[gradeCode]; exists {
		human := fmt.Sprintf("%s %s", grade.Tier, roman(grade.Sub))
		log.Info("Killer grade decoded successfully", "raw_value", gradeCode, "tier", grade.Tier, "sub", grade.Sub)
		return grade, human, roman(grade.Sub)
	}
	
	// Check if this is a survivor grade (DBD_UnlockRanking) - uses different encoding
	if grade, exists := survivorGradeMapping[gradeCode]; exists {
		human := fmt.Sprintf("%s %s", grade.Tier, roman(grade.Sub))
		log.Info("Survivor grade decoded successfully", "raw_value", gradeCode, "tier", grade.Tier, "sub", grade.Sub)
		return grade, human, roman(grade.Sub)
	}
	
	// Unknown grade code - determine if it's likely killer or survivor based on value range
	if gradeCode >= 1000 {
		log.Info("Unknown survivor grade detected", "grade_code", gradeCode, "field_type", "likely_DBD_UnlockRanking")
		return Grade{Tier: "Unknown", Sub: 1}, fmt.Sprintf("Unknown Survivor (%d)", gradeCode), "?"
	} else {
		log.Info("Unknown killer grade detected", "grade_code", gradeCode, "field_type", "likely_DBD_SlasherTierIncrement")
		return Grade{Tier: "Unknown", Sub: 1}, fmt.Sprintf("Unknown Killer (%d)", gradeCode), "?"
	}
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
		"unruled_count":  0,
	}

	var gradeStats []Stat
	var maxPrestige float64
	unruled := 0

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

		// Use alias-based detection for prestige
		if stat.Alias == "highest_prestige" && stat.Value > maxPrestige {
			maxPrestige = stat.Value
		}

		// Surface current grades in summary for header tiles
		if stat.Alias == "killer_grade" {
			summary["killer_grade"] = stat.Formatted
		}
		if stat.Alias == "survivor_grade" {
			summary["survivor_grade"] = stat.Formatted
		}

		// Count unruled stats for troubleshooting
		if stat.MatchedBy == "" {
			unruled++
		}
	}

	summary["grade_stats"] = gradeStats
	summary["prestige_level"] = int(maxPrestige)
	summary["unruled_count"] = unruled

	return summary
}
