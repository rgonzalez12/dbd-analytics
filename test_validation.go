package main

import (
	"fmt"
	"github.com/rgonzalez12/dbd-analytics/internal/api"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

func main() {
	// Test validation directly
	steamID := "76561198000000000"
	fmt.Printf("Testing Steam ID: %s\n", steamID)
	fmt.Printf("Length: %d\n", len(steamID))
	fmt.Printf("First 7: %s\n", steamID[:7])
	
	// Test the validation function
	err := api.ValidateSteamIDOrVanity(steamID)
	if err != nil {
		fmt.Printf("Validation error: %s\n", err.Message)
		fmt.Printf("Error type: %s\n", err.Type)
	} else {
		fmt.Println("Validation passed!")
	}
	
	// Test Steam client
	client := steam.NewClient()
	summary, err := client.GetPlayerSummary(steamID)
	if err != nil {
		fmt.Printf("Steam API error: %s\n", err.Message)
		fmt.Printf("Error type: %s\n", err.Type)
	} else {
		fmt.Printf("Steam API success: %s\n", summary.PersonaName)
	}
}
