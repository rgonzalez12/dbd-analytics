package cache

import (
	"math/rand"
	"time"
)

// addJitter adds random jitter to prevent thundering herd problems
func addJitter(baseTimeout time.Duration, jitterPercent float64) time.Duration {
	if jitterPercent <= 0 || jitterPercent > 1.0 {
		jitterPercent = 0.1 // Default 10% jitter
	}

	maxJitter := float64(baseTimeout) * jitterPercent
	jitter := time.Duration(rand.Float64() * maxJitter)

	return baseTimeout + jitter
}

// _addJitterWithSeed adds jitter with a deterministic seed for testing
// Prefixed with _ to indicate it's intentionally unused but kept for future testing
func _addJitterWithSeed(baseTimeout time.Duration, jitterPercent float64, seed int64) time.Duration {
	if jitterPercent <= 0 || jitterPercent > 1.0 {
		jitterPercent = 0.1
	}

	source := rand.NewSource(seed)
	rng := rand.New(source)

	maxJitter := float64(baseTimeout) * jitterPercent
	jitter := time.Duration(rng.Float64() * maxJitter)

	return baseTimeout + jitter
}
