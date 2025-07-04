# Snapshot

A Go package implementing Redis-style snapshot mechanism for periodic data persistence based on frequency and time intervals. Perfect for creating backups, checkpoints, and persistence layers for in-memory data structures.

## Features

- **Frequency-Based Snapshots**: Trigger snapshots based on the number of changes
- **Time-Based Snapshots**: Trigger snapshots based on time intervals
- **Configurable Policies**: Multiple snapshot policies with different frequencies and intervals
- **Background Processing**: Non-blocking snapshot operations
- **Event-Driven**: Callback-based snapshot execution
- **Graceful Shutdown**: Proper cleanup and final snapshot on termination

## Installation

```bash
go get github.com/alextanhongpin/core/sync/snapshot
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/alextanhongpin/core/sync/snapshot"
)

func main() {
    ctx := context.Background()
    
    // Create snapshot handler
    handler := func(ctx context.Context, evt snapshot.Event) {
        fmt.Printf("Snapshot triggered: Count=%d, Policy=%+v\n", evt.Count, evt.Policy)
        // Here you would save your data to disk, database, etc.
    }
    
    // Create snapshot manager with default policies
    snap, stop := snapshot.New(ctx, handler)
    defer stop()
    
    // Simulate data changes
    for i := 0; i < 1500; i++ {
        snap.Touch() // Notify of a change
        time.Sleep(10 * time.Millisecond)
    }
    
    // Wait for potential snapshots
    time.Sleep(2 * time.Second)
}
```

## API Reference

### Policy Configuration

```go
type Policy struct {
    Every    int           // Number of changes required to trigger snapshot
    Interval time.Duration // Time interval to wait after reaching the count
}
```

### Default Policies

The package provides Redis-like default policies:

```go
policies := snapshot.NewOptions()
// Equivalent to:
// []Policy{
//     {Every: 1000, Interval: 1 * time.Second},
//     {Every: 100, Interval: 10 * time.Second},
//     {Every: 10, Interval: 1 * time.Minute},
//     {Every: 1, Interval: 1 * time.Hour},
// }
```

### Methods

#### `New(ctx context.Context, fn func(context.Context, Event), opts ...Policy) (*Background, func())`
Creates a new snapshot manager with the given handler function and policies.

#### `Touch()`
Notifies the snapshot manager of a data change.

### Event Structure

```go
type Event struct {
    Count  int    // Number of changes since last snapshot
    Policy Policy // The policy that triggered this snapshot
}
```

## Real-World Examples

### In-Memory Database Snapshot

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/snapshot"
)

type InMemoryDB struct {
    data     map[string]interface{}
    mu       sync.RWMutex
    snapshot *snapshot.Background
    stop     func()
}

func NewInMemoryDB(ctx context.Context) *InMemoryDB {
    db := &InMemoryDB{
        data: make(map[string]interface{}),
    }
    
    // Create snapshot manager
    db.snapshot, db.stop = snapshot.New(ctx, db.saveSnapshot,
        snapshot.Policy{Every: 1000, Interval: 1 * time.Second},   // High frequency
        snapshot.Policy{Every: 100, Interval: 10 * time.Second},  // Medium frequency
        snapshot.Policy{Every: 10, Interval: 1 * time.Minute},    // Low frequency
        snapshot.Policy{Every: 1, Interval: 1 * time.Hour},       // Very low frequency
    )
    
    return db
}

func (db *InMemoryDB) Close() {
    db.stop()
}

func (db *InMemoryDB) Set(key string, value interface{}) {
    db.mu.Lock()
    db.data[key] = value
    db.mu.Unlock()
    
    // Notify snapshot manager of change
    db.snapshot.Touch()
}

func (db *InMemoryDB) Get(key string) (interface{}, bool) {
    db.mu.RLock()
    value, exists := db.data[key]
    db.mu.RUnlock()
    return value, exists
}

func (db *InMemoryDB) Delete(key string) {
    db.mu.Lock()
    delete(db.data, key)
    db.mu.Unlock()
    
    // Notify snapshot manager of change
    db.snapshot.Touch()
}

