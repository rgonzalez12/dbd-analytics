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
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	Value       float64 `json:"value"`
	Formatted   string  `json:"formatted"`
	Category    string  `json:"category"`
	ValueType   string  `json:"value_type"`
	SortWeight  int     `json:"sort_weight"`
	Icon        string  `json:"icon,omitempty"`
	Alias       string  `json:"alias,omitempty"`
}

// PlayerStatsResponse represents the complete stats response
type PlayerStatsResponse struct {
	Stats         []Stat                    `json:"stats"`
	Summary       map[string]interface{}    `json:"summary"`
	UnmappedStats []map[string]interface{}  `json:"unmapped_stats,omitempty"`
}

// Aliases map provides display names for DBD stats
var aliases = map[string]string{
	"DBD_CamperSkulls":                      "Survivor Bloodpoints (Skulls)",
	"DBD_KillerSkulls":                      "Killer Bloodpoints (Skulls)", 
	"DBD_SlasherSkulls":                     "Killer Bloodpoints (Skulls)",
	"DBD_GeneratorPct_float":                "Generators Repaired (equivalent)",
	"DBD_HealPct_float":                     "Survivors Healed (equivalent)",
	"DBD_BloodwebPoints":                    "Bloodpoints Earned",
	"DBD_BloodwebMaxLevel":                  "Highest Character Level",
	"DBD_BloodwebMaxPrestigeLevel":          "Highest Prestige Level",
	"DBD_BloodwebPrestige3MaxLevel":         "Prestige 3 Max Level Achievement",
	"DBD_PerksCount_Idx0":                   "Tier 1 Perks Owned",
	"DBD_PerksCount_Idx1":                   "Tier 2 Perks Owned", 
	"DBD_PerksCount_Idx2":                   "Tier 3 Perks Owned",
	"DBD_PerksCount_Idx3":                   "Ultra Rare Perks Owned",
	"DBD_BloodwebPerkMaxLevel":              "Max Perk Level Achieved",
	"DBD_MaxBloodwebPointsOneCategory":      "Max Points in One Category",
	
	"DBD_UnlockRanking":                     "Survivor Grade",
	"DBD_SlasherTierIncrement":              "Killer Grade",
	
	"DBD_Escape":                            "Total Escapes",
	"DBD_EscapeThroughHatch":                "Escapes Through Hatch",
	"DBD_EscapeKO":                          "Escapes While Injured",
	"DBD_AllEscapeThroughHatch":             "All Survivors Escaped Through Hatch",
	"DBD_LastSurvivorGeneratorEscape":       "Last Survivor Generator Escapes",
	"DBD_UnhookOrHeal":                      "Unhooks and Heals Performed",
	"DBD_UnhookOrHeal_PostExit":             "Post-Exit Gate Saves",
	"DBD_SkillCheckSuccess":                 "Successful Skill Checks",
	"DBD_HookedAndEscape":                   "Hooked and Still Escaped",
	"DBD_SaveCounter":                       "Survivors Saved",
	
	"DBD_CamperNewItem":                     "Escaped with New Item",
	"DBD_CamperEscapeWithItemFrom":          "Escaped with Others' Items",
	"DBD_CamperFullLoadout":                 "Survivor Full Loadout Matches",
	"DBD_CamperKeepUltraRare":               "Kept Ultra Rare Items",
	"DBD_CamperMaxScoreByCategory":          "Survivor Max Score by Category",
	
	"DBD_SacrificedCampers":                 "Survivors Sacrificed",
	"DBD_KilledCampers":                     "Survivors Killed (Mori)",
	"DBD_SacrificedCampers_iam":             "Sacrificed (IAM Tracked)",
	"DBD_KilledCampers_iam":                 "Killed (IAM Tracked)",
	"DBD_HitNearHook":                       "Hits Near Hooks",
	"DBD_SlasherFullLoadout":                "Killer Full Loadout Matches",
	"DBD_SlasherMaxScoreByCategory":         "Killer Max Score by Category",
	"DBD_SlasherPowerKillAllCampers":        "4K with Power Ability",
	
	"DBD_ChainsawHit":                       "Chainsaw Hits (Hillbilly/Cannibal)",
	"DBD_UncloakAttack":                     "Uncloak Attacks (Wraith)",
	"DBD_TrapPickup":                        "Bear Trap Catches (Trapper)",
	"DBD_SlasherChainAttack":                "Chain Attacks",
	"DBD_SlasherChainInterruptAfter3":       "Chain Interrupts After 3+",
	
	"DBD_BurnOffering_UltraRare":            "Ultra Rare Offerings Used",
	
	"DBD_EscapeNoBlood_MapAsy_Asylum":       "Escaped Asylum Without Injury",
	"DBD_FixSecondFloorGenerator_MapAsy_Asylum":         "Asylum Second Floor Generator",
	"DBD_FixSecondFloorGenerator_MapSub_Street":         "Haddonfield Second Floor Generator", 
	"DBD_FixSecondFloorGenerator_MapSwp_PaleRose":       "Pale Rose Second Floor Generator",
	"DBD_FixSecondFloorGenerator_MapBrl_MaHouse":        "MacMillan Second Floor Generator",
	"DBD_FixSecondFloorGenerator_MapFin_Hideout":        "Backwater Swamp Second Floor Generator",
	"DBD_FixSecondFloorGenerator_MapAsy_Chapel":         "Chapel Second Floor Generator",
	"DBD_FixSecondFloorGenerator_MapHti_Manor":          "Hawkins Second Floor Generator",
	"DBD_FixSecondFloorGenerator_MapKny_Cottage":        "Cottage Second Floor Generator",
	"DBD_FixSecondFloorGenerator_MapBrl_Temple":         "Temple Second Floor Generator",
	"DBD_FixSecondFloorGenerator_MapQat_Lab":            "Lab Second Floor Generator",
	"DBD_FixSecondFloorGenerator_MapHti_Shrine":         "Shrine Second Floor Generator",
	"DBD_FixSecondFloorGenerator_MapUkr_Saloon":         "Saloon Second Floor Generator",
	"DBD_FixSecondFloorGenerator_MapWal_Level_01":       "Wall Level 01 Generator",
	"DBD_FixSecondFloorGenerator_MapEcl_Level_01":       "Eclipse Level 01 Generator",
	"DBD_FixSecondFloorGenerator_MapIon_Level_01":       "Ion Level 01 Generator",
	"DBD_FixSecondFloorGenerator_MapMtr_Level_1":        "Metro Level 1 Generator",
	"DBD_FixSecondFloorGenerator_MapQtm_Level_01":       "Quantum Level 01 Generator",
	"DBD_FixSecondFloorGenerator_MapInd_Forest":         "Forest Generator",
	"DBD_FixSecondFloorGenerator_MapUba_Level_01":       "Uba Level 01 Generator",
	"DBD_FixSecondFloorGenerator_MapWrm_Level_01":       "Wrm Level 01 Generator",
	"DBD_FixSecondFloorGeneratorMapApl_Level_01":        "Apl Level 01 Generator",
	"DBD_FixSecondFloorGenerator_MapQtm_Level_02":       "Quantum Level 02 Generator",
	"DBD_FixSecondFloorGenerator_MapApl_Shack":          "Apple Shack Generator",
	
	// Special Event Stats
	"DBD_Event1_Stat1":                      "Event 1 Stat 1",
	"DBD_Event1_Stat2":                      "Event 1 Stat 2", 
	"DBD_Event1_Stat3":                      "Event 1 Stat 3",
	"DBD_Racoon_Dog_Triggered":              "Raccoon Dog Event Triggered",
	
	// Character-Specific Stats (Survivors)
	"DBD_Camper8_Stat1":                     "Survivor 8 Stat 1",
	"DBD_Camper8_Stat2":                     "Survivor 8 Stat 2",
	"DBD_Camper9_Stat2":                     "Survivor 9 Stat 2",
	"DBD_Camper38_Stat1":                    "Survivor 38 Stat 1",
	"DBD_Camper38_Stat2":                    "Survivor 38 Stat 2",
	"DBD_Camper40_Stat1":                    "Survivor 40 Stat 1",
	"DBD_Camper40_Stat2":                    "Survivor 40 Stat 2",
	"DBD_Camper43_Stat1":                    "Survivor 43 Stat 1",
	"DBD_Camper43_Stat2":                    "Survivor 43 Stat 2",
	
	// Special Conditions
	"DBD_EscapeNoBlood_Obsession":           "Escaped as Obsession Without Injury",
	
	// Compatibility
	"DBD_MatchesPlayed":                     "Matches Played",
	"DBD_MatchesWon":                        "Matches Won",
	"DBD_PerfectMatch":                      "Perfect Matches",
	"DBD_OfferingsBurnt":                    "Offerings Used",
	"DBD_MysteryBoxes":                      "Mystery Boxes Opened",
	
	// DLC Character Stats
	"DBD_DLC3_Slasher_Stat1":                "Hag: Phantasm Trap Triggers",
	"DBD_DLC3_Slasher_Stat2":                "Hag: Teleport Attacks",
	"DBD_DLC3_Camper_Stat1":                 "Ace: Luck-Based Escapes",
	"DBD_DLC4_Slasher_Stat1":                "Doctor: Shock Therapy Hits",
	"DBD_DLC4_Slasher_Stat2":                "Doctor: Madness Tier 3 Applications",
	"DBD_DLC5_Slasher_Stat1":                "Huntress: Hatchet Hits",
	"DBD_DLC5_Slasher_Stat2":                "Huntress: Long Range Hits",
	"DBD_DLC6_Slasher_Stat1":                "Leatherface: Chainsaw Hits",
	"DBD_DLC6_Slasher_Stat2":                "Leatherface: Multi-Hit Chainsaws",
	"DBD_DLC7_Slasher_Stat1":                "Nightmare: Dream Demon Teleports",
	"DBD_DLC7_Slasher_Stat2":                "Nightmare: Dream State Hits",
	"DBD_DLC7_Camper_Stat1":                 "Quentin: Sleep Resistance",
	"DBD_DLC7_Camper_Stat2":                 "Quentin: Skill Check Bonuses",
	"DBD_DLC8_Slasher_Stat1":                "Pig: Reverse Bear Trap Kills",
	"DBD_DLC8_Slasher_Stat2":                "Pig: Ambush Attacks",
	"DBD_DLC8_Camper_Stat1":                 "David: Protection Hits Taken",
	"DBD_DLC9_Slasher_Stat1":                "Clown: Afterpiece Tonic Hits",
	"DBD_DLC9_Slasher_Stat2":                "Clown: Intoxicated Survivor Downs",
	"DBD_DLC9_Camper_Stat1":                 "Kate: Boil Over Escapes",
	
	// Chapter Character Stats (Post-DLC naming)
	"DBD_Chapter9_Slasher_Stat1":            "Spirit: Yamaoka's Haunting Hits",
	"DBD_Chapter9_Slasher_Stat2":            "Spirit: Phase Walk Attacks",
	"DBD_Chapter9_Camper_Stat1":             "Adam: Deliverance Self-Unhooks",
	"DBD_Chapter10_Slasher_Stat1":           "Legion: Feral Frenzy Hits",
	"DBD_Chapter10_Slasher_Stat2":           "Legion: Deep Wound Downs",
	"DBD_Chapter10_Camper_Stat1":            "Jeff: Aftercare Reveals",
	"DBD_Chapter11_Slasher_Stat1":           "Plague: Corrupt Purge Hits",
	"DBD_Chapter11_Slasher_Stat2":           "Plague: Vile Purge Infections",
	"DBD_Chapter11_Camper_Stat1_float":      "Jane: Head On Stuns",
	"DBD_Chapter12_Slasher_Stat1":           "Ghostface: Marked Survivor Downs",
	"DBD_Chapter12_Slasher_Stat2":           "Ghostface: Stealth Hits",
	"DBD_Chapter12_Camper_Stat1":            "Ash: Mettle of Man Protections",
	"DBD_Chapter12_Camper_Stat2":            "Ash: Flip-Flop Escapes",
	"DBD_Chapter13_Slasher_Stat1":           "Demogorgon: Portal Teleports",
	"DBD_Chapter13_Slasher_Stat2":           "Demogorgon: Shred Attacks",
	"DBD_Chapter14_Slasher_Stat1":           "Oni: Blood Fury Activations",
	"DBD_Chapter14_Slasher_Stat2":           "Oni: Demon Strike Hits",
	"DBD_Chapter14_Camper_Stat1":            "Yui: Any Means Necessary Uses",
	"DBD_Chapter15_Slasher_Stat1":           "Deathslinger: Spear Gun Hits",
	"DBD_Chapter15_Slasher_Stat2":           "Deathslinger: Reeled Survivors",
	"DBD_Chapter15_Camper_Stat1":            "Zarina: Red Herring Activations",
	"DBD_Chapter16_Slasher_Stat1":           "Executioner: Punishment Hits",
	"DBD_Chapter16_Slasher_Stat2":           "Executioner: Cage Placements",
	"DBD_Chapter16_Camper_Stat1_float":      "Cheryl: Soul Guard Protections",
	"DBD_Chapter17_Slasher_Stat1":           "Blight: Lethal Rush Hits",
	"DBD_Chapter17_Slasher_Stat2":           "Blight: Bounce Attacks",
	"DBD_Chapter17_Camper_Stat1":            "Felix: Built to Last Uses",
	"DBD_Chapter17_Camper_Stat2_float":      "Felix: Visionary Reveals",
	"DBD_Chapter18_Slasher_Stat1":           "Twins: Victor Pounces",
	"DBD_Chapter18_Slasher_Stat2":           "Twins: Cooperative Downs",
	"DBD_Chapter18_Camper_Stat1":            "Élodie: Power Struggle Escapes",
	"DBD_Chapter18_Camper_Stat2_float":      "Élodie: Appraisal Uses",
	"DBD_Chapter19_Slasher_Stat1":           "Trickster: Blade Hits",
	"DBD_Chapter19_Slasher_Stat2":           "Trickster: Laceration Meter Fills",
	"DBD_Chapter19_Camper_Stat1":            "Yun-Jin: Fast Track Tokens",
	"DBD_Chapter19_Camper_Stat2":            "Yun-Jin: Self-Preservation Uses",
	"DBD_Chapter20_Slasher_Stat1":           "Nemesis: Tentacle Strike Hits",
	"DBD_Chapter20_Slasher_Stat2":           "Nemesis: Contamination Spreads",
	"DBD_Chapter21_Slasher_Stat1":           "Cenobite: Chain Hunt Activations",
	"DBD_Chapter21_Slasher_Stat2":           "Cenobite: Gateway Summons",
	"DBD_Chapter21_Camper_Stat1":            "Mikaela: Boon Totem Blessings",
	"DBD_Chapter21_Camper_Stat2":            "Mikaela: Circle of Healing Uses",
	"DBD_Chapter22_Slasher_Stat1":           "Artist: Dire Crow Hits",
	"DBD_Chapter22_Slasher_Stat2":           "Artist: Swarm Tracking",
	"DBD_Chapter22_Camper_Stat1":            "Jonah: Overcome Distance",
	"DBD_Chapter23_Slasher_Stat1":           "Onryo: Manifestation Attacks",
	"DBD_Chapter23_Slasher_Stat2":           "Onryo: Condemned Mori Kills",
	"DBD_Chapter23_Camper_Stat1":            "Yoichi: Parental Guidance Uses",
	"DBD_Chapter23_Camper_Stat2":            "Yoichi: Dark Theory Speed Boosts",
	"DBD_Chapter24_Slasher_Stat1":           "Dredge: Remnant Teleports",
	"DBD_Chapter24_Slasher_Stat2":           "Dredge: Nightfall Activations",
	"DBD_Chapter24_Camper_Stat1":            "Haddie: Residual Manifest Uses",
	"DBD_Chapter25_Slasher_Stat1":           "Mastermind: Virulent Bound Hits",
	"DBD_Chapter25_Slasher_Stat2":           "Mastermind: Infection Spreads",
	"DBD_Chapter25_Camper_Stat1":            "Ada: Wiretap Reveals",
	"DBD_Chapter26_Slasher_Stat1":           "Knight: Guard Summons",
	"DBD_Chapter26_Slasher_Stat2":           "Knight: Hunt Completions",
	"DBD_Chapter26_Camper_Stat1":            "Vittorio: Potential Energy Tokens",
	"DBD_Chapter27_Slasher_Stat1":           "Skull Merchant: Drone Detections",
	"DBD_Chapter27_Slasher_Stat2":           "Skull Merchant: Claw Trap Activations",
	"DBD_Chapter28_Slasher_Stat1":           "Singularity: Biopod Teleports",
	"DBD_Chapter28_Slasher_Stat2":           "Singularity: Slipstream Infects",
	"DBD_Chapter28_Slasher_Stat3":           "Singularity: Overclock Uses",
	"DBD_Chapter29_Slasher_Stat1":           "Xenomorph: Tail Attacks",
	"DBD_Chapter29_Slasher_Stat2":           "Xenomorph: Tunnel Ambushes",
	"DBD_Chapter29_Slasher_Stat3":           "Xenomorph: Crawler Mode Hits",
	"DBD_Chapter30_Slasher_Stat1":           "Good Guy: Scamper Uses",
	"DBD_Chapter30_Slasher_Stat2":           "Good Guy: Hidey-Ho Mode Attacks",
	"DBD_Chapter30_Slasher_Stat3":           "Good Guy: Slice & Dice Hits",
	"DBD_Chapter31_Slasher_Stat1":           "Unknown: UVX Weakened Hits",
	"DBD_Chapter31_Slasher_Stat2":           "Unknown: Teleport Attacks",
	"DBD_Chapter31_Camper_Stat1":            "Sable: Invocation Ritual Completions",
	"DBD_Chapter32_Slasher_Stat1":           "Lich: Flight of the Damned Hits",
	"DBD_Chapter32_Slasher_Stat2":           "Lich: Spell Casting Combinations",
	"DBD_Chapter32_Camper_Stat1":            "D&D Survivor: Magic Item Uses",
	"DBD_Chapter33_Slasher_Stat1":           "Dark Lord: Hellfire Hits",
	"DBD_Chapter33_Slasher_Stat2":           "Dark Lord: Bat Form Teleports",
	"DBD_Chapter33_Camper_Stat1":            "Trevor: Moment of Glory Uses",
	"DBD_Chapter33_Camper_Stat2":            "Trevor: Dramatic Entrance Activations",
	"DBD_Chapter34_Slasher_Stat1":           "Houndmaster: Search Commands",
	"DBD_Chapter34_Slasher_Stat2":           "Houndmaster: Redirect Attacks",
	"DBD_Chapter34_Camper_Stat1":            "Taurie: Shoulder the Burden Uses",
	"DBD_Chapter34_Camper_Stat2":            "Taurie: Blood Rush Activations",
	"DBD_Chapter35_Slasher_Stat1":           "Ghoul: Grab Attacks",
	"DBD_Chapter35_Slasher_Stat2":           "Ghoul: Radiation Exposure Spreads",
	"DBD_Chapter35_Survivor_Stat1":          "Orela: Breaking Limits Uses",
	"DBD_Chapter35_Survivor_Stat2":          "Orela: Inner Healing Activations",
	"DBD_Chapter36_Slasher_Stat1":           "Animatronic: Security Cameras Used",
	"DBD_Chapter36_Slasher_Stat2":           "Animatronic: Jump Scare Attacks",
	
	// FinishWithPerks Stats (Character Unlock Progress)
	"DBD_FinishWithPerks_Idx0":              "Dwight Adept Progress",
	"DBD_FinishWithPerks_Idx1":              "Meg Adept Progress", 
	"DBD_FinishWithPerks_Idx2":              "Claudette Adept Progress",
	"DBD_FinishWithPerks_Idx3":              "Jake Adept Progress",
	"DBD_FinishWithPerks_Idx4":              "Nea Adept Progress",
	"DBD_FinishWithPerks_Idx5":              "Laurie Adept Progress",
	"DBD_FinishWithPerks_Idx6":              "Ace Adept Progress",
	"DBD_FinishWithPerks_Idx7":              "Bill Adept Progress",
	"DBD_FinishWithPerks_Idx8":              "Feng Min Adept Progress",
	"DBD_FinishWithPerks_Idx9":              "David Adept Progress",
	"DBD_FinishWithPerks_Idx10":             "Quentin Adept Progress",
	"DBD_FinishWithPerks_Idx11":             "Kate Adept Progress",
	"DBD_FinishWithPerks_Idx12":             "Adam Adept Progress",
	"DBD_FinishWithPerks_Idx13":             "Jeff Adept Progress",
	"DBD_FinishWithPerks_Idx14":             "Jane Adept Progress",
	"DBD_FinishWithPerks_Idx15":             "Ash Adept Progress",
	"DBD_FinishWithPerks_Idx16":             "Nancy Adept Progress",
	"DBD_FinishWithPerks_Idx17":             "Steve Adept Progress",
	"DBD_FinishWithPerks_Idx18":             "Yui Adept Progress",
	"DBD_FinishWithPerks_Idx19":             "Zarina Adept Progress",
	"DBD_FinishWithPerks_Idx20":             "Cheryl Adept Progress",
	"DBD_FinishWithPerks_Idx21":             "Felix Adept Progress",
	"DBD_FinishWithPerks_Idx22":             "Élodie Adept Progress",
	"DBD_FinishWithPerks_Idx23":             "Yun-Jin Adept Progress",
	"DBD_FinishWithPerks_Idx24":             "Jill Adept Progress",
	"DBD_FinishWithPerks_Idx25":             "Leon Adept Progress",
	"DBD_FinishWithPerks_Idx26":             "Mikaela Adept Progress",
	"DBD_FinishWithPerks_Idx27":             "Jonah Adept Progress",
	"DBD_FinishWithPerks_Idx28":             "Yoichi Adept Progress",
	"DBD_FinishWithPerks_Idx29":             "Haddie Adept Progress",
	"DBD_FinishWithPerks_Idx30":             "Ada Adept Progress",
	"DBD_FinishWithPerks_Idx31":             "Rebecca Adept Progress",
	"DBD_FinishWithPerks_Idx32":             "Vittorio Adept Progress",
	"DBD_FinishWithPerks_Idx33":             "Thalita Adept Progress",
	"DBD_FinishWithPerks_Idx34":             "Renato Adept Progress",
	"DBD_FinishWithPerks_Idx35":             "Gabriel Adept Progress",
	"DBD_FinishWithPerks_Idx36":             "Nicolas Cage Adept Progress",
	"DBD_FinishWithPerks_Idx37":             "Ellen Ripley Adept Progress",
	"DBD_FinishWithPerks_Idx38":             "Alan Wake Adept Progress",
	"DBD_FinishWithPerks_Idx39":             "Sable Adept Progress",
	"DBD_FinishWithPerks_Idx40":             "Aestri/Baermar Adept Progress",
	"DBD_FinishWithPerks_Idx41":             "Lara Croft Adept Progress",
	"DBD_FinishWithPerks_Idx42":             "Trevor Belmont Adept Progress",
	"DBD_FinishWithPerks_Idx43":             "Taurie Adept Progress",
	"DBD_FinishWithPerks_Idx44":             "Orela Adept Progress",
	"DBD_FinishWithPerks_Idx45":             "Rick Grimes Adept Progress",
	"DBD_FinishWithPerks_Idx46":             "Michonne Adept Progress",
	"DBD_FinishWithPerks_Idx47":             "Future Survivor 47 Adept Progress",
	
	// Killer FinishWithPerks (using different ID space)
	"DBD_FinishWithPerks_Idx268435456":      "Trapper Adept Progress",
	"DBD_FinishWithPerks_Idx268435457":      "Wraith Adept Progress",
	"DBD_FinishWithPerks_Idx268435458":      "Hillbilly Adept Progress",
	"DBD_FinishWithPerks_Idx268435459":      "Nurse Adept Progress",
	"DBD_FinishWithPerks_Idx268435460":      "Shape/Myers Adept Progress",
	"DBD_FinishWithPerks_Idx268435461":      "Hag Adept Progress",
	"DBD_FinishWithPerks_Idx268435462":      "Doctor Adept Progress",
	"DBD_FinishWithPerks_Idx268435463":      "Huntress Adept Progress",
	"DBD_FinishWithPerks_Idx268435464":      "Cannibal Adept Progress",
	"DBD_FinishWithPerks_Idx268435465":      "Nightmare Adept Progress",
	"DBD_FinishWithPerks_Idx268435466":      "Pig Adept Progress",
	"DBD_FinishWithPerks_Idx268435467":      "Clown Adept Progress",
	"DBD_FinishWithPerks_Idx268435468":      "Spirit Adept Progress",
	"DBD_FinishWithPerks_Idx268435469":      "Legion Adept Progress",
	"DBD_FinishWithPerks_Idx268435470":      "Plague Adept Progress",
	"DBD_FinishWithPerks_Idx268435471":      "Ghostface Adept Progress",
	"DBD_FinishWithPerks_Idx268435472":      "Demogorgon Adept Progress",
	"DBD_FinishWithPerks_Idx268435473":      "Oni Adept Progress",
	"DBD_FinishWithPerks_Idx268435474":      "Deathslinger Adept Progress",
	"DBD_FinishWithPerks_Idx268435475":      "Executioner Adept Progress",
	"DBD_FinishWithPerks_Idx268435476":      "Blight Adept Progress",
	"DBD_FinishWithPerks_Idx268435477":      "Twins Adept Progress",
	"DBD_FinishWithPerks_Idx268435478":      "Trickster Adept Progress",
	"DBD_FinishWithPerks_Idx268435479":      "Nemesis Adept Progress",
	"DBD_FinishWithPerks_Idx268435480":      "Cenobite Adept Progress",
	"DBD_FinishWithPerks_Idx268435481":      "Artist Adept Progress",
	"DBD_FinishWithPerks_Idx268435482":      "Onryo Adept Progress",
	"DBD_FinishWithPerks_Idx268435483":      "Dredge Adept Progress",
	"DBD_FinishWithPerks_Idx268435484":      "Mastermind Adept Progress",
	"DBD_FinishWithPerks_Idx268435485":      "Knight Adept Progress",
	"DBD_FinishWithPerks_Idx268435486":      "Skull Merchant Adept Progress",
	"DBD_FinishWithPerks_Idx268435487":      "Singularity Adept Progress",
	"DBD_FinishWithPerks_Idx268435488":      "Xenomorph Adept Progress",
	"DBD_FinishWithPerks_Idx268435489":      "Good Guy Adept Progress",
	"DBD_FinishWithPerks_Idx268435490":      "Unknown Adept Progress",
	"DBD_FinishWithPerks_Idx268435491":      "Lich Adept Progress",
	"DBD_FinishWithPerks_Idx268435492":      "Dark Lord Adept Progress",
	"DBD_FinishWithPerks_Idx268435493":      "Houndmaster Adept Progress",
	"DBD_FinishWithPerks_Idx268435494":      "Ghoul Adept Progress",
	"DBD_FinishWithPerks_Idx268435495":      "Animatronic Adept Progress",
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
	
	if strings.Contains(id, "_float") || strings.Contains(id, "Pct_float") {
		return "float"
	}
	
	return "count"
}

