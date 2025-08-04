package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/models"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

// MockSteamClientForHardening implements SteamClientInterface for hardening tests
type MockSteamClientForHardening struct {
	GetPlayerSummaryFunc     func(steamIDOrVanity string) (*steam.SteamPlayer, *steam.APIError)
	GetPlayerStatsFunc       func(steamIDOrVanity string) (*steam.SteamPlayerstats, *steam.APIError)
	GetPlayerAchievementsFunc func(steamID, appID string) (*steam.PlayerAchievements, *steam.APIError)
}

func (m *MockSteamClientForHardening) GetPlayerSummary(steamIDOrVanity string) (*steam.SteamPlayer, *steam.APIError) {
	if m.GetPlayerSummaryFunc != nil {
		return m.GetPlayerSummaryFunc(steamIDOrVanity)
	}
	return &steam.SteamPlayer{SteamID: steamIDOrVanity}, nil
}

func (m *MockSteamClientForHardening) GetPlayerStats(steamIDOrVanity string) (*steam.SteamPlayerstats, *steam.APIError) {
	if m.GetPlayerStatsFunc != nil {
		return m.GetPlayerStatsFunc(steamIDOrVanity)
	}
	return &steam.SteamPlayerstats{SteamID: steamIDOrVanity}, nil
}

func (m *MockSteamClientForHardening) GetPlayerAchievements(steamID, appID string) (*steam.PlayerAchievements, *steam.APIError) {
	if m.GetPlayerAchievementsFunc != nil {
		return m.GetPlayerAchievementsFunc(steamID, appID)
	}
	return &steam.PlayerAchievements{SteamID: steamID}, nil
}

// Test Configuration Loading
func TestLoadAPIConfigFromEnv(t *testing.T) {
	// Test with environment variables set
	t.Setenv("API_TIMEOUT_SECS", "30")
	t.Setenv("MAX_RETRIES", "5")
	t.Setenv("BASE_BACKOFF_MS", "1000")
	t.Setenv("MAX_BACKOFF_MS", "60000")
	
	config := LoadAPIConfigFromEnv()
	
	if config.APITimeout != 30*time.Second {
		t.Errorf("Expected APITimeout 30s, got %v", config.APITimeout)
	}
	if config.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries 5, got %d", config.MaxRetries)
	}
	if config.BaseBackoff != 1*time.Second {
		t.Errorf("Expected BaseBackoff 1s, got %v", config.BaseBackoff)
	}
	if config.MaxBackoff != 60*time.Second {
		t.Errorf("Expected MaxBackoff 60s, got %v", config.MaxBackoff)
	}
}

func TestDefaultAPIConfig(t *testing.T) {
	config := DefaultAPIConfig()
	
	// Verify all fields have reasonable defaults
	if config.APITimeoutSecs <= 0 {
		t.Error("APITimeoutSecs should be positive")
	}
	if config.MaxRetries < 0 {
		t.Error("MaxRetries should be non-negative")
	}
	if config.BaseBackoffMs <= 0 {
		t.Error("BaseBackoffMs should be positive")
	}
	if config.MaxBackoffMs <= config.BaseBackoffMs {
		t.Error("MaxBackoffMs should be greater than BaseBackoffMs")
	}
	
	// Test the converted duration fields
	if config.APITimeout <= 0 {
		t.Error("APITimeout should be positive")
	}
	if config.BaseBackoff <= 0 {
		t.Error("BaseBackoff should be positive")
	}
	if config.MaxBackoff <= config.BaseBackoff {
		t.Error("MaxBackoff should be greater than BaseBackoff")
	}
}

