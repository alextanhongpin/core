# HTTP Health Package

The HTTP Health package provides comprehensive health check endpoints for monitoring application status, suitable for load balancers, Kubernetes probes, and observability systems.

## Features

- **Standard Health Endpoints**: Ready-to-use health check HTTP handlers
- **Customizable Checks**: Add custom health checks for databases, services, etc.
- **Status Classification**: Healthy, unhealthy, and degraded states
- **Check Timing**: Latency measurements for each health check
- **Uptime Tracking**: Application uptime reporting
- **Version Information**: Version details included in responses

## Quick Start

```go
package main

import (
    "net/http"
    "database/sql"
    
    "github.com/alextanhongpin/core/http/health"
)

func main() {
    // Create health handler with application version
    healthHandler := health.New("1.0.0")
    
    // Add database health check
    db, _ := sql.Open("postgres", "postgres://localhost:5432/myapp")
    healthHandler.AddCheck("database", func() health.Check {
        start := time.Now()
        err := db.Ping()
        latency := time.Since(start)
        
        if err != nil {
            return health.Check{
                Status:  health.StatusUnhealthy,
                Message: err.Error(),
                Latency: latency,
            }
        }
        
        return health.Check{
            Status:  health.StatusHealthy,
            Message: "Database connection successful",
            Latency: latency,
        }
    })
    
    // Register health endpoints
    http.HandleFunc("/health", healthHandler.Health)
    http.HandleFunc("/health/liveness", healthHandler.Liveness)
    http.HandleFunc("/health/readiness", healthHandler.Readiness)
    
    http.ListenAndServe(":8080", nil)
}
```

## API Reference

### Handler Initialization

#### `New(version string) *Handler`

Creates a new health check handler with the specified version.

```go
healthHandler := health.New("1.0.0")
```

### Adding Health Checks

#### `AddCheck(name string, checker Checker) *Handler`

Adds a named health check to the handler.

```go
healthHandler.AddCheck("redis", func() health.Check {
    // Check Redis connection
    if err := redisClient.Ping().Err(); err != nil {
        return health.Check{
            Status:  health.StatusUnhealthy,
            Message: err.Error(),
        }
    }
    return health.Check{
        Status: health.StatusHealthy,
    }
})
```

### HTTP Handlers

#### `Health(w http.ResponseWriter, r *http.Request)`

Comprehensive health check endpoint that runs all registered checks.

```go
http.HandleFunc("/health", healthHandler.Health)
```

#### `Liveness(w http.ResponseWriter, r *http.Request)`

Lightweight health check for liveness probes (is the application running?).

```go
http.HandleFunc("/health/liveness", healthHandler.Liveness)
```

#### `Readiness(w http.ResponseWriter, r *http.Request)`

Checks if the application is ready to receive traffic.

```go
http.HandleFunc("/health/readiness", healthHandler.Readiness)
```

### Status Types

```go
const (
    StatusHealthy   Status = "healthy"
    StatusUnhealthy Status = "unhealthy"
    StatusDegraded  Status = "degraded"
)
```

### Response Structure

```json
{
  "status": "healthy",
  "timestamp": "2023-05-12T14:30:45.123Z",
  "version": "1.0.0",
  "uptime": "24h0m0s",
  "checks": {
    "database": {
      "status": "healthy",
      "message": "Connected successfully",
      "latency": "15ms"
    },
    "redis": {
      "status": "healthy",
      "message": "PONG",
      "latency": "5ms"
    },
    "external-api": {
      "status": "degraded",
      "message": "High latency detected",
      "latency": "500ms"
    }
  }
}
```

## Common Health Check Implementations

### Database Check

```go
healthHandler.AddCheck("database", func() health.Check {
    start := time.Now()
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()
    
    err := db.PingContext(ctx)
    latency := time.Since(start)
    
    if err != nil {
        return health.Check{
            Status:  health.StatusUnhealthy,
            Message: err.Error(),
            Latency: latency,
        }
    }
    
    return health.Check{
        Status:  health.StatusHealthy,
        Message: "Database connection successful",
        Latency: latency,
    }
})
```

### Redis Check

```go
healthHandler.AddCheck("redis", func() health.Check {
    start := time.Now()
    
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()
    
    result, err := redisClient.Ping(ctx).Result()
    latency := time.Since(start)
    
    if err != nil {
        return health.Check{
            Status:  health.StatusUnhealthy,
            Message: err.Error(),
            Latency: latency,
        }
    }
    
    return health.Check{
        Status:  health.StatusHealthy,
        Message: result,
        Latency: latency,
    }
})
```

### External API Check

```go
healthHandler.AddCheck("payment-service", func() health.Check {
    start := time.Now()
    
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    req, _ := http.NewRequestWithContext(ctx, "GET", "https://payment-api/health", nil)
    resp, err := http.DefaultClient.Do(req)
    latency := time.Since(start)
    
    if err != nil {
        return health.Check{
            Status:  health.StatusUnhealthy,
            Message: err.Error(),
            Latency: latency,
        }
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return health.Check{
            Status:  health.StatusDegraded,
            Message: fmt.Sprintf("Unexpected status code: %d", resp.StatusCode),
            Latency: latency,
        }
    }
    
    return health.Check{
        Status:  health.StatusHealthy,
        Message: "API responded successfully",
        Latency: latency,
    }
})
```

## Kubernetes Integration

The health package works well with Kubernetes health probes:

```yaml
livenessProbe:
  httpGet:
    path: /health/liveness
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /health/readiness
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

## Best Practices

1. **Separate Probes**: Use liveness for detecting crashes and readiness for service availability
2. **Timeouts**: Add timeouts to all health checks to prevent hanging
3. **Degraded State**: Use degraded status for partial outages or performance issues
4. **Metrics Integration**: Use health check results for alerting and dashboards
5. **Check Isolation**: Ensure checks don't affect each other and can fail independently
