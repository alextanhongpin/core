# Metrics Package

A comprehensive Go package for application metrics collection using Prometheus and probabilistic data structures. This package provides HTTP middleware for collecting request metrics, RED (Rate, Error, Duration) metrics, and advanced statistical tracking.

## Features

- **Prometheus Integration**: Standard metrics collection with gauges and histograms
- **RED Metrics**: Rate, Error, Duration tracking for service observability
- **Probabilistic Data Structures**: HyperLogLog, Count-Min Sketch, T-Digest, TopK for memory-efficient analytics
- **HTTP Middleware**: Easy integration with HTTP servers
- **Real-time Analytics**: Live tracking of request patterns and performance

## Quick Start

### Basic Prometheus Metrics

```go
package main

import (
    "net/http"
    "github.com/alextanhongpin/core/metrics"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
    // Register metrics
    reg := prometheus.NewRegistry()
    reg.MustRegister(metrics.InFlightGauge, metrics.RequestDuration, metrics.ResponseSize)
    
    // Create server with metrics middleware
    mux := http.NewServeMux()
    
    // Add metrics endpoint
    mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
    
    // Add your handlers with metrics
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello World"))
    })
    
    instrumentedHandler := metrics.RequestDurationHandler("v1.0", handler)
    mux.Handle("/api/hello", instrumentedHandler)
    
    http.ListenAndServe(":8080", mux)
}
```

### RED Metrics Usage

```go
func processOrder(orderID string) error {
    red := metrics.NewRED("order_service", "process_order")
    defer red.Done()
    
    // Your business logic here
    if err := validateOrder(orderID); err != nil {
        red.Fail()
        return err
    }
    
    return nil
}
```

### Advanced Analytics with Tracker

```go
import (
    "github.com/alextanhongpin/core/metrics"
    "github.com/redis/go-redis/v9"
)

func setupAdvancedMetrics() {
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    tracker := metrics.NewTracker("api_analytics", client)
    
    userExtractor := func(r *http.Request) string {
        return r.Header.Get("X-User-ID")
    }
    
    logger := slog.Default()
    
    handler := metrics.TrackerHandler(
        yourHandler,
        tracker,
        userExtractor,
        logger,
    )
    
    // Get analytics
    mux.Handle("/admin/stats", metrics.TrackerStatsHandler(tracker))
}
```

## Metrics Types

### 1. InFlightGauge
Tracks the number of requests currently being processed.

```go
// Increment when request starts
metrics.InFlightGauge.Inc()

// Decrement when request completes
defer metrics.InFlightGauge.Dec()
```

### 2. RequestDuration
Histogram tracking request duration with method, path, status, and version labels.

```go
handler := metrics.RequestDurationHandler("v1.2.3", yourHandler)
```

### 3. ResponseSize
Tracks the size of HTTP responses.

```go
size := metrics.ObserveResponseSize(request)
```

### 4. RED Metrics
Comprehensive Rate, Error, Duration tracking for services.

```go
red := metrics.NewRED("user_service", "authenticate")
defer red.Done()

if err := authenticate(user); err != nil {
    red.Fail() // or red.SetStatus("auth_failed")
    return err
}
```

## Real-World Examples

### E-commerce API

