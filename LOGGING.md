# Logging & Observability

Structured logging implementation for monitoring, debugging, and operational insights.

## Logging Approach

The application uses structured JSON logging with the following principles:

- Log all important events without exposing sensitive data
- Use structured JSON format for machine-readable analysis
- Include correlation IDs to track requests across components
- Capture performance metrics, cache statistics, and security events
- Monitor circuit breaker state and operational health

## Log Format

### Standard Structure
```json
{
  "timestamp": "2025-08-17T20:35:30.123Z",
  "level": "INFO",
  "source": {
    "function": "github.com/user/dbd-analytics/internal/api.GetPlayer",
    "file": "/app/internal/api/handlers.go",
    "line": 45
  },
  "msg": "player_request_completed",
  "request_id": "req_abc123",
  "steam_id": "7656***5835",
  "duration_ms": 1247.5,
  "cache_hit": true,
  "status_code": 200
}
```

### Field Descriptions
- `timestamp`: ISO 8601 UTC timestamp
- `level`: DEBUG, INFO, WARN, ERROR
- `source`: Code location for debugging
- `msg`: Event type identifier
- `request_id`: Correlation across services
- Custom fields: Context-specific data

## Event Types

### HTTP Request Tracking
```json
// Request started
{
  "msg": "incoming_request",
  "method": "GET",
  "path": "/api/player/76561198215615835",
  "client_ip": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "request_id": "req_abc123"
}

// Request completed
{
  "msg": "request_completed", 
  "request_id": "req_abc123",
  "status_code": 200,
  "duration_ms": 45.2,
  "response_size_bytes": 8192
}
```

### Cache Operations
```json
// Cache hit
{
  "msg": "cache_hit",
  "key": "player_stats:7656***5835",
  "ttl_remaining_seconds": 180,
  "data_size_bytes": 4096
}

// Cache miss - fetch required
{
  "msg": "cache_miss",
  "key": "player_stats:7656***5835", 
  "reason": "expired",
  "last_updated": "2025-08-17T20:30:00Z"
}
```

### Steam API Interactions
```json
// API call started
{
  "msg": "steam_api_request_start",
  "endpoint": "GetPlayerSummaries", 
  "steam_id": "7656***5835",
  "attempt": 1,
  "circuit_state": "closed"
}

// API call completed
{
  "msg": "steam_api_request_completed",
  "endpoint": "GetPlayerSummaries",
  "duration_ms": 234.5,
  "status_code": 200,
  "data_size_bytes": 1024
}

// API call failed
{
  "msg": "steam_api_request_failed",
  "endpoint": "GetPlayerSummaries", 
  "error": "timeout after 5s",
  "attempt": 2,
  "will_retry": true,
  "circuit_state": "closed"
}
```

### Grade Detection
```json
// Grade processing
{
  "msg": "grade_detection",
  "field_id": "DBD_SlasherTierIncrement",
  "raw_value": 439,
  "decoded_grade": "Bronze II",
  "is_killer": true,
  "processing_time_ms": 0.5
}

// Unknown grade encountered
{
  "msg": "unknown_grade_detected",
  "field_id": "DBD_SlasherTierIncrement", 
  "raw_value": 999,
  "fallback_display": "?",
  "action": "using_fallback"
}
```

### Circuit Breaker Events
```json
// Circuit breaker triggered
{
  "msg": "circuit_breaker_opened",
  "failure_count": 5,
  "last_error": "timeout",
  "reset_timeout_seconds": 60,
  "affected_endpoints": ["GetPlayerSummaries", "GetPlayerStats"]
}

// Circuit breaker recovered
{
  "msg": "circuit_breaker_closed",
  "recovery_successes": 3,
  "downtime_duration_seconds": 45,
  "total_stale_responses_served": 23
}
```

### Security Events
```json
// Rate limit hit
{
  "msg": "rate_limit_exceeded",
  "client_ip": "192.168.1.100",
  "requests_per_minute": 150,
  "limit": 100,
  "action": "request_blocked"
}

// Invalid input detected
{
  "msg": "validation_failed",
  "input_type": "steam_id",
  "input_value": "invalid123",
  "validation_rule": "must_be_17_digits",
  "client_ip": "192.168.1.100"
}
```

