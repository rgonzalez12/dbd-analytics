package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/rgonzalez12/dbd-analytics/internal/cache"
	"github.com/rgonzalez12/dbd-analytics/internal/models"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

// Enhanced SteamClient interface that includes achievements
type EnhancedSteamClient interface {
	GetPlayerSummary(steamIDOrVanity string) (*steam.SteamPlayer, *steam.APIError)
	GetPlayerStats(steamIDOrVanity string) (*steam.SteamPlayerstats, *steam.APIError)
	GetPlayerAchievements(steamID, appID string) (*steam.PlayerAchievements, *steam.APIError)
}

// EnhancedMockSteamClient for testing various failure scenarios
type EnhancedMockSteamClient struct {
	shouldFailStats        bool
	shouldFailAchievements bool
	statsError             *steam.APIError
	achievementsError      *steam.APIError
	timeoutDuration        time.Duration
}

func (m *EnhancedMockSteamClient) GetPlayerSummary(steamID string) (*steam.SteamPlayer, *steam.APIError) {
	if m.timeoutDuration > 0 {
		time.Sleep(m.timeoutDuration)
	}
	
	if m.shouldFailStats {
		return nil, m.statsError
	}
	
	return &steam.SteamPlayer{
		SteamID:     steamID,
		PersonaName: "TestPlayer",
	}, nil
}

func (m *EnhancedMockSteamClient) GetPlayerStats(steamID string) (*steam.SteamPlayerstats, *steam.APIError) {
	if m.shouldFailStats {
		return nil, m.statsError
	}
	
	return &steam.SteamPlayerstats{
		SteamID:  steamID,
		GameName: "Dead by Daylight",
		Stats: []steam.SteamStat{
			{Name: "DBD_KillerSkulls", Value: 100},
			{Name: "DBD_CamperPips", Value: 50},
		},
	}, nil
}

func (m *EnhancedMockSteamClient) GetPlayerAchievements(steamID, appID string) (*steam.PlayerAchievements, *steam.APIError) {
	if m.shouldFailAchievements {
		return nil, m.achievementsError
	}
	
	return &steam.PlayerAchievements{
		SteamID:  steamID,
		GameName: "Dead by Daylight",
		Success:  true,
		Achievements: []steam.SteamAchievement{
			{APIName: "ACH_DLC2_50", Achieved: 1}, // dwight
			{APIName: "ACH_DLC2_00", Achieved: 1}, // trapper
		},
	}, nil
}

// Enhanced test handler that uses the enhanced interface
type EnhancedTestHandler struct {
	steamClient  EnhancedSteamClient
	cacheManager *cache.Manager
}

func NewEnhancedTestHandler(client EnhancedSteamClient, cacheManager *cache.Manager) *EnhancedTestHandler {
	return &EnhancedTestHandler{
		steamClient:  client,
		cacheManager: cacheManager,
	}
}

