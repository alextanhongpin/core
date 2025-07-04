# Lock Package

A distributed lock implementation for Go using Redis, designed for coordinating access to shared resources across multiple application instances.

## Features

- **Distributed Locking**: Coordinate access across multiple processes/servers
- **Lock Refresh**: Automatically extend locks during long operations
- **Exponential Backoff**: Intelligent retry mechanism with configurable backoff
- **PubSub Optimization**: Fast lock acquisition using Redis pub/sub notifications
- **Context Support**: Full context cancellation and timeout support
- **Keyed Mutexes**: Prevent local deadlocks with per-key mutexes
- **Structured Logging**: Built-in logging with configurable logger

## Installation

```bash
go get github.com/alextanhongpin/core/dsync/lock
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/alextanhongpin/core/dsync/lock"
    "github.com/redis/go-redis/v9"
)

func main() {
    // Create Redis client
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer client.Close()
    
    // Create locker
    locker := lock.New(client)
    
    // Use the lock
    ctx := context.Background()
    err := locker.Do(ctx, "my-resource", func(ctx context.Context) error {
        // Your critical section here
        log.Println("Executing critical section")
        time.Sleep(2 * time.Second)
        return nil
    }, &lock.LockOption{
        Lock:         30 * time.Second,  // Lock duration
        Wait:         5 * time.Second,   // Wait timeout
        RefreshRatio: 0.8,              // Refresh at 80% of lock duration
    })
    
    if err != nil {
        log.Fatal(err)
    }
}
```

## Usage Patterns

### Basic Locking

```go
// Simple lock without waiting
locker := lock.New(client)
err := locker.Do(ctx, "resource-key", func(ctx context.Context) error {
    // Critical section
    return nil
}, &lock.LockOption{
    Lock: 30 * time.Second,
    Wait: 0, // Don't wait if lock is busy
})

if errors.Is(err, lock.ErrLocked) {
    log.Println("Resource is busy")
}
```

### Lock with Waiting

```go
// Wait up to 10 seconds for lock
err := locker.Do(ctx, "resource-key", func(ctx context.Context) error {
    // Critical section
    return nil
}, &lock.LockOption{
    Lock: 30 * time.Second,
    Wait: 10 * time.Second,
    RefreshRatio: 0.7, // Refresh every 70% of lock duration
})
```

### PubSub Optimization

For better performance when multiple processes are waiting for the same lock:

```go
// Use PubSub for faster lock acquisition
pubsubLocker := lock.NewPubSub(client)
err := pubsubLocker.Do(ctx, "resource-key", func(ctx context.Context) error {
    // Critical section
    return nil
}, &lock.LockOption{
    Lock: 30 * time.Second,
    Wait: 10 * time.Second,
    RefreshRatio: 0.8,
})
```

### Manual Lock Control

```go
locker := lock.New(client)

// Try to acquire lock
token := "my-unique-token"
err := locker.TryLock(ctx, "resource-key", token, 30*time.Second)
if err != nil {
    if errors.Is(err, lock.ErrLocked) {
        log.Println("Resource is locked")
    }
    return
}

// Extend lock if needed
err = locker.Extend(ctx, "resource-key", token, 30*time.Second)
if err != nil {
    log.Printf("Failed to extend lock: %v", err)
}

// Always unlock
err = locker.Unlock(ctx, "resource-key", token)
if err != nil {
    log.Printf("Failed to unlock: %v", err)
}
```

## Configuration Options

### LockOption Fields

- **Lock** (`time.Duration`): Duration for which the lock is held
- **Wait** (`time.Duration`): Maximum time to wait for lock acquisition (0 = no wait)
- **RefreshRatio** (`float64`): Ratio of lock duration at which to refresh (0.8 = refresh at 80%)
- **Token** (`string`): Optional custom token for lock ownership

### Default Values

```go
&lock.LockOption{
    Lock:         30 * time.Second,
    Wait:         5 * time.Second,
    RefreshRatio: 0.8,
    Token:        "", // Auto-generated UUID
}
```

## Error Types

```go
var (
    ErrLocked          = errors.New("lock: another process has acquired the lock")
    ErrExpired         = errors.New("lock: lock expired")
    ErrLockTimeout     = errors.New("lock: exceeded lock duration")
    ErrLockWaitTimeout = errors.New("lock: failed to acquire lock within the wait duration")
)
```

## Best Practices

### 1. Lock Duration Guidelines

- Set lock duration longer than expected operation time
- Use RefreshRatio to automatically extend locks during long operations
- Consider network latency and clock skew between servers

### 2. Wait Time Configuration

- Set reasonable wait times to avoid indefinite blocking
- Use context with timeout for additional safety
- Consider exponential backoff for retry scenarios

### 3. Error Handling

```go
err := locker.Do(ctx, key, fn, opts)
switch {
case errors.Is(err, lock.ErrLocked):
    // Handle busy resource
case errors.Is(err, lock.ErrLockWaitTimeout):
    // Handle timeout waiting for lock
case errors.Is(err, lock.ErrLockTimeout):
    // Handle lock expiration during operation
case errors.Is(err, lock.ErrExpired):
    // Handle lock expiration (e.g., Redis restart)
default:
    // Handle other errors
}
```

### 4. Resource Naming

- Use descriptive, hierarchical lock keys: `user:123:profile`, `order:456:payment`
- Avoid overly long keys (Redis key length limits)
- Consider key expiration for cleanup

### 5. Monitoring and Observability

```go
// Custom logger for lock operations
locker := lock.New(client)
locker.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
```

## Architecture

### Single Redis Instance

The lock package is designed for single Redis node deployments. For Redis Cluster or high availability setups, consider using Redis Sentinel or implementing additional coordination mechanisms.

### Lock Refresh Mechanism

When `RefreshRatio > 0`, the lock is automatically refreshed:

1. Function executes in a goroutine
2. Timer triggers at `RefreshRatio * LockDuration`
3. Lock TTL is extended atomically
4. Process continues until function completes

### Exponential Backoff

The default backoff policy implements exponential backoff with jitter:

- Base: 1 second
- Limit: 1 minute
- Formula: `rand(min(base * 2^attempt, limit))`

## Performance Considerations

- **Throughput**: Depends on Redis latency and lock contention
- **Memory Usage**: ~64 bytes per active lock + keyed mutex overhead
- **Network**: 2-3 Redis operations per lock acquisition
- **PubSub**: Reduces polling overhead for high-contention scenarios

## Testing

Run tests with Redis:

```bash
go test -v ./...
```

Run tests with race detection:

```bash
go test -v -race ./...
```

## Limitations

1. **Single Redis Node**: Not suitable for Redis Cluster
2. **Clock Skew**: Sensitive to time differences between servers
3. **Network Partitions**: No automatic failover mechanism
4. **Memory Growth**: KeyedMutex accumulates mutexes (consider periodic cleanup)

## Contributing

1. Ensure all tests pass
2. Add tests for new features
3. Update documentation
4. Run `go vet` and `golint`

## License

MIT License
