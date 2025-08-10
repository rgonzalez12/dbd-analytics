package steam

import (
	"fmt"
	"net/http"
)

// Predefined error variables for common scenarios
var (
	ErrRateLimit = &APIError{
		Type:       ErrorTypeRateLimit,
		Message:    "Steam API rate-limited, try again later",
		StatusCode: http.StatusTooManyRequests,
		Retryable:  true,
		RetryAfter: 60,
	}
	ErrUpstream = &APIError{
		Type:      ErrorTypeAPIError,
		Message:   "Steam API upstream error",
		Retryable: true,
	}
)

type ErrorType string

const (
	ErrorTypeRateLimit  ErrorType = "rate_limit"
	ErrorTypeNotFound   ErrorType = "not_found"
	ErrorTypeAPIError   ErrorType = "api_error"
	ErrorTypeNetwork    ErrorType = "network_error"
	ErrorTypeValidation ErrorType = "validation_error"
	ErrorTypeInternal   ErrorType = "internal_error"
)

type APIError struct {
	Type       ErrorType `json:"type"`
	Message    string    `json:"error"`
	StatusCode int       `json:"status_code,omitempty"`
	Retryable  bool      `json:"retryable,omitempty"`
	RetryAfter int       `json:"retry_after,omitempty"` // seconds to wait before retrying
}

func (e *APIError) Error() string {
	return e.Message
}

func NewRateLimitError() *APIError {
	return NewRateLimitErrorWithRetryAfter(60) // Default 60 seconds
}

func NewRateLimitErrorWithRetryAfter(retryAfter int) *APIError {
	return &APIError{
		Type:       ErrorTypeRateLimit,
		Message:    "Steam API rate-limited, try again later",
		StatusCode: http.StatusTooManyRequests,
		Retryable:  true,
		RetryAfter: retryAfter,
	}
}

func NewUnauthorizedError(message string) *APIError {
	return &APIError{
		Type:       ErrorTypeValidation,
		Message:    message,
		StatusCode: http.StatusUnauthorized,
		Retryable:  false,
	}
}

func NewForbiddenError(message string) *APIError {
	return &APIError{
		Type:       ErrorTypeValidation,
		Message:    message,
		StatusCode: http.StatusForbidden,
		Retryable:  false,
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
	retryable := isRetryableStatusCode(statusCode)
	return &APIError{
		Type:       ErrorTypeAPIError,
		Message:    fmt.Sprintf("Steam API error: %s", message),
		StatusCode: statusCode,
		Retryable:  retryable,
	}
}

func isRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusBadGateway,         // 502
		http.StatusServiceUnavailable, // 503
		http.StatusGatewayTimeout:     // 504
		return true
	default:
		return false
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