// DBD Grade system structure (0-19 represents the 20 tiers)
type GradeInfo struct {
	Index int
	Tier  string
	Sub   int
}

// Standard DBD grade progression (Ash IV to Iridescent I)
var dbdGrades = []GradeInfo{
	{0, "Ash", 4}, {1, "Ash", 3}, {2, "Ash", 2}, {3, "Ash", 1},
	{4, "Bronze", 4}, {5, "Bronze", 3}, {6, "Bronze", 2}, {7, "Bronze", 1},
	{8, "Silver", 4}, {9, "Silver", 3}, {10, "Silver", 2}, {11, "Silver", 1},
	{12, "Gold", 4}, {13, "Gold", 3}, {14, "Gold", 2}, {15, "Gold", 1},
	{16, "Iridescent", 4}, {17, "Iridescent", 3}, {18, "Iridescent", 2}, {19, "Iridescent", 1},
}

// Known killer grade mappings (DBD_SlasherTierIncrement) with observed Steam values
var killerGradePoints = map[int]int{
	// Sequential pattern for low grades
	16: 0,   // Ash IV - starting point
	17: 1,   // Ash III  
	18: 2,   // Ash II
	19: 3,   // Ash I
	20: 4,   // Bronze IV
	21: 5,   // Bronze III
	22: 6,   // Bronze II
	23: 7,   // Bronze I
	
	// Alternative mappings observed
	73:  4,   // Bronze IV (alternative mapping)
	300: 9,   // Silver III (estimated)
	439: 6,   // Bronze II
	640: 0,   // Ash IV (alternative)
	
	// Additional mappings to handle edge cases
	0:   0,   // Reset/Unranked -> Ash IV
	1:   0,   // Very low values -> Ash IV
	15:  0,   // Below observed range -> Ash IV
	24:  8,   // Silver IV (estimated from pattern)
	25:  9,   // Silver III (estimated from pattern)
	50:  10,  // Silver II (estimated)
	100: 12,  // Gold IV (estimated)
	200: 14,  // Gold II (estimated)
	500: 16,  // Iridescent IV (estimated)
	1000: 19, // Iridescent I (estimated for very high values)
}