// Test Safe Merge Functionality
func TestSafeAchievementMerger_ValidData(t *testing.T) {
	// Create merger with lower thresholds for testing
	merger := NewSafeAchievementMergerWithConfig(1, 1, 24*time.Hour)
	
	// Create a response with existing achievements
	response := &models.PlayerStatsWithAchievements{
		Achievements: &models.AchievementData{
			AdeptSurvivors: map[string]bool{
				"Dwight": true,
				"Meg":    false,
			},
			AdeptKillers: map[string]bool{
				"Trapper": true,
			},
			LastUpdated: time.Now().Add(-1 * time.Hour),
		},
	}
	
	newData := &models.AchievementData{
		AdeptSurvivors: map[string]bool{
			"Dwight":   true,
			"Meg":      true, // Updated
			"Claudette": true, // New
		},
		AdeptKillers: map[string]bool{
			"Trapper": true,
			"Wraith":  true, // New
		},
		LastUpdated: time.Now(),
	}
	
	err := merger.SafeMergeAchievements(response, newData, "12345")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Check merged survivors
	if !response.Achievements.AdeptSurvivors["Dwight"] {
		t.Error("Dwight should remain true")
	}
	if !response.Achievements.AdeptSurvivors["Meg"] {
		t.Error("Meg should be updated to true")
	}
	if !response.Achievements.AdeptSurvivors["Claudette"] {
		t.Error("Claudette should be added as true")
	}
	
	// Check merged killers
	if !response.Achievements.AdeptKillers["Trapper"] {
		t.Error("Trapper should remain true")
	}
	if !response.Achievements.AdeptKillers["Wraith"] {
		t.Error("Wraith should be added as true")
	}
}

func TestSafeAchievementMerger_NilInputs(t *testing.T) {
	// Create merger with lower thresholds for testing
	merger := NewSafeAchievementMergerWithConfig(1, 1, 24*time.Hour)
	
	// Test nil response
	err := merger.SafeMergeAchievements(nil, &models.AchievementData{}, "12345")
	if err == nil {
		t.Error("Expected error with nil response")
	}
	
	// Test response with nil achievements (should be handled gracefully)
	response := &models.PlayerStatsWithAchievements{}
	newData := &models.AchievementData{
		AdeptSurvivors: map[string]bool{"Dwight": true},
		AdeptKillers:   map[string]bool{"Trapper": true},
		LastUpdated:    time.Now(),
	}
	
	err = merger.SafeMergeAchievements(response, newData, "12345")
	if err != nil {
		t.Fatalf("Expected no error with nil achievements, got %v", err)
	}
	if response.Achievements == nil {
		t.Error("Achievements should be initialized")
	}
	
	// Test nil new data (should preserve existing)
	response = &models.PlayerStatsWithAchievements{
		Achievements: &models.AchievementData{
			AdeptSurvivors: map[string]bool{"Meg": true},
			AdeptKillers:   map[string]bool{"Wraith": true},
			LastUpdated:    time.Now(),
		},
	}
	
	err = merger.SafeMergeAchievements(response, nil, "12345")
	if err != nil {
		t.Fatalf("Expected no error with nil new data, got %v", err)
	}
	if !response.Achievements.AdeptSurvivors["Meg"] {
		t.Error("Should preserve existing data when new is nil")
	}
}

// Test Parallel Fetcher
func TestParallelFetcher_SuccessfulFetch(t *testing.T) {
	config := DefaultAPIConfig()
	config.OverallTimeout = 5 * time.Second
	
	mockClient := &MockSteamClientForHardening{
		GetPlayerStatsFunc: func(steamID string) (*steam.SteamPlayerstats, *steam.APIError) {
			// Add small delay to ensure duration is measurable
			time.Sleep(1 * time.Millisecond)
			return &steam.SteamPlayerstats{SteamID: steamID}, nil
		},
		GetPlayerAchievementsFunc: func(steamID, appID string) (*steam.PlayerAchievements, *steam.APIError) {
			time.Sleep(1 * time.Millisecond)
			return &steam.PlayerAchievements{SteamID: steamID}, nil
		},
	}
	
	fetcher := NewParallelFetcher(config, mockClient)
	ctx := context.Background()
	
	result, err := fetcher.FetchPlayerDataParallel(ctx, "12345")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if result.StatsError != nil {
		t.Errorf("Expected no stats error, got %v", result.StatsError)
	}
	if result.AchError != nil {
		t.Errorf("Expected no achievements error, got %v", result.AchError)
	}
	if result.Duration <= 0 {
		t.Errorf("Duration should be positive, got %v", result.Duration)
	}
}

func TestParallelFetcher_StatsFailure(t *testing.T) {
	config := DefaultAPIConfig()
	
	mockClient := &MockSteamClientForHardening{
		GetPlayerStatsFunc: func(steamID string) (*steam.SteamPlayerstats, *steam.APIError) {
			return nil, &steam.APIError{Message: "Stats failed", Type: "api_error"}
		},
		GetPlayerAchievementsFunc: func(steamID, appID string) (*steam.PlayerAchievements, *steam.APIError) {
			return &steam.PlayerAchievements{SteamID: steamID}, nil
		},
	}
	
	fetcher := NewParallelFetcher(config, mockClient)
	ctx := context.Background()
	
	result, err := fetcher.FetchPlayerDataParallel(ctx, "12345")
	
	// Stats failure should cause overall failure
	if err == nil {
		t.Error("Expected error when stats fail")
	}
	if result.StatsError == nil {
		t.Error("Expected stats error to be recorded")
	}
}

