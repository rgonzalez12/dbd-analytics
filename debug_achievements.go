package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Achievement struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Character   string `json:"character"`
	Type        string `json:"type"`
	Unlocked    bool   `json:"unlocked"`
	UnlockTime  int64  `json:"unlock_time,omitempty"`
}

type PlayerResponse struct {
	PlayerID       string            `json:"player_id"`
	PersonaName    string            `json:"persona_name"`
	ProfileURL     string            `json:"profile_url"`
	AvatarURL      string            `json:"avatar_url"`
	PlayerStats    interface{}       `json:"player_stats"`
	AdeptSurvivors map[string]bool   `json:"adept_survivors"`
	AdeptKillers   map[string]bool   `json:"adept_killers"`
	Achievements   []Achievement     `json:"achievements"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run debug_achievements.go <steam_id>")
		fmt.Println("Example: go run debug_achievements.go 76561198000000000")
		return
	}

	steamID := os.Args[1]
	url := fmt.Sprintf("http://localhost:8080/api/player/%s", steamID)

	fmt.Printf("Testing URL: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	
	if resp.StatusCode != 200 {
		fmt.Printf("Error Response: %s\n", string(body))
		return
	}

	var playerResp PlayerResponse
	if err := json.Unmarshal(body, &playerResp); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		fmt.Printf("Raw Response (first 500 chars): %s\n", string(body)[:min(500, len(body))])
		return
	}

	fmt.Printf("Player: %s\n", playerResp.PersonaName)
	fmt.Printf("Total achievements returned: %d\n", len(playerResp.Achievements))

	survivorCount := 0
	killerCount := 0
	unlockedCount := 0

	fmt.Println("\nAchievements breakdown:")
	for _, achievement := range playerResp.Achievements {
		if achievement.Type == "survivor" {
			survivorCount++
		} else if achievement.Type == "killer" {
			killerCount++
		}
		if achievement.Unlocked {
			unlockedCount++
		}
	}

	fmt.Printf("Survivors: %d\n", survivorCount)
	fmt.Printf("Killers: %d\n", killerCount)
	fmt.Printf("Other: %d\n", len(playerResp.Achievements)-survivorCount-killerCount)
	fmt.Printf("Unlocked: %d\n", unlockedCount)
	fmt.Printf("Locked: %d\n", len(playerResp.Achievements)-unlockedCount)

	fmt.Printf("\nAdept Survivors map: %d entries\n", len(playerResp.AdeptSurvivors))
	fmt.Printf("Adept Killers map: %d entries\n", len(playerResp.AdeptKillers))

	fmt.Println("\nFirst 10 achievements:")
	for i, achievement := range playerResp.Achievements {
		if i >= 10 {
			break
		}
		fmt.Printf("- %s (%s): %t\n", achievement.Character, achievement.Type, achievement.Unlocked)
	}

	// Show some specific characters you mentioned
	fmt.Println("\nChecking specific characters from your JSON:")
	characterMap := make(map[string]Achievement)
	for _, achievement := range playerResp.Achievements {
		characterMap[achievement.Character] = achievement
	}

	checkCharacters := []string{"dwight", "trapper", "meg", "artist", "blight"}
	for _, char := range checkCharacters {
		if achievement, exists := characterMap[char]; exists {
			fmt.Printf("- %s: %t (%s)\n", char, achievement.Unlocked, achievement.Type)
		} else {
			fmt.Printf("- %s: NOT FOUND\n", char)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