// Known survivor grade mappings (DBD_UnlockRanking) with observed Steam values
var survivorGradePoints = map[int]int{
	// Ash tier (0-3)
	7:    0,  // Ash IV
	541:  1,  // Ash III
	545:  1,  // Ash III (close variant)
	948:  2,  // Ash II
	949:  2,  // Ash II (close variant)
	1743: 3,  // Ash I
	2115: 0,  // Ash IV (alternative)
	
	// Bronze tier (4-7)
	640:  7,  // Bronze I
	
	// Silver tier (8-11)
	2050: 11, // Silver I
	
	// Gold tier (12-15)
	4226: 15, // Gold I
	4227: 15, // Gold I (close variant)
	
	// Iridescent tier (16-19)
	951:  16, // Iridescent IV
	4228: 16, // Iridescent IV
	4229: 16, // Iridescent IV (close variant)
	4230: 16, // Iridescent IV (close variant)
	4233: 17, // Iridescent III
	4251: 19, // Iridescent I
	8995: 16, // Iridescent IV
	
	// Additional mappings to handle more edge cases
	0:    0,  // Reset/Unranked -> Ash IV
	1:    0,  // Very low values -> Ash IV
	10:   0,  // Low values -> Ash IV
	100:  1,  // Low-mid values -> Ash III
	500:  1,  // Mid values -> Ash III
	1000: 2,  // Higher values -> Ash II
	1500: 3,  // High values -> Ash I
	3000: 12, // Very high values -> Gold IV
	5000: 16, // Extremely high values -> Iridescent IV
	9999: 19, // Maximum observed -> Iridescent I
}

