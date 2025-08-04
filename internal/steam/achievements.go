package steam

import (
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/models"
)

// AdeptAchievementMapping maps Steam achievement API names to character names
var AdeptAchievementMapping = map[string]AdeptCharacter{
	// Survivor Adept Achievements
	"ACH_DLC2_50": {Name: "dwight", Type: "survivor"},
	"ACH_DLC2_51": {Name: "meg", Type: "survivor"},
	"ACH_DLC2_52": {Name: "claudette", Type: "survivor"},
	"ACH_DLC2_53": {Name: "jake", Type: "survivor"},
	"ACH_DLC2_54": {Name: "nea", Type: "survivor"},
	"ACH_DLC2_55": {Name: "laurie", Type: "survivor"},
	"ACH_DLC2_56": {Name: "ace", Type: "survivor"},
	"ACH_DLC2_57": {Name: "bill", Type: "survivor"},
	"ACH_DLC2_58": {Name: "feng", Type: "survivor"},
	"ACH_DLC2_59": {Name: "david", Type: "survivor"},
	"ACH_DLC2_60": {Name: "quentin", Type: "survivor"},
	"ACH_DLC2_61": {Name: "tapp", Type: "survivor"},
	"ACH_DLC2_62": {Name: "kate", Type: "survivor"},
	"ACH_DLC2_63": {Name: "adam", Type: "survivor"},
	"ACH_DLC2_64": {Name: "jeff", Type: "survivor"},
	"ACH_DLC2_65": {Name: "jane", Type: "survivor"},
	"ACH_DLC2_66": {Name: "ash", Type: "survivor"},
	"ACH_DLC2_67": {Name: "nancy", Type: "survivor"},
	"ACH_DLC2_68": {Name: "steve", Type: "survivor"},
	"ACH_DLC2_69": {Name: "yui", Type: "survivor"},
	"ACH_DLC2_70": {Name: "zarina", Type: "survivor"},
	"ACH_DLC2_71": {Name: "cheryl", Type: "survivor"},
	"ACH_DLC2_72": {Name: "felix", Type: "survivor"},
	"ACH_DLC2_73": {Name: "elodie", Type: "survivor"},
	"ACH_DLC2_74": {Name: "yun-jin", Type: "survivor"},
	"ACH_DLC2_75": {Name: "jill", Type: "survivor"},
	"ACH_DLC2_76": {Name: "leon", Type: "survivor"},
	"ACH_DLC2_77": {Name: "mikaela", Type: "survivor"},
	"ACH_DLC2_78": {Name: "jonah", Type: "survivor"},
	"ACH_DLC2_79": {Name: "yoichi", Type: "survivor"},
	"ACH_DLC2_80": {Name: "haddie", Type: "survivor"},
	"ACH_DLC2_81": {Name: "ada", Type: "survivor"},
	"ACH_DLC2_82": {Name: "rebecca", Type: "survivor"},
	"ACH_DLC2_83": {Name: "vittorio", Type: "survivor"},
	"ACH_DLC2_84": {Name: "thalita", Type: "survivor"},
	"ACH_DLC2_85": {Name: "renato", Type: "survivor"},
	"ACH_DLC2_86": {Name: "gabriel", Type: "survivor"},
	"ACH_DLC2_87": {Name: "nicolas", Type: "survivor"},
	"ACH_DLC2_88": {Name: "ellen", Type: "survivor"},
	"ACH_DLC2_89": {Name: "alan", Type: "survivor"},
	"ACH_DLC2_90": {Name: "sable", Type: "survivor"},

	// Killer Adept Achievements
	"ACH_DLC2_00": {Name: "trapper", Type: "killer"},
	"ACH_DLC2_01": {Name: "wraith", Type: "killer"},
	"ACH_DLC2_02": {Name: "hillbilly", Type: "killer"},
	"ACH_DLC2_03": {Name: "nurse", Type: "killer"},
	"ACH_DLC2_04": {Name: "shape", Type: "killer"},
	"ACH_DLC2_05": {Name: "hag", Type: "killer"},
	"ACH_DLC2_06": {Name: "doctor", Type: "killer"},
	"ACH_DLC2_07": {Name: "huntress", Type: "killer"},
	"ACH_DLC2_08": {Name: "cannibal", Type: "killer"},
	"ACH_DLC2_09": {Name: "nightmare", Type: "killer"},
	"ACH_DLC2_10": {Name: "pig", Type: "killer"},
	"ACH_DLC2_11": {Name: "clown", Type: "killer"},
	"ACH_DLC2_12": {Name: "spirit", Type: "killer"},
	"ACH_DLC2_13": {Name: "legion", Type: "killer"},
	"ACH_DLC2_14": {Name: "plague", Type: "killer"},
	"ACH_DLC2_15": {Name: "ghost-face", Type: "killer"},
	"ACH_DLC2_16": {Name: "demogorgon", Type: "killer"},
	"ACH_DLC2_17": {Name: "oni", Type: "killer"},
	"ACH_DLC2_18": {Name: "deathslinger", Type: "killer"},
	"ACH_DLC2_19": {Name: "executioner", Type: "killer"},
	"ACH_DLC2_20": {Name: "blight", Type: "killer"},
	"ACH_DLC2_21": {Name: "twins", Type: "killer"},
	"ACH_DLC2_22": {Name: "trickster", Type: "killer"},
	"ACH_DLC2_23": {Name: "nemesis", Type: "killer"},
	"ACH_DLC2_24": {Name: "cenobite", Type: "killer"},
	"ACH_DLC2_25": {Name: "artist", Type: "killer"},
	"ACH_DLC2_26": {Name: "onryo", Type: "killer"},
	"ACH_DLC2_27": {Name: "dredge", Type: "killer"},
	"ACH_DLC2_28": {Name: "mastermind", Type: "killer"},
	"ACH_DLC2_29": {Name: "knight", Type: "killer"},
	"ACH_DLC2_30": {Name: "skull-merchant", Type: "killer"},
	"ACH_DLC2_31": {Name: "singularity", Type: "killer"},
	"ACH_DLC2_32": {Name: "xenomorph", Type: "killer"},
	"ACH_DLC2_33": {Name: "good-guy", Type: "killer"},
	"ACH_DLC2_34": {Name: "unknown", Type: "killer"},
	"ACH_DLC2_35": {Name: "lich", Type: "killer"},
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

	// Mark unlocked achievements
	for _, achievement := range steamAchievements {
		if character, exists := AdeptAchievementMapping[achievement.APIName]; exists {
			isUnlocked := achievement.Achieved == 1
			
			if character.Type == "survivor" {
				adeptSurvivors[character.Name] = isUnlocked
			} else if character.Type == "killer" {
				adeptKillers[character.Name] = isUnlocked
			}
		}
	}

	return &models.AchievementData{
		AdeptSurvivors: adeptSurvivors,
		AdeptKillers:   adeptKillers,
		LastUpdated:    time.Now(),
	}
}
