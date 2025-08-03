package log

import (
	"log/slog"
	"os"
	"strings"
)

var Logger *slog.Logger

// Initialize sets up the global structured logger
func Initialize() {
	logLevel := getLogLevel()
	
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true, // Add source file and line for debugging
	}))
	
	Logger = logger
	slog.SetDefault(logger)
}

// getLogLevel returns the appropriate log level from environment
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

// Info logs an informational message
func Info(msg string, args ...any) {
	if Logger == nil {
		Initialize()
	}
	Logger.Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	if Logger == nil {
		Initialize()
	}
	Logger.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	if Logger == nil {
		Initialize()
	}
	Logger.Error(msg, args...)
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	if Logger == nil {
		Initialize()
	}
	Logger.Debug(msg, args...)
}

// WithContext returns a logger with additional context fields
func WithContext(args ...any) *slog.Logger {
	if Logger == nil {
		Initialize()
	}
	return Logger.With(args...)
}
