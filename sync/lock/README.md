# Lock Package

A production-ready named lock implementation for Go with advanced features including metrics, timeouts, context support, and automatic cleanup.

## Features

- **Named Locks**: Create locks identified by string keys
- **Lock Types**: Support for both `sync.Mutex` and `sync.RWMutex`
- **Timeout Support**: Lock operations with configurable timeouts
- **Context Support**: Context-aware locking with cancellation
- **Automatic Cleanup**: Unused locks are automatically cleaned up
- **Metrics**: Comprehensive runtime metrics and observability
- **Callbacks**: Customizable hooks for lock events
- **Reference Counting**: Automatic lock lifecycle management
- **Contention Detection**: Detect and monitor lock contention
- **Memory Management**: Configurable limits and cleanup strategies

## Quick Start

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/alextanhongpin/core/sync/lock"
)

func main() {
    // Create a new lock manager
    l := lock.New()
    defer l.Stop()
    
    // Get a named lock
    locker := l.Get("my-resource")
    
    // Use it like a regular mutex
    locker.Lock()
    defer locker.Unlock()
    
    // Do work with the protected resource
    fmt.Println("Working with protected resource")
}
```

## Advanced Configuration

```go
opts := lock.Options{
    DefaultTimeout:  30 * time.Second,
    CleanupInterval: 5 * time.Minute,
    LockType:        lock.RWMutex,
    MaxLocks:        10000,
    OnLockAcquired: func(key string, waitTime time.Duration) {
        log.Printf("Lock acquired for %s after %v", key, waitTime)
    },
    OnLockReleased: func(key string, holdTime time.Duration) {
        log.Printf("Lock released for %s after %v", key, holdTime)
    },
    OnLockContention: func(key string, waitingGoroutines int) {
        log.Printf("Contention detected for %s: %d waiters", key, waitingGoroutines)
    },
}

l := lock.NewWithOptions(opts)
defer l.Stop()
```

## Lock Types

### Standard Mutex Locks

```go
// Create with Mutex locks (default)
l := lock.New()

// Get a standard mutex lock
locker := l.Get("resource-key")
locker.Lock()
defer locker.Unlock()
```

### Read-Write Mutex Locks

```go
// Create with RWMutex locks
opts := lock.Options{
    LockType: lock.RWMutex,
}
l := lock.NewWithOptions(opts)

// Get a read-write mutex
rwMutex := l.GetRW("resource-key")

// Read operations
rwMutex.RLock()
// ... read operations ...
rwMutex.RUnlock()

// Write operations
rwMutex.Lock()
// ... write operations ...
rwMutex.Unlock()
```

## Timeout and Context Support

### Timeout-based Locking

```go
unlock, err := l.LockWithTimeout("resource-key", 5*time.Second)
if err != nil {
    if err == lock.ErrTimeout {
        log.Println("Lock acquisition timed out")
    }
    return err
}
defer unlock()
```

### Context-aware Locking

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

unlock, err := l.LockWithContext(ctx, "resource-key")
if err != nil {
    switch err {
    case lock.ErrTimeout:
        log.Println("Lock acquisition timed out")
    case lock.ErrCanceled:
        log.Println("Lock acquisition canceled")
    }
    return err
}
defer unlock()
```

## Metrics and Observability

### Getting Metrics

```go
metrics := l.Metrics()
fmt.Printf("Active Locks: %d\n", metrics.ActiveLocks)
fmt.Printf("Total Locks: %d\n", metrics.TotalLocks)
fmt.Printf("Lock Acquisitions: %d\n", metrics.LockAcquisitions)
fmt.Printf("Lock Contentions: %d\n", metrics.LockContentions)
fmt.Printf("Average Wait Time: %v\n", metrics.AverageWaitTime)
fmt.Printf("Max Wait Time: %v\n", metrics.MaxWaitTime)
```

### Observability Hooks