// Copy the method we want to test
func (h *EnhancedTestHandler) GetPlayerStatsWithAchievements(w http.ResponseWriter, r *http.Request) {
	steamID := mux.Vars(r)["steamid"]
	
	// Validate Steam ID format
	if err := validateSteamIDOrVanity(steamID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	// Fetch data in parallel for better performance
	type fetchResult struct {
		stats        models.PlayerStats
		achievements *models.AchievementData
		statsError   error
		achError     error
		statsSource  string
		achSource    string
	}

	result := fetchResult{}
	resultChan := make(chan struct{}, 2)

	// Fetch player stats
	go func() {
		defer func() { resultChan <- struct{}{} }()
		result.stats, result.statsError, result.statsSource = h.fetchPlayerStatsWithSource(steamID)
	}()

	// Fetch achievements
	go func() {
		defer func() { resultChan <- struct{}{} }()
		result.achievements, result.achError, result.achSource = h.fetchPlayerAchievementsWithSource(steamID)
	}()

	// Wait for both to complete
	<-resultChan
	<-resultChan

	// Determine response strategy based on what succeeded
	response := models.PlayerStatsWithAchievements{
		PlayerStats: result.stats,
		DataSources: models.DataSourceStatus{
			Stats: models.DataSourceInfo{
				Success:   result.statsError == nil,
				Source:    result.statsSource,
				FetchedAt: time.Now(),
			},
			Achievements: models.DataSourceInfo{
				Success:   result.achError == nil,
				Source:    result.achSource,
				FetchedAt: time.Now(),
			},
		},
	}

	// Handle different failure scenarios
	if result.statsError != nil {
		response.DataSources.Stats.Error = result.statsError.Error()
		writeErrorResponse(w, steam.NewInternalError(result.statsError))
		return
	}

	// Always initialize achievements to prevent frontend errors
	response.Achievements = &models.AchievementData{
		AdeptSurvivors: make(map[string]bool),
		AdeptKillers:   make(map[string]bool),
		LastUpdated:    time.Now(),
	}

	if result.achError != nil {
		response.DataSources.Achievements.Error = result.achError.Error()
	} else {
		response.Achievements = result.achievements
	}

	writeJSONResponse(w, response)
}

func (h *EnhancedTestHandler) fetchPlayerStatsWithSource(steamID string) (models.PlayerStats, error, string) {
	summary, err := h.steamClient.GetPlayerSummary(steamID)
	if err != nil {
		return models.PlayerStats{}, fmt.Errorf("steam summary failed: %w", err), "api"
	}

	rawStats, err := h.steamClient.GetPlayerStats(steamID)
	if err != nil {
		return models.PlayerStats{}, fmt.Errorf("steam stats failed: %w", err), "api"
	}

	playerStats := steam.MapSteamStats(rawStats.Stats, summary.SteamID, summary.PersonaName)
	flatPlayerStats := convertToPlayerStats(playerStats)

	return flatPlayerStats, nil, "api"
}

func (h *EnhancedTestHandler) fetchPlayerAchievementsWithSource(steamID string) (*models.AchievementData, error, string) {
	rawAchievements, err := h.steamClient.GetPlayerAchievements(steamID, steam.DBDAppID)
	if err != nil {
		return nil, fmt.Errorf("steam achievements failed: %w", err), "api"
	}

	processedAchievements := steam.ProcessAchievements(rawAchievements.Achievements)
	return processedAchievements, nil, "api"
}

func TestGetPlayerStatsWithAchievements_HappyPath(t *testing.T) {
	// Setup
	handler := NewEnhancedTestHandler(
		&EnhancedMockSteamClient{},
		createTestCacheManager(t),
	)

	req := httptest.NewRequest("GET", "/player/76561198000000000", nil)
	req = mux.SetURLVars(req, map[string]string{"steamid": "76561198000000000"})
	w := httptest.NewRecorder()

	// Execute
	handler.GetPlayerStatsWithAchievements(w, req)

	// Verify
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.PlayerStatsWithAchievements
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify data structure
	if response.SteamID != "76561198000000000" {
		t.Errorf("Expected SteamID 76561198000000000, got %s", response.SteamID)
	}
	if response.Achievements == nil {
		t.Error("Expected achievements to be present")
	}
	if !response.DataSources.Stats.Success {
		t.Error("Expected stats to be successful")
	}
	if !response.DataSources.Achievements.Success {
		t.Error("Expected achievements to be successful")
	}
}

func TestGetPlayerStatsWithAchievements_AchievementsFailStatsSucceed(t *testing.T) {
	// Setup - achievements fail, stats succeed
	handler := NewEnhancedTestHandler(
		&EnhancedMockSteamClient{
			shouldFailAchievements: true,
			achievementsError:      steam.NewNotFoundError("Private Profile"),
		},
		createTestCacheManager(t),
	)

	req := httptest.NewRequest("GET", "/player/76561198000000000", nil)
	req = mux.SetURLVars(req, map[string]string{"steamid": "76561198000000000"})
	w := httptest.NewRecorder()

	// Execute
	handler.GetPlayerStatsWithAchievements(w, req)

	// Verify
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 (partial success), got %d", w.Code)
	}

	var response models.PlayerStatsWithAchievements
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify partial success behavior
	if !response.DataSources.Stats.Success {
		t.Error("Expected stats to be successful")
	}
	if response.DataSources.Achievements.Success {
		t.Error("Expected achievements to have failed")
	}
	if response.DataSources.Achievements.Error == "" {
		t.Error("Expected achievements error message to be populated")
	}
	
	// Verify achievements is still present but empty
	if response.Achievements == nil {
		t.Error("Expected achievements to be present (but empty)")
	}
	if len(response.Achievements.AdeptSurvivors) != 0 {
		t.Error("Expected empty survivor achievements on failure")
	}
}

