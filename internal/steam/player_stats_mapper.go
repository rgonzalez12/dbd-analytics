package steam

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/cache"
	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

// Stat represents a single player statistic with metadata
type Stat struct {
	ID          string  `json:"id"`           // schema name
	DisplayName string  `json:"display_name"` // schema displayName
	Value       float64 `json:"value"`        // raw numeric
	Formatted   string  `json:"formatted"`    // human string (optional; derived)
	Category    string  `json:"category"`     // "killer" | "survivor" | "general"
	ValueType   string  `json:"value_type"`   // "count" | "float" | "grade" | "level" | "duration"
	SortWeight  int     `json:"sort_weight"`  // for stable ordering in UI
	Icon        string  `json:"icon,omitempty"`
	Alias       string  `json:"alias,omitempty"`      // e.g., killer_grade
	MatchedBy   string  `json:"matched_by,omitempty"` // rule id (for debugging)
}

// PlayerStatsResponse represents the complete stats response
type PlayerStatsResponse struct {
	Stats         []Stat                    `json:"stats"`
	Summary       map[string]interface{}    `json:"summary"` // totals you want (optional)
	UnmappedStats []map[string]interface{}  `json:"unmapped_stats,omitempty"` // stats that fell through rules/alias
}

// Aliases map provides stable display names for important DBD stats
var aliases = map[string]string{
	"DBD_CamperSkulls":                "Survivor Grade (Pips)",
	"DBD_KillerSkulls":                "Killer Grade (Pips)",
	"DBD_GeneratorPct_float":          "Generators Repaired (equivalent)",
	"DBD_HealPct_float":               "Survivors Healed (equivalent)",
	"DBD_BloodwebPoints":              "Bloodpoints Earned",
	"DBD_UnlockRanking":               "Survivor Grade",
	"DBD_SlasherTierIncrement":        "Killer Grade",
	"DBD_BloodwebMaxPrestigeLevel":    "Highest Prestige Level",
	"DBD_Escape":                      "Escapes",
	"DBD_EscapeThroughHatch":          "Escapes Through Hatch",
	"DBD_EscapeKO":                    "Escapes While Injured",
	"DBD_UnhookOrHeal":                "Unhooks and Heals",
	"DBD_UnhookOrHeal_PostExit":       "Post-Exit Saves",
	"DBD_SkillCheckSuccess":           "Successful Skill Checks",
	"DBD_CamperNewItem":               "Escaped with New Item",
	"DBD_CamperEscapeWithItemFrom":    "Escaped with Item from Others",
	"DBD_CamperFullLoadout":           "Survivor Full Loadout Matches",
	"DBD_CamperMaxScoreByCategory":    "Survivor Max Score by Category",
	"DBD_SacrificedCampers":           "Survivors Sacrificed",
	"DBD_KilledCampers":               "Survivors Killed (Mori)",
	"DBD_HitNearHook":                 "Hits Near Hook",
	"DBD_HookedAndEscape":             "Hooked Survivors Who Escaped",
	"DBD_ChainsawHit":                 "Chainsaw Hits",
	"DBD_UncloakAttack":               "Uncloak Attacks",
	"DBD_TrapPickup":                  "Trap Pickups",
	"DBD_SlasherFullLoadout":          "Killer Full Loadout Matches",
	"DBD_SlasherMaxScoreByCategory":   "Killer Max Score by Category",
	"DBD_MatchesPlayed":               "Matches Played",
	"DBD_MatchesWon":                  "Matches Won",
	"DBD_PerfectMatch":                "Perfect Matches",
	"DBD_OfferingsBurnt":              "Offerings Used",
	"DBD_MysteryBoxes":                "Mystery Boxes Opened",
}

// Grade represents decoded grade information
type Grade struct {
	Tier string // Bronze/Silver/Gold/Iridescent/Ash
	Sub  int    // 1..4
}

// isGradeField determines if a stat represents a grade based on ID or display name
func isGradeField(id, displayName string) bool {
	gradePattern := regexp.MustCompile(`(?i)grade|current.*(killer|survivor).*grade`)
	return gradePattern.MatchString(id) || gradePattern.MatchString(displayName)
}

// fallbackDisplayName creates a readable name from raw stat ID
func fallbackDisplayName(id string) string {
	// Strip DBD_ prefix
	name := strings.TrimPrefix(id, "DBD_")
	
	// Strip _float suffix and handle special cases
	name = strings.TrimSuffix(name, "_float")
	
	// Replace underscores with spaces
	name = strings.ReplaceAll(name, "_", " ")
	
	// Replace telemetry terms with proper names
	name = strings.ReplaceAll(name, "Camper", "Survivor")
	name = strings.ReplaceAll(name, "Slasher", "Killer")
	
	// Handle Pct -> equivalent suffix
	if strings.Contains(strings.ToLower(id), "pct") && strings.Contains(id, "_float") {
		name = strings.ReplaceAll(name, "Pct", "")
		name = strings.TrimSpace(name) + " (equivalent)"
	}
	
	// Title case
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	
	return strings.Join(words, " ")
}

