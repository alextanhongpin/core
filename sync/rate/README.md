# Rate Limiting Library

A high-performance, thread-safe Go library for rate limiting, circuit breaking, and error rate tracking using exponential decay algorithms.

## Features

- **Rate Counter**: Exponential decay rate tracking for measuring events per time period
- **Token-based Limiter**: Circuit breaker-style rate limiting based on failure accumulation
- **Error Rate Tracker**: Combined success/failure rate monitoring with time-based decay
- **Thread-safe**: All components are designed for concurrent use
- **Testable**: Injectable time functions for deterministic testing

## Installation

```bash
go get github.com/alextanhongpin/core/sync/rate
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"
    "github.com/alextanhongpin/core/sync/rate"
)

func main() {
    // Create a rate counter for requests per second
    requestRate := rate.New() // 1-second window
    
    // Simulate 10 requests
    for i := 0; i < 10; i++ {
        fmt.Printf("Current rate: %.2f req/s\n", requestRate.Inc())
        time.Sleep(100 * time.Millisecond)
    }
}
```

## Components

### 1. Rate Counter

Tracks the rate of events using exponential decay smoothing.

#### Basic Usage

```go
// Create rate counters for different time windows
rps := rate.NewRate(time.Second)        // Requests per second
rpm := rate.NewRate(time.Minute)       // Requests per minute
rph := rate.NewRate(time.Hour)         // Requests per hour

// Track events
rps.Inc()                              // Increment by 1
rps.Add(5.5)                          // Add 5.5 to the count

// Get current rates
fmt.Printf("RPS: %.2f\n", rps.Count())
fmt.Printf("RPM: %.2f\n", rps.Per(time.Minute))  // Scale to different time unit
```

#### Real-world Example: HTTP Request Monitoring

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "time"
    "github.com/alextanhongpin/core/sync/rate"
)

type RequestMonitor struct {
    requests *rate.Rate
    errors   *rate.Rate
}

func NewRequestMonitor() *RequestMonitor {
    return &RequestMonitor{
        requests: rate.NewRate(time.Minute), // Track requests per minute
        errors:   rate.NewRate(time.Minute), // Track errors per minute
    }
}

func (rm *RequestMonitor) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Wrap ResponseWriter to capture status code
        wrapper := &responseWrapper{ResponseWriter: w, statusCode: 200}
        
        next.ServeHTTP(wrapper, r)
        
        // Track the request
        rm.requests.Inc()
        
        // Track errors (4xx and 5xx responses)
        if wrapper.statusCode >= 400 {
            rm.errors.Inc()
        }
        
        duration := time.Since(start)
        log.Printf("Request: %s %s - Status: %d - Duration: %v", 
            r.Method, r.URL.Path, wrapper.statusCode, duration)
    })
}

func (rm *RequestMonitor) Stats() (requestsPerMin, errorsPerMin, errorRate float64) {
    requests := rm.requests.Count()
    errors := rm.errors.Count()
    
    errorRate = 0
    if requests > 0 {
        errorRate = errors / requests
    }
    
    return requests, errors, errorRate
}

type responseWrapper struct {
    http.ResponseWriter
    statusCode int
}

func (w *responseWrapper) WriteHeader(statusCode int) {
    w.statusCode = statusCode
    w.ResponseWriter.WriteHeader(statusCode)
}

func main() {
    monitor := NewRequestMonitor()
    
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello, World!")
    })
    
    mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
        http.Error(w, "Something went wrong", http.StatusInternalServerError)
    })
    
    mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
        requests, errors, errorRate := monitor.Stats()
        fmt.Fprintf(w, "Requests/min: %.2f\nErrors/min: %.2f\nError rate: %.2f%%\n", 
            requests, errors, errorRate*100)
    })
    
    handler := monitor.Handler(mux)
    
    fmt.Println("Server running on :8080")
    log.Fatal(http.ListenAndServe(":8080", handler))
}
```

### 2. Token-based Limiter

Implements circuit breaker behavior by accumulating failure tokens and blocking operations when limits are exceeded.

#### Basic Usage

```go
// Create a limiter that blocks after accumulating 10 failure tokens
limiter := rate.NewLimiter(10)

// Configure token values (optional)
limiter.FailureToken = 1.0  // Add 1 token per failure
limiter.SuccessToken = 0.5  // Remove 0.5 tokens per success

// Check if operation is allowed
if limiter.Allow() {
    // Perform operation
    err := doSomething()
    if err != nil {
        limiter.Err()  // Record failure
    } else {
        limiter.Ok()   // Record success
    }
}