func TestParallelFetcher_AchievementsFailure(t *testing.T) {
	config := DefaultAPIConfig()
	
	mockClient := &MockSteamClientForHardening{
		GetPlayerStatsFunc: func(steamID string) (*steam.SteamPlayerstats, *steam.APIError) {
			return &steam.SteamPlayerstats{SteamID: steamID}, nil
		},
		GetPlayerAchievementsFunc: func(steamID, appID string) (*steam.PlayerAchievements, *steam.APIError) {
			return nil, &steam.APIError{Message: "Achievements failed", Type: "private_profile"}
		},
	}
	
	fetcher := NewParallelFetcher(config, mockClient)
	ctx := context.Background()
	
	result, err := fetcher.FetchPlayerDataParallel(ctx, "12345")
	
	// Achievements failure should NOT cause overall failure
	if err != nil {
		t.Errorf("Expected no error when only achievements fail, got %v", err)
	}
	if result.AchError == nil {
		t.Error("Expected achievements error to be recorded")
	}
	if result.StatsError != nil {
		t.Error("Stats should succeed")
	}
}

func TestParallelFetcher_ContextTimeout(t *testing.T) {
	config := DefaultAPIConfig()
	config.OverallTimeout = 100 * time.Millisecond
	
	mockClient := &MockSteamClientForHardening{
		GetPlayerStatsFunc: func(steamID string) (*steam.SteamPlayerstats, *steam.APIError) {
			time.Sleep(200 * time.Millisecond) // Longer than timeout
			return &steam.SteamPlayerstats{SteamID: steamID}, nil
		},
	}
	
	fetcher := NewParallelFetcher(config, mockClient)
	ctx := context.Background()
	
	result, err := fetcher.FetchPlayerDataParallel(ctx, "12345")
	
	// Should timeout
	if err == nil {
		t.Error("Expected timeout error")
	}
	if result.Duration < 100*time.Millisecond {
		t.Error("Duration should reflect timeout period")
	}
}