// determineValueType determines the appropriate value type for a stat
func determineValueType(id, displayName string, _ float64) string {
	idLower := strings.ToLower(id)
	displayLower := strings.ToLower(displayName)
	
	// Grade detection
	if isGradeField(id, displayName) {
		return "grade"
	}
	
	// Duration fields
	if strings.Contains(idLower, "time") || strings.Contains(idLower, "duration") ||
		strings.Contains(displayLower, "time") || strings.Contains(displayLower, "duration") {
		return "duration"
	}
	
	// Level fields
	if strings.Contains(idLower, "level") || strings.Contains(idLower, "prestige") ||
		strings.Contains(displayLower, "level") || strings.Contains(displayLower, "prestige") {
		return "level"
	}
	
	// Float fields (including _Pct_float which are equivalents, not percentages)
	if strings.Contains(id, "_float") || strings.Contains(id, "Pct_float") {
		return "float"
	}
	
	// Default to count
	return "count"
}

// Grade mappings for killer grades (DBD_SlasherTierIncrement)
var gradeMapping = map[int]Grade{
	// Based on field-aware detection and actual Steam API values
	16:  {Tier: "Ash", Sub: 4},      // Ash IV
	17:  {Tier: "Ash", Sub: 3},      // Ash III
	18:  {Tier: "Ash", Sub: 2},      // Ash II
	19:  {Tier: "Ash", Sub: 1},      // Ash I
	20:  {Tier: "Bronze", Sub: 4},   // Bronze IV
	21:  {Tier: "Bronze", Sub: 3},   // Bronze III
	22:  {Tier: "Bronze", Sub: 2},   // Bronze II
	23:  {Tier: "Bronze", Sub: 1},   // Bronze I
	73:  {Tier: "Bronze", Sub: 4},   // Bronze IV (alternative)
	439: {Tier: "Bronze", Sub: 2},   // Bronze II
	640: {Tier: "Ash", Sub: 4},      // Ash IV
}

// survivorGradeMapping maps DBD_UnlockRanking values to survivor grades
var survivorGradeMapping = map[int]Grade{
	// Survivor grade mappings based on actual Steam API data
	7:    {Tier: "Ash", Sub: 4},        // Ash IV
	541:  {Tier: "Ash", Sub: 3},        // Ash III
	640:  {Tier: "Bronze", Sub: 1},     // Bronze I
	948:  {Tier: "Ash", Sub: 2},        // Ash II
	949:  {Tier: "Ash", Sub: 2},        // Ash II
	951:  {Tier: "Iridescent", Sub: 4}, // Iridescent IV
	1743: {Tier: "Ash", Sub: 1},        // Ash I
	2050: {Tier: "Silver", Sub: 1},     // Silver I
	2115: {Tier: "Ash", Sub: 4},        // Ash IV
	4226: {Tier: "Gold", Sub: 1},       // Gold I
	4227: {Tier: "Gold", Sub: 1},       // Gold I
	4228: {Tier: "Iridescent", Sub: 4}, // Iridescent IV
	4229: {Tier: "Iridescent", Sub: 4}, // Iridescent IV
	4230: {Tier: "Iridescent", Sub: 4}, // Iridescent IV
	4233: {Tier: "Iridescent", Sub: 3}, // Iridescent III
	8995: {Tier: "Iridescent", Sub: 4}, // Iridescent IV
}

