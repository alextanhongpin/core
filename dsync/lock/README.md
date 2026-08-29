# Lock Package

A distributed lock implementation for Go using Redis, designed for coordinating access to shared resources across multiple application instances.

## Features

- **Distributed Locking**: Coordinate access across multiple processes/servers with Redis
- **Automatic Lock Refresh**: Automatically extend locks during long operations using `RefreshRatio`
- **Context Support**: Full context cancellation and timeout support
- **Keyed Mutexes**: Prevent local deadlocks with per-key in-process mutexes
- **Structured Logging**: Built-in `slog.Logger` for debugging
- **Exponential Backoff**: Intelligent retry mechanism with jitter when waiting for a lock
- **Configurable TTL**: Separate wait timeout and lock TTL

## Installation

```bash
go get github.com/alextanhongpin/core/dsync/lock
```

Requires Go 1.27+ and a single-node Redis instance.

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
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer redisClient.Close()
    
    // Create locker
    client := lock.NewClient(redisClient)
    locker := lock.New(client, &lock.Config{
        LockTTL:      30 * time.Second, // Duration for which the lock is held
        WaitTTL:      5 * time.Second,  // Max time to wait for acquisition
        RefreshRatio: 0.8,              // Refresh at 80% of LockTTL
    })
    
    // Use the lock
    ctx := context.Background()
    err := locker.Do(ctx, "my-resource", func(ctx context.Context) error {
        // Your critical section here
        log.Println("Executing critical section")
        time.Sleep(2 * time.Second)
        return nil
    })
    
    if err != nil {
        log.Fatal(err)
    }
}
```

## Configuration

`lock.Config` holds the lock behavior:

```go
type Config struct {
    // WaitTTL is the duration to wait for the lock to become available.
    // 0 means don't wait.
    WaitTTL time.Duration
    // LockTTL is the duration for which the lock is held in Redis.
    LockTTL time.Duration
    // RefreshRatio is the ratio of LockTTL at which the lock is refreshed.
    // Set to 0 or negative to disable refresh. The operation will then be
    // bounded by a context timeout equal to LockTTL.
    RefreshRatio float64
}
```

Defaults:

```go
lock.DefaultConfig()
// WaitTTL:      5 * time.Second
// LockTTL:      30 * time.Second
// RefreshRatio: 0.8
```

## Usage Patterns

### Basic Locking

```go
locker := lock.New(client, &lock.Config{
    LockTTL: 30 * time.Second,
    WaitTTL: 0, // Don't wait if lock is busy
})

// No wait
err := locker.Do(ctx, "resource-key", func(ctx context.Context) error {
    // Critical section
    return nil
})

if errors.Is(err, lock.ErrLocked) {
    log.Println("Resource is busy")
}
```

### Lock with Waiting

```go
locker := lock.New(client, &lock.Config{
    LockTTL:      30 * time.Second,
    WaitTTL:      10 * time.Second,
    RefreshRatio: 0.7,
})

err := locker.Do(ctx, "resource-key", func(ctx context.Context) error {
    // Critical section
    return nil
})
```

### Manual Lock Control

The underlying `lock.Client` exposes explicit acquire/extend/release operations:

```go
c := lock.NewClient(redisClient)

ctx := context.Background()
token := "my-unique-token"
err := c.Lock(ctx, "resource-key", token, 30*time.Second, 5*time.Second)
if err != nil {
    if errors.Is(err, lock.ErrLocked) {
        log.Println("Resource is locked")
    }
    return
}
defer c.Unlock(ctx, "resource-key", token)

// Extend if needed
err = c.Extend(ctx, "resource-key", token, 30*time.Second)
```

### Func Helper

Wrap any function with locking:

```go
fn := func(ctx context.Context, id int) (string, error) {
    return fmt.Sprintf("result-%d", id), nil
}

lockedFn := lock.Func(fn, locker, func(id int) string {
    return fmt.Sprintf("key:%d", id)
})

res, err := lockedFn(ctx, 123)
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

Error handling example:

```go
err := locker.Do(ctx, key, fn)
switch {
case errors.Is(err, lock.ErrLocked):
    // Busy resource
case errors.Is(err, lock.ErrLockWaitTimeout):
    // Timeout waiting for lock
case errors.Is(err, lock.ErrLockTimeout):
    // Lock expired during operation
case errors.Is(err, lock.ErrExpired):
    // Lock expired, e.g., Redis restart
default:
    // Other errors
}
```

## Best Practices

### 1. Lock Duration Guidelines
- Set `LockTTL` longer than expected operation time
- Use `RefreshRatio` > 0 to automatically extend locks during long operations
- Consider network latency and clock skew between servers

### 2. Wait Time Configuration
- Set `WaitTTL` to avoid indefinite blocking
- Use context with timeout for additional safety
- Backoff with jitter is applied automatically while waiting

### 3. Resource Naming
- Use descriptive, hierarchical lock keys: `user:123:profile`, `order:456:payment`
- Avoid overly long keys

### 4. Monitoring and Observability

```go
locker := lock.New(client, cfg)
locker.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
```

## Architecture

### Single Redis Instance

Designed for a single Redis node. Not suitable for Redis Cluster without additional coordination.

### Lock Refresh Mechanism

When `RefreshRatio > 0`:
1. Function executes in a goroutine
2. Ticker triggers at `RefreshRatio * LockTTL`
3. `Extend` is called atomically to refresh TTL
4. Loop continues until function completes or context is cancelled

When `RefreshRatio <= 0`:
- No refresh is performed
- A context with timeout `LockTTL` is applied, causing `ErrLockTimeout` if the operation exceeds the TTL

### Keyed Mutex

An in-process `cache.Cache[string, sync.Mutex]` ensures only one goroutine per key proceeds in the same process, preventing local deadlocks.

## Performance Considerations

- Throughput depends on Redis latency and lock contention
- Memory: ~64 bytes per active Redis lock + in-process mutex overhead
- Each acquisition uses `SET NX` and each refresh uses `SET IF DEQ` via Redis

## Testing

Run tests with Redis:

```bash
go test -v ./...
```

Run with race detection:

```bash
go test -v -race ./...
```

## Limitations

1. **Single Redis Node**: Not designed for Redis Cluster
2. **Clock Skew**: Sensitive to time differences between servers
3. **Network Partitions**: No automatic failover mechanism
4. **Mutex Growth**: In-process keyed mutexes accumulate; consider periodic cleanup for long-running processes

## License

MIT License
