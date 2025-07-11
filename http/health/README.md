# HTTP Health Package

The HTTP Health package provides comprehensive health check endpoints for monitoring application status, suitable for load balancers, Kubernetes probes, and observability systems.

## Features

- **Standard Health Endpoints**: Ready-to-use health check HTTP handlers
- **Customizable Checks**: Add custom health checks for databases, services, etc.
- **Status Classification**: Healthy, unhealthy, and degraded states
- **Check Timing**: Latency measurements for each health check
- **Uptime Tracking**: Application uptime reporting
- **Version Information**: Version details included in responses
- **Cross-Package Integration**: Works with `server`, `chain`, and logging middleware

## Quick Start

```go
package main

import (
    "net/http"
    "database/sql"
    "time"
    
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

### Health Handler

#### `New(version string) *Handler`

Creates a new health handler with version info.

#### `AddCheck(name string, fn func() Check)`

Adds a custom health check.

#### `Health(w http.ResponseWriter, r *http.Request)`

Main health endpoint.

#### `Liveness(w http.ResponseWriter, r *http.Request)`

Liveness probe endpoint.

#### `Readiness(w http.ResponseWriter, r *http.Request)`

Readiness probe endpoint.

## Best Practices

- Add checks for all critical dependencies (DB, cache, external APIs).
- Use version info for deployment tracking.
- Integrate with orchestration systems for automated health monitoring.

## Related Packages

- [`server`](../server/README.md): HTTP server utilities
- [`chain`](../chain/README.md): Middleware chaining
- [`handler`](../handler/README.md): Base handler utilities

## License

MIT
