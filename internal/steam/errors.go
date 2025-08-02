package steam

import (
	"fmt"
	"net/http"
)

type ErrorType string

const (
	ErrorTypeRateLimit   ErrorType = "rate_limit"
	ErrorTypeNotFound    ErrorType = "not_found"
	ErrorTypeAPIError    ErrorType = "api_error"
	ErrorTypeNetwork     ErrorType = "network_error"
	ErrorTypeValidation  ErrorType = "validation_error"
	ErrorTypeInternal    ErrorType = "internal_error"
)

type APIError struct {
	Type       ErrorType `json:"type"`
	Message    string    `json:"error"`
	StatusCode int       `json:"status_code,omitempty"`
	Retryable  bool      `json:"retryable,omitempty"`
}

func (e *APIError) Error() string {
	return e.Message
}

func NewRateLimitError() *APIError {
	return &APIError{
		Type:       ErrorTypeRateLimit,
		Message:    "Steam API rate-limited, try again later",
		StatusCode: http.StatusTooManyRequests,
		Retryable:  true,
	}
}

func NewNotFoundError(resource string) *APIError {
	return &APIError{
		Type:       ErrorTypeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		StatusCode: http.StatusNotFound,
		Retryable:  false,
	}
}

func NewAPIError(statusCode int, message string) *APIError {
	return &APIError{
		Type:       ErrorTypeAPIError,
		Message:    fmt.Sprintf("Steam API error: %s", message),
		StatusCode: statusCode,
		Retryable:  statusCode >= 500,
	}
}

func NewNetworkError(err error) *APIError {
	return &APIError{
		Type:      ErrorTypeNetwork,
		Message:   fmt.Sprintf("Network error: %v", err),
		Retryable: true,
	}
}

func NewValidationError(message string) *APIError {
	return &APIError{
		Type:       ErrorTypeValidation,
		Message:    message,
		StatusCode: http.StatusBadRequest,
		Retryable:  false,
	}
}

func NewInternalError(err error) *APIError {
	return &APIError{
		Type:      ErrorTypeInternal,
		Message:   fmt.Sprintf("Internal error: %v", err),
		Retryable: false,
	}
}
