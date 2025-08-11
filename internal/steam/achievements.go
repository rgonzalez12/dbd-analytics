package steam

import (
	"strings"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/models"
)

// AdeptAchievementMapping maps Steam achievement API names to character names and types
var AdeptAchievementMapping = map[string]AdeptCharacter{
	// Base Game Survivors
	"ACH_UNLOCK_DWIGHT_PERKS":    {Name: "dwight", Type: "survivor"},
	"ACH_UNLOCK_MEG_PERKS":       {Name: "meg", Type: "survivor"},
	"ACH_UNLOCK_CLAUDETTE_PERKS": {Name: "claudette", Type: "survivor"},
	"ACH_USE_JAKE_PERKS":         {Name: "jake", Type: "survivor"},

	// DLC Survivors
	"ACH_USE_NEA_PERKS":           {Name: "nea", Type: "survivor"},
	"ACH_DLC2_SURVIVOR_1":         {Name: "laurie", Type: "survivor"},
	"ACH_DLC3_SURVIVOR_3":         {Name: "ace", Type: "survivor"},
	"SURVIVOR7_ACHIEVEMENT_3":     {Name: "bill", Type: "survivor"},
	"ACH_DLC4_SURVIVOR_3":         {Name: "feng", Type: "survivor"},
	"ACH_DLC5_SURVIVOR_3":         {Name: "david", Type: "survivor"},
	"ACH_DLC7_SURVIVOR_3":         {Name: "quentin", Type: "survivor"},
	"ACH_DLC8_SURVIVOR_3":         {Name: "tapp", Type: "survivor"},
	"ACH_DLC9_SURVIVOR_3":         {Name: "kate", Type: "survivor"},
	"ACH_CHAPTER9_SURVIVOR_3":     {Name: "adam", Type: "survivor"},
	"ACH_CHAPTER10_SURVIVOR_3":    {Name: "jeff", Type: "survivor"},
	"ACH_CHAPTER11_SURVIVOR_3":    {Name: "jane", Type: "survivor"},
	"ACH_CHAPTER12_SURVIVOR_3":    {Name: "ash", Type: "survivor"},
	"ACH_CHAPTER14_SURVIVOR_3":    {Name: "yui", Type: "survivor"},
	"NEW_ACHIEVEMENT_146_31":      {Name: "zarina", Type: "survivor"},
	"ACH_CHAPTER16_SURVIVOR_3":    {Name: "cheryl", Type: "survivor"},
	"ACH_CHAPTER17_SURVIVOR_3":    {Name: "felix", Type: "survivor"},
	"ACH_CHAPTER18_SURVIVOR_3":    {Name: "elodie", Type: "survivor"},
	"ACH_CHAPTER19_SURVIVOR_3":    {Name: "yun-jin", Type: "survivor"},
	"ACH_CHAPTER20_SURVIVOR_3":    {Name: "jill", Type: "survivor"},
	"ACH_CHAPTER20_SURVIVOR_2":    {Name: "leon", Type: "survivor"},
	"NEW_ACHIEVEMENT_211_3":       {Name: "mikaela", Type: "survivor"},
	"ACH_CHAPTER22_SURVIVOR_3":    {Name: "jonah", Type: "survivor"},
	"NEW_ACHIEVEMENT_211_15":      {Name: "yoichi", Type: "survivor"},
	"NEW_ACHIEVEMENT_211_21":      {Name: "haddie", Type: "survivor"},
	"NEW_ACHIEVEMENT_211_26_NAME": {Name: "ada", Type: "survivor"},
	"NEW_ACHIEVEMENT_211_27_NAME": {Name: "rebecca", Type: "survivor"},
	"NEW_ACHIEVEMENT_245_1":       {Name: "vittorio", Type: "survivor"},
	"NEW_ACHIEVEMENT_245_6":       {Name: "thalita", Type: "survivor"},
	"NEW_ACHIEVEMENT_245_7":       {Name: "renato", Type: "survivor"},
	"NEW_ACHIEVEMENT_245_13":      {Name: "gabriel", Type: "survivor"},
	"NEW_ACHIEVEMENT_245_17":      {Name: "nicolas", Type: "survivor"},
	"NEW_ACHIEVEMENT_245_23":      {Name: "ellen", Type: "survivor"},
	"NEW_ACHIEVEMENT_245_29":      {Name: "alan", Type: "survivor"},
	"NEW_ACHIEVEMENT_280_3":       {Name: "sable", Type: "survivor"},
	"NEW_ACHIEVEMENT_280_10":      {Name: "troupe", Type: "survivor"},
	"NEW_ACHIEVEMENT_280_13":      {Name: "lara", Type: "survivor"},
	"NEW_ACHIEVEMENT_280_19":      {Name: "trevor", Type: "survivor"},
	"NEW_ACHIEVEMENT_280_25":      {Name: "taurie", Type: "survivor"},
	"NEW_ACHIEVEMENT_280_31":      {Name: "orela", Type: "survivor"},
	"NEW_ACHIEVEMENT_312_2":       {Name: "animatronic", Type: "killer"}, // CORRECTED: Animatronic is a killer (William Afton/Springtrap)
	"NEW_ACHIEVEMENT_312_4":       {Name: "rick", Type: "survivor"},
	"NEW_ACHIEVEMENT_312_5":       {Name: "michonne", Type: "survivor"},

	// Base Game Killers
	"ACH_UNLOCK_CHUCKLES_PERKS": {Name: "trapper", Type: "killer"},
	"ACH_UNLOCKBANSHEE_PERKS":   {Name: "wraith", Type: "killer"},
	"ACH_UNLOCKHILLBILY_PERKS":  {Name: "hillbilly", Type: "killer"},
	"ACH_DLC1_KILLER_3":         {Name: "nurse", Type: "killer"},

	// DLC Killers
	"ACH_DLC2_KILLER_1":           {Name: "shape", Type: "killer"},
	"ACH_DLC3_KILLER_3":           {Name: "hag", Type: "killer"},
	"ACH_DLC4_KILLER_3":           {Name: "doctor", Type: "killer"},
	"ACH_DLC5_KILLER_3":           {Name: "huntress", Type: "killer"},
	"ACH_DLC6_KILLER_3":           {Name: "cannibal", Type: "killer"},
	"ACH_DLC7_KILLER_3":           {Name: "nightmare", Type: "killer"},
	"ACH_DLC8_KILLER_3":           {Name: "pig", Type: "killer"},
	"ACH_DLC9_KILLER_3":           {Name: "clown", Type: "killer"},
	"ACH_CHAPTER9_KILLER_3":       {Name: "spirit", Type: "killer"},
	"ACH_CHAPTER10_KILLER_3":      {Name: "legion", Type: "killer"},
	"ACH_CHAPTER11_KILLER_3":      {Name: "plague", Type: "killer"},
	"ACH_CHAPTER12_KILLER_3":      {Name: "ghostface", Type: "killer"},
	"ACH_CHAPTER14_KILLER_3":      {Name: "oni", Type: "killer"},
	"NEW_ACHIEVEMENT_146_28":      {Name: "deathslinger", Type: "killer"},
	"ACH_CHAPTER16_KILLER_3":      {Name: "executioner", Type: "killer"},
	"ACH_CHAPTER17_KILLER_3":      {Name: "blight", Type: "killer"},
	"ACH_CHAPTER18_KILLER_3":      {Name: "twins", Type: "killer"},
	"ACH_CHAPTER19_KILLER_3":      {Name: "trickster", Type: "killer"},
	"ACH_CHAPTER20_KILLER_3":      {Name: "nemesis", Type: "killer"},
	"ACH_CHAPTER21_KILLER_3":      {Name: "cenobite", Type: "killer"},
	"ACH_CHAPTER22_KILLER_3":      {Name: "artist", Type: "killer"},
	"NEW_ACHIEVEMENT_211_12":      {Name: "onryo", Type: "killer"},
	"NEW_ACHIEVEMENT_211_18":      {Name: "dredge", Type: "killer"},
	"NEW_ACHIEVEMENT_211_24_NAME": {Name: "mastermind", Type: "killer"},
	"NEW_ACHIEVEMENT_211_30":      {Name: "knight", Type: "killer"},
	"NEW_ACHIEVEMENT_245_4":       {Name: "skull-merchant", Type: "killer"},
	"NEW_ACHIEVEMENT_245_10":      {Name: "singularity", Type: "killer"},
	"NEW_ACHIEVEMENT_245_20":      {Name: "xenomorph", Type: "killer"},
	"NEW_ACHIEVEMENT_245_26":      {Name: "chucky", Type: "killer"},
	"NEW_ACHIEVEMENT_280_0":       {Name: "unknown", Type: "killer"},
	"NEW_ACHIEVEMENT_280_7":       {Name: "vecna", Type: "killer"},
	"NEW_ACHIEVEMENT_280_16":      {Name: "dark-lord", Type: "killer"},
	"NEW_ACHIEVEMENT_280_22":      {Name: "houndmaster", Type: "killer"},
	"NEW_ACHIEVEMENT_312_1":       {Name: "lich", Type: "killer"},
	"NEW_ACHIEVEMENT_312_8":       {Name: "ghoul", Type: "killer"}, // Adept The Ghoul (killer)
}

