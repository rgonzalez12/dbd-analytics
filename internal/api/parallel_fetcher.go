package api

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/log"
	"github.com/rgonzalez12/dbd-analytics/internal/models"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
	"golang.org/x/sync/errgroup"
)

// SteamClientInterface defines the interface for Steam API operations
type SteamClientInterface interface {
	GetPlayerSummary(steamIDOrVanity string) (*steam.SteamPlayer, *steam.APIError)
	GetPlayerStats(steamIDOrVanity string) (*steam.SteamPlayerstats, *steam.APIError)
	GetPlayerAchievements(steamID, appID string) (*steam.PlayerAchievements, *steam.APIError)
}

// ParallelFetcher handles safe parallel fetching of player data
type ParallelFetcher struct {
	config      APIConfig
	merger      *SafeAchievementMerger
	steamClient SteamClientInterface
}

// NewParallelFetcher creates a new parallel fetcher with configuration
func NewParallelFetcher(config APIConfig, steamClient SteamClientInterface) *ParallelFetcher {
	return &ParallelFetcher{
		config:      config,
		merger:      NewSafeAchievementMerger(),
		steamClient: steamClient,
	}
}

// FetchResult holds the results of parallel fetch operations with guaranteed shape consistency
type FetchResult struct {
	Stats        models.PlayerStats
	Achievements *models.AchievementData // Always non-nil, may be empty on failure
	StatsError   error
	AchError     error
	StatsSource  string // "cache" | "api" | "fallback"
	AchSource    string // "cache" | "api" | "fallback" | "unavailable"
	Duration     time.Duration
	DataSources  models.DataSourceStatus // Detailed source tracking
}

// FetchPlayerDataParallel fetches stats and achievements with fail-soft error handling
func (f *ParallelFetcher) FetchPlayerDataParallel(ctx context.Context, steamID string) (*FetchResult, error) {
	start := time.Now()

	// Create context with overall timeout
	ctx, cancel := context.WithTimeout(ctx, f.config.OverallTimeout)
	defer cancel()

	// Initialize result with safe defaults - ensures consistent data shape
	result := &FetchResult{
		Achievements: &models.AchievementData{
			AdeptSurvivors: make(map[string]bool),
			AdeptKillers:   make(map[string]bool),
			LastUpdated:    time.Now(),
		},
		DataSources: models.DataSourceStatus{
			Stats: models.DataSourceInfo{
				Success:   false,
				Source:    "unknown",
				FetchedAt: time.Now(),
			},
			Achievements: models.DataSourceInfo{
				Success:   false,
				Source:    "unknown",
				FetchedAt: time.Now(),
			},
		},
	}

	// Use errgroup for safer parallel execution
	g, gCtx := errgroup.WithContext(ctx)

	// Fetch player stats - CRITICAL PATH
	g.Go(func() error {
		result.Stats, result.StatsSource, result.StatsError = f.fetchStatsWithRetry(gCtx, steamID)

		// Update data source tracking
		result.DataSources.Stats = models.DataSourceInfo{
			Success:   result.StatsError == nil,
			Source:    result.StatsSource,
			FetchedAt: time.Now(),
		}

		if result.StatsError != nil {
			result.DataSources.Stats.Error = result.StatsError.Error()
			log.Error("Critical: Stats fetch failed - blocking operation",
				"steam_id", steamID,
				"error", result.StatsError,
				"error_type", classifyError(result.StatsError),
				"source", result.StatsSource,
				"impact", "user_will_see_error")
			return fmt.Errorf("stats fetch failed: %w", result.StatsError)
		}

		log.Info("Stats fetch successful",
			"steam_id", steamID,
			"source", result.StatsSource,
			"duration_ms", time.Since(start).Milliseconds())

		return nil
	})

	// Fetch achievements - NON-CRITICAL PATH (fail-soft)
	g.Go(func() error {
		achievementData, achSource, achErr := f.fetchAchievementsWithRetry(gCtx, steamID)

		result.AchError = achErr
		result.AchSource = achSource

		// Update data source tracking
		result.DataSources.Achievements = models.DataSourceInfo{
			Success:   achErr == nil,
			Source:    achSource,
			FetchedAt: time.Now(),
		}

		if achErr != nil {
			// FAIL-SOFT: Log warning but continue with empty achievements
			result.DataSources.Achievements.Error = achErr.Error()
			result.AchSource = "unavailable"

			errorType := classifyError(achErr)
			log.Warn("Non-critical: Achievements fetch failed - continuing with empty data",
				"steam_id", steamID,
				"error", achErr,
				"error_type", errorType,
				"source", achSource,
				"impact", "user_gets_stats_without_achievements",
				"fallback_behavior", "empty_achievement_data")
		} else {
			// Success: Use fetched data
			if achievementData != nil {
				result.Achievements = achievementData
			}
			log.Info("Achievements fetch successful",
				"steam_id", steamID,
				"source", achSource,
				"survivor_count", len(result.Achievements.AdeptSurvivors),
				"killer_count", len(result.Achievements.AdeptKillers))
		}

		// Never return error for achievements - always allow graceful degradation
		return nil
	})

	// Wait for all operations to complete
	if err := g.Wait(); err != nil {
		result.Duration = time.Since(start)

		// Even on stats failure, ensure achievements object exists
		if result.Achievements == nil {
			result.Achievements = &models.AchievementData{
				AdeptSurvivors: make(map[string]bool),
				AdeptKillers:   make(map[string]bool),
				LastUpdated:    time.Now(),
			}
		}

		return result, err
	}

	result.Duration = time.Since(start)

	log.Info("Parallel fetch completed successfully",
		"steam_id", steamID,
		"stats_success", result.StatsError == nil,
		"achievements_success", result.AchError == nil,
		"stats_source", result.StatsSource,
		"achievements_source", result.AchSource,
		"total_duration_ms", result.Duration.Milliseconds(),
		"data_consistency", "guaranteed")

	return result, nil
}