```go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "net/http"
    "time"
    
    "github.com/alextanhongpin/core/metrics"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/redis/go-redis/v9"
)

type EcommerceAPI struct {
    tracker *metrics.Tracker
    logger  *slog.Logger
}

func NewEcommerceAPI() *EcommerceAPI {
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    return &EcommerceAPI{
        tracker: metrics.NewTracker("ecommerce_api", client),
        logger:  slog.Default(),
    }
}

func (api *EcommerceAPI) SetupRoutes() *http.ServeMux {
    mux := http.NewServeMux()
    
    // Metrics endpoint
    reg := prometheus.NewRegistry()
    reg.MustRegister(
        metrics.InFlightGauge,
        metrics.RequestDuration,
        metrics.ResponseSize,
        metrics.RED,
    )
    mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
    
    // Analytics endpoint
    mux.Handle("/admin/analytics", metrics.TrackerStatsHandler(api.tracker))
    
    // Business endpoints with full instrumentation
    mux.Handle("/api/products", api.instrumentHandler("list_products", api.listProducts))
    mux.Handle("/api/orders", api.instrumentHandler("create_order", api.createOrder))
    mux.Handle("/api/users/login", api.instrumentHandler("user_login", api.userLogin))
    
    return mux
}

func (api *EcommerceAPI) instrumentHandler(action string, handler http.HandlerFunc) http.Handler {
    // Add RED metrics
    wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        red := metrics.NewRED("ecommerce_api", action)
        defer red.Done()
        
        // Track in-flight requests
        metrics.InFlightGauge.Inc()
        defer metrics.InFlightGauge.Dec()
        
        handler(w, r)
        
        // Check for errors based on status code
        if w.Header().Get("X-Error") != "" {
            red.Fail()
        }
    })
    
    // Add request duration tracking
    durationHandler := metrics.RequestDurationHandler("v2.1", wrappedHandler)
    
    // Add advanced analytics
    userExtractor := func(r *http.Request) string {
        if userID := r.Header.Get("X-User-ID"); userID != "" {
            return userID
        }
        return r.RemoteAddr // fallback to IP
    }
    
    return metrics.TrackerHandler(durationHandler, api.tracker, userExtractor, api.logger)
}

func (api *EcommerceAPI) listProducts(w http.ResponseWriter, r *http.Request) {
    // Simulate product listing
    time.Sleep(50 * time.Millisecond)
    
    products := `[
        {"id": 1, "name": "Widget A", "price": 19.99},
        {"id": 2, "name": "Widget B", "price": 29.99}
    ]`
    
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(products))
    
    // Track response size
    metrics.ObserveResponseSize(r)
}

func (api *EcommerceAPI) createOrder(w http.ResponseWriter, r *http.Request) {
    // Simulate order processing
    time.Sleep(200 * time.Millisecond)
    
    // Simulate occasional errors
    if time.Now().UnixNano()%10 == 0 {
        w.Header().Set("X-Error", "inventory_unavailable")
        http.Error(w, "Product unavailable", http.StatusConflict)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"order_id": "ORD-12345", "status": "created"}`))
    metrics.ObserveResponseSize(r)
}

func (api *EcommerceAPI) userLogin(w http.ResponseWriter, r *http.Request) {
    // Simulate authentication
    time.Sleep(100 * time.Millisecond)
    
    // Simulate authentication failures
    if time.Now().UnixNano()%5 == 0 {
        w.Header().Set("X-Error", "auth_failed")
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"token": "jwt-token-here", "user_id": "user-123"}`))
    metrics.ObserveResponseSize(r)
}

func main() {
    api := NewEcommerceAPI()
    mux := api.SetupRoutes()
    
    fmt.Println("E-commerce API running on :8080")
    fmt.Println("Metrics: http://localhost:8080/metrics")
    fmt.Println("Analytics: http://localhost:8080/admin/analytics")
    
    http.ListenAndServe(":8080", mux)
}
```

### Microservice Example

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    
    "github.com/alextanhongpin/core/metrics"
)

type UserService struct {
    db *sql.DB
}

func (s *UserService) CreateUser(ctx context.Context, email string) error {
    red := metrics.NewRED("user_service", "create_user")
    defer red.Done()
    
    // Validate email
    if err := s.validateEmail(email); err != nil {
        red.SetStatus("validation_failed")
        return fmt.Errorf("validation failed: %w", err)
    }
    
    // Check if user exists
    exists, err := s.userExists(ctx, email)
    if err != nil {
        red.Fail()
        return fmt.Errorf("database error: %w", err)
    }
    
    if exists {
        red.SetStatus("user_exists")
        return fmt.Errorf("user already exists")
    }
    
    // Create user
    if err := s.insertUser(ctx, email); err != nil {
        red.Fail()
        return fmt.Errorf("failed to create user: %w", err)
    }
    
    return nil
}

func (s *UserService) validateEmail(email string) error {
    // Email validation logic
    time.Sleep(10 * time.Millisecond)
    return nil
}

