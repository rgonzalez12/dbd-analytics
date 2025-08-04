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

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func main() {
	log.Initialize()

	if workDir := os.Getenv("WORKDIR"); workDir != "" {
		if err := os.Chdir(workDir); err != nil {
			log.Warn("Failed to change working directory", "dir", workDir, "error", err.Error())
		}
	}

	loadEnvironment()

	port := getPort()
	r := setupRouter()
	server := &http.Server{Addr: port, Handler: r}

	startServer(server, port)
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
	r.Use(loggingMiddleware)

	h := api.NewHandler()

	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/stats", statsHandler).Methods("GET")
	r.HandleFunc("/debug/routes", debugRoutesHandler(r)).Methods("GET")

	r.HandleFunc("/api/player/{steamid}/summary", h.GetPlayerSummary).Methods("GET")
	r.HandleFunc("/api/player/{steamid}/stats", h.GetPlayerStats).Methods("GET")
	r.HandleFunc("/api/player/{steamid}", h.GetPlayerStatsWithAchievements).Methods("GET")

	r.HandleFunc("/api/cache/stats", h.GetCacheStats).Methods("GET")
	r.HandleFunc("/api/cache/evict", h.EvictExpiredEntries).Methods("POST")

	r.HandleFunc("/metrics", h.GetMetrics).Methods("GET")

	return r
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		vars := mux.Vars(req)
		steamID := vars["steamid"]

		log.Info("incoming_request",
			"method", req.Method,
			"path", req.URL.Path,
			"steam_id", steamID,
			"user_agent", req.UserAgent(),
			"remote_addr", req.RemoteAddr,
			"request_id", req.Header.Get("X-Request-ID"))

		next.ServeHTTP(wrappedWriter, req)

		duration := time.Since(start)
		log.Info("request_completed",
			"method", req.Method,
			"path", req.URL.Path,
			"steam_id", steamID,
			"status_code", wrappedWriter.statusCode,
			"duration", duration,
			"duration_ms", fmt.Sprintf("%.2f", duration.Seconds()*1000))
	})
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if host == "" {
		port := getPort()
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
	fmt.Fprintln(w, "GET /api/player/{steamID} - Get player stats WITH achievements (JSON)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Example URLs:")
	fmt.Fprintf(w, "  http://%s/api/player/76561198000000000/summary\n", host)
	fmt.Fprintf(w, "  http://%s/api/player/76561198000000000/stats\n", host)
	fmt.Fprintf(w, "  http://%s/api/player/76561198000000000 - 🏆 ACHIEVEMENTS JSON\n", host)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Stats and Analytics coming soon...")
}

func debugRoutesHandler(router *mux.Router) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "Registered routes:")
		err := router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
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
	}
}

func startServer(server *http.Server, port string) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

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

	<-quit
	log.Info("Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err.Error())
		os.Exit(1)
	}

	log.Info("Server stopped gracefully")
}
