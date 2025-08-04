package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
)

// Steam API structures for GetSchemaForGame
type SchemaResponse struct {
	Game *GameSchema `json:"game"`
}

type GameSchema struct {
	GameName            string            `json:"gameName"`
	GameVersion         string            `json:"gameVersion"`
	AvailableGameStats  *AvailableStats   `json:"availableGameStats"`
}

type AvailableStats struct {
	Stats        []StatDefinition        `json:"stats"`
	Achievements []AchievementDefinition `json:"achievements"`
}

type AchievementDefinition struct {
	Name         string `json:"name"`
	DefaultValue int    `json:"defaultvalue"`
	DisplayName  string `json:"displayName"`
	Hidden       int    `json:"hidden"`
	Description  string `json:"description"`
	Icon         string `json:"icon"`
	IconGray     string `json:"icongray"`
}

type StatDefinition struct {
	Name         string `json:"name"`
	DefaultValue int    `json:"defaultvalue"`
	DisplayName  string `json:"displayName"`
}

const (
	DbdAppID = "381210" // Dead by Daylight Steam App ID
	SchemaURL = "https://api.steampowered.com/ISteamUserStats/GetSchemaForGame/v2/"
)

func main() {
	// Get Steam API key from environment
	apiKey := os.Getenv("STEAM_API_KEY")
	if apiKey == "" {
		log.Fatal("STEAM_API_KEY environment variable is required")
	}

	// Fetch achievement schema from Steam
	url := fmt.Sprintf("%s?key=%s&appid=%s", SchemaURL, apiKey, DbdAppID)
	
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Failed to fetch schema: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Steam API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	var schema SchemaResponse
	if err := json.Unmarshal(body, &schema); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	if schema.Game == nil || schema.Game.AvailableGameStats == nil {
		log.Fatal("Invalid schema response structure")
	}

	achievements := schema.Game.AvailableGameStats.Achievements
	fmt.Printf("Found %d achievements for Dead by Daylight\n\n", len(achievements))

	// Sort achievements by name for easier reading
	sort.Slice(achievements, func(i, j int) bool {
		return achievements[i].Name < achievements[j].Name
	})

	// Categorize achievements
	var survivors, killers, general []AchievementDefinition
	
	for _, ach := range achievements {
		name := strings.ToLower(ach.Name)
		displayName := strings.ToLower(ach.DisplayName)
		
		// Categorize based on naming patterns
		if strings.Contains(name, "survivor") || 
		   strings.Contains(displayName, "survivor") ||
		   isKnownSurvivorAchievement(ach.Name) {
			survivors = append(survivors, ach)
		} else if strings.Contains(name, "killer") || 
				  strings.Contains(displayName, "killer") ||
				  isKnownKillerAchievement(ach.Name) {
			killers = append(killers, ach)
		} else {
			general = append(general, ach)
		}
	}

	// Print categorized results
	fmt.Println("=== SURVIVOR ACHIEVEMENTS ===")
	printAchievements(survivors)
	
	fmt.Println("\n=== KILLER ACHIEVEMENTS ===")
	printAchievements(killers)
	
	fmt.Println("\n=== GENERAL ACHIEVEMENTS ===")
	printAchievements(general)

	// Generate Go code for mapping
	fmt.Println("\n=== GO CODE FOR ACHIEVEMENT MAPPING ===")
	generateGoMapping(achievements)

	// Generate JSON config
	fmt.Println("\n=== JSON CONFIG FOR CONFIGURABLE SYSTEM ===")
	generateJSONConfig(survivors, killers, general)
}

func printAchievements(achievements []AchievementDefinition) {
	for _, ach := range achievements {
		hidden := ""
		if ach.Hidden == 1 {
			hidden = " [HIDDEN]"
		}
		fmt.Printf("%-50s | %s%s\n", ach.Name, ach.DisplayName, hidden)
		if ach.Description != "" {
			fmt.Printf("%-50s   Description: %s\n", "", ach.Description)
		}
		fmt.Println()
	}
}

func generateGoMapping(achievements []AchievementDefinition) {
	fmt.Println("// Generated achievement mapping for Go code")
	fmt.Println("var achievementMapping = map[string]AchievementInfo{")
	
	for _, ach := range achievements {
		category := "general"
		name := strings.ToLower(ach.Name)
		displayName := strings.ToLower(ach.DisplayName)
		
		if strings.Contains(name, "survivor") || 
		   strings.Contains(displayName, "survivor") ||
		   isKnownSurvivorAchievement(ach.Name) {
			category = "survivor"
		} else if strings.Contains(name, "killer") || 
				  strings.Contains(displayName, "killer") ||
				  isKnownKillerAchievement(ach.Name) {
			category = "killer"
		}
		
		fmt.Printf("    \"%s\": {\n", ach.Name)
		fmt.Printf("        DisplayName: \"%s\",\n", strings.ReplaceAll(ach.DisplayName, "\"", "\\\""))
		fmt.Printf("        Description: \"%s\",\n", strings.ReplaceAll(ach.Description, "\"", "\\\""))
		fmt.Printf("        Category: \"%s\",\n", category)
		fmt.Printf("        Hidden: %t,\n", ach.Hidden == 1)
		fmt.Printf("    },\n")
	}
	
	fmt.Println("}")
}

func generateJSONConfig(survivors, killers, general []AchievementDefinition) {
	config := map[string]interface{}{
		"version": "1.0",
		"app_id": DbdAppID,
		"achievements": map[string]interface{}{
			"survivors": extractAchievementNames(survivors),
			"killers":   extractAchievementNames(killers),
			"general":   extractAchievementNames(general),
		},
	}
	
	jsonData, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		log.Printf("Failed to generate JSON: %v", err)
		return
	}
	
	fmt.Println(string(jsonData))
}

func extractAchievementNames(achievements []AchievementDefinition) []string {
	names := make([]string, len(achievements))
	for i, ach := range achievements {
		names[i] = ach.Name
	}
	return names
}

// Helper functions for known achievement patterns
func isKnownSurvivorAchievement(name string) bool {
	survivorPatterns := []string{
		"ACH_CHAPTER", // Many chapter achievements are survivor-focused
		"ADEPT_",      // Adept achievements for survivors
	}
	
	for _, pattern := range survivorPatterns {
		if strings.Contains(name, pattern) {
			// Additional logic could be added here for specific patterns
			return strings.Contains(strings.ToLower(name), "survivor")
		}
	}
	return false
}

func isKnownKillerAchievement(name string) bool {
	killerPatterns := []string{
		"ADEPT_", // Adept achievements for killers
		"KILLER",
	}
	
	for _, pattern := range killerPatterns {
		if strings.Contains(name, pattern) {
			return strings.Contains(strings.ToLower(name), "killer")
		}
	}
	return false
}