// MapPlayerStats maps raw Steam stats to structured response using schema + user stats union
func MapPlayerStats(ctx context.Context, steamID string, cacheManager cache.Cache, client *Client) (*PlayerStatsResponse, error) {
	if client == nil {
		return nil, fmt.Errorf("steam client is required")
	}

	start := time.Now()
	log.Info("Starting player stats mapping", "steam_id", steamID)

	// 1) Fetch schema for stats definitions with forced English
	schema, err := client.GetSchemaForGame(DBDAppID)
	if err != nil {
		log.Warn("Failed to get stats schema, proceeding with user stats only", "error", err, "steam_id", steamID)
		// Don't fail completely - continue with user stats only
	}

	// 2) Fetch user's actual stat values
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
	schemaCount := 0
	if schema != nil && schema.AvailableGameStats.Stats != nil {
		for _, ss := range schema.AvailableGameStats.Stats {
			if ss.DisplayName != "" && ss.DisplayName != "Unknown" {
				schemaByID[ss.Name] = ss.DisplayName
			}
			schemaCount++
		}
	}

	// 4) Build user stats lookup map
	userByID := map[string]float64{}
	if userStats != nil && userStats.Stats != nil {
		for _, us := range userStats.Stats {
			userByID[us.Name] = us.Value
		}
	}

	// 5) Build union keyset: schemaStats ∪ userStats
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

	unionCount := len(keys)
	userOnlyCount := len(userByID) - len(seen) + len(keys) - len(schemaByID)

	// 6) Map each stat in the union with comprehensive rule detection
	mapped := make([]Stat, 0, len(keys))
	unmappedStats := make([]map[string]interface{}, 0)
	gradeStats := make([]map[string]interface{}, 0)

	for _, id := range keys {
		value, hasValue := userByID[id]
		if !hasValue {
			continue // Skip schema-only stats with no user value
		}

		schemaDisplayName := schemaByID[id]
		
		// Resolve display name priority: schema → alias → fallback
		var displayName, alias, matchedBy string
		var category, valueType string
		var sortWeight int

		if aliasName, hasAlias := aliases[id]; hasAlias {
			displayName = aliasName
			alias = id
			matchedBy = "alias"
		} else if schemaDisplayName != "" {
			displayName = schemaDisplayName
			matchedBy = "schema"
		} else {
			displayName = fallbackDisplayName(id)
			matchedBy = "fallback"
		}

		// Determine value type
		valueType = determineValueType(id, displayName, value)

		// Categorize stat
		category = categorizeStats(id, displayName)

		// Set sort weight
		sortWeight = getSortWeight(category, id)

		// Log grade-like stats for debugging
		gradePattern := regexp.MustCompile(`(?i)grade|skull|rank`)
		if gradePattern.MatchString(id) || gradePattern.MatchString(displayName) {
			gradeStats = append(gradeStats, map[string]interface{}{
				"id":           id,
				"display_name": displayName,
				"raw_value":    value,
				"value_type":   valueType,
			})
		}

		// Format value based on type
		formatted := formatValue(value, valueType, id)

		// Set specific aliases for important stats
		switch id {
		case "DBD_UnlockRanking":
			alias = "survivor_grade"
		case "DBD_SlasherTierIncrement":
			alias = "killer_grade"
		case "DBD_CamperSkulls":
			alias = "survivor_grade_pips"
		case "DBD_KillerSkulls":
			alias = "killer_grade_pips"
		case "DBD_BloodwebMaxPrestigeLevel":
			alias = "highest_prestige"
		}

		stat := Stat{
			ID:          id,
			DisplayName: displayName,
			Value:       value,
			Formatted:   formatted,
			Category:    category,
			ValueType:   valueType,
			SortWeight:  sortWeight,
			Alias:       alias,
			MatchedBy:   matchedBy,
		}

		mapped = append(mapped, stat)

		// Track unmapped stats (those using fallback naming)
		if matchedBy == "fallback" {
			unmappedStats = append(unmappedStats, map[string]interface{}{
				"id":           id,
				"display_name": displayName,
			})
		}
	}

	// 7) Sort stats: killer → survivor → general, then by weight, then by display name
	sort.Slice(mapped, func(i, j int) bool {
		if mapped[i].Category != mapped[j].Category {
			return categoryOrder(mapped[i].Category) < categoryOrder(mapped[j].Category)
		}
		if mapped[i].SortWeight != mapped[j].SortWeight {
			return mapped[i].SortWeight < mapped[j].SortWeight
		}
		return mapped[i].DisplayName < mapped[j].DisplayName
	})

	// 8) Build summary with important values
	summary := make(map[string]interface{})
	for _, stat := range mapped {
		switch stat.Alias {
		case "killer_grade":
			if stat.ValueType == "grade" {
				summary["killer_grade"] = stat.Formatted
			}
		case "survivor_grade":
			if stat.ValueType == "grade" {
				summary["survivor_grade"] = stat.Formatted
			}
		case "killer_grade_pips":
			summary["killer_pips"] = int(stat.Value)
		case "survivor_grade_pips":
			summary["survivor_pips"] = int(stat.Value)
		case "highest_prestige":
			// Clamp prestige at 100
			prestige := int(stat.Value)
			if prestige > 100 {
				prestige = 100
			}
			summary["prestige_max"] = prestige
		}
	}

	unmappedCount := len(unmappedStats)
	gradeCount := len(gradeStats)

	// Log comprehensive statistics
	duration := time.Since(start)
	log.Info("Player stats mapping completed",
		"steam_id", steamID,
		"schema_count", schemaCount,
		"user_only_count", userOnlyCount,
		"union_count", unionCount,
		"unmapped_count", unmappedCount,
		"grade_like_count", gradeCount,
		"duration", duration)

	if gradeCount > 0 {
		log.Debug("Grade-like stats detected", "steam_id", steamID, "grade_stats", gradeStats)
	}

	if unmappedCount > 0 {
		// Log first 10 unmapped stats
		logLimit := 10
		if unmappedCount < logLimit {
			logLimit = unmappedCount
		}
		log.Info("Unmapped stats detected", "steam_id", steamID, "sample_unmapped", unmappedStats[:logLimit])
	}

	response := &PlayerStatsResponse{
		Stats:   mapped,
		Summary: summary,
	}

	// Include unmapped stats in response if any exist
	if unmappedCount > 0 {
		response.UnmappedStats = unmappedStats
	}

	return response, nil
}