## ⚙️ Configuration

### Log Levels
```bash
# Environment variable
export LOG_LEVEL=info

# Available levels (in order)
DEBUG    # Detailed debugging info
INFO     # General operational messages  
WARN     # Warning conditions
ERROR    # Error conditions
```

### Development vs Production
```go
// Development: Human-readable
log.SetOutput(os.Stdout)
log.SetFormatter(&log.TextFormatter{
    FullTimestamp: true,
    ForceColors:   true,
})

// Production: JSON for log aggregation
log.SetOutput(os.Stdout)
log.SetFormatter(&log.JSONFormatter{
    TimestampFormat: time.RFC3339Nano,
})
```

## 📈 Operational Insights

### Performance Monitoring
```bash
# Find slow requests
jq 'select(.duration_ms > 1000)' app.log

# Cache hit rate analysis
jq 'select(.msg == "cache_hit" or .msg == "cache_miss")' app.log | \
  jq -s 'group_by(.msg) | map({type: .[0].msg, count: length})'

# Steam API health
jq 'select(.msg == "steam_api_request_failed")' app.log | \
  jq -s 'group_by(.error) | map({error: .[0].error, count: length})'
```

### Error Analysis
```bash
# Top error types
jq 'select(.level == "ERROR")' app.log | \
  jq -s 'group_by(.msg) | map({error_type: .[0].msg, count: length}) | sort_by(.count) | reverse'

# Circuit breaker events
jq 'select(.msg | contains("circuit_breaker"))' app.log
```

### Security Monitoring
```bash
# Rate limit violations
jq 'select(.msg == "rate_limit_exceeded")' app.log | \
  jq -s 'group_by(.client_ip) | map({ip: .[0].client_ip, violations: length})'

# Invalid requests by IP
jq 'select(.msg == "validation_failed")' app.log | \
  jq -s 'group_by(.client_ip) | map({ip: .[0].client_ip, attempts: length})'
```

## 🔧 Log Management

### Structured Query Examples
```bash
# All requests for a specific Steam ID
jq 'select(.steam_id == "7656***5835")' app.log

# Requests taking longer than 5 seconds
jq 'select(.duration_ms > 5000)' app.log

# Cache performance over time
jq 'select(.msg == "cache_hit" or .msg == "cache_miss") | {time: .timestamp, type: .msg}' app.log

# Grade detection events
jq 'select(.msg == "grade_detection")' app.log
```

### Production Log Aggregation
```yaml
# Example Fluentd/Logstash configuration
<filter dbd-analytics.**>
  @type parser
  format json
  key_name message
  reserve_data true
</filter>

<match dbd-analytics.**>
  @type elasticsearch
  host elasticsearch.local
  port 9200
  index_name dbd-analytics-${+YYYY.MM.dd}
</match>
```

## Metrics Extraction

### Key Performance Indicators
```bash
# Average response time
jq 'select(.duration_ms) | .duration_ms' app.log | jq -s 'add/length'

# Error rate percentage
errors=$(jq 'select(.level == "ERROR")' app.log | wc -l)
total=$(jq 'select(.status_code)' app.log | wc -l)
echo "scale=2; $errors * 100 / $total" | bc

# Cache hit rate
hits=$(jq 'select(.msg == "cache_hit")' app.log | wc -l)
misses=$(jq 'select(.msg == "cache_miss")' app.log | wc -l)
echo "scale=2; $hits * 100 / ($hits + $misses)" | bc
```

### Alerting Conditions
```yaml
# Example alerting rules
alerts:
  high_error_rate:
    query: 'level:"ERROR"'
    threshold: '>5% in 5m'
    
  slow_responses:
    query: 'duration_ms:>5000'
    threshold: '>10 in 1m'
    
  circuit_breaker_open:
    query: 'msg:"circuit_breaker_opened"'
    threshold: '>0 in 1m'
```

This logging system provides comprehensive observability into the DBD Analytics application, enabling effective monitoring, debugging, and operational management in production environments.
