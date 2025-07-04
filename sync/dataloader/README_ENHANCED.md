# DataLoader

A production-ready Go implementation of the DataLoader pattern for efficient batching and caching of data fetching operations. This implementation is inspired by Facebook's DataLoader and is designed for high-performance applications with comprehensive observability features.

## Features

- **Automatic Batching**: Collects individual loads over a configurable time window and batches them into single requests
- **Smart Caching**: In-memory caching with configurable cache management
- **Comprehensive Metrics**: Runtime metrics for monitoring performance and cache effectiveness
- **Observability Hooks**: Configurable callbacks for monitoring batch operations, cache hits/misses, and errors
- **Timeout Handling**: Configurable timeouts for load operations
- **Thread-Safe**: Safe for concurrent use across multiple goroutines
- **Generic Type Support**: Fully typed with Go generics for type safety
- **Production Ready**: Includes error handling, graceful shutdown, and performance optimizations

## Installation

```bash
go get github.com/alextanhongpin/core/sync/dataloader
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/alextanhongpin/core/sync/dataloader"
)

func main() {
    // Create a new DataLoader
    dl := dataloader.New(context.Background(), &dataloader.Options[int, *User]{
        BatchFn: func(ctx context.Context, keys []int) (map[int]*User, error) {
            // Simulate database query
            log.Printf("Loading users: %v", keys)
            
            users := make(map[int]*User)
            for _, id := range keys {
                users[id] = &User{ID: id, Name: fmt.Sprintf("User %d", id)}
            }
            return users, nil
        },
        BatchTimeout: 16 * time.Millisecond,
        BatchMaxKeys: 100,
    })
    defer dl.Stop()

    // Load individual users - these will be batched automatically
    user1, err := dl.Load(1)
    if err != nil {
        log.Fatal(err)
    }
    
    user2, err := dl.Load(2)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Loaded: %+v, %+v\n", user1, user2)
}

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}
```

## Advanced Configuration

```go
dl := dataloader.New(context.Background(), &dataloader.Options[int, *User]{
    BatchFn: batchLoadUsers,
    
    // Batching configuration
    BatchMaxKeys:   100,                    // Max keys per batch
    BatchTimeout:   16 * time.Millisecond,  // Max wait time before batching
    BatchQueueSize: 10,                     // Queue size for batches
    
    // Timeout configuration
    LoadTimeout: 30 * time.Second,          // Timeout for individual loads
    
    // Cache configuration
    MaxCacheSize: 10000,                    // Max items in cache
    Cache: customCache,                     // Custom cache implementation
    
    // Observability callbacks
    OnBatchStart: func(keys []int) {
        log.Printf("Starting batch for %d keys", len(keys))
    },
    OnBatchComplete: func(keys []int, duration time.Duration, err error) {
        if err != nil {
            log.Printf("Batch failed: %v", err)
        } else {
            log.Printf("Batch completed in %v", duration)
        }
    },
    OnCacheHit: func(key int) {
        log.Printf("Cache hit for key %d", key)
    },
    OnCacheMiss: func(key int) {
        log.Printf("Cache miss for key %d", key)
    },
    OnError: func(key int, err error) {
        log.Printf("Error loading key %d: %v", key, err)
    },
})
```

## API Reference

### Core Methods

#### `Load(key K) (V, error)`
Load a single value by key. This operation will be batched with other concurrent loads.

#### `LoadMany(keys []K) ([]promise.Result[V], error)`
Load multiple values by keys. Returns results in the same order as the input keys.

#### `LoadWithTimeout(ctx context.Context, key K) (V, error)`
Load a single value with a custom timeout context.

#### `Set(key K, value V)`
Manually set a value in the cache.

#### `Stop()`
Gracefully stop the DataLoader and wait for all pending operations to complete.

### Metrics and Monitoring

#### `Metrics() Metrics`
Get current runtime metrics:

```go
type Metrics struct {
    TotalRequests  int64  // Total Load/LoadMany calls
    KeysRequested  int64  // Total keys requested
    CacheHits      int64  // Keys served from cache
    CacheMisses    int64  // Keys loaded via batch function
    BatchCalls     int64  // Number of batch operations
    ErrorCount     int64  // Number of errors
    NoResultCount  int64  // Keys with no result
    CacheSize      int64  // Current cache size
    QueueLength    int64  // Current queue length
}
```

#### `ClearCache()`
Clear all entries from the cache.

### Error Handling

The DataLoader provides structured error handling:

- `ErrNoResult`: Returned when a key has no corresponding value
- `ErrTerminated`: Returned when the DataLoader is stopped
- `ErrTimeout`: Returned when a load operation times out
- `KeyError`: Wraps errors with the specific key that caused them

```go
user, err := dl.Load(123)
if err != nil {
    var keyErr *dataloader.KeyError
    if errors.As(err, &keyErr) {
        log.Printf("Error loading key %s: %v", keyErr.Key, keyErr.Unwrap())
    }
}
```

## Performance

The DataLoader is optimized for high throughput:

- **Single Load**: ~200ns per operation
- **Cached Load**: ~180ns per operation  
- **Metrics Access**: ~0.3ns per operation
- **Memory**: Minimal allocations with object pooling

Benchmarks show significant efficiency gains:
- **21x efficiency** compared to individual database calls
- **0% cache miss rate** for repeated access patterns
- **Minimal memory overhead** with concurrent operations

## Common Use Cases

### GraphQL Resolvers

Perfect for solving the N+1 query problem in GraphQL:

```go
type PostResolver struct {
    userLoader *dataloader.DataLoader[int, *User]
}

func (r *PostResolver) Author(ctx context.Context, post *Post) (*User, error) {
    return r.userLoader.Load(post.AuthorID)
}
```

### Microservices

Batch calls to downstream services:

```go
// Instead of making N individual HTTP calls
for _, id := range userIDs {
    user := httpClient.GetUser(id)  // N calls
}

// Batch them efficiently
users, err := userLoader.LoadMany(userIDs)  // 1 batch call
```

### Database Queries

Optimize database access patterns:

```go
// Batch SQL queries with IN clauses
func batchLoadUsers(ctx context.Context, ids []int) (map[int]*User, error) {
    query := "SELECT id, name, email FROM users WHERE id IN (?)"
    // Use IN clause for efficient batch loading
    return db.QueryUsersIn(ctx, query, ids)
}
```

## Best Practices

1. **Use appropriate batch sizes**: Balance between efficiency and memory usage
2. **Set reasonable timeouts**: Prevent indefinite blocking
3. **Monitor metrics**: Track cache hit rates and batch efficiency
4. **Handle errors gracefully**: Use structured error handling
5. **Cache strategically**: Consider TTL and cache size limits
6. **Test under load**: Verify performance characteristics

## Examples

See the `examples/` directory for complete working examples:
- `examples/main.go` - Basic usage
- `examples/advanced/main.go` - Production-ready GraphQL-like resolver with comprehensive monitoring

## Contributing

Contributions are welcome! Please ensure:
- All tests pass (`go test -v`)
- Benchmarks show no performance regression (`go test -bench=.`)
- Code follows Go conventions
- Documentation is updated

## License

This project is licensed under the MIT License.