func (db *InMemoryDB) saveSnapshot(ctx context.Context, evt snapshot.Event) {
    fmt.Printf("Saving snapshot: %d changes, triggered by policy %+v\n", evt.Count, evt.Policy)
    
    // Create snapshot data
    db.mu.RLock()
    dataCopy := make(map[string]interface{})
    for k, v := range db.data {
        dataCopy[k] = v
    }
    db.mu.RUnlock()
    
    // Save to file
    filename := fmt.Sprintf("snapshot_%d.json", time.Now().Unix())
    data, err := json.MarshalIndent(dataCopy, "", "  ")
    if err != nil {
        fmt.Printf("Error marshaling snapshot: %v\n", err)
        return
    }
    
    err = os.WriteFile(filename, data, 0644)
    if err != nil {
        fmt.Printf("Error saving snapshot: %v\n", err)
        return
    }
    
    fmt.Printf("Snapshot saved to %s\n", filename)
}

func main() {
    ctx := context.Background()
    db := NewInMemoryDB(ctx)
    defer db.Close()
    
    // Simulate database operations
    for i := 0; i < 1500; i++ {
        key := fmt.Sprintf("key_%d", i)
        value := fmt.Sprintf("value_%d", i)
        db.Set(key, value)
        
        // Occasionally delete some keys
        if i%100 == 0 && i > 0 {
            deleteKey := fmt.Sprintf("key_%d", i-50)
            db.Delete(deleteKey)
        }
        
        time.Sleep(5 * time.Millisecond)
    }
    
    // Wait for final snapshots
    time.Sleep(3 * time.Second)
    
    fmt.Println("Database operations completed")
}
```

### Configuration Manager with Snapshots

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/snapshot"
)

type ConfigManager struct {
    config   map[string]string
    mu       sync.RWMutex
    snapshot *snapshot.Background
    stop     func()
    filename string
}

func NewConfigManager(ctx context.Context, filename string) *ConfigManager {
    cm := &ConfigManager{
        config:   make(map[string]string),
        filename: filename,
    }
    
    // Load existing config if it exists
    cm.loadConfig()
    
    // Create snapshot manager with conservative policies for config
    cm.snapshot, cm.stop = snapshot.New(ctx, cm.saveConfig,
        snapshot.Policy{Every: 10, Interval: 30 * time.Second},   // Save after 10 changes, wait 30s
        snapshot.Policy{Every: 1, Interval: 5 * time.Minute},     // Save after 1 change, wait 5m
    )
    
    return cm
}

func (cm *ConfigManager) Close() {
    // Force final snapshot before closing
    cm.saveConfig(context.Background(), snapshot.Event{Count: 1})
    cm.stop()
}

func (cm *ConfigManager) Set(key, value string) {
    cm.mu.Lock()
    cm.config[key] = value
    cm.mu.Unlock()
    
    fmt.Printf("Config updated: %s = %s\n", key, value)
    cm.snapshot.Touch()
}

func (cm *ConfigManager) Get(key string) (string, bool) {
    cm.mu.RLock()
    value, exists := cm.config[key]
    cm.mu.RUnlock()
    return value, exists
}

func (cm *ConfigManager) Delete(key string) {
    cm.mu.Lock()
    delete(cm.config, key)
    cm.mu.Unlock()
    
    fmt.Printf("Config deleted: %s\n", key)
    cm.snapshot.Touch()
}

func (cm *ConfigManager) loadConfig() {
    data, err := os.ReadFile(cm.filename)
    if err != nil {
        fmt.Printf("No existing config file, starting fresh\n")
        return
    }
    
    err = json.Unmarshal(data, &cm.config)
    if err != nil {
        fmt.Printf("Error loading config: %v\n", err)
        return
    }
    
    fmt.Printf("Loaded config with %d entries\n", len(cm.config))
}

func (cm *ConfigManager) saveConfig(ctx context.Context, evt snapshot.Event) {
    fmt.Printf("Saving config: %d changes, triggered by policy %+v\n", evt.Count, evt.Policy)
    
    cm.mu.RLock()
    configCopy := make(map[string]string)
    for k, v := range cm.config {
        configCopy[k] = v
    }
    cm.mu.RUnlock()
    
    data, err := json.MarshalIndent(configCopy, "", "  ")
    if err != nil {
        fmt.Printf("Error marshaling config: %v\n", err)
        return
    }
    
    err = os.WriteFile(cm.filename, data, 0644)
    if err != nil {
        fmt.Printf("Error saving config: %v\n", err)
        return
    }
    
    fmt.Printf("Config saved to %s\n", cm.filename)
}

func main() {
    ctx := context.Background()
    cm := NewConfigManager(ctx, "config.json")
    defer cm.Close()
    
    // Simulate configuration changes
    configs := []struct {
        key   string
        value string
    }{
        {"database.host", "localhost"},
        {"database.port", "5432"},
        {"database.name", "myapp"},
        {"redis.host", "localhost"},
        {"redis.port", "6379"},
        {"api.timeout", "30s"},
        {"log.level", "info"},
        {"debug.enabled", "false"},
        {"cache.ttl", "1h"},
        {"max.connections", "100"},
    }
    
    for i, cfg := range configs {
        cm.Set(cfg.key, cfg.value)
        time.Sleep(2 * time.Second)
        
        // Occasionally update existing configs
        if i%3 == 0 && i > 0 {
            cm.Set("database.host", "updated-host")
            time.Sleep(1 * time.Second)
        }
    }
    
    // Wait for potential snapshots
    time.Sleep(10 * time.Second)
    
    fmt.Println("Configuration management completed")
}
```

