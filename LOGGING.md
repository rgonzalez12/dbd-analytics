# Structured Logging

JSON-based structured logging for enhanced observability and debugging.

## Configuration

```bash
# Set log level (default: info)
export LOG_LEVEL=debug
```

Example log output:
```json
{
  "time": "2025-08-03T01:41:47.6116719-04:00",
  "level": "INFO",
  "source": {"function": "main.main", "file": "main.go", "line": 92},
  "msg": "incoming_request",
  "method": "GET",
  "path": "/api/player/[steam_id]",
  "steam_id": "[steam_id]",
  "remote_addr": "[::1]:63035"
}
```

## Logged Events

### HTTP Requests
- `incoming_request`: Every HTTP request with method, path, Steam ID, client IP
- `request_completed`: Request completion with status code and duration

### Steam API
- `steam_api_request_start`: Outgoing Steam API requests
- `steam_api_request_completed`: Steam API responses with timing
- `steam_api_request_success`: Successful calls with response size
- `steam_api_request_failed`: Failed calls with error details

### Errors
- `API error response generated`: Error responses with request IDs
- `Invalid Steam ID format`: Validation errors
- `Steam API rate limit exceeded`: Rate limiting events
- `Network error during Steam API request`: Connection failures

### Success Events
- `successful_response_sent`: Successful JSON responses with size
- `Successfully retrieved player summary`: Steam API data retrieval

## Log Fields

Standard fields in all logs:
- `time`: RFC3339 timestamp
- `level`: Log level (INFO, WARN, ERROR, DEBUG)
- `msg`: Human-readable message
- `source`: Source code location (function, file, line)

Request-specific fields:
- `method`: HTTP method
- `path`: Request path
- `steam_id`: Steam ID from URL parameter
- `player_id`: Steam ID or vanity URL
- `user_agent`: Client user agent
- `remote_addr`: Client IP address
- `status_code`: HTTP response status
- `duration_ms`: Request duration in milliseconds
- `request_id`: Unique request identifier

Steam API fields:
- `endpoint`: Steam API endpoint URL
- `response_size`: Response body size in bytes
- `error_type`: Categorized error type

## Error Tracing

Each error response includes a unique `request_id` in:
1. HTTP response header (`X-Request-ID`)
2. JSON response body (`request_id` field)
3. Server logs for correlation

Example error response:
```json
{
  "error": "Invalid vanity URL format",
  "type": "validation_error",
  "details": "Invalid request parameters",
  "request_id": "faaca1bb184b7116",
  "source": "client_error"
}
```

## Log Analysis

Use `jq` to parse and analyze logs:

```bash
# Filter error logs
cat logs.json | jq 'select(.level == "ERROR")'

# Extract request timings
cat logs.json | jq 'select(.msg == "request_completed") | {path, duration_ms, status_code}'

# Find Steam API performance
cat logs.json | jq 'select(.msg == "steam_api_request_success") | {endpoint, duration_ms}'

# Trace specific request
cat logs.json | jq 'select(.request_id == "faaca1bb184b7116")'
```

## Environment Variables

```bash
LOG_LEVEL=debug                           # Log level
ACHIEVEMENTS_TIMEOUT_SECS=5               # Steam API timeout
CACHE_PLAYER_ACHIEVEMENTS_TTL=30m         # Cache TTL
```
