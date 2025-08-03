package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rgonzalez12/dbd-analytics/internal/api"
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
	// Set working directory (helpful for Docker/deployment)
	if workDir := os.Getenv("WORKDIR"); workDir != "" {
		if err := os.Chdir(workDir); err != nil {
			slog.Warn("Failed to change working directory", slog.String("dir", workDir), slog.String("error", err.Error()))
		}
	}

	// Initialize structured logging with JSON output for better observability
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}
	
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
		AddSource: true, // Add source file and line for debugging
	}))
	slog.SetDefault(logger)

	// Load environment variables from multiple possible locations
	envFiles := []string{".env", ".env.local", "../.env"}
	envLoaded := false
	
	for _, envFile := range envFiles {
		if err := godotenv.Load(envFile); err == nil {
			slog.Info("Loaded environment file", slog.String("file", envFile))
			envLoaded = true
			break
		}
	}
	
	if !envLoaded {
		slog.Warn("No environment file found, continuing with system environment variables")
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
			slog.Info("incoming_request",
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.String("steam_id", steamID),
				slog.String("user_agent", req.UserAgent()),
				slog.String("remote_addr", req.RemoteAddr),
				slog.String("request_id", req.Header.Get("X-Request-ID")))
			
			// Process request
			next.ServeHTTP(wrappedWriter, req)
			
			// Log response
			duration := time.Since(start)
			slog.Info("request_completed",
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.String("steam_id", steamID),
				slog.Int("status_code", wrappedWriter.statusCode),
				slog.Duration("duration", duration),
				slog.String("duration_ms", fmt.Sprintf("%.2f", duration.Seconds()*1000)))
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
		slog.Info("Starting Dead by Daylight Analytics server", 
			slog.String("port", port),
			slog.String("version", "1.0.0"))
		
		fmt.Printf("🚀 Server running on http://localhost%s\n", port)
		fmt.Printf("💡 Try: http://localhost%s/api/player/76561198000000000/summary\n", port)
		fmt.Printf("🐛 Debug routes: http://localhost%s/debug/routes\n", port)
		fmt.Println("⏹️  Press Ctrl+C to stop the server gracefully")
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", slog.String("error", err.Error()))
			log.Fatal("Server failed to start:", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	slog.Info("Shutting down server gracefully...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", slog.String("error", err.Error()))
		log.Fatal("Server forced to shutdown:", err)
	}

	slog.Info("Server stopped gracefully")
}
