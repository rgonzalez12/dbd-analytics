package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

func TestStructuredLoggingValidation(t *testing.T) {
	// Capture log output for testing
	var logOutput bytes.Buffer
	
	// Setup JSON logger for testing
	logger := slog.New(slog.NewJSONHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	
	tests := []struct {
		name              string
		steamID           string
		expectedStatus    int
		expectedLogFields []string
	}{
		{
			name:           "Invalid Steam ID",
			steamID:        "invalid@id",
			expectedStatus: 400,
			expectedLogFields: []string{
				"incoming_request",
				"Invalid Steam ID format",
				"API error response generated",
				"request_completed",
			},
		},
		{
			name:           "Short Steam ID",
			steamID:        "123",
			expectedStatus: 400,
			expectedLogFields: []string{
				"incoming_request",
				"Invalid Steam ID format",
				"API error response generated",
				"request_completed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear log output
			logOutput.Reset()
			
			// Create handler
			handler := NewHandler()
			router := mux.NewRouter()
			
			// Add the same logging middleware as main.go
			router.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					vars := mux.Vars(req)
					steamID := vars["steamid"]
					
					slog.Info("incoming_request",
						slog.String("method", req.Method),
						slog.String("path", req.URL.Path),
						slog.String("steam_id", steamID))
					
					next.ServeHTTP(w, req)
					
					slog.Info("request_completed",
						slog.String("method", req.Method),
						slog.String("path", req.URL.Path),
						slog.String("steam_id", steamID))
				})
			})
			
			router.HandleFunc("/api/player/{steamid}/summary", handler.GetPlayerSummary).Methods("GET")

			req := httptest.NewRequest("GET", "/api/player/"+tt.steamID+"/summary", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Parse and verify log output
			logLines := strings.Split(strings.TrimSpace(logOutput.String()), "\n")
			logStr := logOutput.String()
			
			// Check that expected log fields are present
			for _, expectedField := range tt.expectedLogFields {
				if !strings.Contains(logStr, expectedField) {
					t.Errorf("Expected log field '%s' not found in logs", expectedField)
				}
			}

			// Verify JSON structure of logs
			for _, line := range logLines {
				if line == "" {
					continue
				}
				var logEntry map[string]interface{}
				if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
					t.Errorf("Log line is not valid JSON: %s", line)
					continue
				}
				
				// Verify required fields
				requiredFields := []string{"time", "level", "msg"}
				for _, field := range requiredFields {
					if logEntry[field] == nil {
						t.Errorf("Log entry missing required field '%s': %s", field, line)
					}
				}
			}

			t.Logf("Generated %d log lines for test case", len(logLines))
		})
	}
}

func TestLogOutputFormat(t *testing.T) {
	// Test that we're using JSON format
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	
	slog.Info("test_message", 
		slog.String("test_field", "test_value"),
		slog.Int("test_number", 42))
	
	logLine := strings.TrimSpace(logOutput.String())
	
	// Verify it's valid JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(logLine), &logEntry); err != nil {
		t.Fatalf("Log output is not valid JSON: %s", logLine)
	}
	
	// Verify structure
	if logEntry["msg"] != "test_message" {
		t.Errorf("Expected msg to be 'test_message', got %v", logEntry["msg"])
	}
	
	if logEntry["test_field"] != "test_value" {
		t.Errorf("Expected test_field to be 'test_value', got %v", logEntry["test_field"])
	}
	
	if logEntry["test_number"] != float64(42) {
		t.Errorf("Expected test_number to be 42, got %v", logEntry["test_number"])
	}
}

func TestErrorLogging(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	
	// Test error response logging
	w := httptest.NewRecorder()
	apiErr := &steam.APIError{
		Type:       steam.ErrorTypeValidation,
		Message:    "Test validation error",
		StatusCode: 400,
		Retryable:  false,
	}
	
	writeErrorResponse(w, apiErr)
	
	logStr := logOutput.String()
	
	// Should contain error response log
	if !strings.Contains(logStr, "API error response generated") {
		t.Error("Expected 'API error response generated' log not found")
	}
	
	// Should contain request ID
	if !strings.Contains(logStr, "request_id") {
		t.Error("Expected 'request_id' field not found in error logs")
	}
	
	// Parse the log line to verify structure
	lines := strings.Split(strings.TrimSpace(logStr), "\n")
	for _, line := range lines {
		if strings.Contains(line, "API error response generated") {
			var logEntry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
				t.Errorf("Error log line is not valid JSON: %s", line)
			}
			
			if logEntry["error_type"] != "validation_error" {
				t.Errorf("Expected error_type to be 'validation_error', got %v", logEntry["error_type"])
			}
			
			if logEntry["status_code"] != float64(400) {
				t.Errorf("Expected status_code to be 400, got %v", logEntry["status_code"])
			}
		}
	}
}