// MapPlayerStats maps raw Steam stats to structured response using schema + user stats union
func MapPlayerStats(ctx context.Context, steamID string, cacheManager cache.Cache, client *Client) (*PlayerStatsResponse, error) {
	if client == nil {
		return nil, fmt.Errorf("steam client is required")
	}

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

	// 6) Map each stat in the union with comprehensive rule detection
	mapped := make([]Stat, 0, len(keys))
	unmappedStats := make([]map[string]interface{}, 0)

	for _, id := range keys {
		value, hasValue := userByID[id]
		if !hasValue {
			continue // Skip schema-only stats with no user value
		}

		schemaDisplayName := schemaByID[id]
		
		// Resolve display name priority
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

		// Format value based on type
		formatted := formatValue(value, valueType, id)

		// Set aliases for stats
		switch id {
		case "DBD_UnlockRanking":
			alias = "survivor_grade"
		case "DBD_SlasherTierIncrement":
			alias = "killer_grade"
		case "DBD_CamperSkulls":
			alias = "survivor_bloodpoints"
		case "DBD_SlasherSkulls":
			alias = "killer_bloodpoints"
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
		}

		mapped = append(mapped, stat)

		// Track unmapped stats
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

	// 8) Build summary
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

	response := &PlayerStatsResponse{
		Stats:         mapped,
		Summary:       summary,
		UnmappedStats: unmappedStats,
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

// decodeGrade converts raw grade value to human readable format using improved mapping
func decodeGrade(v float64, fieldID string) (Grade, string, string) {
	gradeCode := int(v)

	// Determine role based on field name
	isKillerGrade := strings.Contains(strings.ToLower(fieldID), "slasher") || 
					strings.Contains(strings.ToLower(fieldID), "killer")
	isSurvivorGrade := strings.Contains(strings.ToLower(fieldID), "unlock") || 
					  strings.Contains(strings.ToLower(fieldID), "survivor") ||
					  strings.Contains(strings.ToLower(fieldID), "camper")

	var gradeIndex int
	var found bool

	// Try killer grade mapping if it's a killer field
	if isKillerGrade {
		if index, exists := killerGradePoints[gradeCode]; exists {
			gradeIndex = index
			found = true
		} else {
			// Fallback: try to estimate based on value ranges for killer grades
			gradeIndex = estimateKillerGrade(gradeCode)
			found = gradeIndex >= 0
		}
	}

	// Try survivor grade mapping if it's a survivor field
	if isSurvivorGrade {
		if index, exists := survivorGradePoints[gradeCode]; exists {
			gradeIndex = index
			found = true
		} else {
			// Fallback: try to estimate based on value ranges for survivor grades
			gradeIndex = estimateSurvivorGrade(gradeCode)
			found = gradeIndex >= 0
		}
	}

	// If we found a valid grade index, convert it to the DBD grade structure
	if found && gradeIndex >= 0 && gradeIndex < len(dbdGrades) {
		gradeInfo := dbdGrades[gradeIndex]
		grade := Grade{Tier: gradeInfo.Tier, Sub: gradeInfo.Sub}
		
		if gradeInfo.Tier == "Unranked" {
			return grade, "Unranked", ""
		}
		
		human := fmt.Sprintf("%s %s", gradeInfo.Tier, roman(gradeInfo.Sub))
		return grade, human, roman(gradeInfo.Sub)
	}

	// Unknown grade - return question mark
	return Grade{Tier: "Unknown", Sub: 1}, "?", "?"
}

// estimateKillerGrade attempts to estimate killer grade based on value patterns
func estimateKillerGrade(value int) int {
	switch {
	case value >= 16 && value <= 23: // Sequential pattern for low grades
		return value - 16
	case value >= 50 && value <= 100: // Mid-range values (Bronze/Silver)
		return 4 + ((value - 50) * 8 / 50) // Map to Bronze/Silver range
	case value >= 200 && value <= 500: // Higher values (Silver/Gold)
		return 8 + ((value - 200) * 8 / 300) // Map to Silver/Gold range
	case value >= 600: // Very high values (Gold/Iridescent)
		index := 16 + ((value - 600) * 4 / 1000) // Map to Gold/Iridescent range
		if index > 19 {
			return 19
		}
		return index
	default:
		return -1 // Unknown
	}
}

// estimateSurvivorGrade attempts to estimate survivor grade based on value patterns
func estimateSurvivorGrade(value int) int {
	switch {
	case value >= 0 && value <= 10: // Very low values (Ash IV)
		return 0
	case value >= 500 && value <= 1000: // Low values (Ash range)
		return ((value - 500) * 4 / 500) // Map to Ash range (0-3)
	case value >= 1000 && value <= 2500: // Mid values (Bronze/Silver range)
		return 4 + ((value - 1000) * 8 / 1500) // Map to Bronze/Silver range (4-11)
	case value >= 2500 && value <= 5000: // High values (Gold/Iridescent range)
		return 12 + ((value - 2500) * 8 / 2500) // Map to Gold/Iridescent range (12-19)
	case value >= 5000: // Very high values (Iridescent range)
		index := 16 + ((value - 5000) * 4 / 5000) // Map to Iridescent range (16-19)
		if index > 19 {
			return 19
		}
		return index
	default:
		return -1 // Unknown
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