// fetchStatsWithRetry fetches player stats with exponential backoff retry
func (f *ParallelFetcher) fetchStatsWithRetry(ctx context.Context, steamID string) (models.PlayerStats, string, error) {
	var lastErr error

	for attempt := 0; attempt <= f.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate backoff with jitter
			backoff := f.calculateBackoff(attempt)
			log.Debug("Retrying stats fetch after backoff",
				"steam_id", steamID,
				"attempt", attempt,
				"backoff", backoff)

			select {
			case <-time.After(backoff):
				// Continue with retry
			case <-ctx.Done():
				return models.PlayerStats{}, "timeout", ctx.Err()
			}
		}

		// Create per-request context with timeout
		reqCtx, cancel := context.WithTimeout(ctx, f.config.APITimeout)

		stats, source, err := f.fetchStatsOnce(reqCtx, steamID)
		cancel()

		if err == nil {
			if attempt > 0 {
				log.Info("Stats fetch succeeded after retry",
					"steam_id", steamID,
					"attempt", attempt,
					"source", source)
			}
			return stats, source, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			log.Debug("Non-retryable error, stopping retries",
				"steam_id", steamID,
				"error", err,
				"attempt", attempt)
			break
		}
	}

	return models.PlayerStats{}, "api", lastErr
}

// fetchAchievementsWithRetry fetches achievements with exponential backoff retry
func (f *ParallelFetcher) fetchAchievementsWithRetry(ctx context.Context, steamID string) (*models.AchievementData, string, error) {
	var lastErr error

	for attempt := 0; attempt <= f.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := f.calculateBackoff(attempt)
			log.Debug("Retrying achievements fetch after backoff",
				"steam_id", steamID,
				"attempt", attempt,
				"backoff", backoff)

			select {
			case <-time.After(backoff):
				// Continue with retry
			case <-ctx.Done():
				return nil, "timeout", ctx.Err()
			}
		}

		reqCtx, cancel := context.WithTimeout(ctx, f.config.APITimeout)

		achievements, source, err := f.fetchAchievementsOnce(reqCtx, steamID)
		cancel()

		if err == nil {
			if attempt > 0 {
				log.Info("Achievements fetch succeeded after retry",
					"steam_id", steamID,
					"attempt", attempt,
					"source", source)
			}
			return achievements, source, nil
		}

		lastErr = err

		// For achievements, be more permissive with retries since it's non-critical
		if !isRetryableError(err) && !isAchievementSpecificError(err) {
			log.Debug("Non-retryable achievement error, stopping retries",
				"steam_id", steamID,
				"error", err,
				"attempt", attempt)
			break
		}
	}

	return nil, "api", lastErr
}

// fetchStatsOnce performs a single stats fetch attempt
func (f *ParallelFetcher) fetchStatsOnce(_ context.Context, steamID string) (models.PlayerStats, string, error) {
	// Use the Steam client to fetch stats
	steamStats, apiErr := f.steamClient.GetPlayerStats(steamID)
	if apiErr != nil {
		return models.PlayerStats{}, "api", apiErr
	}

	// Convert steam stats to models.PlayerStats using existing mappers
	return models.PlayerStats{
		SteamID: steamStats.SteamID,
		// Integration with steam.MapSteamStats would be implemented here
	}, "api", nil
}

// fetchAchievementsOnce performs a single achievements fetch attempt
func (f *ParallelFetcher) fetchAchievementsOnce(_ context.Context, steamID string) (*models.AchievementData, string, error) {
	// Use the Steam client to fetch achievements
	_, apiErr := f.steamClient.GetPlayerAchievements(steamID, "381210") // DBD App ID
	if apiErr != nil {
		return nil, "api", apiErr
	}

	// Convert steam achievements to models.AchievementData using existing mappers
	return &models.AchievementData{
		AdeptSurvivors: make(map[string]bool),
		AdeptKillers:   make(map[string]bool),
		LastUpdated:    time.Now(),
	}, "api", nil
}

// calculateBackoff calculates exponential backoff with jitter
func (f *ParallelFetcher) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: base * (2^attempt)
	backoff := f.config.BaseBackoff * time.Duration(1<<uint(attempt))

	// Cap at maximum
	if backoff > f.config.MaxBackoff {
		backoff = f.config.MaxBackoff
	}

	// Add jitter (±25%) only if backoff is greater than 0
	if backoff > 0 {
		jitter := time.Duration(rand.Int63n(int64(backoff / 4)))
		if rand.Intn(2) == 0 {
			backoff += jitter
		} else {
			backoff -= jitter
		}
	}

	// Ensure minimum
	if backoff < f.config.BaseBackoff {
		backoff = f.config.BaseBackoff
	}

	return backoff
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errorType := classifyError(err)
	switch errorType {
	case "rate_limited", "timeout", "network_error", "steam_api_down":
		return true
	case "private_profile", "validation_error", "no_achievements":
		return false
	default:
		return true // Be optimistic about unknown errors
	}
}

// isAchievementSpecificError checks for achievement-specific errors that might be retryable
func isAchievementSpecificError(err error) bool {
	errorType := classifyError(err)
	return errorType == "no_achievements" || errorType == "private_profile"
}
