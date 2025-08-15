package cache

import (
	"context"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

// WarmUpConfig defines cache warm-up behavior
type WarmUpConfig struct {
	Enabled          bool          `json:"enabled"`
	Timeout          time.Duration `json:"timeout"`
	ConcurrentJobs   int           `json:"concurrent_jobs"`
	RetryAttempts    int           `json:"retry_attempts"`
	RetryDelay       time.Duration `json:"retry_delay"`
	PopularPlayerIDs []string      `json:"popular_player_ids"`
}

func DefaultWarmUpConfig() WarmUpConfig {
	return WarmUpConfig{
		Enabled:          false, // Disabled by default - enable via env var
		Timeout:          30 * time.Second,
		ConcurrentJobs:   3,
		RetryAttempts:    2,
		RetryDelay:       1 * time.Second,
		PopularPlayerIDs: []string{
			// Add popular player IDs here when available
			// "76561198012345678",
			// "76561198087654321",
		},
	}
}

// WarmUpJob represents a single cache warm-up task
type WarmUpJob struct {
	Key        string
	Fetcher    func() (interface{}, error)
	Priority   int // Higher = more important
	MaxRetries int
}

// CacheWarmer handles pre-loading cache with common data
type CacheWarmer struct {
	manager *Manager
	config  WarmUpConfig
}

func NewCacheWarmer(manager *Manager, config WarmUpConfig) *CacheWarmer {
	return &CacheWarmer{
		manager: manager,
		config:  config,
	}
}

// WarmUp pre-loads cache with common lookups
func (cw *CacheWarmer) WarmUp(ctx context.Context, jobs []WarmUpJob) error {
	if !cw.config.Enabled {
		log.Info("Cache warm-up disabled, skipping")
		return nil
	}

	log.Info("Starting cache warm-up",
		"jobs", len(jobs),
		"concurrent_workers", cw.config.ConcurrentJobs,
		"timeout", cw.config.Timeout)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, cw.config.Timeout)
	defer cancel()

	// Create job channel
	jobChan := make(chan WarmUpJob, len(jobs))
	resultChan := make(chan WarmUpResult, len(jobs))

	// Start workers
	for i := 0; i < cw.config.ConcurrentJobs; i++ {
		go cw.warmUpWorker(ctx, jobChan, resultChan)
	}

	// Send jobs
	for _, job := range jobs {
		select {
		case jobChan <- job:
		case <-ctx.Done():
			log.Warn("Cache warm-up timeout reached while queuing jobs")
			break
		}
	}
	close(jobChan)

	// Collect results
	var successful, failed int
	for i := 0; i < len(jobs); i++ {
		select {
		case result := <-resultChan:
			if result.Success {
				successful++
			} else {
				failed++
				log.Debug("Cache warm-up job failed",
					"key", result.Key,
					"error", result.Error,
					"attempts", result.Attempts)
			}
		case <-ctx.Done():
			log.Warn("Cache warm-up timeout reached while collecting results")
			break
		}
	}

	log.Info("Cache warm-up completed",
		"successful", successful,
		"failed", failed,
		"total", len(jobs),
		"duration", time.Since(time.Now().Add(-cw.config.Timeout)))

	return nil
}

// WarmUpResult represents the outcome of a warm-up job
type WarmUpResult struct {
	Key      string
	Success  bool
	Error    error
	Attempts int
	Duration time.Duration
}

// warmUpWorker processes warm-up jobs
func (cw *CacheWarmer) warmUpWorker(ctx context.Context, jobs <-chan WarmUpJob, results chan<- WarmUpResult) {
	for job := range jobs {
		start := time.Now()
		success := false
		var lastErr error
		attempts := 0

		// Retry logic
		for attempts < cw.config.RetryAttempts {
			attempts++

			select {
			case <-ctx.Done():
				results <- WarmUpResult{
					Key:      job.Key,
					Success:  false,
					Error:    ctx.Err(),
					Attempts: attempts,
					Duration: time.Since(start),
				}
				return
			default:
			}

			// Try to fetch and cache
			data, err := job.Fetcher()
			if err != nil {
				lastErr = err
				if attempts < cw.config.RetryAttempts {
					time.Sleep(cw.config.RetryDelay)
				}
				continue
			}

			// Store in cache (use default TTL)
			if err := cw.manager.GetCache().Set(job.Key, data, 0); err != nil {
				lastErr = err
				continue
			}

			success = true
			break
		}

		results <- WarmUpResult{
			Key:      job.Key,
			Success:  success,
			Error:    lastErr,
			Attempts: attempts,
			Duration: time.Since(start),
		}
	}
}

func CreatePlayerStatsWarmUpJobs(playerIDs []string, fetcher func(playerID string) (interface{}, error)) []WarmUpJob {
	jobs := make([]WarmUpJob, len(playerIDs))

	for i, playerID := range playerIDs {
		playerID := playerID // Capture for closure
		jobs[i] = WarmUpJob{
			Key:      GenerateKey(PlayerStatsPrefix, playerID),
			Priority: 1,
			Fetcher: func() (interface{}, error) {
				return fetcher(playerID)
			},
			MaxRetries: 2,
		}
	}

	return jobs
}