// categorizeStats determines the category (killer/survivor/general) for a stat
func categorizeStats(id, displayName string) string {
	idLower := strings.ToLower(id)
	displayLower := strings.ToLower(displayName)
	
	// Killer indicators
	if strings.Contains(idLower, "slasher") || strings.Contains(idLower, "killer") ||
		strings.Contains(displayLower, "killer") || strings.Contains(displayLower, "slasher") ||
		strings.Contains(idLower, "chainsaw") || strings.Contains(idLower, "uncloak") ||
		strings.Contains(idLower, "trap") || strings.Contains(idLower, "sacrifice") ||
		strings.Contains(idLower, "hook") {
		return "killer"
	}
	
	// Survivor indicators
	if strings.Contains(idLower, "camper") || strings.Contains(idLower, "survivor") ||
		strings.Contains(displayLower, "survivor") || strings.Contains(displayLower, "camper") ||
		strings.Contains(idLower, "escape") || strings.Contains(idLower, "generator") ||
		strings.Contains(idLower, "heal") || strings.Contains(idLower, "unhook") ||
		strings.Contains(idLower, "skill") {
		return "survivor"
	}
	
	return "general"
}

// getSortWeight determines sort weight based on category and importance
func getSortWeight(category, id string) int {
	// Grades get highest priority
	if strings.Contains(strings.ToLower(id), "grade") || 
		strings.Contains(strings.ToLower(id), "unlock") ||
		strings.Contains(strings.ToLower(id), "tier") {
		return 0
	}
	
	// Prestige gets high priority
	if strings.Contains(strings.ToLower(id), "prestige") {
		return 5
	}
	
	// Category-based weights
	switch category {
	case "killer":
		if strings.Contains(strings.ToLower(id), "skull") {
			return 1 // Killer pips
		}
		return 10
	case "survivor":
		if strings.Contains(strings.ToLower(id), "skull") {
			return 1 // Survivor pips
		}
		return 15
	default:
		return 20
	}
}

// categoryOrder returns numeric order for sorting categories
func categoryOrder(category string) int {
	switch category {
	case "killer":
		return 0
	case "survivor":
		return 1
	default:
		return 2
	}
}

// formatValue formats a raw value according to its type
func formatValue(v float64, valueType string, fieldID string) string {
	switch valueType {
	case "float":
		// Format floats with 1 decimal place
		return fmt.Sprintf("%.1f", v)
	case "grade":
		// Decode grade using existing logic
		_, human, _ := decodeGrade(v, fieldID)
		return human
	case "level":
		return strconv.Itoa(int(v))
	case "duration":
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

// decodeGrade converts raw grade value to human readable format using existing mappings
func decodeGrade(v float64, fieldID string) (Grade, string, string) {
	gradeCode := int(v)

	// Determine if this is killer or survivor based on field name
	isKillerGrade := strings.Contains(strings.ToLower(fieldID), "slasher") || 
					strings.Contains(strings.ToLower(fieldID), "killer")
	isSurvivorGrade := strings.Contains(strings.ToLower(fieldID), "unlock") || 
					  strings.Contains(strings.ToLower(fieldID), "survivor") ||
					  strings.Contains(strings.ToLower(fieldID), "camper")

	// Try killer grade mapping if it's a killer field
	if isKillerGrade {
		if grade, exists := gradeMapping[gradeCode]; exists {
			if grade.Tier == "Unranked" {
				return grade, "Unranked", ""
			}
			human := fmt.Sprintf("%s %s", grade.Tier, roman(grade.Sub))
			return grade, human, roman(grade.Sub)
		}
	}

	// Try survivor grade mapping if it's a survivor field
	if isSurvivorGrade {
		if grade, exists := survivorGradeMapping[gradeCode]; exists {
			human := fmt.Sprintf("%s %s", grade.Tier, roman(grade.Sub))
			return grade, human, roman(grade.Sub)
		}
	}

	// Unknown grade - return question mark
	return Grade{Tier: "Unknown", Sub: 1}, "?", "?"
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