type AdeptCharacter struct {
	Name string // Character name
	Type string // "survivor" or "killer"
}

// ProcessAchievements converts Steam achievements to our structured format
func ProcessAchievements(steamAchievements []SteamAchievement) *models.AchievementData {
	adeptSurvivors := make(map[string]bool)
	adeptKillers := make(map[string]bool)

	// Initialize all characters as not unlocked
	for _, character := range AdeptAchievementMapping {
		if character.Type == "survivor" {
			adeptSurvivors[character.Name] = false
		} else if character.Type == "killer" {
			adeptKillers[character.Name] = false
		}
	}

	// Track achievement processing stats
	achievementCount := 0
	adeptCount := 0
	achievedCount := 0
	unknownAchievements := 0

	// Mark unlocked achievements
	for _, achievement := range steamAchievements {
		achievementCount++

		if achievement.Achieved == 1 {
			achievedCount++

			// Check if this is a known Adept achievement
			if character, exists := AdeptAchievementMapping[achievement.APIName]; exists {
				adeptCount++
				if character.Type == "survivor" {
					adeptSurvivors[character.Name] = true
				} else if character.Type == "killer" {
					adeptKillers[character.Name] = true
				}

				logSteamInfo("Mapped achieved adept achievement",
					"api_name", achievement.APIName,
					"character", character.Name,
					"type", character.Type)
			} else if strings.HasPrefix(achievement.APIName, "NEW_ACHIEVEMENT_") {
				// Track unknown achievements that might be newer character Adepts
				unknownAchievements++
			}
		}
	}

	// Log summary of achievement processing
	logSteamInfo("Processed achievements",
		"total_achievements", achievementCount,
		"achieved_achievements", achievedCount,
		"mapped_adept_achievements", adeptCount,
		"unknown_new_achievements", unknownAchievements,
		"survivor_count", len(adeptSurvivors),
		"killer_count", len(adeptKillers))

	return &models.AchievementData{
		AdeptSurvivors: adeptSurvivors,
		AdeptKillers:   adeptKillers,
		LastUpdated:    time.Now(),
	}
}