```go
opts := lock.Options{
    OnLockAcquired: func(key string, waitTime time.Duration) {
        // Called when a lock is successfully acquired
        if waitTime > 100*time.Millisecond {
            log.Printf("Slow lock acquisition for %s: %v", key, waitTime)
        }
    },
    OnLockReleased: func(key string, holdTime time.Duration) {
        // Called when a lock is released
        if holdTime > 1*time.Second {
            log.Printf("Long lock hold for %s: %v", key, holdTime)
        }
    },
    OnLockContention: func(key string, waitingGoroutines int) {
        // Called when lock contention is detected
        if waitingGoroutines > 10 {
            log.Printf("High contention for %s: %d waiters", key, waitingGoroutines)
        }
    },
}
```

## Real-World Examples

### User Service with Per-User Locking

```go
type UserService struct {
    locks *lock.Lock
    users map[string]*User
    mu    sync.RWMutex
}

func (s *UserService) UpdateUser(id string, updates map[string]interface{}) error {
    lockKey := fmt.Sprintf("user:%s", id)
    
    unlock, err := s.locks.LockWithTimeout(lockKey, 2*time.Second)
    if err != nil {
        return fmt.Errorf("failed to acquire lock for user %s: %w", id, err)
    }
    defer unlock()
    
    // Update user safely
    user := s.getUser(id)
    if user == nil {
        return fmt.Errorf("user %s not found", id)
    }
    
    // Apply updates...
    return nil
}
```

### Safe Money Transfer with Deadlock Prevention

```go
func (s *UserService) TransferMoney(fromID, toID string, amount int64) error {
    // Order locks by ID to prevent deadlocks
    firstID, secondID := fromID, toID
    if fromID > toID {
        firstID, secondID = toID, fromID
    }
    
    firstUnlock, err := s.locks.LockWithTimeout(fmt.Sprintf("user:%s", firstID), 2*time.Second)
    if err != nil {
        return err
    }
    defer firstUnlock()
    
    secondUnlock, err := s.locks.LockWithTimeout(fmt.Sprintf("user:%s", secondID), 2*time.Second)
    if err != nil {
        return err
    }
    defer secondUnlock()
    
    // Perform transfer safely
    return s.performTransfer(fromID, toID, amount)
}
```

### Cache with Read-Write Locks

```go
type Cache struct {
    locks *lock.Lock
    data  map[string]interface{}
    mu    sync.RWMutex
}

func (c *Cache) Get(key string) interface{} {
    lockKey := fmt.Sprintf("cache:%s", key)
    rwMutex := c.locks.GetRW(lockKey)
    
    rwMutex.RLock()
    defer rwMutex.RUnlock()
    
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    return c.data[key]
}

func (c *Cache) Set(key string, value interface{}) {
    lockKey := fmt.Sprintf("cache:%s", key)
    rwMutex := c.locks.GetRW(lockKey)
    
    rwMutex.Lock()
    defer rwMutex.Unlock()
    
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.data[key] = value
}
```

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `DefaultTimeout` | `time.Duration` | `30s` | Default timeout for lock operations |
| `CleanupInterval` | `time.Duration` | `5m` | How often to run cleanup for unused locks |
| `LockType` | `LockType` | `Mutex` | Type of locks to create (`Mutex` or `RWMutex`) |
| `MaxLocks` | `int` | `10000` | Maximum number of locks to keep in memory |
| `OnLockAcquired` | `func(string, time.Duration)` | `nil` | Called when a lock is acquired |
| `OnLockReleased` | `func(string, time.Duration)` | `nil` | Called when a lock is released |
| `OnLockContention` | `func(string, int)` | `nil` | Called when lock contention is detected |

## Performance Considerations

### Memory Usage

- Locks are automatically cleaned up when unused
- Configure `MaxLocks` to limit memory usage
- Use `CleanupInterval` to control cleanup frequency

### Contention

- Use different lock keys for different resources
- Consider using `RWMutex` for read-heavy workloads
- Monitor contention via metrics and callbacks

### Timeouts

- Set appropriate timeouts to avoid deadlocks
- Use context cancellation for request-scoped operations
- Monitor timeout occurrences via error handling

## Error Handling

The package defines several error types:

- `lock.ErrTimeout`: Returned when a lock operation times out
- `lock.ErrCanceled`: Returned when a context is canceled
- `lock.ErrInvalidKey`: Returned when an invalid key is provided