func TestGetPlayerStatsWithAchievements_StatsFailAchievementsSucceed(t *testing.T) {
	// Setup - stats fail (should return error)
	handler := NewEnhancedTestHandler(
		&EnhancedMockSteamClient{
			shouldFailStats: true,
			statsError:      steam.NewNotFoundError("Player Not Found"),
		},
		createTestCacheManager(t),
	)

	req := httptest.NewRequest("GET", "/player/76561198000000000", nil)
	req = mux.SetURLVars(req, map[string]string{"steamid": "76561198000000000"})
	w := httptest.NewRecorder()

	// Execute
	handler.GetPlayerStatsWithAchievements(w, req)

	// Verify - should return error since stats are critical
	if w.Code == http.StatusOK {
		t.Error("Expected error status when stats fail, got 200")
	}
}

func TestGetPlayerStatsWithAchievements_InvalidSteamID(t *testing.T) {
	handler := NewEnhancedTestHandler(
		&EnhancedMockSteamClient{},
		createTestCacheManager(t),
	)

	req := httptest.NewRequest("GET", "/player/", nil)
	req = mux.SetURLVars(req, map[string]string{"steamid": ""}) // Empty string should fail
	w := httptest.NewRecorder()

	// Execute
	handler.GetPlayerStatsWithAchievements(w, req)

	// Verify
	if w.Code == http.StatusOK {
		t.Error("Expected error status for empty Steam ID, got 200")
	}
}

func TestGetPlayerStatsWithAchievements_Timeout(t *testing.T) {
	// Setup with timeout
	handler := NewEnhancedTestHandler(
		&EnhancedMockSteamClient{
			timeoutDuration: 100 * time.Millisecond, // Simulate slow API
		},
		createTestCacheManager(t),
	)

	req := httptest.NewRequest("GET", "/player/76561198000000000", nil)
	req = mux.SetURLVars(req, map[string]string{"steamid": "76561198000000000"})
	w := httptest.NewRecorder()

	start := time.Now()
	handler.GetPlayerStatsWithAchievements(w, req)
	duration := time.Since(start)

	// Verify it took some time (but didn't hang forever)
	if duration < 100*time.Millisecond {
		t.Error("Expected request to take some time due to mock delay")
	}
	if duration > 5*time.Second {
		t.Error("Request took too long, might be hanging")
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil error", nil, "none"},
		{"rate limit", &steam.APIError{Message: "Rate limit exceeded"}, "rate_limited"},
		{"timeout", &steam.APIError{Message: "Request timeout"}, "timeout"},
		{"private profile", &steam.APIError{Message: "Private profile"}, "private_profile"},
		{"network error", &steam.APIError{Message: "Network connection failed"}, "network_error"},
		{"steam api down", &steam.APIError{Message: "Steam API server error"}, "steam_api_down"},
		{"unknown error", &steam.APIError{Message: "Something unexpected"}, "unknown_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCountUnlocked(t *testing.T) {
	achievements := map[string]bool{
		"dwight":    true,
		"meg":       false,
		"claudette": true,
		"jake":      false,
		"nea":       true,
	}

	count := countUnlocked(achievements)
	expected := 3
	
	if count != expected {
		t.Errorf("Expected %d unlocked achievements, got %d", expected, count)
	}
}

// Helper function to create test cache manager
func createTestCacheManager(t *testing.T) *cache.Manager {
	config := cache.DevelopmentConfig()
	manager, err := cache.NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create test cache manager: %v", err)
	}
	return manager
}