### Cache with Persistence

```go
package main

import (
    "context"
    "encoding/gob"
    "fmt"
    "os"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/snapshot"
)

type CacheItem struct {
    Value     interface{}
    ExpiresAt time.Time
}

type PersistentCache struct {
    items    map[string]CacheItem
    mu       sync.RWMutex
    snapshot *snapshot.Background
    stop     func()
    filename string
}

func NewPersistentCache(ctx context.Context, filename string) *PersistentCache {
    pc := &PersistentCache{
        items:    make(map[string]CacheItem),
        filename: filename,
    }
    
    // Load existing cache if it exists
    pc.loadCache()
    
    // Create snapshot manager
    pc.snapshot, pc.stop = snapshot.New(ctx, pc.saveCache,
        snapshot.Policy{Every: 50, Interval: 10 * time.Second},   // Frequent saves
        snapshot.Policy{Every: 5, Interval: 1 * time.Minute},     // Medium frequency
        snapshot.Policy{Every: 1, Interval: 10 * time.Minute},    // Low frequency
    )
    
    // Start cleanup goroutine
    go pc.cleanup(ctx)
    
    return pc
}

func (pc *PersistentCache) Close() {
    pc.stop()
    // Save final snapshot
    pc.saveCache(context.Background(), snapshot.Event{Count: 1})
}

func (pc *PersistentCache) Set(key string, value interface{}, ttl time.Duration) {
    pc.mu.Lock()
    pc.items[key] = CacheItem{
        Value:     value,
        ExpiresAt: time.Now().Add(ttl),
    }
    pc.mu.Unlock()
    
    pc.snapshot.Touch()
}

func (pc *PersistentCache) Get(key string) (interface{}, bool) {
    pc.mu.RLock()
    item, exists := pc.items[key]
    pc.mu.RUnlock()
    
    if !exists || time.Now().After(item.ExpiresAt) {
        return nil, false
    }
    
    return item.Value, true
}

func (pc *PersistentCache) Delete(key string) {
    pc.mu.Lock()
    delete(pc.items, key)
    pc.mu.Unlock()
    
    pc.snapshot.Touch()
}

func (pc *PersistentCache) cleanup(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            pc.cleanupExpired()
        }
    }
}

func (pc *PersistentCache) cleanupExpired() {
    pc.mu.Lock()
    now := time.Now()
    cleaned := 0
    for key, item := range pc.items {
        if now.After(item.ExpiresAt) {
            delete(pc.items, key)
            cleaned++
        }
    }
    pc.mu.Unlock()
    
    if cleaned > 0 {
        fmt.Printf("Cleaned up %d expired items\n", cleaned)
        pc.snapshot.Touch()
    }
}

func (pc *PersistentCache) loadCache() {
    file, err := os.Open(pc.filename)
    if err != nil {
        fmt.Printf("No existing cache file, starting fresh\n")
        return
    }
    defer file.Close()
    
    decoder := gob.NewDecoder(file)
    err = decoder.Decode(&pc.items)
    if err != nil {
        fmt.Printf("Error loading cache: %v\n", err)
        return
    }
    
    fmt.Printf("Loaded cache with %d items\n", len(pc.items))
}

func (pc *PersistentCache) saveCache(ctx context.Context, evt snapshot.Event) {
    fmt.Printf("Saving cache: %d changes, triggered by policy %+v\n", evt.Count, evt.Policy)
    
    pc.mu.RLock()
    itemsCopy := make(map[string]CacheItem)
    for k, v := range pc.items {
        itemsCopy[k] = v
    }
    pc.mu.RUnlock()
    
    file, err := os.Create(pc.filename)
    if err != nil {
        fmt.Printf("Error creating cache file: %v\n", err)
        return
    }
    defer file.Close()
    
    encoder := gob.NewEncoder(file)
    err = encoder.Encode(itemsCopy)
    if err != nil {
        fmt.Printf("Error encoding cache: %v\n", err)
        return
    }
    
    fmt.Printf("Cache saved to %s\n", pc.filename)
}

func main() {
    ctx := context.Background()
    cache := NewPersistentCache(ctx, "cache.gob")
    defer cache.Close()
    
    // Simulate cache operations
    for i := 0; i < 100; i++ {
        key := fmt.Sprintf("key_%d", i)
        value := fmt.Sprintf("value_%d", i)
        ttl := time.Duration(i%10+1) * time.Minute
        
        cache.Set(key, value, ttl)
        
        // Occasionally get and delete items
        if i%10 == 0 && i > 0 {
            getKey := fmt.Sprintf("key_%d", i-5)
            if val, exists := cache.Get(getKey); exists {
                fmt.Printf("Retrieved: %s = %s\n", getKey, val)
            }
            
            deleteKey := fmt.Sprintf("key_%d", i-10)
            cache.Delete(deleteKey)
        }
        
        time.Sleep(100 * time.Millisecond)
    }
    
    // Wait for snapshots
    time.Sleep(5 * time.Second)
    
    fmt.Println("Cache operations completed")
}
```

