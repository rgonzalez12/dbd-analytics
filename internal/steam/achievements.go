package steam

import (
	"strings"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/models"
)

// AdeptAchievementMapping maps Steam achievement API names to character names
// All API names verified from Steam's GetSchemaForGame endpoint
var AdeptAchievementMapping = map[string]AdeptCharacter{
	// ===== SURVIVOR ADEPT ACHIEVEMENTS =====
	// Base Game Survivors
	"ACH_UNLOCK_DWIGHT_PERKS":    {Name: "dwight", Type: "survivor"},
	"ACH_UNLOCK_MEG_PERKS":       {Name: "meg", Type: "survivor"},
	"ACH_UNLOCK_CLAUDETTE_PERKS": {Name: "claudette", Type: "survivor"},
	"ACH_UNLOCK_JACK_PERKS":      {Name: "jake", Type: "survivor"}, // Jack = Jake
	
	// DLC Survivors - Verified API names from Steam
	"ACH_USE_NEA_PERKS":              {Name: "nea", Type: "survivor"},
	"ACH_DLC2_SURVIVOR_1":            {Name: "laurie", Type: "survivor"},    // Halloween Chapter
	"ACH_DLC3_SURVIVOR_3":            {Name: "ace", Type: "survivor"},       // Last Breath Chapter  
	"SURVIVOR7_ACHIEVEMENT_3":        {Name: "bill", Type: "survivor"},      // Left Behind Chapter
	"ACH_DLC4_SURVIVOR_3":            {Name: "feng", Type: "survivor"},      // Spark of Madness (Min)
	"ACH_DLC5_SURVIVOR_3":            {Name: "david", Type: "survivor"},     // Bloodstained Sack
	"ACH_DLC7_SURVIVOR_3":            {Name: "quentin", Type: "survivor"},   // Nightmare on Elm Street
	"ACH_DLC8_SURVIVOR_3":            {Name: "tapp", Type: "survivor"},      // SAW Chapter
	"ACH_DLC9_SURVIVOR_3":            {Name: "kate", Type: "survivor"},      // Curtain Call
	"ACH_CHAPTER9_SURVIVOR_3":        {Name: "adam", Type: "survivor"},      // Shattered Bloodline
	"ACH_CHAPTER10_SURVIVOR_3":       {Name: "jeff", Type: "survivor"},      // Darkness Among Us
	"ACH_CHAPTER11_SURVIVOR_3":       {Name: "jane", Type: "survivor"},      // Demise of the Faithful
	"ACH_CHAPTER12_SURVIVOR_3":       {Name: "ash", Type: "survivor"},       // Ash vs Evil Dead
	"ACH_CHAPTER14_SURVIVOR_3":       {Name: "yui", Type: "survivor"},       // Cursed Legacy
	"NEW_ACHIEVEMENT_146_31":         {Name: "zarina", Type: "survivor"},    // Chains of Hate
	"ACH_CHAPTER16_SURVIVOR_3":       {Name: "cheryl", Type: "survivor"},    // Silent Hill
	"ACH_CHAPTER17_SURVIVOR_3":       {Name: "felix", Type: "survivor"},     // Descend Beyond
	"ACH_CHAPTER18_SURVIVOR_3":       {Name: "elodie", Type: "survivor"},    // A Binding of Kin
	"ACH_CHAPTER19_SURVIVOR_3":       {Name: "yun-jin", Type: "survivor"},   // All-Kill
	"ACH_CHAPTER20_SURVIVOR_3":       {Name: "jill", Type: "survivor"},      // Resident Evil
	"ACH_CHAPTER20_SURVIVOR_2":       {Name: "leon", Type: "survivor"},      // Resident Evil
	"NEW_ACHIEVEMENT_211_3":          {Name: "mikaela", Type: "survivor"},   // Hour of the Witch
	"ACH_CHAPTER22_SURVIVOR_3":       {Name: "jonah", Type: "survivor"},     // Portrait of a Murder
	"NEW_ACHIEVEMENT_211_15":         {Name: "yoichi", Type: "survivor"},    // Sadako Rising
	"NEW_ACHIEVEMENT_211_21":         {Name: "haddie", Type: "survivor"},    // Roots of Dread
	"NEW_ACHIEVEMENT_211_26_NAME":    {Name: "ada", Type: "survivor"},       // Resident Evil: PROJECT W
	"NEW_ACHIEVEMENT_211_27_NAME":    {Name: "rebecca", Type: "survivor"},   // Resident Evil: PROJECT W
	"NEW_ACHIEVEMENT_245_1":          {Name: "vittorio", Type: "survivor"},  // Forged in Fog
	"NEW_ACHIEVEMENT_245_6":          {Name: "thalita", Type: "survivor"},   // Tools of Torment
	"NEW_ACHIEVEMENT_245_7":          {Name: "renato", Type: "survivor"},    // Tools of Torment
	"NEW_ACHIEVEMENT_245_13":         {Name: "gabriel", Type: "survivor"},   // Nicolas Cage Chapter
	"NEW_ACHIEVEMENT_245_17":         {Name: "nicolas", Type: "survivor"},   // Nicolas Cage Chapter
	"NEW_ACHIEVEMENT_245_23":         {Name: "ellen", Type: "survivor"},     // Alien Chapter (Ripley)
	"NEW_ACHIEVEMENT_245_29":         {Name: "alan", Type: "survivor"},      // Alan Wake Chapter
	"NEW_ACHIEVEMENT_280_3":          {Name: "sable", Type: "survivor"},     // All Things Wicked
	"NEW_ACHIEVEMENT_280_10":         {Name: "troupe", Type: "survivor"},    // Dungeons & Dragons (Aestri/Baermar)
	"NEW_ACHIEVEMENT_280_13":         {Name: "lara", Type: "survivor"},      // Tomb Raider Chapter
	"NEW_ACHIEVEMENT_280_19":         {Name: "trevor", Type: "survivor"},    // Casting of Frank Stone
	"NEW_ACHIEVEMENT_280_25":         {Name: "taurie", Type: "survivor"},    // Houndmaster Chapter
	"NEW_ACHIEVEMENT_280_31":         {Name: "orela", Type: "survivor"},     // Ghoul Chapter
	"NEW_ACHIEVEMENT_312_4":          {Name: "rick", Type: "survivor"},      // The Walking Dead Chapter
	"NEW_ACHIEVEMENT_312_5":          {Name: "michonne", Type: "survivor"},  // The Walking Dead Chapter

	// ===== KILLER ADEPT ACHIEVEMENTS =====
	// Base Game Killers  
	"ACH_UNLOCK_CHUCKLES_PERKS":      {Name: "trapper", Type: "killer"},     // Chuckles = Trapper
	"ACH_UNLOCKBANSHEE_PERKS":        {Name: "wraith", Type: "killer"},      // Banshee = Wraith  
	"ACH_UNLOCKHILLBILY_PERKS":       {Name: "hillbilly", Type: "killer"},   // Note: HILLBILY not HILLBILLY
	"ACH_DLC1_KILLER_3":              {Name: "nurse", Type: "killer"},       // Nurse
	
	// DLC Killers - Verified API names from Steam
	"ACH_DLC2_KILLER_1":              {Name: "shape", Type: "killer"},       // Michael Myers - Halloween
	"ACH_DLC3_KILLER_3":              {Name: "hag", Type: "killer"},         // Hag - Last Breath
	"ACH_DLC4_KILLER_3":              {Name: "doctor", Type: "killer"},      // Doctor - Spark of Madness
	"ACH_DLC5_KILLER_3":              {Name: "huntress", Type: "killer"},    // Huntress - A Lullaby for the Dark
	"ACH_DLC6_KILLER_3":              {Name: "cannibal", Type: "killer"},    // Leatherface - Leatherface
	"ACH_DLC7_KILLER_3":              {Name: "nightmare", Type: "killer"},   // Freddy - Nightmare on Elm Street
	"ACH_DLC8_KILLER_3":              {Name: "pig", Type: "killer"},         // Pig - SAW Chapter
	"ACH_DLC9_KILLER_3":              {Name: "clown", Type: "killer"},       // Clown - Curtain Call
	"ACH_CHAPTER9_KILLER_3":          {Name: "spirit", Type: "killer"},      // Spirit - Shattered Bloodline
	"ACH_CHAPTER10_KILLER_3":         {Name: "legion", Type: "killer"},      // Legion - Darkness Among Us
	"ACH_CHAPTER11_KILLER_3":         {Name: "plague", Type: "killer"},      // Plague - Demise of the Faithful
	"ACH_CHAPTER12_KILLER_3":         {Name: "ghostface", Type: "killer"},   // Ghost Face - Ghost Face
	"ACH_CHAPTER14_KILLER_3":         {Name: "oni", Type: "killer"},         // Oni - Cursed Legacy
	"NEW_ACHIEVEMENT_146_28":         {Name: "deathslinger", Type: "killer"}, // Deathslinger - Chains of Hate
	"ACH_CHAPTER16_KILLER_3":         {Name: "executioner", Type: "killer"}, // Pyramid Head - Silent Hill
	"ACH_CHAPTER17_KILLER_3":         {Name: "blight", Type: "killer"},      // Blight - Descend Beyond
	"ACH_CHAPTER18_KILLER_3":         {Name: "twins", Type: "killer"},       // Twins - A Binding of Kin
	"ACH_CHAPTER19_KILLER_3":         {Name: "trickster", Type: "killer"},   // Trickster - All-Kill
	"ACH_CHAPTER20_KILLER_3":         {Name: "nemesis", Type: "killer"},     // Nemesis - Resident Evil
	"ACH_CHAPTER22_KILLER_3":         {Name: "artist", Type: "killer"},      // Artist - Portrait of a Murder
	"NEW_ACHIEVEMENT_211_12":         {Name: "onryo", Type: "killer"},       // Sadako - Sadako Rising
	"NEW_ACHIEVEMENT_211_18":         {Name: "dredge", Type: "killer"},      // Dredge - Roots of Dread
	"NEW_ACHIEVEMENT_211_24_NAME":    {Name: "mastermind", Type: "killer"},  // Wesker - Resident Evil: PROJECT W
	"NEW_ACHIEVEMENT_211_30":         {Name: "knight", Type: "killer"},      // Knight - Forged in Fog
	"NEW_ACHIEVEMENT_245_4":          {Name: "skull-merchant", Type: "killer"}, // Skull Merchant - Tools of Torment
	"NEW_ACHIEVEMENT_245_10":         {Name: "singularity", Type: "killer"}, // Singularity - End Transmission
	"NEW_ACHIEVEMENT_245_20":         {Name: "xenomorph", Type: "killer"},   // Xenomorph - Alien Chapter
	"NEW_ACHIEVEMENT_245_26":         {Name: "chucky", Type: "killer"},      // Chucky - Chucky Chapter (Good Guy)
	"NEW_ACHIEVEMENT_280_0":          {Name: "unknown", Type: "killer"},     // Unknown - All Things Wicked
	"NEW_ACHIEVEMENT_280_7":          {Name: "vecna", Type: "killer"},       // Vecna - Dungeons & Dragons (Lich)
	"NEW_ACHIEVEMENT_280_16":         {Name: "dark-lord", Type: "killer"},   // Dracula - Castlevania Chapter
	"NEW_ACHIEVEMENT_280_22":         {Name: "houndmaster", Type: "killer"}, // Houndmaster Chapter
	"NEW_ACHIEVEMENT_280_28":         {Name: "ghoul", Type: "killer"},       // Ghoul Chapter
	"NEW_ACHIEVEMENT_312_2":          {Name: "animatronic", Type: "killer"}, // Five Nights at Freddy's Chapter
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
