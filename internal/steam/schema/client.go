package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

const (
	SchemaEndpoint = "https://partner.steam-api.com/ISteamUserStats/GetSchemaForGame/v2"
	DefaultAppID   = "381210" // Dead by Daylight
	DefaultLang    = "en"
	DefaultTTL     = 24 * time.Hour
)

// SchemaClient handles Steam game schema operations
type SchemaClient struct {
	httpClient *http.Client
	cache      *SchemaCache
	apiKey     string
}

// SchemaCache provides in-memory caching with TTL
type SchemaCache struct {
	mu       sync.RWMutex
	data     map[string]*CacheEntry
	ttl      time.Duration
	cleanup  *time.Ticker
	stopChan chan struct{}
}

// CacheEntry represents a cached schema with metadata
type CacheEntry struct {
	Schema    *Schema
	FetchedAt time.Time
	ETag      string
	ExpiresAt time.Time
}

// Schema represents the Steam game schema
type Schema struct {
	AppID        string                     `json:"appid"`
	Language     string                     `json:"language"`
	Achievements map[string]AchievementMeta `json:"achievements"`
	Stats        map[string]string          `json:"stats"`
	FetchedAt    time.Time                  `json:"fetched_at"`
}

// AchievementMeta contains humanized achievement metadata
type AchievementMeta struct {
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	IconGray    string `json:"icon_gray"`
	Hidden      bool   `json:"hidden"`
}

// Raw Steam API response types
type schemaResponse struct {
	Game GameSchema `json:"game"`
}

type GameSchema struct {
	GameName     string                   `json:"gameName"`
	GameVersion  string                   `json:"gameVersion"`
	Achievements []RawAchievementSchema   `json:"availableGameStats"`
	Stats        []RawStatSchema          `json:"stats"`
}

type RawAchievementSchema struct {
	Name         string `json:"name"`
	DefaultValue int    `json:"defaultvalue"`
	DisplayName  string `json:"displayName"`
	Hidden       int    `json:"hidden"`
	Description  string `json:"description"`
	Icon         string `json:"icon"`
	IconGray     string `json:"icongray"`
}

type RawStatSchema struct {
	Name        string `json:"name"`
	DefaultValue int   `json:"defaultvalue"`
	DisplayName string `json:"displayName"`
}

// NewSchemaClient creates a new schema client with caching
func NewSchemaClient() *SchemaClient {
	apiKey := os.Getenv("STEAM_API_KEY")
	if apiKey == "" {
		log.Info("STEAM_API_KEY not set for schema client")
	}

	// Parse TTL from environment
	ttl := DefaultTTL
	if ttlHours := os.Getenv("STEAM_SCHEMA_TTL_HOURS"); ttlHours != "" {
		if hours, err := strconv.Atoi(ttlHours); err == nil && hours > 0 {
			ttl = time.Duration(hours) * time.Hour
		}
	}

	cache := NewSchemaCache(ttl)
	
	return &SchemaClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:  cache,
		apiKey: apiKey,
	}
}