func (s *UserService) userExists(ctx context.Context, email string) (bool, error) {
    // Database check
    time.Sleep(50 * time.Millisecond)
    return false, nil
}

func (s *UserService) insertUser(ctx context.Context, email string) error {
    // Database insert
    time.Sleep(100 * time.Millisecond)
    return nil
}
```

## Configuration

### Custom Histogram Buckets

```go
import "github.com/prometheus/client_golang/prometheus"

// For API latencies (milliseconds to seconds)
apiLatencyBuckets := []float64{
    0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

// For file sizes (bytes to megabytes)
fileSizeBuckets := []float64{
    1024, 4096, 16384, 65536, 262144, 1048576, 4194304, 16777216,
}

// Custom histogram
customDuration := prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "custom_duration_seconds",
        Help:    "Custom operation duration",
        Buckets: apiLatencyBuckets,
    },
    []string{"operation", "status"},
)
```

## Best Practices

### 1. Resource Management

```go
func (tracker *Tracker) WithTimeout(timeout time.Duration) *Tracker {
    // Create tracker with context timeout
    // Implement proper cleanup
}
```

### 2. Error Handling

```go
red := metrics.NewRED("service", "action")
defer func() {
    if r := recover(); r != nil {
        red.SetStatus("panic")
        red.Done()
        panic(r) // re-panic
    }
}()
```

### 3. Label Cardinality

```go
// Good: Limited cardinality
RequestDuration.WithLabelValues("GET", "/api/users", "200", "v1.0")

// Bad: High cardinality (user IDs)
// RequestDuration.WithLabelValues("GET", "/api/users/12345", "200", "v1.0")
```

## Monitoring and Alerting

### Prometheus Queries

```promql
# Error rate
rate(red_count{status="err"}[5m]) / rate(red_count[5m])

# 95th percentile latency
histogram_quantile(0.95, rate(request_duration_seconds_bucket[5m]))

# Requests per second
rate(request_duration_seconds_count[5m])

