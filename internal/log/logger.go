package log

import (
	"log/slog"
	"os"
	"strings"
)

var Logger *slog.Logger

func Initialize() {
	logLevel := getLogLevel()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	}))

	Logger = logger
	slog.SetDefault(logger)
}

func getLogLevel() slog.Level {
	level := strings.ToLower(os.Getenv("LOG_LEVEL"))
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func Info(msg string, args ...any) {
	if Logger == nil {
		Initialize()
	}
	Logger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	if Logger == nil {
		Initialize()
	}
	Logger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	if Logger == nil {
		Initialize()
	}
	Logger.Error(msg, args...)
}

func Debug(msg string, args ...any) {
	if Logger == nil {
		Initialize()
	}
	Logger.Debug(msg, args...)
}

func WithContext(args ...any) *slog.Logger {
	if Logger == nil {
		Initialize()
	}
	return Logger.With(args...)
}

func PlayerContext(playerID string) *slog.Logger {
	return WithContext("player_id", playerID)
}

func SteamAPIContext(playerID, endpoint string) *slog.Logger {
	return WithContext(
		"player_id", playerID,
		"endpoint", endpoint,
		"api_provider", "steam",
	)
}

func HTTPRequestContext(method, path, playerID, clientIP string) *slog.Logger {
	return WithContext(
		"method", method,
		"path", path,
		"player_id", playerID,
		"client_ip", clientIP,
		"request_type", "http",
	)
}

func ErrorContext(errorType, playerID string) *slog.Logger {
	return WithContext(
		"error_type", errorType,
		"player_id", playerID,
		"severity", "error",
	)
}

func PerformanceContext(operation, playerID string, durationMs float64) *slog.Logger {
	return WithContext(
		"operation", operation,
		"player_id", playerID,
		"duration_ms", durationMs,
		"metric_type", "performance",
	)
}
