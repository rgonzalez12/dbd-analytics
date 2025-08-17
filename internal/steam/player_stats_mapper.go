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
	ValueType   string  `json:"value_type"`   // "count" | "percent" | "grade" | "level" | "duration"
	SortWeight  int     `json:"sort_weight"`  // for stable ordering in UI
	Icon        string  `json:"icon,omitempty"`
	Alias       string  `json:"alias,omitempty"`      // e.g., killer_grade
	MatchedBy   string  `json:"matched_by,omitempty"` // rule id (for debugging)
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

	// Survivor-specific stats
	{ // Escapes
		ID: "DBD_Escape", Category: "survivor", ValueType: "count", Weight: 10,
		Match: func(id, dn string) bool {
			return id == "DBD_Escape"
		},
	},
	{ // Escapes through hatch
		ID: "DBD_EscapeThroughHatch", Category: "survivor", ValueType: "count", Weight: 10,
		Match: func(id, dn string) bool {
			return id == "DBD_EscapeThroughHatch"
		},
	},
	{ // Escapes while injured
		ID: "DBD_EscapeKO", Category: "survivor", ValueType: "count", Weight: 10,
		Match: func(id, dn string) bool {
			return id == "DBD_EscapeKO"
		},
	},
	{ // Generator progress
		ID: "DBD_GeneratorPct_float", Category: "survivor", ValueType: "percent", Weight: 15,
		Match: func(id, dn string) bool {
			return id == "DBD_GeneratorPct_float"
		},
	},
	{ // Healing progress
		ID: "DBD_HealPct_float", Category: "survivor", ValueType: "percent", Weight: 15,
		Match: func(id, dn string) bool {
			return id == "DBD_HealPct_float"
		},
	},
	{ // Unhooks/saves
		ID: "DBD_UnhookOrHeal", Category: "survivor", ValueType: "count", Weight: 15,
		Match: func(id, dn string) bool {
			return id == "DBD_UnhookOrHeal"
		},
	},
	{ // Post-exit saves
		ID: "DBD_UnhookOrHeal_PostExit", Category: "survivor", ValueType: "count", Weight: 15,
		Match: func(id, dn string) bool {
			return id == "DBD_UnhookOrHeal_PostExit"
		},
	},
	{ // Skill checks
		ID: "DBD_SkillCheckSuccess", Category: "survivor", ValueType: "count", Weight: 20,
		Match: func(id, dn string) bool {
			return id == "DBD_SkillCheckSuccess"
		},
	},
	{ // Escaped with new item
		ID: "DBD_CamperNewItem", Category: "survivor", ValueType: "count", Weight: 25,
		Match: func(id, dn string) bool {
			return id == "DBD_CamperNewItem"
		},
	},
	{ // Escaped with item from someone else
		ID: "DBD_CamperEscapeWithItemFrom", Category: "survivor", ValueType: "count", Weight: 25,
		Match: func(id, dn string) bool {
			return id == "DBD_CamperEscapeWithItemFrom"
		},
	},
	{ // Survivor full loadout matches
		ID: "DBD_CamperFullLoadout", Category: "survivor", ValueType: "count", Weight: 30,
		Match: func(id, dn string) bool {
			return id == "DBD_CamperFullLoadout"
		},
	},
	{ // Max score by category (survivor)
		ID: "DBD_CamperMaxScoreByCategory", Category: "survivor", ValueType: "count", Weight: 35,
		Match: func(id, dn string) bool {
			return id == "DBD_CamperMaxScoreByCategory"
		},
	},

	// Killer-specific stats
	{ // Sacrifices
		ID: "DBD_SacrificedCampers", Category: "killer", ValueType: "count", Weight: 10,
		Match: func(id, dn string) bool {
			return id == "DBD_SacrificedCampers"
		},
	},
	{ // Mori kills
		ID: "DBD_KilledCampers", Category: "killer", ValueType: "count", Weight: 10,
		Match: func(id, dn string) bool {
			return id == "DBD_KilledCampers"
		},
	},
	{ // Hits near hook
		ID: "DBD_HitNearHook", Category: "killer", ValueType: "count", Weight: 15,
		Match: func(id, dn string) bool {
			return id == "DBD_HitNearHook"
		},
	},
	{ // Hooked and escaped (killer perspective)
		ID: "DBD_HookedAndEscape", Category: "killer", ValueType: "count", Weight: 15,
		Match: func(id, dn string) bool {
			return id == "DBD_HookedAndEscape"
		},
	},
	{ // Chainsaw hits
		ID: "DBD_ChainsawHit", Category: "killer", ValueType: "count", Weight: 20,
		Match: func(id, dn string) bool {
			return id == "DBD_ChainsawHit"
		},
	},
	{ // Uncloak attacks
		ID: "DBD_UncloakAttack", Category: "killer", ValueType: "count", Weight: 20,
		Match: func(id, dn string) bool {
			return id == "DBD_UncloakAttack"
		},
	},
	{ // Trap pickups
		ID: "DBD_TrapPickup", Category: "killer", ValueType: "count", Weight: 20,
		Match: func(id, dn string) bool {
			return id == "DBD_TrapPickup"
		},
	},
	{ // Killer full loadout matches
		ID: "DBD_SlasherFullLoadout", Category: "killer", ValueType: "count", Weight: 30,
		Match: func(id, dn string) bool {
			return id == "DBD_SlasherFullLoadout"
		},
	},
	{ // Max score by category (killer)
		ID: "DBD_SlasherMaxScoreByCategory", Category: "killer", ValueType: "count", Weight: 35,
		Match: func(id, dn string) bool {
			return id == "DBD_SlasherMaxScoreByCategory"
		},
	},

	// General stats
	{ // Bloodpoints
		ID: "DBD_BloodwebPoints", Category: "general", ValueType: "count", Weight: 5,
		Match: func(id, dn string) bool {
			return id == "DBD_BloodwebPoints"
		},
	},
	{ // Max bloodweb level
		ID: "DBD_BloodwebMaxLevel", Category: "general", ValueType: "level", Weight: 10,
		Match: func(id, dn string) bool {
			return id == "DBD_BloodwebMaxLevel"
		},
	},
	{ // Max perk level
		ID: "DBD_BloodwebPerkMaxLevel", Category: "general", ValueType: "level", Weight: 10,
		Match: func(id, dn string) bool {
			return id == "DBD_BloodwebPerkMaxLevel"
		},
	},
	{ // Ultra rare offerings
		ID: "DBD_BurnOffering_UltraRare", Category: "general", ValueType: "count", Weight: 25,
		Match: func(id, dn string) bool {
			return id == "DBD_BurnOffering_UltraRare"
		},
	},
	{ // Max bloodpoints in single category
		ID: "DBD_MaxBloodwebPointsOneCategory", Category: "general", ValueType: "count", Weight: 30,
		Match: func(id, dn string) bool {
			return id == "DBD_MaxBloodwebPointsOneCategory"
		},
	},
}

