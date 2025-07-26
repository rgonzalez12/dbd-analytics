package handler

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/rgonzalez12/dbd-analytics/internal/models"
)

func GetPlayerStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	steamID := vars["steamID"]


// mock data for testing
	 player := models.PlayerStats{
        SteamID:              steamID,
        DisplayName:          "RayTheRank14",
        KillerPips:           123,
        SurvivorPips:         87,
        KilledCampers:        250,
        SacrificedCampers:    320,
        GeneratorPct:         73.6,
        HealPct:              51.2,
        Escapes:              64,
        SkillCheckSuccess:    421,
        BloodwebPoints:       312000,
        EscapeThroughHatch:   5,
        UnhookOrHeal:         76,
        CamperFullLoadout:    12,
        KillerPerfectGames:   7,
    }

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}