```go
unlock, err := l.LockWithTimeout("key", 1*time.Second)
if err != nil {
    switch err {
    case lock.ErrTimeout:
        // Handle timeout
    case lock.ErrInvalidKey:
        // Handle invalid key
    default:
        // Handle other errors
    }
    return err
}
defer unlock()
```

## Best Practices

1. **Always call `Stop()`** when done with the lock manager
2. **Use ordered locking** to prevent deadlocks when acquiring multiple locks
3. **Set appropriate timeouts** to avoid indefinite blocking
4. **Monitor metrics** to detect performance issues
5. **Use descriptive lock keys** for better observability
6. **Consider lock granularity** - too fine-grained can hurt performance, too coarse-grained can hurt concurrency
7. **Use RWMutex** for read-heavy workloads
8. **Handle errors properly** and implement retry logic where appropriate

## Thread Safety

The lock package is fully thread-safe and can be used concurrently from multiple goroutines. All operations are atomic and protected by appropriate synchronization primitives.

## Benchmarks

The package includes comprehensive benchmarks. Run them with:

```bash
go test -bench=. -benchmem
```

Example results:
```
BenchmarkLockBasic-8                    10000000    150 ns/op    0 B/op    0 allocs/op
BenchmarkLockMultipleKeys-8             5000000     280 ns/op    0 B/op    0 allocs/op
BenchmarkRWMutexRead-8                  20000000    75 ns/op     0 B/op    0 allocs/op
BenchmarkStandardMutex-8                50000000    25 ns/op     0 B/op    0 allocs/op
```

## Value and Values Utilities

The package also includes utilities for protected value storage:

### Value[T] - Single Value Protection

```go
// Create a protected value store
values := lock.NewValue[string]()

// Store a value with a key
values.LoadOrStore("config", func() string {
    return "default-config"
})

// The function is called only once per key
result := values.LoadOrStore("config", func() string {
    return "this-wont-be-called"
})
```

### Values[T] - Error-Returning Value Protection

```go
// Create a protected value store that can return errors
values := lock.NewValues[*User]()

// Store a value that might fail
user, err := values.LoadOrStore("user:123", func() (*User, error) {
    return loadUserFromDB("123")
})
if err != nil {
    log.Printf("Failed to load user: %v", err)
}
```

## License

This package is part of the `github.com/alextanhongpin/core` module and follows the same license terms.
            fmt.Printf("Goroutine %d - Counter: %d\n", i, counter)
        })
    }
    
    time.Sleep(100 * time.Millisecond)
}
```

### Protected Values

```go
package main

import (
    "fmt"
    "sync"

    "github.com/alextanhongpin/core/sync/lock"
)

func main() {
    // Create a protected value
    protectedValue := lock.NewValue(42)
    
    // Read the value safely
    value := protectedValue.Load()
    fmt.Printf("Value: %d\n", value)
    
    // Update the value safely
    protectedValue.Store(100)
    
    // Perform atomic operations
    protectedValue.Update(func(current int) int {
        return current * 2
    })
    
    newValue := protectedValue.Load()
    fmt.Printf("Updated value: %d\n", newValue)
}
```

### Protected Maps

```go
package main

import (
    "fmt"

    "github.com/alextanhongpin/core/sync/lock"
)