// displayNameOverrides provides human-readable names for common stats
var displayNameOverrides = map[string]string{
	"DBD_BloodwebMaxPrestigeLevel":        "Highest Prestige Level",
	"DBD_BloodwebPrestige3MaxLevel":       "Legacy Prestige (Old System)",
	"DBD_DailyRitualsRerolled":            "Daily Rituals Rerolled",
	"DBD_CamperMaxLevel":                  "Survivor Max Level",
	"DBD_SlasherMaxLevel":                 "Killer Max Level",
	"DBD_Chapter9Slasher_Stat1":           "Plague Vomit Hits",
	"DBD_Chapter9Camper_Stat1":            "Cleansing Pool Uses",
	"DBD_Chapter9Camper_Stat2":            "Infection Cleanses",
	"DBD_Stat_Camper_DB_TotalDistanceRun": "Total Distance Run",
	"DBD_Stat_Slasher_TotalHooks":         "Total Hooks",
	"DBD_RankResetLastTime":               "Last Rank Reset",
	"DBD_Stat_Camper_DB_Unhooks":          "Unhooks Performed",
	"DBD_Stat_Camper_DB_HealOthers":       "Others Healed",
	"DBD_Stat_Camper_DB_Escapes":          "Successful Escapes",
	"DBD_Stat_Slasher_DB_KillKillers":     "Kills as Killer",
	"DBD_Stat_Slasher_DB_TimesKilled":     "Times Killed Others",
	"DBD_UnlockRanking":                   "Survivor Grade",
	"DBD_SlasherTierIncrement":            "Killer Grade",
	"DBD_Bloodwebpoints":                  "Bloodpoints",
	"DBD_Bloodwebmaxlevel":                "Max Bloodweb Level",
	"DBD_Bloodwebperkmaxlevel":            "Max Perk Level",
	"DBD_Chainsawhit":                     "Chainsaw Hits",
	"DBD_Skillchecksuccess":               "Skill Checks Succeeded",
	"DBD_Uncloakattack":                   "Uncloak Attacks",
	"DBD_Trappickup":                      "Trap Pickups",
	"DBD_Burnoffering_Ultrarare":          "Ultra Rare Offerings Used",
	"DBD_Maxbloodwebpointsonecategory":    "Max Bloodpoints Single Category",
}

