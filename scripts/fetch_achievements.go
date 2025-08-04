package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type SchemaResponse struct {
	Game struct {
		AvailableGameStats struct {
			Achievements []Achievement `json:"achievements"`
		} `json:"availableGameStats"`
	} `json:"game"`
}

type Achievement struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Hidden      int    `json:"hidden"`
}

func main() {
	apiKey := os.Getenv("STEAM_API_KEY")
	if apiKey == "" {
		fmt.Println("STEAM_API_KEY environment variable not set")
		return
	}

	url := fmt.Sprintf("https://api.steampowered.com/ISteamUserStats/GetSchemaForGame/v2/?key=%s&appid=381210", apiKey)
	
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error fetching schema: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	var schema SchemaResponse
	if err := json.Unmarshal(body, &schema); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	fmt.Println("=== POTENTIAL ADEPT ACHIEVEMENTS ===")
	for _, achievement := range schema.Game.AvailableGameStats.Achievements {
		name := strings.ToUpper(achievement.Name)
		displayName := strings.ToLower(achievement.DisplayName)
		
		// Look for achievements that might be Adept achievements
		if strings.Contains(name, "UNLOCK") && strings.Contains(name, "PERKS") ||
		   strings.Contains(name, "USE") && strings.Contains(name, "PERKS") ||
		   strings.Contains(displayName, "adept") {
			fmt.Printf("API Name: %s\n", achievement.Name)
			fmt.Printf("Display: %s\n", achievement.DisplayName)
			fmt.Printf("Description: %s\n", achievement.Description)
			fmt.Println("---")
		}
	}
}
