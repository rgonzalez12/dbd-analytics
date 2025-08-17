package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rgonzalez12/dbd-analytics/internal/api"
	"github.com/rgonzalez12/dbd-analytics/internal/log"
	"github.com/rgonzalez12/dbd-analytics/internal/security"
)

func main() {
	log.Initialize()

	// Load environment variables first
	loadEnvironment()

	// Validate security configuration on startup
	if err := security.ValidateEnvironment(); err != nil {
		log.Error("Security validation failed", "error", err.Error())
		os.Exit(1)
	}

	port := getPort()
	r := setupRouter()

	fmt.Printf("🚀 Server running on http://localhost%s\n", port)
	fmt.Printf("💡 Try: http://localhost%s/api/player/76561198000000000\n", port)

	if err := http.ListenAndServe(port, r); err != nil {
		log.Error("Server failed", "error", err.Error())
		os.Exit(1)
	}
}

func loadEnvironment() {
	envFiles := []string{".env", ".env.local", "../.env"}
	for _, envFile := range envFiles {
		if err := godotenv.Load(envFile); err == nil {
			log.Info("Loaded environment file", "file", envFile)
			return
		}
	}
	log.Warn("No environment file found, using system environment variables")
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if port[0] != ':' {
		port = ":" + port
	}
	return port
}

func setupRouter() *mux.Router {
	r := mux.NewRouter()

	// Basic CORS middleware for development
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if req.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, req)
		})
	})

	h := api.NewHandler()

	// Simple routes
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "🎮 DBD Analytics API - TypeScript client test ready!")
	}).Methods("GET")

	r.HandleFunc("/api/player/{steamid}", h.GetPlayerStatsWithAchievements).Methods("GET", "OPTIONS")

	return r
}