// tokenDictionary maps common DBD terms to human-readable equivalents
var tokenDictionary = map[string]string{
	// Common DBD terms
	"bloodwebpoints":            "Bloodpoints",
	"bloodwebmaxlevel":          "Max Bloodweb Level",
	"bloodwebperkmaxlevel":      "Max Perk Level",
	"ultrarare":                 "Ultra Rare",
	"obsession":                 "Obsession",
	"hatch":                     "Hatch",
	"basement":                  "Basement",
	"generator":                 "Generator",
	"generators":                "Generators",
	"sacrificed":                "Sacrificed",
	"sacrificedcampers":         "Survivors Sacrificed",
	"mori":                      "Mori",
	"healdying":                 "Healed From Dying",
	"unhook":                    "Unhook",
	"unhooks":                   "Unhooks",
	"escape":                    "Escape",
	"escapes":                   "Escapes",
	"escapeko":                  "Escape While Injured",
	"escapethroughhatch":        "Escape Through Hatch",
	"skillcheck":                "Skill Check",
	"skillchecks":               "Skill Checks",
	"skillchecksuccess":         "Skill Checks Succeeded",
	"chainsaw":                  "Chainsaw",
	"chainsawhit":               "Chainsaw Hits",
	"uncloakattack":             "Uncloak Attacks",
	"trappickup":                "Trap Pickups",
	"hookedandescape":           "Hooked And Escaped",
	"hitnearhook":               "Hits Near Hook",
	"killedcampers":             "Survivors Killed",
	"camperfullloadout":         "Survivor Full Loadout",
	"slasherfullloadout":        "Killer Full Loadout",
	"maxscorebycategory":        "Max Score By Category",
	"campermaxscorebycategory":  "Survivor Max Score By Category",
	"slashermaxscorebycategory": "Killer Max Score By Category",
	"generatorpct":              "Generator Progress",
	"healpct":                   "Healing Progress",
	"prestige":                  "Prestige",
	"prestigelevel":             "Prestige Level",
	"maxprestigelevel":          "Max Prestige Level",
	"camper":                    "Survivor",
	"slasher":                   "Killer",
	"chapter":                   "Chapter",
	"dlc":                       "DLC",
	"event":                     "Event",
	"stat":                      "",
	"dbd":                       "",
	"db":                        "",
	"count":                     "",
	"maxlevel":                  "Max Level",
	"iam":                       "",
	"idx":                       "",
	"float":                     "",
}

// labelForStat chooses the best display name for a stat
func labelForStat(id, schemaDisplayName string) (label string, matched bool) {
	// 1) If schemaDisplayName is non-empty, use it
	if schemaDisplayName != "" {
		return schemaDisplayName, true
	}

	// 2) Check displayNameOverrides
	if override, exists := displayNameOverrides[id]; exists {
		return override, true
	}

	// 3) Construct from ID using normalizeIDToTitle
	return normalizeIDToTitle(id), false
}