# Top endpoints by volume
topk(10, sum(rate(request_duration_seconds_count[5m])) by (path))
```

### Grafana Dashboard Example

```json
{
  "dashboard": {
    "title": "Application Metrics",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [
          {
            "expr": "rate(request_duration_seconds_count[5m])"
          }
        ]
      },
      {
        "title": "Error Rate",
        "targets": [
          {
            "expr": "rate(red_count{status=\"err\"}[5m]) / rate(red_count[5m])"
          }
        ]
      }
    ]
  }
}
```

## Performance Considerations

- **Memory Usage**: Probabilistic data structures use constant memory
- **Redis Connection**: Pool connections for high-throughput applications
- **Metrics Cardinality**: Keep label combinations under 10,000 per metric
- **Sampling**: Consider sampling for very high-traffic applications

## Troubleshooting

### Common Issues

1. **High Memory Usage**: Check metric cardinality
2. **Slow Performance**: Review Redis connection settings
3. **Missing Data**: Verify metric registration
4. **Test Flakiness**: Use deterministic time in tests

### Debug Mode

```go
// Enable debug logging
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
```

## Best Practices and Edge Cases

### Thread Safety

All metrics operations are thread-safe and can be used concurrently:

```go
// Safe for concurrent use
func concurrentHandler(w http.ResponseWriter, r *http.Request) {
    red := metrics.NewRED("service", "operation")
    defer red.Done()
    
    // Multiple goroutines can safely use metrics
    go func() {
        metrics.InFlightGauge.Inc()
        defer metrics.InFlightGauge.Dec()
        // ... work
    }()
}
```

### Error Handling

Always handle edge cases gracefully:

```go
func robustHandler(w http.ResponseWriter, r *http.Request) {
    red := metrics.NewRED("service", "operation")
    defer func() {
        // Always call Done() even if panic occurs
        if r := recover(); r != nil {
            red.SetStatus("panic")
            red.Done()
            panic(r) // re-panic after recording
        } else {
            red.Done()
        }
    }()
    
    // Handle nil requests safely
    if r == nil {
        red.SetStatus("nil_request")
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }
    
    // Your logic here
}
```

### Resource Management

#### Proper Cleanup

```go
func serviceLevelMetrics() {
    // Use isolated registries for tests
    registry := prometheus.NewRegistry()
    
    duration := prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "service_duration",
            Help: "Service operation duration",
        },
        []string{"operation"},
    )
    
    registry.MustRegister(duration)
    
    // Always unregister when done (especially in tests)
    defer registry.Unregister(duration)
}
```

#### Memory Leak Prevention

```go
func preventLeaks() {
    // Limit label cardinality
    red := metrics.NewRED("service", "operation")
    
    // Avoid high-cardinality labels
    // BAD: red.SetStatus(fmt.Sprintf("user_%d", userID))
    // GOOD: red.SetStatus("user_operation")
    
    red.Done()
}
```

### Configuration Best Practices

#### Prometheus Registry Setup

```go
func setupPrometheusRegistry() *prometheus.Registry {
    registry := prometheus.NewRegistry()
    
    // Register only necessary metrics
    registry.MustRegister(
        metrics.InFlightGauge,
        metrics.RequestDuration,
        metrics.ResponseSize,
        metrics.RED,
        // Add collectors for runtime metrics
        prometheus.NewGoCollector(),
        prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
    )
    
    return registry
}
```

#### Redis Configuration for High Throughput

```go
func setupRedisForMetrics() *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:         "localhost:6379",
        DB:           1, // Use dedicated DB for metrics
        PoolSize:     20, // Increase for high concurrency
        MinIdleConns: 5,
        MaxRetries:   3,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
        PoolTimeout:  4 * time.Second,
    })
}
```

### Testing Best Practices

#### Isolated Test Metrics

```go
func TestMetricsIsolated(t *testing.T) {
    // Create isolated registry for tests
    registry := prometheus.NewRegistry()
    
    // Create test-specific metrics
    testDuration := prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "test_duration",
            Help: "Test duration metric",
        },
        []string{"test_case"},
    )
    
    registry.MustRegister(testDuration)
    defer registry.Unregister(testDuration)
    
    // Test logic here
    testDuration.WithLabelValues("test_case_1").Observe(1.0)
    
    // Verify metrics
    count := testutil.CollectAndCount(testDuration)
    assert.Equal(t, 1, count)
}
```

#### Deterministic Tests

```go
func TestWithDeterministicTime(t *testing.T) {
    // Use fixed time for reproducible tests
    fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
    
    red := &metrics.REDTracker{
        Service: "test_service",
        Action:  "test_action",
        Status:  "ok",
        Now:     fixedTime,
    }
    
    // Simulate passage of time
    time.Sleep(100 * time.Millisecond)
    
    // Test completion with known duration
    red.Done()
}
```

### Monitoring and Alerting

#### Key Metrics to Monitor

```promql
# Error Rate (should be < 1%)
rate(red_count{status="err"}[5m]) / rate(red_count[5m]) * 100

# 95th Percentile Latency (should be < 200ms)
histogram_quantile(0.95, rate(request_duration_seconds_bucket[5m]))

# Request Rate
rate(request_duration_seconds_count[5m])

# In-Flight Requests (monitor for spikes)
in_flight_requests
```

#### Sample Grafana Alerts

```yaml
groups:
  - name: application.rules
    rules:
      - alert: HighErrorRate
        expr: rate(red_count{status="err"}[5m]) / rate(red_count[5m]) > 0.01
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          
      - alert: HighLatency
        expr: histogram_quantile(0.95, rate(request_duration_seconds_bucket[5m])) > 0.2
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High latency detected"
```

### Common Pitfalls

1. **High Cardinality Labels**: Avoid user IDs, timestamps in labels
2. **Missing Defer**: Always use `defer red.Done()` to ensure metrics are recorded
3. **Global State in Tests**: Use isolated registries for test isolation
4. **Resource Leaks**: Unregister metrics in tests and clean up Redis connections
5. **Blocking Operations**: Don't perform blocking operations in metric collection
6. **Label Consistency**: Use consistent label names across all metrics
