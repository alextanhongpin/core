# Redis Cache Implementation

A robust, thread-safe Redis-based cache implementation with atomic operations and comprehensive error handling.

## Features

- **Thread-safe operations** with Redis-backed atomic guarantees
- **Comprehensive error handling** with sentinel errors for reliable error checking
- **Type safety** with safe type assertions to prevent panics
- **JSON wrapper** for automatic serialization/deserialization
- **Atomic operations** like Compare-and-Swap and Load-or-Store
- **Convenience methods** for common cache operations
- **Full test coverage** with examples

## Installation

```go
import "github.com/alextanhongpin/core/dsync/cache"
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "time"
    
    "github.com/alextanhongpin/core/dsync/cache"
    "github.com/redis/go-redis/v9"
)

func main() {
    // Initialize Redis client
    client := redis.NewClient(&redis.Options{
        Addr: ":6379",
    })
    defer client.Close()
    
    // Create cache instance
    c := cache.New(client)
    ctx := context.Background()
    
    // Store a value
    err := c.Store(ctx, "user:123", []byte("John Doe"), time.Hour)
    if err != nil {
        // handle error
    }
    
    // Load a value
    value, err := c.Load(ctx, "user:123")
    if err != nil {
        // handle error
    }
}
```

### JSON Cache Usage

```go
type User struct {
    ID   int64  `json:"id"`
    Name string `json:"name"`
}

jsonCache := cache.NewJSON(client)

// Store JSON object
user := User{ID: 123, Name: "John Doe"}
err := jsonCache.Store(ctx, "user:123", user, time.Hour)

// Load JSON object
var loadedUser User
err = jsonCache.Load(ctx, "user:123", &loadedUser)

// Load-or-Store with getter function
var user User
loaded, err := jsonCache.LoadOrStore(ctx, "user:456", &user, func() (any, error) {
    // This function is called only if the key doesn't exist
    return fetchUserFromDatabase(456)
}, time.Hour)
```

## Error Handling

The library provides sentinel errors for reliable error checking:

```go
import "errors"

_, err := c.Load(ctx, "nonexistent-key")
switch {
case errors.Is(err, cache.ErrNotExist):
    // Key doesn't exist - handle cache miss
case errors.Is(err, cache.ErrValueMismatch):
    // Compare operation failed due to value mismatch
case errors.Is(err, cache.ErrUnexpectedType):
    // Redis returned unexpected data type
case errors.Is(err, cache.ErrExists):
    // Key already exists (from StoreOnce)
case errors.Is(err, cache.ErrOperationNotSupported):
    // Operation not supported by underlying implementation
case err != nil:
    // Other Redis or network errors
}
```

## Available Operations

### Core Cache Interface

| Method | Description |
|--------|-------------|
| `Load(key)` | Retrieve value for a key |
| `Store(key, value, ttl)` | Set key-value with TTL |
| `StoreOnce(key, value, ttl)` | Store only if key doesn't exist |
| `LoadOrStore(key, value, ttl)` | Atomic load-or-store operation |
| `LoadAndDelete(key)` | Atomic get-and-delete operation |
| `CompareAndDelete(key, expected)` | Delete only if value matches |
| `CompareAndSwap(key, old, new, ttl)` | Update only if value matches |
| `Exists(key)` | Check if key exists |
| `TTL(key)` | Get remaining time to live |
| `Expire(key, ttl)` | Set expiration on existing key |
| `Delete(keys...)` | Remove one or more keys |

### JSON Cache Additional Methods

All core methods plus automatic JSON marshaling/unmarshaling.

## Atomic Operations

### Compare-and-Swap (CAS)

```go
// Only update if current value matches expected
oldValue := []byte("old")
newValue := []byte("new")
err := c.CompareAndSwap(ctx, "key", oldValue, newValue, time.Hour)
if errors.Is(err, cache.ErrValueMismatch) {
    // Value was modified by another process
}
```

### Load-or-Store

```go
// Atomically load existing value or store new one
value := []byte("default")
current, loaded, err := c.LoadOrStore(ctx, "key", value, time.Hour)
if loaded {
    // Value was already in cache
} else {
    // Value was stored
}
```

## Best Practices

### 1. Use Sentinel Errors

Always use `errors.Is()` for error checking:

```go
// ✅ Good
if errors.Is(err, cache.ErrNotExist) {
    // handle cache miss
}

// ❌ Bad
if err.Error() == "redis: nil" {
    // fragile string matching
}
```

### 2. Handle Cache Misses Gracefully

```go
value, err := c.Load(ctx, key)
switch {
case errors.Is(err, cache.ErrNotExist):
    // Load from database, then cache it
    value = loadFromDatabase(key)
    c.Store(ctx, key, value, time.Hour)
case err != nil:
    // Handle other errors
    return err
default:
    // Use cached value
}
```

### 3. Use Appropriate TTLs

```go
// Short TTL for frequently changing data
c.Store(ctx, "session:"+sessionID, sessionData, 15*time.Minute)

// Longer TTL for stable data
c.Store(ctx, "user:"+userID, userData, 24*time.Hour)

// No expiration for permanent data (use 0)
c.Store(ctx, "config:"+key, configData, 0)
```

### 4. Leverage Atomic Operations

```go
// Use LoadOrStore to prevent race conditions
_, loaded, err := c.LoadOrStore(ctx, key, expensiveValue, time.Hour)
if !loaded {
    log.Printf("Computed and cached expensive value for %s", key)
}
```

## Thread Safety

All operations are thread-safe and provide strong consistency guarantees through Redis. The implementation handles:

- **Concurrent reads and writes** safely
- **Atomic operations** that prevent race conditions
- **Safe type assertions** that never panic
- **Proper error propagation** from Redis

## Performance Considerations

- **Pipeline operations** when possible for bulk operations
- **Appropriate TTLs** to balance freshness and performance
- **Connection pooling** through Redis client configuration
- **Lua scripts** for atomic multi-step operations

## Examples

See the `examples/` directory for comprehensive usage examples:

- `examples/improved_usage.go` - Basic usage and convenience methods
- `examples/sentinel_errors.go` - Error handling patterns
- `examples/main.go` - JSON cache with real-world patterns

## Contributing

When contributing:

1. Maintain thread safety guarantees
2. Add appropriate tests for new functionality
3. Use sentinel errors for consistent error handling
4. Document new methods with godoc comments
5. Ensure backward compatibility
