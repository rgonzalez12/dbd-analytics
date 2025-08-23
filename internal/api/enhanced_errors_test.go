package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/rgonzalez12/dbd-analytics/internal/steam"
)

func TestCriticalErrorHandling(t *testing.T) {
	t.Run("RateLimitErrorIncludesRetryAfter", func(t *testing.T) {
		apiErr := &steam.APIError{
			Type:       steam.ErrorTypeRateLimit,
			Message:    "Rate limit exceeded",
			StatusCode: 429,
			Retryable:  true,
		}

		w := httptest.NewRecorder()
		writeErrorResponse(w, apiErr)

		if w.Code != 429 {
			t.Errorf("Expected status 429, got %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["retry_after"] == nil {
			t.Error("Rate limit response must include retry_after field")
		}
		if response["retryable"] != true {
			t.Error("Rate limit response must be marked as retryable")
		}
	})

	t.Run("ValidationErrorsProvideDetails", func(t *testing.T) {
		apiErr := &steam.APIError{
			Type:       steam.ErrorTypeValidation,
			Message:    "Invalid Steam ID",
			StatusCode: 400,
			Retryable:  false,
		}

		w := httptest.NewRecorder()
		writeErrorResponse(w, apiErr)

		if w.Code != 400 {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["details"] == nil {
			t.Error("Validation errors must include details field")
		}
		// Validation errors should not include retryAfter (indicating they're not retryable)
		if response["retryAfter"] != nil {
			t.Error("Validation errors must not be retryable (should not have retryAfter)")
		}
	})
}
