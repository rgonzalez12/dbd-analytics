package steam

import (
	"strings"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/models"
)

// AdeptAchievementMapping maps Steam achievement API names to character names
var AdeptAchievementMapping = map[string]AdeptCharacter{
	// ===== SURVIVOR ADEPT ACHIEVEMENTS =====
	// Base Game Survivors
	"ACH_UNLOCK_DWIGHT_PERKS":   {Name: "dwight", Type: "survivor"},
	"ACH_UNLOCK_MEG_PERKS":      {Name: "meg", Type: "survivor"},
	"ACH_UNLOCK_CLAUDETTE_PERKS": {Name: "claudette", Type: "survivor"},
	"ACH_UNLOCK_JACK_PERKS":     {Name: "jake", Type: "survivor"}, // Jack = Jake
	
	// DLC Survivors - Using pattern ACH_USE_{CHARACTER}_PERKS or ACH_UNLOCK_{CHARACTER}_PERKS
	"ACH_USE_NEA_PERKS":         {Name: "nea", Type: "survivor"},
	"ACH_USE_LAURIE_PERKS":      {Name: "laurie", Type: "survivor"},    // Halloween Chapter
	"ACH_USE_ACE_PERKS":         {Name: "ace", Type: "survivor"},       // Last Breath Chapter  
	"ACH_USE_BILL_PERKS":        {Name: "bill", Type: "survivor"},      // Left Behind Chapter
	"ACH_USE_FENG_PERKS":        {Name: "feng", Type: "survivor"},      // Spark of Madness
	"ACH_USE_DAVID_PERKS":       {Name: "david", Type: "survivor"},     // Bloodstained Sack
	"ACH_USE_QUENTIN_PERKS":     {Name: "quentin", Type: "survivor"},   // Nightmare on Elm Street
	"ACH_USE_TAPP_PERKS":        {Name: "tapp", Type: "survivor"},      // SAW Chapter
	"ACH_USE_KATE_PERKS":        {Name: "kate", Type: "survivor"},      // Curtain Call
	"ACH_USE_ADAM_PERKS":        {Name: "adam", Type: "survivor"},      // Shattered Bloodline
	"ACH_USE_JEFF_PERKS":        {Name: "jeff", Type: "survivor"},      // Darkness Among Us
	"ACH_USE_JANE_PERKS":        {Name: "jane", Type: "survivor"},      // Demise of the Faithful
	"ACH_USE_ASH_PERKS":         {Name: "ash", Type: "survivor"},       // Ash vs Evil Dead
	"ACH_USE_NANCY_PERKS":       {Name: "nancy", Type: "survivor"},     // Stranger Things
	"ACH_USE_STEVE_PERKS":       {Name: "steve", Type: "survivor"},     // Stranger Things
	"ACH_USE_YUI_PERKS":         {Name: "yui", Type: "survivor"},       // Cursed Legacy
	"ACH_USE_ZARINA_PERKS":      {Name: "zarina", Type: "survivor"},    // Chains of Hate
	"ACH_USE_CHERYL_PERKS":      {Name: "cheryl", Type: "survivor"},    // Silent Hill
	"ACH_USE_FELIX_PERKS":       {Name: "felix", Type: "survivor"},     // Descend Beyond
	"ACH_USE_ELODIE_PERKS":      {Name: "elodie", Type: "survivor"},    // A Binding of Kin
	"ACH_USE_YUNJIN_PERKS":      {Name: "yun-jin", Type: "survivor"},   // All-Kill
	"ACH_USE_JILL_PERKS":        {Name: "jill", Type: "survivor"},      // Resident Evil
	"ACH_USE_LEON_PERKS":        {Name: "leon", Type: "survivor"},      // Resident Evil
	"ACH_USE_MIKAELA_PERKS":     {Name: "mikaela", Type: "survivor"},   // Hour of the Witch
	"ACH_USE_JONAH_PERKS":       {Name: "jonah", Type: "survivor"},     // Portrait of a Murder
	"ACH_USE_YOICHI_PERKS":      {Name: "yoichi", Type: "survivor"},    // Sadako Rising
	"ACH_USE_HADDIE_PERKS":      {Name: "haddie", Type: "survivor"},    // Roots of Dread
	"ACH_USE_ADA_PERKS":         {Name: "ada", Type: "survivor"},       // Resident Evil: PROJECT W
	"ACH_USE_REBECCA_PERKS":     {Name: "rebecca", Type: "survivor"},   // Resident Evil: PROJECT W
	"ACH_USE_VITTORIO_PERKS":    {Name: "vittorio", Type: "survivor"},  // Forged in Fog
	"ACH_USE_THALITA_PERKS":     {Name: "thalita", Type: "survivor"},   // Tools of Torment
	"ACH_USE_RENATO_PERKS":      {Name: "renato", Type: "survivor"},    // Tools of Torment
	"ACH_USE_GABRIEL_PERKS":     {Name: "gabriel", Type: "survivor"},   // Nicolas Cage Chapter
	"ACH_USE_NICOLAS_PERKS":     {Name: "nicolas", Type: "survivor"},   // Nicolas Cage Chapter
	"ACH_USE_ELLEN_PERKS":       {Name: "ellen", Type: "survivor"},     // Alien Chapter
	"ACH_USE_ALAN_PERKS":        {Name: "alan", Type: "survivor"},      // Alan Wake Chapter
	"ACH_USE_SABLE_PERKS":       {Name: "sable", Type: "survivor"},     // All Things Wicked
	"ACH_USE_AESTRI_PERKS":      {Name: "aestri", Type: "survivor"},    // Dungeons & Dragons
	"ACH_USE_BAERMAR_PERKS":     {Name: "baermar", Type: "survivor"},   // Dungeons & Dragons
	"ACH_USE_LARA_PERKS":        {Name: "lara", Type: "survivor"},      // Tomb Raider Chapter
	"ACH_USE_TREVOR_PERKS":      {Name: "trevor", Type: "survivor"},    // Casting of Frank Stone
	"ACH_USE_DARYL_PERKS":       {Name: "daryl", Type: "survivor"},     // The Walking Dead Chapter
	"ACH_USE_RICK_PERKS":        {Name: "rick", Type: "survivor"},      // The Walking Dead Chapter

	// ===== KILLER ADEPT ACHIEVEMENTS =====
	// Base Game Killers  
	"ACH_UNLOCK_CHUCKLES_PERKS":  {Name: "trapper", Type: "killer"},   // Chuckles = Trapper
	"ACH_UNLOCKBANSHEE_PERKS":    {Name: "wraith", Type: "killer"},    // Banshee = Wraith  
	"ACH_UNLOCKHILLBILY_PERKS":   {Name: "hillbilly", Type: "killer"}, // Note: HILLBILY not HILLBILLY
	"ACH_UNLOCKNURSE_PERKS":      {Name: "nurse", Type: "killer"},     // Nurse
	
	// DLC Killers - Using pattern ACH_UNLOCK{CHARACTER}_PERKS (note inconsistent underscores)
	"ACH_UNLOCKSHAPE_PERKS":       {Name: "shape", Type: "killer"},        // Michael Myers - Halloween
	"ACH_UNLOCKHAG_PERKS":         {Name: "hag", Type: "killer"},          // Hag - Last Breath
	"ACH_UNLOCKDOCTOR_PERKS":      {Name: "doctor", Type: "killer"},       // Doctor - Spark of Madness
	"ACH_UNLOCKHUNTRESS_PERKS":    {Name: "huntress", Type: "killer"},     // Huntress - A Lullaby for the Dark
	"ACH_UNLOCKCANNIBAL_PERKS":    {Name: "cannibal", Type: "killer"},     // Leatherface - Leatherface
	"ACH_UNLOCKNIGHTMARE_PERKS":   {Name: "nightmare", Type: "killer"},    // Freddy - Nightmare on Elm Street
	"ACH_UNLOCKPIG_PERKS":         {Name: "pig", Type: "killer"},          // Pig - SAW Chapter
	"ACH_UNLOCKCLOWN_PERKS":       {Name: "clown", Type: "killer"},        // Clown - Curtain Call
	"ACH_UNLOCKSPIRIT_PERKS":      {Name: "spirit", Type: "killer"},       // Spirit - Shattered Bloodline
	"ACH_UNLOCKLEGION_PERKS":      {Name: "legion", Type: "killer"},       // Legion - Darkness Among Us
	"ACH_UNLOCKPLAGUE_PERKS":      {Name: "plague", Type: "killer"},       // Plague - Demise of the Faithful
	"ACH_UNLOCKGHOSTFACE_PERKS":   {Name: "ghostface", Type: "killer"},    // Ghost Face - Ghost Face
	"ACH_UNLOCKDEMOGORGON_PERKS":  {Name: "demogorgon", Type: "killer"},   // Demogorgon - Stranger Things
	"ACH_UNLOCKONI_PERKS":         {Name: "oni", Type: "killer"},          // Oni - Cursed Legacy
	"ACH_UNLOCKDEATHSLINGER_PERKS": {Name: "deathslinger", Type: "killer"}, // Deathslinger - Chains of Hate
	"ACH_UNLOCKEXECUTIONER_PERKS": {Name: "executioner", Type: "killer"},  // Pyramid Head - Silent Hill
	"ACH_UNLOCKBLIGHT_PERKS":      {Name: "blight", Type: "killer"},       // Blight - Descend Beyond
	"ACH_UNLOCKTWINS_PERKS":       {Name: "twins", Type: "killer"},        // Twins - A Binding of Kin
	"ACH_UNLOCKTRICKSTER_PERKS":   {Name: "trickster", Type: "killer"},    // Trickster - All-Kill
	"ACH_UNLOCKNEMESIS_PERKS":     {Name: "nemesis", Type: "killer"},      // Nemesis - Resident Evil
	"ACH_UNLOCKCENTOBITE_PERKS":   {Name: "cenobite", Type: "killer"},     // Pinhead - Hellraiser
	"ACH_UNLOCKARTIST_PERKS":      {Name: "artist", Type: "killer"},       // Artist - Portrait of a Murder
	"ACH_UNLOCKONRYO_PERKS":       {Name: "onryo", Type: "killer"},        // Sadako - Sadako Rising
	"ACH_UNLOCKDREDGE_PERKS":      {Name: "dredge", Type: "killer"},       // Dredge - Roots of Dread
	"ACH_UNLOCKMASTERMIND_PERKS":  {Name: "mastermind", Type: "killer"},   // Wesker - Resident Evil: PROJECT W
	"ACH_UNLOCKKNIGHT_PERKS":      {Name: "knight", Type: "killer"},       // Knight - Forged in Fog
	"ACH_UNLOCKSKULLMERCHANT_PERKS": {Name: "skull-merchant", Type: "killer"}, // Skull Merchant - Tools of Torment
	"ACH_UNLOCKSINGULARITY_PERKS": {Name: "singularity", Type: "killer"},  // Singularity - End Transmission
	"ACH_UNLOCKXENOMORPH_PERKS":   {Name: "xenomorph", Type: "killer"},    // Xenomorph - Alien Chapter
	"ACH_UNLOCKGOODGUY_PERKS":     {Name: "chucky", Type: "killer"},       // Chucky - Chucky Chapter
	"ACH_UNLOCKUNKNOWN_PERKS":     {Name: "unknown", Type: "killer"},      // Unknown - All Things Wicked
	"ACH_UNLOCKLICH_PERKS":        {Name: "vecna", Type: "killer"},        // Vecna - Dungeons & Dragons
	"ACH_UNLOCKDARKGOD_PERKS":     {Name: "dark-lord", Type: "killer"},    // Dracula - Castlevania Chapter
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