// Or use the convenience method
err := limiter.Do(func() error {
    return doSomething()  // Automatically tracks success/failure
})
```

#### Real-world Example: External API Circuit Breaker

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
    "github.com/alextanhongpin/core/sync/rate"
)

type APIClient struct {
    baseURL    string
    client     *http.Client
    limiter    *rate.Limiter
}

func NewAPIClient(baseURL string) *APIClient {
    return &APIClient{
        baseURL: baseURL,
        client:  &http.Client{Timeout: 10 * time.Second},
        limiter: rate.NewLimiter(5), // Block after 5 failures
    }
}

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func (c *APIClient) GetUser(ctx context.Context, userID int) (*User, error) {
    return c.makeRequest(ctx, fmt.Sprintf("/users/%d", userID))
}

func (c *APIClient) makeRequest(ctx context.Context, endpoint string) (*User, error) {
    // Use the limiter to prevent requests when circuit is open
    err := c.limiter.Do(func() error {
        return c.doRequest(ctx, endpoint)
    })
    
    if err == rate.ErrLimitExceeded {
        return nil, fmt.Errorf("circuit breaker open: too many recent failures")
    }
    
    return c.lastUser, err
}

var lastUser *User // Simplified for example

func (c *APIClient) doRequest(ctx context.Context, endpoint string) error {
    url := c.baseURL + endpoint
    
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return fmt.Errorf("creating request: %w", err)
    }
    
    resp, err := c.client.Do(req)
    if err != nil {
        return fmt.Errorf("making request: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 500 {
        // Server errors should trigger circuit breaker
        return fmt.Errorf("server error: %d", resp.StatusCode)
    }
    
    if resp.StatusCode >= 400 {
        // Client errors don't trigger circuit breaker
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("client error %d: %s", resp.StatusCode, body)
    }
    
    var user User
    if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
        return fmt.Errorf("decoding response: %w", err)
    }
    
    lastUser = &user
    return nil
}

func (c *APIClient) Stats() (success, failure, total int) {
    return c.limiter.Success(), c.limiter.Failure(), c.limiter.Total()
}

func main() {
    client := NewAPIClient("https://jsonplaceholder.typicode.com")
    
    // Simulate multiple requests
    for i := 1; i <= 20; i++ {
        user, err := client.GetUser(context.Background(), i)
        if err != nil {
            fmt.Printf("Request %d failed: %v\n", i, err)
        } else {
            fmt.Printf("Request %d success: User %s\n", i, user.Name)
        }
        
        success, failure, total := client.Stats()
        fmt.Printf("Stats - Success: %d, Failure: %d, Total: %d\n\n", 
            success, failure, total)
        
        time.Sleep(100 * time.Millisecond)
    }
}
```

### 3. Error Rate Tracker

Combines success and failure tracking with time-based decay for monitoring error rates.

#### Basic Usage

```go
// Create an error tracker with 5-minute window
tracker := rate.NewErrors(5 * time.Minute)

// Record events
tracker.Success().Inc()     // Record success
tracker.Failure().Add(2)    // Record 2 failures

// Get current error rate
errorRate := tracker.Rate()
fmt.Printf("Success rate: %.2f/min\n", errorRate.Success())
fmt.Printf("Failure rate: %.2f/min\n", errorRate.Failure())
fmt.Printf("Error ratio: %.2f%%\n", errorRate.Ratio()*100)
```

#### Real-world Example: Database Connection Pool Monitor

