package api

import (
	"github.com/gorilla/mux"
)

func RegisterRoutes(router *mux.Router) {
	handler := NewHandler()

	// Player data endpoints
	router.HandleFunc("/player/{steamid}/summary", handler.GetPlayerSummary).Methods("GET")
	router.HandleFunc("/player/{steamid}/stats", handler.GetPlayerStats).Methods("GET")
	router.HandleFunc("/player/{steamid}", handler.GetPlayerStatsWithAchievements).Methods("GET")

	// Cache management endpoints (useful for monitoring and debugging)
	router.HandleFunc("/cache/stats", handler.GetCacheStats).Methods("GET")
	router.HandleFunc("/cache/evict", handler.EvictExpiredEntries).Methods("POST")
}
