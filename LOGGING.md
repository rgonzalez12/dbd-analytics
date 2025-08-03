# Structured Logging

The Dead by Daylight Analytics API uses structured JSON logging for enhanced observability and debugging.

## Log Configuration

The application uses Go's `log/slog` package with JSON output for structured logging:

```bash
# Set log level (default: info)
export LOG_LEVEL=debug

# Example log output
{"time":"2025-08-03T01:41:47.6116719-04:00","level":"INFO","source":{"function":"main.main.func1.1","file":"C:/Users/Ray/dbd-analytics/cmd/app/main.go","line":92},"msg":"incoming_request","method":"GET","path":"/api/player/76561198000000000/summary","steam_id":"76561198000000000","user_agent":"Mozilla/5.0 (Windows NT; Windows NT 10.0; en-US) WindowsPowerShell/5.1.26100.4652","remote_addr":"[::1]:63035","request_id":""}
```

## Logged Events

### HTTP Request Lifecycle
- **incoming_request**: Every incoming HTTP request with method, path, Steam ID, user agent, and client IP
- **request_completed**: Request completion with response status code and duration

### Steam API Calls
- **steam_api_request_start**: Outgoing Steam API requests with endpoint and method
- **steam_api_request_completed**: Steam API response with status code and timing
- **steam_api_request_success**: Successful Steam API calls with response size
- **steam_api_request_failed**: Failed Steam API calls with error details

### Error Handling
- **API error response generated**: Error responses sent to clients with request IDs for tracing
- **Invalid Steam ID format**: Validation errors with client details
- **Steam API rate limit exceeded**: Rate limiting events
- **Network error during Steam API request**: Connection failures

### Success Events
- **successful_response_sent**: Successful JSON responses with size information
- **Successfully retrieved player summary**: Steam API data retrieval success

## Log Fields

All logs include standard fields:
- `time`: RFC3339 timestamp
- `level`: Log level (INFO, WARN, ERROR, DEBUG)
- `msg`: Human-readable message
- `source`: Source code location (function, file, line)

Request-specific fields:
- `method`: HTTP method
- `path`: Request path
- `steam_id`: Steam ID from URL parameter
- `user_agent`: Client user agent
- `remote_addr`: Client IP address
- `status_code`: HTTP response status
- `duration`: Request duration in nanoseconds
- `duration_ms`: Request duration in milliseconds
- `request_id`: Unique request identifier for error tracing

Steam API fields:
- `endpoint`: Steam API endpoint URL
- `response_size`: Response body size in bytes
- `error_type`: Categorized error type (validation_error, network_error, etc.)

## Error Tracing

Each error response includes a unique `request_id` that appears in both:
1. HTTP response header (`X-Request-ID`)
2. JSON response body (`request_id` field)
3. Server logs for correlation

Example error response:
```json
{
  "error": "Invalid vanity URL format. Must be 3-32 characters, alphanumeric with underscore/hyphen only",
  "type": "validation_error",
  "details": "Invalid request parameters",
  "request_id": "faaca1bb184b7116",
  "source": "client_error"
}
```

## Log Analysis

Use tools like `jq` to parse and analyze logs:

```bash
# Filter error logs
cat logs.json | jq 'select(.level == "ERROR")'

# Extract request timings
cat logs.json | jq 'select(.msg == "request_completed") | {path, duration_ms, status_code}'

# Find Steam API performance
cat logs.json | jq 'select(.msg == "steam_api_request_success") | {endpoint, duration_ms, response_size}'

# Trace specific request by ID
cat logs.json | jq 'select(.request_id == "faaca1bb184b7116")'
```

## Monitoring Integration

The structured logs are designed for integration with monitoring tools:
- **Prometheus**: Extract metrics from duration and status code fields
- **Grafana**: Visualize request patterns and error rates
- **ELK Stack**: Index and search log events
- **Datadog/New Relic**: Application performance monitoring