// Test Enhanced Retry Logic
func TestEnhancedRetrier_SuccessFirstAttempt(t *testing.T) {
	policy := DefaultRetryPolicy()
	config := DefaultAPIConfig()
	retrier := NewEnhancedRetrier(policy, config)
	
	attempts := 0
	err := retrier.Execute(context.Background(), "test", func(ctx context.Context, attempt int) error {
		attempts++
		return nil // Success on first try
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestEnhancedRetrier_RetryableError(t *testing.T) {
	policy := DefaultRetryPolicy()
	policy.MaxRetries = 2
	config := DefaultAPIConfig()
	retrier := NewEnhancedRetrier(policy, config)
	
	attempts := 0
	err := retrier.Execute(context.Background(), "test", func(ctx context.Context, attempt int) error {
		attempts++
		if attempts <= 2 {
			return &steam.APIError{Type: "rate_limit", Message: "Rate limited"}
		}
		return nil // Success on third try
	})
	
	if err != nil {
		t.Errorf("Expected no error after retry, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestEnhancedRetrier_NonRetryableError(t *testing.T) {
	policy := DefaultRetryPolicy()
	config := DefaultAPIConfig()
	retrier := NewEnhancedRetrier(policy, config)
	
	attempts := 0
	err := retrier.Execute(context.Background(), "test", func(ctx context.Context, attempt int) error {
		attempts++
		return &steam.APIError{Type: "private_profile", Message: "Private profile"}
	})
	
	if err == nil {
		t.Error("Expected error for non-retryable error")
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestEnhancedRetrier_MaxRetriesExceeded(t *testing.T) {
	policy := DefaultRetryPolicy()
	policy.MaxRetries = 2
	policy.BaseBackoff = 1 * time.Millisecond // Fast for testing
	config := DefaultAPIConfig()
	retrier := NewEnhancedRetrier(policy, config)
	
	attempts := 0
	err := retrier.Execute(context.Background(), "test", func(ctx context.Context, attempt int) error {
		attempts++
		return &steam.APIError{Type: "network_error", Message: "Network error"}
	})
	
	if err == nil {
		t.Error("Expected error when max retries exceeded")
	}
	if attempts != 3 { // MaxRetries + 1
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestEnhancedRetrier_ContextCancellation(t *testing.T) {
	policy := DefaultRetryPolicy()
	policy.BaseBackoff = 100 * time.Millisecond
	config := DefaultAPIConfig()
	retrier := NewEnhancedRetrier(policy, config)
	
	ctx, cancel := context.WithCancel(context.Background())
	
	attempts := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	
	err := retrier.Execute(ctx, "test", func(ctx context.Context, attempt int) error {
		attempts++
		return &steam.APIError{Type: "network_error", Message: "Network error"}
	})
	
	if err == nil {
		t.Error("Expected context cancellation error")
	}
	if attempts == 0 {
		t.Error("Should have made at least one attempt")
	}
}

// Test Steam API Retrier
func TestSteamAPIRetrier_StatsSuccess(t *testing.T) {
	policy := DefaultRetryPolicy()
	config := DefaultAPIConfig()
	
	mockClient := &MockSteamClientForHardening{
		GetPlayerStatsFunc: func(steamID string) (*steam.SteamPlayerstats, *steam.APIError) {
			return &steam.SteamPlayerstats{SteamID: steamID}, nil
		},
	}
	
	retrier := NewSteamAPIRetrier(policy, config, mockClient)
	ctx := context.Background()
	
	stats, err := retrier.GetPlayerStatsWithRetry(ctx, "12345")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if stats.SteamID != "12345" {
		t.Errorf("Expected SteamID '12345', got '%s'", stats.SteamID)
	}
}

func TestSteamAPIRetrier_AchievementsWithRetry(t *testing.T) {
	policy := DefaultRetryPolicy()
	policy.MaxRetries = 1
	config := DefaultAPIConfig()
	
	attempts := 0
	mockClient := &MockSteamClientForHardening{
		GetPlayerAchievementsFunc: func(steamID, appID string) (*steam.PlayerAchievements, *steam.APIError) {
			attempts++
			if attempts == 1 {
				return nil, &steam.APIError{Type: "timeout", Message: "Timeout"}
			}
			return &steam.PlayerAchievements{SteamID: steamID}, nil
		},
	}
	
	retrier := NewSteamAPIRetrier(policy, config, mockClient)
	ctx := context.Background()
	
	achievements, err := retrier.GetPlayerAchievementsWithRetry(ctx, "12345")
	if err != nil {
		t.Errorf("Expected no error after retry, got %v", err)
	}
	if achievements.SteamID != "12345" {
		t.Errorf("Expected SteamID '12345', got '%s'", achievements.SteamID)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

// Benchmark Tests
func BenchmarkParallelFetcher_SuccessCase(b *testing.B) {
	config := DefaultAPIConfig()
	mockClient := &MockSteamClientForHardening{
		GetPlayerStatsFunc: func(steamID string) (*steam.SteamPlayerstats, *steam.APIError) {
			return &steam.SteamPlayerstats{SteamID: steamID}, nil
		},
		GetPlayerAchievementsFunc: func(steamID, appID string) (*steam.PlayerAchievements, *steam.APIError) {
			return &steam.PlayerAchievements{SteamID: steamID}, nil
		},
	}
	
	fetcher := NewParallelFetcher(config, mockClient)
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		steamID := fmt.Sprintf("steam_%d", i)
		_, err := fetcher.FetchPlayerDataParallel(ctx, steamID)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkSafeAchievementMerger(b *testing.B) {
	merger := NewSafeAchievementMergerWithConfig(1, 1, 24*time.Hour)
	
	response := &models.PlayerStatsWithAchievements{
		Achievements: &models.AchievementData{
			AdeptSurvivors: map[string]bool{
				"Dwight": true,
				"Meg":    false,
			},
			AdeptKillers: map[string]bool{
				"Trapper": true,
			},
			LastUpdated: time.Now().Add(-1 * time.Hour),
		},
	}
	
	newData := &models.AchievementData{
		AdeptSurvivors: map[string]bool{
			"Dwight":   true,
			"Meg":      true,
			"Claudette": true,
		},
		AdeptKillers: map[string]bool{
			"Trapper": true,
			"Wraith":  true,
		},
		LastUpdated: time.Now(),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := merger.SafeMergeAchievements(response, newData, fmt.Sprintf("steam_%d", i))
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}