func main() {
    // Create a protected map
    protectedMap := lock.NewValues[string, int]()
    
    // Store values safely
    protectedMap.Store("key1", 10)
    protectedMap.Store("key2", 20)
    
    // Load values safely
    if value, ok := protectedMap.Load("key1"); ok {
        fmt.Printf("key1: %d\n", value)
    }
    
    // Update values atomically
    protectedMap.Update("key1", func(current int) int {
        return current + 5
    })
    
    // Range over all values
    protectedMap.Range(func(key string, value int) bool {
        fmt.Printf("%s: %d\n", key, value)
        return true // continue iteration
    })
}
```

## 🏗️ API Reference

### Functions

#### Do

```go
func Do(mu *sync.Mutex, fn func())
```

Executes a function with automatic mutex locking and unlocking.

**Parameters:**
- `mu`: Mutex to lock
- `fn`: Function to execute while holding the lock

#### DoRW

```go
func DoRW(mu *sync.RWMutex, fn func())
```

Executes a function with automatic read-write mutex locking and unlocking.

#### DoRead

```go
func DoRead(mu *sync.RWMutex, fn func())
```

Executes a function with automatic read lock.

### Types

#### Value[T]

```go
type Value[T any] struct {
    // Contains filtered or unexported fields
}
```

A type-safe protected value with automatic locking.

**Methods:**
```go
func NewValue[T any](initial T) *Value[T]
func (v *Value[T]) Load() T
func (v *Value[T]) Store(value T)
func (v *Value[T]) Update(fn func(T) T)
func (v *Value[T]) Swap(new T) (old T)
```

#### Values[K, V]

```go
type Values[K comparable, V any] struct {
    // Contains filtered or unexported fields
}
```

A type-safe protected map with automatic locking.

**Methods:**
```go
func NewValues[K comparable, V any]() *Values[K, V]
func (v *Values[K, V]) Load(key K) (V, bool)
func (v *Values[K, V]) Store(key K, value V)
func (v *Values[K, V]) Delete(key K)
func (v *Values[K, V]) Update(key K, fn func(V) V)
func (v *Values[K, V]) Range(fn func(K, V) bool)
func (v *Values[K, V]) Len() int
```

## 🌟 Real-World Examples

### Thread-Safe Counter Service

```go
package main

import (
    "fmt"
    "sync"
    "time"

    "github.com/alextanhongpin/core/sync/lock"
)

type CounterService struct {
    counters *lock.Values[string, int64]
    stats    *lock.Value[ServiceStats]
}

type ServiceStats struct {
    TotalIncrements int64
    TotalDecrements int64
    ActiveCounters  int
}

func NewCounterService() *CounterService {
    return &CounterService{
        counters: lock.NewValues[string, int64](),
        stats:    lock.NewValue(ServiceStats{}),
    }
}

func (cs *CounterService) Increment(name string, delta int64) int64 {
    // Update counter
    var newValue int64
    cs.counters.Update(name, func(current int64) int64 {
        newValue = current + delta
        return newValue
    })
    
    // Update stats
    cs.stats.Update(func(current ServiceStats) ServiceStats {
        current.TotalIncrements++
        current.ActiveCounters = cs.counters.Len()
        return current
    })
    
    return newValue
}

func (cs *CounterService) Decrement(name string, delta int64) int64 {
    var newValue int64
    cs.counters.Update(name, func(current int64) int64 {
        newValue = current - delta
        return newValue
    })
    
    // Update stats
    cs.stats.Update(func(current ServiceStats) ServiceStats {
        current.TotalDecrements++
        current.ActiveCounters = cs.counters.Len()
        return current
    })
    
    return newValue
}

func (cs *CounterService) Get(name string) (int64, bool) {
    return cs.counters.Load(name)
}

func (cs *CounterService) Reset(name string) {
    cs.counters.Delete(name)
    
    cs.stats.Update(func(current ServiceStats) ServiceStats {
        current.ActiveCounters = cs.counters.Len()
        return current
    })
}

func (cs *CounterService) GetStats() ServiceStats {
    return cs.stats.Load()
}

func (cs *CounterService) ListCounters() map[string]int64 {
    result := make(map[string]int64)
    
    cs.counters.Range(func(key string, value int64) bool {
        result[key] = value
        return true
    })
    
    return result
}

func main() {
    service := NewCounterService()
    
    fmt.Println("=== Thread-Safe Counter Service Demo ===")
    
    // Simulate concurrent counter operations
    var wg sync.WaitGroup
    
    // Multiple goroutines incrementing different counters
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            counterName := fmt.Sprintf("counter_%d", id%3) // 3 different counters
            
            for j := 0; j < 100; j++ {
                newValue := service.Increment(counterName, 1)
                if j%20 == 0 {
                    fmt.Printf("Goroutine %d: %s = %d\n", id, counterName, newValue)
                }
                time.Sleep(1 * time.Millisecond)
            }
        }(i)
    }
    
    // One goroutine occasionally decrementing
    wg.Add(1)
    go func() {
        defer wg.Done()
        
        for i := 0; i < 50; i++ {
            counterName := fmt.Sprintf("counter_%d", i%3)
            service.Decrement(counterName, 2)
            time.Sleep(10 * time.Millisecond)
        }
    }()
    
    // Monitor stats
    wg.Add(1)
    go func() {
        defer wg.Done()
        
        for i := 0; i < 10; i++ {
            stats := service.GetStats()
            fmt.Printf("📊 Stats - Increments: %d, Decrements: %d, Active: %d\n", 
                stats.TotalIncrements, stats.TotalDecrements, stats.ActiveCounters)
            time.Sleep(100 * time.Millisecond)
        }
    }()
    
    wg.Wait()
    
    // Final results
    fmt.Println("\n=== Final Results ===")
    counters := service.ListCounters()
    for name, value := range counters {
        fmt.Printf("%s: %d\n", name, value)
    }
    
    finalStats := service.GetStats()
    fmt.Printf("\nFinal Stats: Increments: %d, Decrements: %d\n", 
        finalStats.TotalIncrements, finalStats.TotalDecrements)
}
```

### Configuration Manager

```go
package main

