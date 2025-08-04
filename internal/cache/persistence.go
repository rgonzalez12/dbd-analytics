package cache

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

// PersistentCircuitBreakerState represents the state that should survive restarts
type PersistentCircuitBreakerState struct {
	State           CircuitState `json:"state"`
	Failures        int          `json:"failures"`
	LastFailureTime time.Time    `json:"last_failure_time"`
	OpenCount       int64        `json:"open_count"`
	LastSaved       time.Time    `json:"last_saved"`
}

// StatePersistence handles saving and loading circuit breaker state
type StatePersistence struct {
	filePath string
	mu       sync.RWMutex
}

// NewStatePersistence creates a new state persistence handler
func NewStatePersistence(filePath string) *StatePersistence {
	return &StatePersistence{
		filePath: filePath,
	}
}

// SaveState persists circuit breaker state to disk
func (sp *StatePersistence) SaveState(cb *CircuitBreaker) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	state := PersistentCircuitBreakerState{
		State:           cb.state,
		Failures:        cb.failures,
		LastFailureTime: cb.lastFailureTime,
		OpenCount:       cb.metrics.CircuitOpenCount,
		LastSaved:       time.Now(),
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file first, then atomic rename
	tempFile := sp.filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return err
	}

	return os.Rename(tempFile, sp.filePath)
}

// LoadState restores circuit breaker state from disk
func (sp *StatePersistence) LoadState() (*PersistentCircuitBreakerState, error) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	data, err := os.ReadFile(sp.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return default state
			return &PersistentCircuitBreakerState{
				State: CircuitClosed,
			}, nil
		}
		return nil, err
	}

	var state PersistentCircuitBreakerState
	if err := json.Unmarshal(data, &state); err != nil {
		log.Warn("Failed to parse circuit breaker state file, using defaults", "error", err)
		return &PersistentCircuitBreakerState{
			State: CircuitClosed,
		}, nil
	}

	// Validate loaded state - if it's too old, reset to closed
	if time.Since(state.LastSaved) > 24*time.Hour {
		log.Info("Circuit breaker state file is stale, resetting to closed",
			"age", time.Since(state.LastSaved))
		return &PersistentCircuitBreakerState{
			State: CircuitClosed,
		}, nil
	}

	return &state, nil
}

// RestoreState applies persisted state to a circuit breaker
func (cb *CircuitBreaker) RestoreState(persistence *StatePersistence) error {
	state, err := persistence.LoadState()
	if err != nil {
		return err
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = state.State
	cb.failures = state.Failures
	cb.lastFailureTime = state.LastFailureTime
	cb.metrics.CircuitOpenCount = state.OpenCount

	log.Info("Circuit breaker state restored",
		"state", cb.getStateString(),
		"failures", cb.failures,
		"last_failure", cb.lastFailureTime)

	return nil
}

// EnableStatePersistence adds automatic state persistence to a circuit breaker
func (cb *CircuitBreaker) EnableStatePersistence(filePath string) *StatePersistence {
	persistence := NewStatePersistence(filePath)

	// Try to restore existing state
	if err := cb.RestoreState(persistence); err != nil {
		log.Warn("Failed to restore circuit breaker state", "error", err)
	}

	// Save state periodically (every 30 seconds)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if err := persistence.SaveState(cb); err != nil {
				log.Error("Failed to save circuit breaker state", "error", err)
			}
		}
	}()

	return persistence
}
