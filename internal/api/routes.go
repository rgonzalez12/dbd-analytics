package api

import (
	"time"
	"github.com/gorilla/mux"
)

func RegisterRoutes(router *mux.Router) {
	handler := NewHandler()

	// Create rate limiter (100 requests per minute per client)
	rateLimiter := NewRequestLimiter(100, time.Minute)

	// Apply global middleware for all routes
	router.Use(RequestIDMiddleware())
	router.Use(SecurityMiddleware())
	router.Use(RateLimitMiddleware(rateLimiter))
	router.Use(APIKeyMiddleware())

	// Player data endpoints
	router.HandleFunc("/player/{steamid}/summary", handler.GetPlayerSummary).Methods("GET")
	router.HandleFunc("/player/{steamid}/stats", handler.GetPlayerStats).Methods("GET")
	router.HandleFunc("/player/{steamid}", handler.GetPlayerStatsWithAchievements).Methods("GET")

	// Cache management endpoints (useful for monitoring and debugging)
	router.HandleFunc("/cache/stats", handler.GetCacheStats).Methods("GET")
	router.HandleFunc("/cache/evict", handler.EvictExpiredEntries).Methods("POST")

	// PATCH START - Production-ready health and metrics endpoints
	// Health and metrics endpoints
	router.HandleFunc("/health", handler.HealthCheck).Methods("GET")
	router.HandleFunc("/healthz", handler.HealthCheck).Methods("GET") // Kubernetes-style healthcheck
	router.HandleFunc("/metrics", handler.GetMetrics).Methods("GET")
	// PATCH END
}