import (
    "fmt"
    "sync"
    "time"

    "github.com/alextanhongpin/core/sync/lock"
)

type Config struct {
    DatabaseURL    string
    MaxConnections int
    Timeout        time.Duration
    Features       map[string]bool
    Version        string
}

type ConfigManager struct {
    config    *lock.Value[Config]
    listeners *lock.Values[string, func(Config)]
    version   *lock.Value[int]
}

func NewConfigManager(initial Config) *ConfigManager {
    return &ConfigManager{
        config:    lock.NewValue(initial),
        listeners: lock.NewValues[string, func(Config)](),
        version:   lock.NewValue(1),
    }
}

func (cm *ConfigManager) GetConfig() Config {
    return cm.config.Load()
}

func (cm *ConfigManager) UpdateConfig(updater func(Config) Config) {
    oldConfig := cm.config.Load()
    
    newConfig := cm.config.Update(func(current Config) Config {
        updated := updater(current)
        // Update version string
        cm.version.Update(func(v int) int {
            updated.Version = fmt.Sprintf("v%d", v+1)
            return v + 1
        })
        return updated
    })
    
    // Notify all listeners
    cm.notifyListeners(oldConfig, newConfig)
}

func (cm *ConfigManager) AddListener(id string, listener func(Config)) {
    cm.listeners.Store(id, listener)
}

func (cm *ConfigManager) RemoveListener(id string) {
    cm.listeners.Delete(id)
}

func (cm *ConfigManager) notifyListeners(oldConfig, newConfig Config) {
    if oldConfig == newConfig {
        return
    }
    
    cm.listeners.Range(func(id string, listener func(Config)) bool {
        go listener(newConfig) // Notify asynchronously
        return true
    })
}

func (cm *ConfigManager) SetDatabaseURL(url string) {
    cm.UpdateConfig(func(c Config) Config {
        c.DatabaseURL = url
        return c
    })
}

func (cm *ConfigManager) SetMaxConnections(max int) {
    cm.UpdateConfig(func(c Config) Config {
        c.MaxConnections = max
        return c
    })
}

func (cm *ConfigManager) SetTimeout(timeout time.Duration) {
    cm.UpdateConfig(func(c Config) Config {
        c.Timeout = timeout
        return c
    })
}

func (cm *ConfigManager) EnableFeature(feature string) {
    cm.UpdateConfig(func(c Config) Config {
        if c.Features == nil {
            c.Features = make(map[string]bool)
        }
        c.Features[feature] = true
        return c
    })
}

func (cm *ConfigManager) DisableFeature(feature string) {
    cm.UpdateConfig(func(c Config) Config {
        if c.Features == nil {
            c.Features = make(map[string]bool)
        }
        c.Features[feature] = false
        return c
    })
}

func (cm *ConfigManager) IsFeatureEnabled(feature string) bool {
    config := cm.GetConfig()
    return config.Features[feature]
}