## Custom Snapshot Policies

You can create custom snapshot policies tailored to your needs:

```go
// High-frequency snapshots for critical data
criticalPolicies := []snapshot.Policy{
    {Every: 100, Interval: 5 * time.Second},
    {Every: 10, Interval: 30 * time.Second},
    {Every: 1, Interval: 2 * time.Minute},
}

// Low-frequency snapshots for less critical data
backgroundPolicies := []snapshot.Policy{
    {Every: 10000, Interval: 1 * time.Minute},
    {Every: 1000, Interval: 10 * time.Minute},
    {Every: 100, Interval: 1 * time.Hour},
}
```

## Error Handling

Handle errors gracefully in your snapshot function:

```go
func saveSnapshot(ctx context.Context, evt snapshot.Event) {
    defer func() {
        if r := recover(); r != nil {
            fmt.Printf("Snapshot panic recovered: %v\n", r)
        }
    }()
    
    // Your snapshot logic here
    if err := performSnapshot(ctx, evt); err != nil {
        fmt.Printf("Snapshot failed: %v\n", err)
        // Consider logging, metrics, or retry logic
    }
}
```

## Testing

```go
func TestSnapshot(t *testing.T) {
    ctx := context.Background()
    
    var capturedEvents []snapshot.Event
    handler := func(ctx context.Context, evt snapshot.Event) {
        capturedEvents = append(capturedEvents, evt)
    }
    
    // Create snapshot with test policies
    snap, stop := snapshot.New(ctx, handler,
        snapshot.Policy{Every: 5, Interval: 100 * time.Millisecond},
    )
    defer stop()
    
    // Trigger changes
    for i := 0; i < 10; i++ {
        snap.Touch()
        time.Sleep(50 * time.Millisecond)
    }
    
    // Wait for snapshot
    time.Sleep(200 * time.Millisecond)
    
    // Verify snapshot was triggered
    assert.True(t, len(capturedEvents) > 0)
    assert.True(t, capturedEvents[0].Count >= 5)
}
```

## Best Practices

1. **Choose Appropriate Policies**: Balance between data safety and performance
2. **Handle Errors**: Always handle errors in snapshot functions
3. **Atomic Operations**: Ensure snapshot data is consistent
4. **Monitor Performance**: Track snapshot frequency and duration
5. **Graceful Shutdown**: Always call stop() to ensure final snapshots
6. **File Management**: Consider rotation and cleanup of old snapshots

## Performance Considerations

- Snapshot operations should be non-blocking
- Consider using background goroutines for heavy snapshot operations
- Monitor the overhead of frequent snapshots
- Use appropriate data structures for efficient copying

## License

MIT License. See [LICENSE](../../LICENSE) for details.
