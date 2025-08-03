package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rgonzalez12/dbd-analytics/internal/api"
	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func main() {
	// Initialize centralized structured logging
	log.Initialize()
	
	// Set working directory (helpful for Docker/deployment)
	if workDir := os.Getenv("WORKDIR"); workDir != "" {
		if err := os.Chdir(workDir); err != nil {
			log.Warn("Failed to change working directory", "dir", workDir, "error", err.Error())
		}
	}

	// Load environment variables from multiple possible locations
	envFiles := []string{".env", ".env.local", "../.env"}
	envLoaded := false
	
	for _, envFile := range envFiles {
		if err := godotenv.Load(envFile); err == nil {
			log.Info("Loaded environment file", "file", envFile)
			envLoaded = true
			break
		}
	}
	
	if !envLoaded {
		log.Warn("No environment file found, continuing with system environment variables")
	}

	// Get port configuration early
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"  // Default port
	}
	if port[0] != ':' {
		port = ":" + port  // Add colon if missing
	}

	// Initialize router
	r := mux.NewRouter()

	// Add comprehensive logging middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			
			// Create a response writer wrapper to capture status code
			wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			
			// Extract Steam ID from URL if present
			vars := mux.Vars(req)
			steamID := vars["steamid"]
			
			// Log incoming request
			log.Info("incoming_request",
				"method", req.Method,
				"path", req.URL.Path,
				"steam_id", steamID,
				"user_agent", req.UserAgent(),
				"remote_addr", req.RemoteAddr,
				"request_id", req.Header.Get("X-Request-ID"))
			
			// Process request
			next.ServeHTTP(wrappedWriter, req)
			
			// Log response
			duration := time.Since(start)
			log.Info("request_completed",
				"method", req.Method,
				"path", req.URL.Path,
				"steam_id", steamID,
				"status_code", wrappedWriter.statusCode,
				"duration", duration,
				"duration_ms", fmt.Sprintf("%.2f", duration.Seconds()*1000))
		})
	})

	// Initialize handlers
	h := api.NewHandler()

	// Home route
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Get the current host for dynamic URLs
		host := r.Host
		if host == "" {
			// If host is empty, use localhost with the configured port
			if strings.HasPrefix(port, ":") {
				host = "localhost" + port
			} else {
				host = "localhost:" + port
			}
		}
		
		fmt.Fprintln(w, "🎮 Dead by Daylight Analytics API")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "Available endpoints:")
		fmt.Fprintln(w, "GET /api/player/{steamID}/summary - Get player summary")
		fmt.Fprintln(w, "GET /api/player/{steamID}/stats - Get player DBD stats")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "Example URLs:")
		fmt.Fprintf(w, "  http://%s/api/player/invalid/summary\n", host)
		fmt.Fprintf(w, "  http://%s/api/player/76561198000000000/stats\n", host)
	}).Methods("GET")

	r.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Stats and Analytics coming soon...")
	}).Methods("GET")

	// Add debug route to list all routes
	r.HandleFunc("/debug/routes", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "Registered routes:")
		err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			pathTemplate, err := route.GetPathTemplate()
			if err == nil {
				methods, _ := route.GetMethods()
				fmt.Fprintf(w, "  %v %s\n", methods, pathTemplate)
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(w, "Error walking routes: %v\n", err)
		}
	}).Methods("GET")

	// Player routes (make sure these are exactly right)
	r.HandleFunc("/api/player/{steamid}/summary", h.GetPlayerSummary).Methods("GET")
	r.HandleFunc("/api/player/{steamid}/stats", h.GetPlayerStats).Methods("GET")

	// Start server with graceful shutdown
	// Port is already configured above
	
	server := &http.Server{
		Addr:    port,
		Handler: r,
	}

	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Info("Starting Dead by Daylight Analytics server", 
			"port", port,
			"version", "1.0.0")
		
		fmt.Printf("🚀 Server running on http://localhost%s\n", port)
		fmt.Printf("💡 Try: http://localhost%s/api/player/76561198000000000/summary\n", port)
		fmt.Printf("🐛 Debug routes: http://localhost%s/debug/routes\n", port)
		fmt.Println("⏹️  Press Ctrl+C to stop the server gracefully")
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Server failed to start", "error", err.Error())
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Info("Shutting down server gracefully...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err.Error())
		os.Exit(1)
	}

	log.Info("Server stopped gracefully")
}