func main() {
    // Initial configuration
    initialConfig := Config{
        DatabaseURL:    "postgres://localhost:5432/mydb",
        MaxConnections: 10,
        Timeout:        30 * time.Second,
        Features:       make(map[string]bool),
        Version:        "v1",
    }
    
    manager := NewConfigManager(initialConfig)
    
    fmt.Println("=== Configuration Manager Demo ===")
    
    // Add some listeners
    manager.AddListener("logger", func(config Config) {
        fmt.Printf("🔧 Config updated: %s\n", config.Version)
    })
    
    manager.AddListener("database", func(config Config) {
        fmt.Printf("💾 Database config: %s (max: %d)\n", 
            config.DatabaseURL, config.MaxConnections)
    })
    
    manager.AddListener("features", func(config Config) {
        fmt.Printf("🎯 Features: %v\n", config.Features)
    })
    
    // Simulate concurrent configuration updates
    var wg sync.WaitGroup
    
    // Update database settings
    wg.Add(1)
    go func() {
        defer wg.Done()
        
        for i := 0; i < 5; i++ {
            manager.SetMaxConnections(10 + i*5)
            time.Sleep(200 * time.Millisecond)
            
            newURL := fmt.Sprintf("postgres://localhost:5432/db_%d", i+1)
            manager.SetDatabaseURL(newURL)
            time.Sleep(200 * time.Millisecond)
        }
    }()
    
    // Update timeout settings
    wg.Add(1)
    go func() {
        defer wg.Done()
        
        timeouts := []time.Duration{
            30 * time.Second,
            60 * time.Second,
            45 * time.Second,
            90 * time.Second,
        }
        
        for _, timeout := range timeouts {
            manager.SetTimeout(timeout)
            time.Sleep(500 * time.Millisecond)
        }
    }()
    
    // Update feature flags
    wg.Add(1)
    go func() {
        defer wg.Done()
        
        features := []string{"feature_a", "feature_b", "feature_c"}
        
        for _, feature := range features {
            manager.EnableFeature(feature)
            time.Sleep(300 * time.Millisecond)
            
            if feature == "feature_b" {
                time.Sleep(200 * time.Millisecond)
                manager.DisableFeature(feature)
            }
        }
    }()
    
    // Monitor configuration
    wg.Add(1)
    go func() {
        defer wg.Done()
        
        for i := 0; i < 10; i++ {
            config := manager.GetConfig()
            fmt.Printf("📋 Current config - Connections: %d, Timeout: %v, Features: %d\n",
                config.MaxConnections, config.Timeout, len(config.Features))
            time.Sleep(400 * time.Millisecond)
        }
    }()
    
    wg.Wait()
    
    // Final configuration
    fmt.Println("\n=== Final Configuration ===")
    finalConfig := manager.GetConfig()
    fmt.Printf("Database URL: %s\n", finalConfig.DatabaseURL)
    fmt.Printf("Max Connections: %d\n", finalConfig.MaxConnections)
    fmt.Printf("Timeout: %v\n", finalConfig.Timeout)
    fmt.Printf("Version: %s\n", finalConfig.Version)
    fmt.Printf("Features: %v\n", finalConfig.Features)
}
```

### Cache with LRU and TTL

```go
package main

import (
    "fmt"
    "sync"
    "time"

    "github.com/alextanhongpin/core/sync/lock"
)

type CacheEntry[T any] struct {
    Value     T
    ExpiresAt time.Time
    AccessedAt time.Time
}

type LRUCache[K comparable, V any] struct {
    entries   *lock.Values[K, CacheEntry[V]]
    maxSize   *lock.Value[int]
    defaultTTL *lock.Value[time.Duration]
    stats     *lock.Value[CacheStats]
}

type CacheStats struct {
    Hits        int64
    Misses      int64
    Evictions   int64
    Expirations int64
}

func NewLRUCache[K comparable, V any](maxSize int, defaultTTL time.Duration) *LRUCache[K, V] {
    cache := &LRUCache[K, V]{
        entries:    lock.NewValues[K, CacheEntry[V]](),
        maxSize:    lock.NewValue(maxSize),
        defaultTTL: lock.NewValue(defaultTTL),
        stats:      lock.NewValue(CacheStats{}),
    }
    
    // Start cleanup goroutine
    go cache.cleanup()
    
    return cache
}