// NewSchemaCache creates a new schema cache with TTL and cleanup
func NewSchemaCache(ttl time.Duration) *SchemaCache {
	cache := &SchemaCache{
		data:     make(map[string]*CacheEntry),
		ttl:      ttl,
		cleanup:  time.NewTicker(time.Hour), // Clean up every hour
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// GetSchemaForGame fetches the game schema with caching
func (c *SchemaClient) GetSchemaForGame(ctx context.Context, appID, lang string) (*Schema, error) {
	if appID == "" {
		appID = DefaultAppID
	}
	if lang == "" {
		lang = DefaultLang
	}

	cacheKey := fmt.Sprintf("%s:%s", appID, lang)

	// Try cache first
	if schema := c.cache.Get(cacheKey); schema != nil {
		log.Info("Schema cache hit", "app_id", appID, "language", lang, "fetched_at", schema.FetchedAt)
		return schema, nil
	}

	// Fetch from Steam API
	schema, err := c.fetchSchemaFromAPI(ctx, appID, lang)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schema from Steam API: %w", err)
	}

	// Cache the result
	c.cache.Set(cacheKey, schema, "")

	log.Info("Schema fetched and cached", "app_id", appID, "language", lang, "achievements", len(schema.Achievements), "stats", len(schema.Stats))

	return schema, nil
}

// ForceRefresh forces a refresh of the schema bypassing cache
func (c *SchemaClient) ForceRefresh(ctx context.Context, appID, lang string) (*Schema, error) {
	if appID == "" {
		appID = DefaultAppID
	}
	if lang == "" {
		lang = DefaultLang
	}

	cacheKey := fmt.Sprintf("%s:%s", appID, lang)

	// Clear cache entry
	c.cache.Delete(cacheKey)

	// Fetch fresh data
	return c.GetSchemaForGame(ctx, appID, lang)
}

// fetchSchemaFromAPI makes the actual HTTP request to Steam
func (c *SchemaClient) fetchSchemaFromAPI(ctx context.Context, appID, lang string) (*Schema, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("STEAM_API_KEY environment variable not set")
	}

	// Build URL
	baseURL, err := url.Parse(SchemaEndpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid schema endpoint URL: %w", err)
	}

	params := url.Values{}
	params.Set("key", c.apiKey)
	params.Set("appid", appID)
	params.Set("l", lang)
	baseURL.RawQuery = params.Encode()

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", "dbd-analytics/1.0")

	// Make request with retries
	resp, err := c.doRequestWithRetries(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var apiResp schemaResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our schema format
	schema := &Schema{
		AppID:        appID,
		Language:     lang,
		Achievements: make(map[string]AchievementMeta),
		Stats:        make(map[string]string),
		FetchedAt:    time.Now(),
	}

	// Process achievements
	for _, ach := range apiResp.Game.Achievements {
		schema.Achievements[ach.Name] = AchievementMeta{
			DisplayName: ach.DisplayName,
			Description: ach.Description,
			Icon:        ach.Icon,
			IconGray:    ach.IconGray,
			Hidden:      ach.Hidden == 1,
		}
	}

	// Process stats
	for _, stat := range apiResp.Game.Stats {
		schema.Stats[stat.Name] = stat.DisplayName
	}

	return schema, nil
}

// doRequestWithRetries implements exponential backoff retry logic
func (c *SchemaClient) doRequestWithRetries(req *http.Request) (*http.Response, error) {
	maxRetries := 3
	baseDelay := 100 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err == nil {
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return resp, nil
			}
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		// Don't retry on client errors (4xx)
		if resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return nil, lastErr
		}

		// Wait before retrying
		if attempt < maxRetries-1 {
			delay := time.Duration(1<<uint(attempt)) * baseDelay
			log.Info("Schema API retry", "attempt", attempt+1, "delay_ms", delay.Milliseconds(), "error", lastErr.Error())
			time.Sleep(delay)
		}
	}

	return nil, fmt.Errorf("schema fetch failed after %d attempts: %w", maxRetries, lastErr)
}

// Cache methods

// Get retrieves a schema from cache if not expired
func (c *SchemaCache) Get(key string) *Schema {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.data[key]
	if !exists || time.Now().After(entry.ExpiresAt) {
		return nil
	}

	return entry.Schema
}

// Set stores a schema in cache with TTL
func (c *SchemaCache) Set(key string, schema *Schema, etag string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = &CacheEntry{
		Schema:    schema,
		FetchedAt: time.Now(),
		ETag:      etag,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes a schema from cache
func (c *SchemaCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, key)
}

// cleanupLoop runs periodic cleanup of expired entries
func (c *SchemaCache) cleanupLoop() {
	for {
		select {
		case <-c.cleanup.C:
			c.cleanupExpired()
		case <-c.stopChan:
			c.cleanup.Stop()
			return
		}
	}
}

// cleanupExpired removes expired entries from cache
func (c *SchemaCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.data {
		if now.After(entry.ExpiresAt) {
			delete(c.data, key)
		}
	}
}

// Close stops the cleanup goroutine
func (c *SchemaCache) Close() {
	close(c.stopChan)
}