// normalizeIDToTitle converts DBD-style IDs to human titles
func normalizeIDToTitle(id string) string {
	if id == "" {
		return "Unknown Stat"
	}

	// Convert to lowercase for processing
	normalized := strings.ToLower(id)

	// Remove common prefixes
	prefixes := []string{"dbd_", "stat_", "db_", "dbdstat_"}
	for _, prefix := range prefixes {
		normalized = strings.TrimPrefix(normalized, prefix)
	}

	// Split on underscores and camelCase boundaries
	parts := splitIDParts(normalized)

	// Process each part through token dictionary and filtering
	var cleanParts []string
	for _, part := range parts {
		// Skip noisy tokens
		if isNoisyToken(part) {
			continue
		}

		// Apply token dictionary
		if replacement, exists := tokenDictionary[part]; exists {
			if replacement != "" { // Skip empty replacements
				cleanParts = append(cleanParts, replacement)
			}
		} else {
			// Capitalize first letter
			cleanParts = append(cleanParts, strings.Title(part))
		}
	}

	// Join and clean up
	result := strings.Join(cleanParts, " ")

	// Remove duplicate words
	result = removeDuplicateWords(result)

	// Fallback if result is empty
	if strings.TrimSpace(result) == "" {
		return strings.Title(strings.ReplaceAll(id, "_", " "))
	}

	return result
}

// splitIDParts splits an ID on underscores and camelCase boundaries
func splitIDParts(s string) []string {
	// First split on underscores
	parts := strings.Split(s, "_")

	var result []string
	for _, part := range parts {
		// Further split on camelCase boundaries
		camelParts := splitCamelCase(part)
		result = append(result, camelParts...)
	}

	return result
}

// splitCamelCase splits a string on camelCase boundaries
func splitCamelCase(s string) []string {
	if s == "" {
		return []string{}
	}

	// Use regex to split on camelCase boundaries
	re := regexp.MustCompile(`([a-z])([A-Z])`)
	s = re.ReplaceAllString(s, `${1}_${2}`)

	return strings.Split(strings.ToLower(s), "_")
}

// isNoisyToken checks if a token should be filtered out
func isNoisyToken(token string) bool {
	// Filter out common noise
	noisePatterns := []string{
		"^event\\d+$",
		"^idx\\d+$",
		"^stat\\d+$",
		"^\\d+$", // pure numbers
	}

	for _, pattern := range noisePatterns {
		matched, _ := regexp.MatchString(pattern, token)
		if matched {
			return true
		}
	}

	// Also filter very short meaningless tokens
	if len(token) <= 1 {
		return true
	}

	return false
}

