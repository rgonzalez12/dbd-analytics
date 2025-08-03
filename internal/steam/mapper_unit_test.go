package steam_test

import (
    "testing"

    "github.com/rgonzalez12/dbd-analytics/internal/steam"
)

func TestMapSteamStats(t *testing.T) {
    t.Run("BasicMapping", func(t *testing.T) {
        fakeData := []steam.SteamStat{
            {Name: "DBD_KilledCampers", Value: 5},
            {Name: "DBD_Escapes", Value: 10},
            {Name: "DBD_BloodwebPoints", Value: 2000},
        }

        mapped := steam.MapSteamStats(fakeData, "123456789", "TestUser")

        if mapped.Killer.TotalKills != 5 {
            t.Errorf("Expected 5 kills, got %d", mapped.Killer.TotalKills)
        }

        if mapped.Survivor.TotalEscapes != 10 {
            t.Errorf("Expected 10 escapes, got %d", mapped.Survivor.TotalEscapes)
        }

        if mapped.General.BloodwebPoints != 2000 {
            t.Errorf("Expected 2000 bloodpoints, got %d", mapped.General.BloodwebPoints)
        }
    })

    t.Run("UnknownStatIgnored", func(t *testing.T) {
        fakeData := []steam.SteamStat{
            {Name: "DBD_UnknownStatKey", Value: 99},
        }

        mapped := steam.MapSteamStats(fakeData, "987654321", "TestUser")

        // Unknown stats should not populate anything
        if mapped.General.BloodwebPoints != 0 {
            t.Errorf("Expected 0 bloodweb points, got %d", mapped.General.BloodwebPoints)
        }
        if mapped.Killer.TotalKills != 0 {
            t.Errorf("Expected 0 kills, got %d", mapped.Killer.TotalKills)
        }
        if mapped.Survivor.TotalEscapes != 0 {
            t.Errorf("Expected 0 escapes, got %d", mapped.Survivor.TotalEscapes)
        }
    })
}