func (c *LRUCache[K, V]) Get(key K) (V, bool) {
    var zero V
    
    entry, exists := c.entries.Load(key)
    if !exists {
        c.stats.Update(func(s CacheStats) CacheStats {
            s.Misses++
            return s
        })
        return zero, false
    }
    
    // Check expiration
    if time.Now().After(entry.ExpiresAt) {
        c.entries.Delete(key)
        c.stats.Update(func(s CacheStats) CacheStats {
            s.Misses++
            s.Expirations++
            return s
        })
        return zero, false
    }
    
    // Update access time
    c.entries.Update(key, func(current CacheEntry[V]) CacheEntry[V] {
        current.AccessedAt = time.Now()
        return current
    })
    
    c.stats.Update(func(s CacheStats) CacheStats {
        s.Hits++
        return s
    })
    
    return entry.Value, true
}

func (c *LRUCache[K, V]) Set(key K, value V) {
    c.SetWithTTL(key, value, c.defaultTTL.Load())
}

func (c *LRUCache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
    now := time.Now()
    entry := CacheEntry[V]{
        Value:      value,
        ExpiresAt:  now.Add(ttl),
        AccessedAt: now,
    }
    
    c.entries.Store(key, entry)
    
    // Check if we need to evict
    maxSize := c.maxSize.Load()
    if c.entries.Len() > maxSize {
        c.evictLRU()
    }
}

func (c *LRUCache[K, V]) evictLRU() {
    var oldestKey K
    var oldestTime time.Time
    first := true
    
    c.entries.Range(func(key K, entry CacheEntry[V]) bool {
        if first || entry.AccessedAt.Before(oldestTime) {
            oldestKey = key
            oldestTime = entry.AccessedAt
            first = false
        }
        return true
    })
    
    if !first {
        c.entries.Delete(oldestKey)
        c.stats.Update(func(s CacheStats) CacheStats {
            s.Evictions++
            return s
        })
    }
}

func (c *LRUCache[K, V]) cleanup() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        now := time.Now()
        var expiredKeys []K
        
        c.entries.Range(func(key K, entry CacheEntry[V]) bool {
            if now.After(entry.ExpiresAt) {
                expiredKeys = append(expiredKeys, key)
            }
            return true
        })
        
        for _, key := range expiredKeys {
            c.entries.Delete(key)
            c.stats.Update(func(s CacheStats) CacheStats {
                s.Expirations++
                return s
            })
        }
    }
}

func (c *LRUCache[K, V]) GetStats() CacheStats {
    return c.stats.Load()
}

func (c *LRUCache[K, V]) Size() int {
    return c.entries.Len()
}

func (c *LRUCache[K, V]) Clear() {
    c.entries = lock.NewValues[K, CacheEntry[V]]()
}

func main() {
    cache := NewLRUCache[string, string](5, 2*time.Second)
    
    fmt.Println("=== LRU Cache with TTL Demo ===")
    
    // Add some values
    cache.Set("key1", "value1")
    cache.Set("key2", "value2")
    cache.Set("key3", "value3")
    
    fmt.Printf("Cache size after adding 3 items: %d\n", cache.Size())
    
    // Test retrieval
    if value, found := cache.Get("key1"); found {
        fmt.Printf("Retrieved key1: %s\n", value)
    }
    
    // Add more values to trigger eviction
    cache.Set("key4", "value4")
    cache.Set("key5", "value5")
    cache.Set("key6", "value6") // Should trigger eviction
    
    fmt.Printf("Cache size after adding more items: %d\n", cache.Size())
    
    // Test access pattern
    var wg sync.WaitGroup
    
    // Goroutine 1: Keep accessing some keys
    wg.Add(1)
    go func() {
        defer wg.Done()
        for i := 0; i < 10; i++ {
            cache.Get("key1")
            cache.Get("key2")
            time.Sleep(100 * time.Millisecond)
        }
    }()
    
    // Goroutine 2: Add new keys
    wg.Add(1)
    go func() {
        defer wg.Done()
        for i := 7; i <= 12; i++ {
            key := fmt.Sprintf("key%d", i)
            value := fmt.Sprintf("value%d", i)
            cache.Set(key, value)
            time.Sleep(150 * time.Millisecond)
        }
    }()
    
    // Goroutine 3: Random access
    wg.Add(1)
    go func() {
        defer wg.Done()
        keys := []string{"key1", "key2", "key3", "key4", "key5", "nonexistent"}
        for i := 0; i < 20; i++ {
            key := keys[i%len(keys)]
            if value, found := cache.Get(key); found {
                fmt.Printf("Found %s: %s\n", key, value)
            }
            time.Sleep(80 * time.Millisecond)
        }
    }()
    
    wg.Wait()
    
    // Wait for some TTL expiration
    fmt.Println("\nWaiting for TTL expiration...")
    time.Sleep(3 * time.Second)
    
    // Final stats
    stats := cache.GetStats()
    fmt.Printf("\n=== Final Cache Stats ===\n")
    fmt.Printf("Hits: %d\n", stats.Hits)
    fmt.Printf("Misses: %d\n", stats.Misses)
    fmt.Printf("Evictions: %d\n", stats.Evictions)
    fmt.Printf("Expirations: %d\n", stats.Expirations)
    fmt.Printf("Final size: %d\n", cache.Size())
    
    hitRate := float64(stats.Hits) / float64(stats.Hits+stats.Misses) * 100
    fmt.Printf("Hit rate: %.2f%%\n", hitRate)
}
```

## 📊 Performance Considerations

### Benchmarks

```go
func BenchmarkValue(b *testing.B) {
    value := lock.NewValue(0)
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            value.Update(func(v int) int {
                return v + 1
            })
        }
    })
}

