# Quick Usage Guide

## When to Use Each Component

### Rate Counter (`rate.Rate`)
**Use when you need to:**
- Monitor request rates (req/s, req/min, etc.)
- Track events over time with smooth decay
- Implement rate-based alerting
- Measure throughput

```go
// HTTP request rate monitoring
requestRate := rate.NewRate(time.Minute)
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    currentRate := requestRate.Inc()
    log.Printf("Current rate: %.2f req/min", currentRate)
    // ... handle request
})
```

### Token Limiter (`rate.Limiter`)
**Use when you need to:**
- Implement circuit breaker patterns
- Block operations after too many failures
- Prevent cascade failures
- Control access based on accumulated errors

```go
// Circuit breaker for external API
breaker := rate.NewLimiter(5) // Open after 5 failures

err := breaker.Do(func() error {
    return callExternalAPI()
})
if err == rate.ErrLimitExceeded {
    return errors.New("service temporarily unavailable")
}
```

### Error Tracker (`rate.Errors`)
**Use when you need to:**
- Monitor error rates over time
- Track success/failure ratios
- Implement health checks
- Generate monitoring metrics

```go
// Database health monitoring
dbHealth := rate.NewErrors(5 * time.Minute)

func queryDatabase() error {
    err := db.Query("SELECT ...")
    if err != nil {
        dbHealth.Failure().Inc()
        return err
    }
    dbHealth.Success().Inc()
    return nil
}

func isHealthy() bool {
    health := dbHealth.Rate()
    return health.Ratio() < 0.1 // < 10% error rate
}
```

## Configuration Tips

### Rate Counter Periods
- **1 second**: Real-time monitoring, immediate response
- **1 minute**: Operational dashboards, alerting
- **5-15 minutes**: Trend analysis, capacity planning
- **1 hour+**: Long-term monitoring, SLA tracking

### Limiter Token Values
- **Conservative**: `FailureToken=0.5, SuccessToken=1.0` (slow to fail, quick to recover)
- **Balanced**: `FailureToken=1.0, SuccessToken=0.5` (default, moderate response)
- **Aggressive**: `FailureToken=2.0, SuccessToken=1.0` (quick to fail, slow to recover)

### Common Patterns

#### API Gateway Rate Limiting
```go
type APIGateway struct {
    globalRate   *rate.Rate
    errorTracker *rate.Errors
    breakers     map[string]*rate.Limiter
}

func (gw *APIGateway) HandleRequest(service string, req *http.Request) error {
    // Global rate tracking
    gw.globalRate.Inc()
    
    // Per-service circuit breaker
    breaker := gw.getBreaker(service)
    
    return breaker.Do(func() error {
        err := gw.callService(service, req)
        
        // Track for monitoring
        if err != nil {
            gw.errorTracker.Failure().Inc()
        } else {
            gw.errorTracker.Success().Inc()
        }
        
        return err
    })
}
```

#### Database Connection Pool
```go
type DBPool struct {
    connections *rate.Limiter  // Limit concurrent connections
    health      *rate.Errors   // Track query success/failure
    queryRate   *rate.Rate     // Monitor query rate
}

func (pool *DBPool) Query(sql string) (*sql.Rows, error) {
    pool.queryRate.Inc()
    
    return pool.connections.Do(func() error {
        rows, err := pool.db.Query(sql)
        
        if err != nil {
            pool.health.Failure().Inc()
            return err
        }
        
        pool.health.Success().Inc()
        return nil
    })
}
```

#### Microservice Health Check
```go
type ServiceHealth struct {
    services map[string]*rate.Errors
    mu       sync.RWMutex
}

func (sh *ServiceHealth) RecordCall(service string, success bool) {
    sh.mu.Lock()
    if _, exists := sh.services[service]; !exists {
        sh.services[service] = rate.NewErrors(time.Minute)
    }
    tracker := sh.services[service]
    sh.mu.Unlock()
    
    if success {
        tracker.Success().Inc()
    } else {
        tracker.Failure().Inc()
    }
}

func (sh *ServiceHealth) GetStatus() map[string]string {
    sh.mu.RLock()
    defer sh.mu.RUnlock()
    
    status := make(map[string]string)
    for service, tracker := range sh.services {
        health := tracker.Rate()
        if health.Total() < 5 {
            status[service] = "UNKNOWN"
        } else if health.Ratio() < 0.05 {
            status[service] = "HEALTHY"
        } else if health.Ratio() < 0.2 {
            status[service] = "DEGRADED"
        } else {
            status[service] = "UNHEALTHY"
        }
    }
    return status
}
```

## Testing Strategies

### Unit Testing with Mocked Time
```go
func TestRateCounter(t *testing.T) {
    now := time.Now()
    counter := rate.NewRate(time.Second)
    
    // Control time for deterministic tests
    counter.Now = func() time.Time { return now }
    
    assert.Equal(t, 1.0, counter.Inc())
    
    // Advance time by 500ms
    counter.Now = func() time.Time { return now.Add(500 * time.Millisecond) }
    assert.InDelta(t, 1.6, counter.Inc(), 0.1)
}
```

### Integration Testing
```go
func TestCircuitBreakerIntegration(t *testing.T) {
    breaker := rate.NewLimiter(3)
    failures := 0
    
    // Force 5 failures
    for i := 0; i < 5; i++ {
        err := breaker.Do(func() error {
            return errors.New("simulated failure")
        })
        if err != nil {
            failures++
        }
    }
    
    // Circuit should be open now
    assert.False(t, breaker.Allow())
    assert.Equal(t, 3, failures) // Only 3 reached the function
}
```

## Performance Considerations

- Each operation involves mutex locking - consider batching for high-throughput scenarios
- Memory usage is constant regardless of time period or event volume  
- Exponential decay calculations are O(1) and computationally efficient
- For extreme performance, consider per-goroutine instances with periodic aggregation

## Monitoring and Alerting

### Key Metrics to Track
```go
// Rate metrics
currentRPS := requestRate.Count()
currentRPM := requestRate.Per(time.Minute)

// Error metrics  
errorStats := errorTracker.Rate()
errorRate := errorStats.Ratio()
errorCount := errorStats.Failure()

// Circuit breaker metrics
circuitState := breaker.Allow() // true = closed, false = open
successCount := breaker.Success()
failureCount := breaker.Failure()
```

### Alerting Thresholds
- **Error Rate**: > 5% for 5 minutes (warning), > 20% for 2 minutes (critical)
- **Request Rate**: > 1000 req/min (scaling needed), < 10 req/min (potential issue)
- **Circuit Breaker**: Open for > 30 seconds (service degradation)
