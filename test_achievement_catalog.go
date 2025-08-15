// Quick test to verify complete achievement catalog guarantee
package main

import (
	"fmt"
	"log"

	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

func main() {
	emptyAchievements := &steam.PlayerAchievements{
		Achievements: []steam.SteamAchievement{},
	}

	mappedData := steam.GetMappedAchievementsWithCache(emptyAchievements, nil)
	mapped := mappedData["achievements"].([]steam.AchievementMapping)
	summary := mappedData["summary"].(map[string]interface{})

	fmt.Printf("=== Achievement Catalog Completeness Test ===\n")
	fmt.Printf("Total achievements returned: %d\n", len(mapped))
	fmt.Printf("Unlocked achievements: %d\n", summary["unlocked_count"])
	fmt.Printf("Survivor adepts: %d\n", summary["survivor_count"])
	fmt.Printf("Killer adepts: %d\n", summary["killer_count"])
	fmt.Printf("General achievements: %d\n", summary["general_count"])
	fmt.Printf("Completion rate: %.1f%%\n", summary["completion_rate"])

	// Count by type and unlock status for verification
	unlockedGeneral := 0
	totalGeneral := 0
	for _, achievement := range mapped {
		if achievement.Type == "general" {
			totalGeneral++
			if achievement.Unlocked {
				unlockedGeneral++
			}
		}
	}

	fmt.Printf("\n=== General Achievement Breakdown ===\n")
	fmt.Printf("General achievements found: %d\n", totalGeneral)
	fmt.Printf("General achievements unlocked: %d\n", unlockedGeneral)
	fmt.Printf("General achievements locked: %d\n", totalGeneral-unlockedGeneral)

	// List some general achievements to verify they're present
	fmt.Printf("\n=== Sample General Achievements ===\n")
	count := 0
	for _, achievement := range mapped {
		if achievement.Type == "general" && count < 5 {
			status := "🔒"
			if achievement.Unlocked {
				status = "🏆"
			}
			fmt.Printf("%s %s - %s\n", status, achievement.DisplayName, achievement.Description)
			count++
		}
	}

	// Verify the guarantee: Should have at least 10 general achievements all unlocked=false
	if totalGeneral < 10 {
		log.Fatalf("❌ FAILED: Expected at least 10 general achievements, got %d", totalGeneral)
	}
	if unlockedGeneral > 0 {
		log.Fatalf("❌ FAILED: Expected 0 unlocked general achievements (empty account), got %d", unlockedGeneral)
	}
	if summary["unlocked_count"].(int) > 0 {
		log.Fatalf("❌ FAILED: Expected 0 total unlocked achievements (empty account), got %d", summary["unlocked_count"])
	}

	fmt.Printf("\n✅ SUCCESS: Complete achievement catalog guarantee verified!\n")
	fmt.Printf("   - All %d adept achievements present with unlocked=false\n", summary["survivor_count"].(int)+summary["killer_count"].(int))
	fmt.Printf("   - All %d general achievements present with unlocked=false\n", totalGeneral)
	fmt.Printf("   - Total catalog size: %d achievements\n", len(mapped))

	// Test with one unlocked general achievement
	fmt.Printf("\n=== Testing with Unlocked General Achievement ===\n")
	oneUnlockedAchievements := &steam.PlayerAchievements{
		Achievements: []steam.SteamAchievement{
			{APIName: "ACH_PERFECT_KILLER", Achieved: 1, UnlockTime: 1634567890},
		},
	}

	mappedData2 := steam.GetMappedAchievementsWithCache(oneUnlockedAchievements, nil)
	mapped2 := mappedData2["achievements"].([]steam.AchievementMapping)
	summary2 := mappedData2["summary"].(map[string]interface{})

	unlockedGeneral2 := 0
	for _, achievement := range mapped2 {
		if achievement.Type == "general" && achievement.Unlocked {
			unlockedGeneral2++
			fmt.Printf("🏆 %s - %s (Unlocked!)\n", achievement.DisplayName, achievement.Description)
		}
	}

	if len(mapped2) != len(mapped) {
		log.Fatalf("❌ FAILED: Catalog size changed with unlocked achievement: %d vs %d", len(mapped2), len(mapped))
	}
	if unlockedGeneral2 != 1 {
		log.Fatalf("❌ FAILED: Expected 1 unlocked general achievement, got %d", unlockedGeneral2)
	}
	if summary2["unlocked_count"].(int) != 1 {
		log.Fatalf("❌ FAILED: Expected 1 total unlocked achievement, got %d", summary2["unlocked_count"])
	}

	fmt.Printf("✅ SUCCESS: Unlocked achievement properly marked while catalog remains complete!\n")
	fmt.Printf("   - Catalog size consistent: %d achievements\n", len(mapped2))
	fmt.Printf("   - Unlocked general achievements: %d\n", unlockedGeneral2)
	fmt.Printf("   - Total unlocked: %d\n", summary2["unlocked_count"])
}