func BenchmarkValues(b *testing.B) {
    values := lock.NewValues[string, int]()
    
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            key := fmt.Sprintf("key%d", i%100)
            values.Update(key, func(v int) int {
                return v + 1
            })
            i++
        }
    })
}
```

### Memory Usage

- Minimal overhead over standard mutex
- No memory leaks with proper usage
- Efficient for both single values and collections

## 🔧 Best Practices

### 1. Choose Appropriate Lock Type

```go
// For single values
counter := lock.NewValue(0)

// For collections
cache := lock.NewValues[string, CacheEntry]()

// For custom critical sections
var mu sync.Mutex
lock.Do(&mu, func() {
    // Custom logic
})
```

### 2. Avoid Long-Running Operations

```go
// Good: Short critical section
value.Update(func(v int) int {
    return v + 1
})

// Bad: Long operation in critical section
value.Update(func(v Data) Data {
    // Don't do this - expensive operation
    result := expensiveComputation(v)
    return result
})
```

### 3. Use Read Locks When Possible

```go
var mu sync.RWMutex

// Use read lock for read-only operations
lock.DoRead(&mu, func() {
    // Read operations only
    value := data.Load()
    processValue(value)
})

// Use write lock only when necessary
lock.DoRW(&mu, func() {
    // Write operations
    data.Store(newValue)
})
```

### 4. Handle Errors Appropriately

```go
value.Update(func(current Data) Data {
    if err := validateData(current); err != nil {
        // Log error but return current to avoid data corruption
        log.Printf("Validation failed: %v", err)
        return current
    }
    return updateData(current)
})
```

## 🧪 Testing

### Unit Tests

```go
func TestValue(t *testing.T) {
    value := lock.NewValue(10)
    
    // Test Load
    if got := value.Load(); got != 10 {
        t.Errorf("Expected 10, got %d", got)
    }
    
    // Test Store
    value.Store(20)
    if got := value.Load(); got != 20 {
        t.Errorf("Expected 20, got %d", got)
    }
    
    // Test Update
    result := value.Update(func(v int) int {
        return v * 2
    })
    if result != 40 {
        t.Errorf("Expected 40, got %d", result)
    }
}

func TestValuesConcurrency(t *testing.T) {
    values := lock.NewValues[string, int]()
    
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            key := fmt.Sprintf("key%d", i%10)
            values.Update(key, func(v int) int {
                return v + 1
            })
        }(i)
    }
    
    wg.Wait()
    
    // Verify total updates
    total := 0
    values.Range(func(key string, value int) bool {
        total += value
        return true
    })
    
    if total != 100 {
        t.Errorf("Expected total 100, got %d", total)
    }
}
```

## 🔗 Related Packages

- [`background`](../background/) - Background processing
- [`throttle`](../throttle/) - Load control
- [`singleflight`](../singleflight/) - Request deduplication

## 📄 License

This package is part of the `github.com/alextanhongpin/core/sync` module and is licensed under the MIT License.

---

**Built with ❤️ for thread-safe Go applications**