```go
package main

import (
    "database/sql"
    "fmt"
    "log"
    "time"
    _ "github.com/lib/pq"
    "github.com/alextanhongpin/core/sync/rate"
)

type DBMonitor struct {
    db      *sql.DB
    errors  *rate.Errors
}

func NewDBMonitor(db *sql.DB) *DBMonitor {
    return &DBMonitor{
        db:     db,
        errors: rate.NewErrors(time.Minute), // Track errors per minute
    }
}

func (m *DBMonitor) Query(query string, args ...interface{}) (*sql.Rows, error) {
    start := time.Now()
    rows, err := m.db.Query(query, args...)
    duration := time.Since(start)
    
    if err != nil {
        m.errors.Failure().Inc()
        log.Printf("Query failed (%v): %s", duration, query)
        return nil, err
    }
    
    m.errors.Success().Inc()
    log.Printf("Query succeeded (%v): %s", duration, query)
    return rows, nil
}

func (m *DBMonitor) Exec(query string, args ...interface{}) (sql.Result, error) {
    start := time.Now()
    result, err := m.db.Exec(query, args...)
    duration := time.Since(start)
    
    if err != nil {
        m.errors.Failure().Inc()
        log.Printf("Exec failed (%v): %s", duration, query)
        return nil, err
    }
    
    m.errors.Success().Inc()
    log.Printf("Exec succeeded (%v): %s", duration, query)
    return result, nil
}

func (m *DBMonitor) HealthCheck() error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    err := m.db.PingContext(ctx)
    if err != nil {
        m.errors.Failure().Inc()
        return fmt.Errorf("database health check failed: %w", err)
    }
    
    m.errors.Success().Inc()
    return nil
}

func (m *DBMonitor) ErrorRate() *rate.ErrorRate {
    return m.errors.Rate()
}

func (m *DBMonitor) IsHealthy() bool {
    errorRate := m.errors.Rate()
    
    // Consider unhealthy if error rate > 10% and we have enough data
    if errorRate.Total() > 10 && errorRate.Ratio() > 0.1 {
        return false
    }
    
    return true
}

func main() {
    // In a real application, use proper connection string
    db, err := sql.Open("postgres", "postgres://user:pass@localhost/dbname?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    monitor := NewDBMonitor(db)
    
    // Simulate database operations
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            // Perform health check
            if err := monitor.HealthCheck(); err != nil {
                log.Printf("Health check failed: %v", err)
            }
            
            // Get and log error rate
            errorRate := monitor.ErrorRate()
            fmt.Printf("DB Stats - Success: %.2f/min, Failure: %.2f/min, Error Rate: %.2f%%, Healthy: %v\n",
                errorRate.Success(), errorRate.Failure(), errorRate.Ratio()*100, monitor.IsHealthy())
        }
    }
}
```

## Advanced Usage

### Custom Token Configuration

```go
// Create a more aggressive circuit breaker
limiter := rate.NewLimiter(5)
limiter.FailureToken = 2.0  // Failures count double
limiter.SuccessToken = 1.0  // Successes remove one token

// Create a more forgiving circuit breaker
limiter2 := rate.NewLimiter(20)
limiter2.FailureToken = 0.5  // Failures count as half
limiter2.SuccessToken = 1.0  // Successes remove one token
```

### Combining Rate Limiting Strategies

```go
type SmartLimiter struct {
    rateLimiter   *rate.Limiter  // Token-based limiting
    errorTracker  *rate.Errors   // Error rate tracking
    lastReset     time.Time
}

func NewSmartLimiter() *SmartLimiter {
    return &SmartLimiter{
        rateLimiter:  rate.NewLimiter(10),
        errorTracker: rate.NewErrors(time.Minute),
        lastReset:    time.Now(),
    }
}

func (sl *SmartLimiter) Allow() bool {
    // Check token-based limit
    if !sl.rateLimiter.Allow() {
        return false
    }
    
    // Check error rate (block if > 50% errors in last minute)
    errorRate := sl.errorTracker.Rate()
    if errorRate.Total() > 10 && errorRate.Ratio() > 0.5 {
        return false
    }
    
    return true
}

func (sl *SmartLimiter) RecordSuccess() {
    sl.rateLimiter.Ok()
    sl.errorTracker.Success().Inc()
}

func (sl *SmartLimiter) RecordFailure() {
    sl.rateLimiter.Err()
    sl.errorTracker.Failure().Inc()
}
```

## Testing

The library provides injectable time functions for deterministic testing:

```go
func TestRateCounter(t *testing.T) {
    now := time.Now()
    counter := rate.NewRate(time.Second)
    
    // Inject custom time function
    counter.Now = func() time.Time { return now }
    
    // Test at time 0
    assert.Equal(t, 1.0, counter.Inc())
    
    // Test at time +500ms
    counter.Now = func() time.Time { return now.Add(500 * time.Millisecond) }
    assert.InDelta(t, 1.6, counter.Inc(), 0.1)
}
```

## Performance Considerations

- All operations are thread-safe but involve mutex locking
- Rate counters use exponential decay which is computationally efficient
- Memory usage is constant regardless of time period or event volume
- For high-throughput scenarios, consider batching operations or using separate instances per goroutine

## Error Handling

The library uses panics for configuration errors (invalid periods/limits) but returns errors for operational issues:

```go
// This will panic - invalid configuration
// limiter := rate.NewLimiter(0)

// This returns an error - operational issue
err := limiter.Do(func() error {
    return someOperation()
})
if err == rate.ErrLimitExceeded {
    // Handle rate limit exceeded
}
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `go test -v`
5. Submit a pull request

## License

MIT License - see LICENSE file for details.