// removeDuplicateWords removes consecutive duplicate words from a string
func removeDuplicateWords(s string) string {
	words := strings.Fields(s)
	if len(words) <= 1 {
		return s
	}

	var result []string
	result = append(result, words[0])

	for i := 1; i < len(words); i++ {
		if !strings.EqualFold(words[i], words[i-1]) {
			result = append(result, words[i])
		}
	}

	return strings.Join(result, " ")
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
// Based on confirmed user data from in-game screenshots
var gradeMapping = map[int]Grade{
	// Killer grade mappings - confirmed values
	0:   {Tier: "Unranked", Sub: 0}, // Unranked players
	16:  {Tier: "Ash", Sub: 4},      // Ash IV
	29:  {Tier: "Ash", Sub: 3},      // Ash III
	65:  {Tier: "Bronze", Sub: 2},   // Bronze II
	73:  {Tier: "Bronze", Sub: 4},   // Bronze IV
	93:  {Tier: "Ash", Sub: 4},      // Ash IV - confirmed user data
	300: {Tier: "Ash", Sub: 4},      // Ash IV - confirmed user data
	439: {Tier: "Bronze", Sub: 2},   // Bronze II

	// Pattern observed: Lower values for lower tiers
}

// survivorGradeMapping maps DBD_UnlockRanking values to survivor grades
// Based on confirmed user data from in-game screenshots
// NOTE: DBD ranking goes IV→III→II→I (4 is lowest, 1 is highest within each tier)
var survivorGradeMapping = map[int]Grade{
	// Survivor grade mappings - confirmed values
	7:    {Tier: "Ash", Sub: 4},        // Ash IV
	541:  {Tier: "Ash", Sub: 3},        // Ash III - confirmed user data
	948:  {Tier: "Ash", Sub: 2},        // Ash II
	949:  {Tier: "Ash", Sub: 2},        // Ash II
	951:  {Tier: "Iridescent", Sub: 4}, // Iridescent IV - confirmed current
	1743: {Tier: "Ash", Sub: 1},        // Ash I
	2050: {Tier: "Silver", Sub: 1},     // Silver I
	2115: {Tier: "Ash", Sub: 4},        // Ash IV
	4226: {Tier: "Gold", Sub: 1},       // Gold I
	4227: {Tier: "Gold", Sub: 1},       // Gold I
	4228: {Tier: "Iridescent", Sub: 4}, // Iridescent IV
	4229: {Tier: "Iridescent", Sub: 4}, // Iridescent IV
	4230: {Tier: "Iridescent", Sub: 4}, // Iridescent IV
	4233: {Tier: "Iridescent", Sub: 3}, // Iridescent III - confirmed user data
	8995: {Tier: "Iridescent", Sub: 4}, // Iridescent IV - confirmed user data

	// Pattern observed: Values can vary within same rank tiers
	// TODO: Need more data for complete mapping coverage
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

	// 6) Map each stat in the union
	mapped := make([]Stat, 0, len(keys))
	unlabeledStats := make([]map[string]interface{}, 0, 30) // For diagnostics
	unruledCount := 0

	for _, id := range keys {
		schemaDisplayName := schemaByID[id]
		raw := userByID[id] // 0 if missing

		// Use new labeling pipeline
		displayName, labelMatched := labelForStat(id, schemaDisplayName)

		// Try to find explicit rule first
		rule, ruleMatched := findRule(id, schemaDisplayName)
		if !ruleMatched {
			rule = inferStatRule(id, schemaDisplayName)
		}

		// Track unruled stats
		if rule.ID == "" {
			unruledCount++
		}

		st := Stat{
			ID:          id,
			DisplayName: displayName,
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

		// Collect unlabeled examples for diagnostics
		if !labelMatched && len(unlabeledStats) < 30 {
			unlabeledStats = append(unlabeledStats, map[string]interface{}{
				"id":               id,
				"value":            raw,
				"normalized_label": normalizeIDToTitle(id),
			})
		}

		// Diagnostic: "looks like grade" but not typed as grade
		if strings.Contains(strings.ToLower(id+"|"+displayName), "grade") && st.ValueType != "grade" {
			log.Debug("Looks like grade but not typed as grade", "id", id, "name", displayName, "value", raw)
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

	// 8) Build summary with diagnostics
	summary := buildStatsSummary(mapped, unruledCount, unlabeledStats)

	// Log diagnostic information
	log.Debug("Stats processing complete",
		"total_stats", len(mapped),
		"unruled_count", unruledCount,
		"unlabeled_examples_count", len(unlabeledStats),
		"killer_grade", summary["killer_grade"],
		"survivor_grade", summary["survivor_grade"],
		"prestige_level", summary["prestige_level"])

	// Log top unlabeled examples for debugging
	for i, example := range unlabeledStats {
		if i >= 10 { // Limit log spam
			break
		}
		log.Debug("Unlabeled stat example",
			"id", example["id"],
			"value", example["value"],
			"normalized_label", example["normalized_label"])
	}

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
		// Handle special case for unranked
		if grade.Tier == "Unranked" {
			log.Info("Killer grade decoded as unranked", "raw_value", gradeCode)
			return grade, "Unranked", ""
		}
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

// buildStatsSummary creates aggregate statistics with diagnostics
func buildStatsSummary(stats []Stat, unruledCount int, unlabeledStats []map[string]interface{}) map[string]interface{} {
	summary := map[string]interface{}{
		"total_stats":        len(stats),
		"killer_count":       0,
		"survivor_count":     0,
		"general_count":      0,
		"grade_stats":        []Stat{},
		"prestige_level":     0,
		"unruled_count":      unruledCount,
		"unlabeled_examples": unlabeledStats,
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
	}

	summary["grade_stats"] = gradeStats
	// Cap prestige level at 100 as specified
	prestigeLevel := int(maxPrestige)
	if prestigeLevel > 100 {
		prestigeLevel = 100
	}
	summary["prestige_level"] = prestigeLevel

	// Log grade warnings if missing
	if summary["killer_grade"] == nil {
		log.Warn("Killer grade missing from summary", "unruled_count", unruledCount)
		summary["killer_grade"] = "Unranked"
	}
	if summary["survivor_grade"] == nil {
		log.Warn("Survivor grade missing from summary", "unruled_count", unruledCount)
		summary["survivor_grade"] = "Unranked"
	}

	return summary
}
