# Cache Package Examples

This directory contains real-world examples of using the cache package.

## Examples

### 1. Real-World User Service (`realworld/`)

A complete user service implementation demonstrating:
- Cache-aside pattern with automatic fallback to database
- Proper cache invalidation on updates
- Error handling and logging
- Concurrent request handling

```bash
cd realworld
# Ensure Redis is running on :6379
redis-server
# Run the example
go run main.go
```

### 2. Advanced Patterns (`patterns/`)

Advanced caching patterns including:
- Distributed locking for expensive computations
- Write-through and write-behind caching
- Refresh-ahead pattern
- Cache coordination between instances

```bash
cd patterns
# Ensure Redis is running on :6379
redis-server
# Run the example
go run main.go
```

### 3. Basic Examples

- `improved_usage.go` - Basic operations and convenience methods
- `sentinel_errors.go` - Comprehensive error handling patterns
- `main.go` - JSON cache usage with real-world patterns

## Prerequisites

All examples require a Redis server running on localhost:6379.

### Installing Redis

**macOS (Homebrew):**
```bash
brew install redis
redis-server
```

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install redis-server
sudo systemctl start redis-server
```

**Docker:**
```bash
docker run -d -p 6379:6379 redis:alpine
```

## Key Patterns Demonstrated

### Cache-Aside Pattern
```go
var user *User
loaded, err := cache.LoadOrStore(ctx, key, &user, func() (any, error) {
    return database.GetUser(ctx, userID)
}, time.Hour)
```

### Distributed Locking
```go
lockKey := "lock:" + key
err := cache.StoreOnce(ctx, lockKey, lockValue, lockTTL)
if errors.Is(err, cache.ErrExists) {
    // Another process is working, wait for result
}
defer cache.CompareAndDelete(ctx, lockKey, lockValue)
```

### Cache Invalidation
```go
func UpdateUser(ctx context.Context, user *User) error {
    if err := database.Update(ctx, user); err != nil {
        return err
    }
    
    // Invalidate related cache keys
    keys := []string{
        fmt.Sprintf("user:%d", user.ID),
        fmt.Sprintf("user:email:%s", user.Email),
    }
    cache.Delete(ctx, keys...)
    return nil
}
```

### Write-Through Caching
```go
func CreateUser(ctx context.Context, user *User) error {
    // Write to database first
    if err := database.Create(ctx, user); err != nil {
        return err
    }
    
    // Then update cache
    return cache.Store(ctx, key, user, ttl)
}
```

### Refresh-Ahead Pattern
```go
ttl, err := cache.TTL(ctx, key)
if ttl < refreshThreshold {
    // Start background refresh
    go refreshCache(ctx, key)
}
return cache.Load(ctx, key, &result)
```

## Performance Tips

1. **Use appropriate TTLs** - Balance freshness vs. performance
2. **Batch operations** when possible
3. **Handle cache failures gracefully** - Don't let cache issues break your app
4. **Monitor cache hit rates** - Optimize based on actual usage patterns
5. **Use atomic operations** - Prevent race conditions in distributed environments

## Error Handling Best Practices

Always use sentinel errors for reliable error checking:

```go
_, err := cache.Load(ctx, key)
switch {
case errors.Is(err, cache.ErrNotExist):
    // Handle cache miss
case errors.Is(err, cache.ErrValueMismatch):
    // Handle compare operation failure
case err != nil:
    // Handle other errors
}
```
