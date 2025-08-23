package cache

import (
	"math/rand"
	"time"
)

func addJitter(baseTimeout time.Duration, jitterPercent float64) time.Duration {
	if jitterPercent <= 0 || jitterPercent > 1.0 {
		jitterPercent = 0.1 // Default 10%
	}

	maxJitter := float64(baseTimeout) * jitterPercent
	jitter := time.Duration(rand.Float64() * maxJitter)

	return baseTimeout + jitter
}
