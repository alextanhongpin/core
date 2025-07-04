# Sync - Concurrent Programming Utilities

[![Go Reference](https://pkg.go.dev/badge/github.com/alextanhongpin/core/sync.svg)](https://pkg.go.dev/github.com/alextanhongpin/core/sync)
[![Go Report Card](https://goreportcard.com/badge/github.com/alextanhongpin/core/sync)](https://goreportcard.com/report/github.com/alextanhongpin/core/sync)

A comprehensive collection of Go packages for concurrent programming, synchronization, and coordination between goroutines. This workspace provides battle-tested utilities for building robust, scalable concurrent applications.

## 🚀 Overview

The `sync` workspace contains production-ready utilities for:

- **🔄 Background Processing**: Worker pools and background task management
- **📦 Batch Operations**: Batching, queuing, and data loading utilities
- **🔌 Circuit Breakers**: Fault tolerance and failure handling
- **📊 Data Loading**: Efficient data loading with caching and deduplication
- **⏱️ Debouncing**: Rate limiting and event throttling
- **🔒 Locking**: Distributed and local locking mechanisms
- **🔄 Pipelines**: Stream processing and data transformation pipelines
- **🎯 Polling**: Periodic task execution and monitoring
- **🤝 Promises**: Async programming patterns and futures
- **📈 Rate Limiting**: Traffic control and API rate limiting
- **🔄 Retry Logic**: Intelligent retry mechanisms with backoff
- **🔀 SingleFlight**: Request deduplication and coalescing
- **📸 Snapshots**: State capturing and versioning
- **⏳ Throttling**: Adaptive load control
- **⏰ Timers**: Advanced timing utilities

## 📦 Packages

### 🔄 [Background](./background/) - Worker Pool Management
Worker pools for concurrent background task processing.

```go
import "github.com/alextanhongpin/core/sync/background"

worker, stop := background.New(ctx, 4, func(ctx context.Context, task Task) {
    // Process task
    log.Printf("Processing task: %v", task)
})
defer stop()

// Send tasks to worker pool
worker.Send(Task{ID: "task1", Data: "example"})
```

**Key Features**: Worker pools, graceful shutdown, context support, configurable concurrency  
**Use Cases**: Background job processing, async task execution, worker pools

### 📦 [Batch](./batch/) - Batch Processing
Efficient batching and queuing for bulk operations.

```go
import "github.com/alextanhongpin/core/sync/batch"

// Create a batch processor
processor := batch.NewProcessor(100, 5*time.Second, func(items []Item) error {
    // Process batch of items
    return database.BulkInsert(items)
})

// Add items to batch
processor.Add(item1)
processor.Add(item2)
```

**Key Features**: Automatic batching, timeout-based flushing, error handling, backpressure  
**Use Cases**: Database bulk operations, API batching, message queuing

### 🔌 [Circuit Breaker](./circuitbreaker/) - Fault Tolerance
Prevent cascading failures with circuit breaker pattern.

```go
import "github.com/alextanhongpin/core/sync/circuitbreaker"

cb := circuitbreaker.New()
err := cb.Do(func() error {
    // Call to external service
    return httpClient.Get(url)
})
```

**Key Features**: Open/closed/half-open states, failure thresholds, slow call detection, observability hooks  
**Use Cases**: Service resilience, API fault tolerance, dependency failure handling

### 📊 [DataLoader](./dataloader/) - Efficient Data Loading
Batch loading with caching to solve N+1 query problems.

```go
import "github.com/alextanhongpin/core/sync/dataloader"

loader := dataloader.New(ctx, &dataloader.Options[int, User]{
    BatchFn: func(ctx context.Context, userIDs []int) (map[int]User, error) {
        return database.LoadUsers(ctx, userIDs)
    },
})

user, err := loader.Load(userID)
```

**Key Features**: Automatic batching, caching, error handling, generic types  
**Use Cases**: GraphQL resolvers, database optimization, API efficiency

### ⏱️ [Debounce](./debounce/) - Event Throttling
Prevent excessive calls with debouncing mechanisms.

```go
import "github.com/alextanhongpin/core/sync/debounce"

debouncer := debounce.New(500*time.Millisecond, func() {
    // Handle search request
    performSearch(query)
})

// Rapid calls will be debounced
debouncer.Call()
```

**Key Features**: Time-based debouncing, count-based debouncing, flexible triggers  
**Use Cases**: Search input handling, API rate limiting, event processing

### 🔒 [Lock](./lock/) - Synchronization Primitives
Advanced locking mechanisms for coordination.

```go
import "github.com/alextanhongpin/core/sync/lock"

// Value-based locking
lock := lock.NewValue()
if acquired := lock.TryLock("resource-key"); acquired {
    defer lock.Unlock("resource-key")
    // Process resource
}
```

**Key Features**: Value-based locking, try-lock patterns, deadlock prevention  
**Use Cases**: Resource synchronization, distributed locking, critical sections

### 🔄 [Pipeline](./pipeline/) - Stream Processing
Build data processing pipelines with parallel stages.

```go
import "github.com/alextanhongpin/core/sync/pipeline"

// Create processing pipeline
numbers := pipeline.Generator(ctx, 100)
strings := pipeline.Map(numbers, strconv.Itoa)
processed := pipeline.Pool(5, strings, processString)

for result := range processed {
    log.Printf("Result: %v", result)
}
```

**Key Features**: Parallel processing, rate limiting, monitoring, transformations  
**Use Cases**: Data ETL, stream processing, parallel computations

### 🎯 [Poll](./poll/) - Periodic Task Execution
Robust polling with backoff and failure handling.

```go
import "github.com/alextanhongpin/core/sync/poll"

poller := poll.New()
events, stop := poller.Poll(func(ctx context.Context) error {
    // Polling logic
    return processQueue(ctx)
})
defer stop()
```

**Key Features**: Configurable backoff, failure thresholds, concurrent workers  
**Use Cases**: Queue processing, health checks, periodic tasks

### 🤝 [Promise](./promise/) - Async Programming
JavaScript-style promises for Go.

```go
import "github.com/alextanhongpin/core/sync/promise"

p := promise.New(func() (string, error) {
    // Async operation
    return fetchData()
})

result, err := p.Await()
```

**Key Features**: Async execution, deferred resolution, generic types  
**Use Cases**: Async operations, concurrent programming, futures pattern

### 📈 [Rate](./rate/) - Rate Limiting
Flexible rate limiting with multiple algorithms.

```go
import "github.com/alextanhongpin/core/sync/rate"

limiter := rate.NewLimiter(100, time.Second) // 100 requests per second
if limiter.Allow() {
    // Process request
}
```

**Key Features**: Token bucket, sliding window, distributed rate limiting  
**Use Cases**: API rate limiting, traffic control, resource protection

### 📈 [RateLimit](./ratelimit/) - Advanced Rate Limiting
Multiple rate limiting algorithms and strategies.

```go
import "github.com/alextanhongpin/core/sync/ratelimit"

// Fixed window rate limiter
limiter := ratelimit.NewFixedWindow(100, time.Minute)
if limiter.Allow("user:123") {
    // Process request
}
```

**Key Features**: Multiple algorithms, per-key limiting, time window management  
**Use Cases**: User-based limiting, API quotas, abuse prevention

### 🔄 [Retry](./retry/) - Intelligent Retry Logic
Configurable retry mechanisms with backoff strategies.

```go
import "github.com/alextanhongpin/core/sync/retry"

err := retry.Do(func() error {
    return httpClient.Get(url)
}, retry.WithMaxRetries(3), retry.WithBackoff(retry.ExponentialBackoff))
```

**Key Features**: Multiple backoff strategies, jitter, conditional retries  
**Use Cases**: Network resilience, API reliability, fault tolerance

### 🔀 [SingleFlight](./singleflight/) - Request Deduplication
Prevent duplicate work with request coalescing.

```go
import "github.com/alextanhongpin/core/sync/singleflight"

group := singleflight.New[string]()
result, shared, err := group.Do(ctx, "cache-key", func(ctx context.Context) (string, error) {
    // Expensive operation
    return fetchFromDatabase(ctx)
})
```

**Key Features**: Request deduplication, result sharing, generic types  
**Use Cases**: Cache stampede prevention, database optimization, API efficiency

### 📸 [Snapshot](./snapshot/) - State Persistence
Redis-style snapshot mechanism for data persistence.

```go
import "github.com/alextanhongpin/core/sync/snapshot"

snap, stop := snapshot.New(ctx, func(ctx context.Context, evt snapshot.Event) {
    // Save snapshot
    saveToFile(data)
})
defer stop()

// Notify of changes
snap.Touch()
```

**Key Features**: Frequency-based snapshots, time-based triggers, background processing  
**Use Cases**: Data persistence, backup systems, checkpoint mechanisms

### ⏳ [Throttle](./throttle/) - Adaptive Load Control
Control concurrent operations with adaptive throttling.

```go
import "github.com/alextanhongpin/core/sync/throttle"

throttler := throttle.New(&throttle.Options{
    Limit: 10,
    BacklogLimit: 50,
    BacklogTimeout: 30 * time.Second,
})

err := throttler.Do(ctx, func() error {
    // Throttled operation
    return processRequest()
})
```

**Key Features**: Backlog management, timeout handling, graceful degradation  
**Use Cases**: Load control, resource protection, capacity management

### ⏰ [Timer](./timer/) - Advanced Timing
JavaScript-style timer utilities for Go.

```go
import "github.com/alextanhongpin/core/sync/timer"

// setTimeout equivalent
cancel := timer.SetTimeout(func() {
    log.Println("Timer fired!")
}, 5*time.Second)

// setInterval equivalent
stop := timer.SetInterval(func() {
    log.Println("Interval tick")
}, 1*time.Second)
```

**Key Features**: setTimeout/setInterval, cancellation, resource management  
**Use Cases**: Delayed execution, periodic tasks, timeout handling

```go
import "github.com/alextanhongpin/core/sync/batch"

// Create batch loader
loader := batch.NewLoader(func(keys []string) ([]User, error) {
    return userService.LoadMany(keys)
})

// Load individual items (automatically batched)
user, err := loader.Load("user123")
```

**Key Features**: Automatic batching, caching, wait groups, error handling  
**Use Cases**: Database batch loading, API request batching, N+1 query prevention

### 🔌 [Circuit Breaker](./circuitbreaker/) - Fault Tolerance
Circuit breaker pattern for fault tolerance and graceful degradation.

```go
import "github.com/alextanhongpin/core/sync/circuitbreaker"

cb := circuitbreaker.New(&circuitbreaker.Config{
    MaxFailures: 5,
    Timeout:     30 * time.Second,
})

err := cb.Call(ctx, func() error {
    return externalService.Call()
})
```

**Key Features**: Multiple states (Open/Closed/Half-Open), failure thresholds, timeout handling  
**Use Cases**: External API calls, database connections, microservice communication

### 📊 [DataLoader](./dataloader/) - Efficient Data Loading
Batching and caching for efficient data loading patterns.

```go
import "github.com/alextanhongpin/core/sync/dataloader"

loader := dataloader.New(func(keys []string) ([]User, []error) {
    return userService.BatchLoad(keys)
})

user, err := loader.Load("user123")
```

**Key Features**: Request batching, result caching, error handling, deduplication  
**Use Cases**: GraphQL resolvers, database optimization, API response caching

### ⏱️ [Debounce](./debounce/) - Event Throttling
Debouncing for rate limiting and event throttling.

```go
import "github.com/alextanhongpin/core/sync/debounce"

debouncer := &debounce.Group{
    Every:   10,           // Execute every 10 calls
    Timeout: 5 * time.Second, // Or after 5 seconds
}

debouncer.Do(func() {
    // This will be called after 10 invocations or 5 seconds
    log.Println("Debounced execution")
})
```

**Key Features**: Count-based and time-based debouncing, thread-safe  
**Use Cases**: Input validation, API rate limiting, event aggregation

### 🔒 [Lock](./lock/) - Synchronization Primitives
Local and distributed locking mechanisms.

```go
import "github.com/alextanhongpin/core/sync/lock"

var mu sync.Mutex
lock.Do(&mu, func() {
    // Critical section
    sharedResource.Update()
})
```

**Key Features**: Mutex helpers, value protection, concurrent access control  
**Use Cases**: Resource protection, critical sections, state synchronization

### 🔄 [Pipeline](./pipeline/) - Stream Processing
Stream processing pipelines for data transformation.

```go
import "github.com/alextanhongpin/core/sync/pipeline"

source := pipeline.NewSource(inputData)
sink := pipeline.NewSink()

pipeline := pipeline.New(source, 
    pipeline.Transform(processData),
    pipeline.Filter(validateData),
    sink,
)

results := pipeline.Run(ctx)
```

**Key Features**: Stream processing, data transformation, filtering, parallel processing  
**Use Cases**: Data ETL, real-time processing, stream analytics

### 🎯 [Poll](./poll/) - Periodic Operations
Polling utilities for periodic task execution.

```go
import "github.com/alextanhongpin/core/sync/poll"

poller := poll.New(1*time.Second, func(ctx context.Context) error {
    return checkHealth()
})

poller.Start(ctx)
defer poller.Stop()
```

**Key Features**: Configurable intervals, rate limiting, context support  
**Use Cases**: Health checks, monitoring, periodic cleanup

### 🤝 [Promise](./promise/) - Async Programming
Promise patterns for asynchronous programming.

```go
import "github.com/alextanhongpin/core/sync/promise"

p := promise.New(func() (string, error) {
    return expensiveOperation()
})

result, err := p.Await() // Blocks until complete
```

**Key Features**: Async execution, result caching, error handling, futures  
**Use Cases**: Async operations, concurrent computations, lazy evaluation

### 📈 [Rate](./rate/) - Rate Limiting
Advanced rate limiting with multiple algorithms.

```go
import "github.com/alextanhongpin/core/sync/rate"

limiter := rate.NewLimiter(100, time.Second) // 100 requests per second

if limiter.Allow() {
    // Request allowed
    handleRequest()
} else {
    // Request denied
    http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
}
```

**Key Features**: Token bucket, sliding window, distributed rate limiting  
**Use Cases**: API rate limiting, traffic control, abuse prevention

### 📊 [RateLimit](./ratelimit/) - Traffic Control
Comprehensive rate limiting with Redis backend.

```go
import "github.com/alextanhongpin/core/sync/ratelimit"

limiter := ratelimit.New(client, &ratelimit.Config{
    Window: time.Minute,
    Limit:  100,
})

allowed, err := limiter.Allow(ctx, "user123")
```

**Key Features**: Redis-based, distributed, multiple algorithms, sliding windows  
**Use Cases**: Distributed systems, API gateways, microservice rate limiting

### 🔄 [Retry](./retry/) - Intelligent Retries
Retry mechanisms with configurable backoff strategies.

```go
import "github.com/alextanhongpin/core/sync/retry"

err := retry.Do(ctx, func() error {
    return unreliableOperation()
}, retry.WithBackoff(retry.ExponentialBackoff(time.Second)))
```

**Key Features**: Multiple backoff strategies, context support, error filtering  
**Use Cases**: API calls, database operations, network requests

### 🔀 [SingleFlight](./singleflight/) - Request Deduplication
Request deduplication and coalescing for cache stampede prevention.

```go
import "github.com/alextanhongpin/core/sync/singleflight"

var sf singleflight.Group

result, err, shared := sf.Do("key", func() (interface{}, error) {
    return expensiveOperation()
})
```

**Key Features**: Request deduplication, shared results, memory efficient  
**Use Cases**: Cache warming, API deduplication, expensive computations

### 📸 [Snapshot](./snapshot/) - State Management
State capturing and versioning utilities.

```go
import "github.com/alextanhongpin/core/sync/snapshot"

snap := snapshot.New()
snap.Save("state1", currentState)

// Later...
previousState := snap.Load("state1")
```

**Key Features**: State versioning, rollback support, memory management  
**Use Cases**: State machines, undo operations, debugging

### ⏳ [Throttle](./throttle/) - Adaptive Load Control
Adaptive throttling for load control and backpressure.

```go
import "github.com/alextanhongpin/core/sync/throttle"

throttler := throttle.New(&throttle.Options{
    Limit:        100,
    BacklogLimit: 50,
})

err := throttler.Do(ctx, func() error {
    return processRequest()
})
```

**Key Features**: Adaptive throttling, backlog management, context support  
**Use Cases**: Load shedding, backpressure handling, service protection

### ⏰ [Timer](./timer/) - Advanced Timing
Advanced timing utilities and schedulers.

```go
import "github.com/alextanhongpin/core/sync/timer"

timer := timer.New(5*time.Second, func() {
    log.Println("Timer fired")
})

timer.Start()
defer timer.Stop()
```

**Key Features**: Configurable timers, schedulers, recurring tasks  
**Use Cases**: Scheduled tasks, timeout handling, periodic operations

## 🛠️ Installation

Install individual packages:

```bash
go get github.com/alextanhongpin/core/sync/background
go get github.com/alextanhongpin/core/sync/batch
go get github.com/alextanhongpin/core/sync/circuitbreaker
# ... other packages
```

Or install all packages:

```bash
go get github.com/alextanhongpin/core/sync/...
```

## 🚀 Quick Start

### Building a Robust Service

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/alextanhongpin/core/sync/background"
    "github.com/alextanhongpin/core/sync/circuitbreaker"
    "github.com/alextanhongpin/core/sync/rate"
    "github.com/alextanhongpin/core/sync/retry"
)

func main() {
    ctx := context.Background()
    
    // Rate limiter for API calls
    limiter := rate.NewLimiter(100, time.Second)
    
    // Circuit breaker for external service
    cb := circuitbreaker.New(&circuitbreaker.Config{
        MaxFailures: 5,
        Timeout:     30 * time.Second,
    })
    
    // Background worker pool
    worker, stop := background.New(ctx, 4, func(ctx context.Context, task Task) {
        // Rate limited processing
        if !limiter.Allow() {
            log.Println("Rate limit exceeded, skipping task")
            return
        }
        
        // Circuit breaker protected call
        err := cb.Call(ctx, func() error {
            return processTaskWithRetry(ctx, task)
        })
        
        if err != nil {
            log.Printf("Task failed: %v", err)
        }
    })
    defer stop()
    
    // Send tasks
    worker.Send(Task{ID: "task1", Data: "example"})
}

func processTaskWithRetry(ctx context.Context, task Task) error {
    return retry.Do(ctx, func() error {
        return externalService.Process(task)
    }, retry.WithBackoff(retry.ExponentialBackoff(time.Second)))
}
```

### Data Loading with Caching

```go
package main

import (
    "context"
    "log"

    "github.com/alextanhongpin/core/sync/batch"
    "github.com/alextanhongpin/core/sync/dataloader"
    "github.com/alextanhongpin/core/sync/singleflight"
)

func main() {
    ctx := context.Background()
    
    // Single flight for deduplication
    var sf singleflight.Group
    
    // Batch loader for database queries
    loader := batch.NewLoader(func(keys []string) ([]User, error) {
        result, err, _ := sf.Do("users", func() (interface{}, error) {
            return userService.LoadMany(keys)
        })
        return result.([]User), err
    })
    
    // DataLoader for GraphQL resolvers
    userLoader := dataloader.New(func(keys []string) ([]User, []error) {
        users, err := loader.LoadMany(keys)
        errors := make([]error, len(keys))
        for i := range errors {
            errors[i] = err
        }
        return users, errors
    })
    
    // Load users efficiently
    user, err := userLoader.Load("user123")
    if err != nil {
        log.Printf("Failed to load user: %v", err)
    }
    
    log.Printf("Loaded user: %+v", user)
}
```

## 🏗️ Architecture Patterns

### Worker Pool Pattern

```go
// High-throughput background processing
worker, stop := background.New(ctx, runtime.NumCPU(), processTask)
defer stop()

for task := range taskStream {
    worker.Send(task)
}
```

### Circuit Breaker Pattern

```go
// Fault tolerance for external services
cb := circuitbreaker.New(&circuitbreaker.Config{
    MaxFailures: 5,
    Timeout:     30 * time.Second,
})

err := cb.Call(ctx, func() error {
    return externalService.Call()
})
```

### Rate Limiting Pattern

```go
// API rate limiting
limiter := rate.NewLimiter(100, time.Second)

http.HandleFunc("/api/endpoint", func(w http.ResponseWriter, r *http.Request) {
    if !limiter.Allow() {
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
    }
    // Handle request
})
```

### Retry Pattern

```go
// Intelligent retry with backoff
err := retry.Do(ctx, func() error {
    return unreliableOperation()
}, retry.WithBackoff(retry.ExponentialBackoff(time.Second)))
```

## 📊 Performance Considerations

### Benchmarks

All packages include comprehensive benchmarks:

```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific package benchmarks
go test -bench=. ./background
go test -bench=. ./batch
go test -bench=. ./circuitbreaker
```

### Memory Usage

- **Background**: Minimal overhead, configurable worker pools
- **Batch**: Efficient batching with memory pooling
- **Circuit Breaker**: Lightweight state tracking
- **Rate Limiting**: Token bucket with minimal memory footprint

### Concurrency

All packages are designed for high concurrency:

- Thread-safe operations
- Context-aware cancellation
- Efficient synchronization primitives
- Minimal lock contention

## 🧪 Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run race detection
go test -race ./...
```

### Integration Tests

```bash
# Run integration tests
go test -tags=integration ./...
```

## 🔧 Configuration

### Environment Variables

```bash
# Worker pool size
WORKER_POOL_SIZE=8

# Rate limiting
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60s

# Circuit breaker
CIRCUIT_BREAKER_TIMEOUT=30s
CIRCUIT_BREAKER_MAX_FAILURES=5
```

### Best Practices

1. **Use Context**: Always pass context for cancellation
2. **Handle Errors**: Implement proper error handling
3. **Monitor Resources**: Track memory and CPU usage
4. **Test Thoroughly**: Include unit and integration tests
5. **Document Config**: Clearly document configuration options

## 📈 Monitoring

### Metrics

Each package can be instrumented with metrics:

```go
// Add metrics to your implementation
import "github.com/prometheus/client_golang/prometheus"

var (
    requestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "requests_total",
            Help: "Total number of requests",
        },
        []string{"method", "status"},
    )
)
```

### Logging

```go
// Structured logging
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Info("Worker started", "workers", numWorkers)
```

## 🤝 Contributing

1. **Follow Go Conventions**: Use standard Go formatting and conventions
2. **Add Tests**: Include comprehensive tests for new functionality
3. **Document Changes**: Update documentation and examples
4. **Benchmark**: Include performance benchmarks for critical paths
5. **Maintain Compatibility**: Ensure backward compatibility

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🔗 Related Projects

- [Go Context](https://golang.org/pkg/context/) - Understanding Go context
- [Go Concurrency Patterns](https://go.dev/blog/pipelines) - Concurrency patterns
- [Effective Go](https://go.dev/doc/effective_go) - Go best practices
- [Go Memory Model](https://go.dev/ref/mem) - Memory model and synchronization

---

**Built with ❤️ for the Go community**
