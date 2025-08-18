package api

import (
	"github.com/gorilla/mux"
	"time"
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
	router.HandleFunc("/player/{steamid}", handler.GetPlayerStatsWithAchievements).Methods("GET")

	// Health endpoints
	router.HandleFunc("/health", handler.HealthCheck).Methods("GET")
	router.HandleFunc("/healthz", handler.HealthCheck).Methods("GET") // Kubernetes-style healthcheck
}
