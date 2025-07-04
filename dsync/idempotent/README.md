# Idempotent Package

A Redis-based idempotent request handler for Go that ensures requests are executed only once, even when received multiple times. This package provides distributed idempotency across multiple application instances.

## Features

- **Distributed Idempotency**: Works across multiple application instances using Redis
- **Type Safety**: Generic handler with type-safe request/response handling
- **Concurrent Safety**: Handles concurrent requests for the same key gracefully
- **Lock Extension**: Automatically extends locks for long-running operations
- **Memory Efficient**: Smart memory management with cleanup mechanisms
- **Request Validation**: Ensures request consistency using SHA-256 hashing
- **Flexible Configuration**: Customizable lock and storage TTL settings

## Installation

```bash
go get github.com/alextanhongpin/core/dsync/idempotent
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/alextanhongpin/core/dsync/idempotent"
    "github.com/redis/go-redis/v9"
)

type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

type CreateUserResponse struct {
    UserID int64  `json:"user_id"`
    Name   string `json:"name"`
    Email  string `json:"email"`
}

func main() {
    // Create Redis client
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer client.Close()

    // Define your business logic
    createUser := func(ctx context.Context, req CreateUserRequest) (*CreateUserResponse, error) {
        // Simulate user creation (database call, etc.)
        time.Sleep(100 * time.Millisecond)
        
        return &CreateUserResponse{
            UserID: 12345,
            Name:   req.Name,
            Email:  req.Email,
        }, nil
    }

    // Create idempotent handler
    handler := idempotent.NewHandler(client, createUser, nil)

    ctx := context.Background()
    req := CreateUserRequest{
        Name:  "John Doe",
        Email: "john@example.com",
    }

    // First request - will execute the function
    resp1, shared1, err := handler.Handle(ctx, "create-user-123", req)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("First request: %+v (shared: %v)\n", resp1, shared1)

    // Second request - will return cached result
    resp2, shared2, err := handler.Handle(ctx, "create-user-123", req)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Second request: %+v (shared: %v)\n", resp2, shared2)
}
```

### Advanced Configuration

```go
// Custom configuration
handler := idempotent.NewHandler(client, businessLogic, &idempotent.HandlerOptions{
    LockTTL: 30 * time.Second,  // Lock expires after 30 seconds
    KeepTTL: 24 * time.Hour,    // Results cached for 24 hours
})
```

### Using the Store Interface Directly

```go
store := idempotent.NewRedisStore(client)

fn := func(ctx context.Context, req []byte) ([]byte, error) {
    // Your business logic here
    return []byte("response"), nil
}

result, shared, err := store.Do(
    ctx, 
    "operation-key", 
    fn, 
    []byte("request-data"), 
    time.Minute,  // Lock TTL
    time.Hour,    // Keep TTL
)
```

## How It Works

1. **Request Hashing**: Each request is hashed using SHA-256 for comparison
2. **Lock Acquisition**: A distributed lock is acquired using Redis
3. **Duplicate Detection**: Checks if the same request was already processed
4. **Result Caching**: Stores the result with configurable TTL
5. **Lock Extension**: Automatically extends locks for long-running operations

## Error Handling

The package provides specific error types for different scenarios:

```go
resp, shared, err := handler.Handle(ctx, key, req)
if err != nil {
    switch {
    case errors.Is(err, idempotent.ErrRequestInFlight):
        // Another request with the same key is currently being processed
        log.Println("Request already in flight")
    case errors.Is(err, idempotent.ErrRequestMismatch):
        // Same key but different request content
        log.Println("Request mismatch for key")
    case errors.Is(err, idempotent.ErrLockConflict):
        // Lock expired or conflict occurred
        log.Println("Lock conflict")
    case errors.Is(err, idempotent.ErrEmptyKey):
        // Empty key provided
        log.Println("Key cannot be empty")
    default:
        log.Printf("Other error: %v", err)
    }
}
```

## Performance Characteristics

Based on benchmarks:
- **Throughput**: ~27k operations/second for different keys
- **Latency**: ~37µs average for new requests, ~125µs for cached results
- **Memory**: ~1.4KB per operation with 33 allocations
- **Concurrent Performance**: Handles high concurrency gracefully

## Best Practices

### 1. Choose Good Keys
Use meaningful, unique keys that identify your operations:
```go
key := fmt.Sprintf("create-user:%s", userEmail)
key := fmt.Sprintf("payment:%s", transactionID)
```

### 2. Configure TTL Appropriately
```go
opts := &idempotent.HandlerOptions{
    LockTTL: 30 * time.Second,  // Should be > expected operation time
    KeepTTL: 24 * time.Hour,    // Based on business requirements
}
```

### 3. Handle Errors Gracefully
```go
resp, shared, err := handler.Handle(ctx, key, req)
if err != nil {
    if errors.Is(err, idempotent.ErrRequestInFlight) {
        // Maybe retry after a delay
        time.Sleep(100 * time.Millisecond)
        return handler.Handle(ctx, key, req)
    }
    return nil, false, err
}
```

### 4. Monitor Performance
Use the `shared` return value to monitor cache hit rates:
```go
if shared {
    cacheHitCounter.Inc()
} else {
    cacheMissCounter.Inc()
}
```

## Examples

### HTTP API with Idempotency

```go
func createUserHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Use idempotency key from header
    key := r.Header.Get("Idempotency-Key")
    if key == "" {
        http.Error(w, "Missing Idempotency-Key header", http.StatusBadRequest)
        return
    }

    resp, shared, err := handler.Handle(r.Context(), key, req)
    if err != nil {
        if errors.Is(err, idempotent.ErrRequestInFlight) {
            http.Error(w, "Request in progress", http.StatusConflict)
            return
        }
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("X-Idempotent-Replayed", fmt.Sprintf("%t", shared))
    json.NewEncoder(w).Encode(resp)
}
```

### Batch Operations

```go
func processBatch(ctx context.Context, items []BatchItem) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(items))

    for _, item := range items {
        wg.Add(1)
        go func(item BatchItem) {
            defer wg.Done()
            
            key := fmt.Sprintf("batch-item:%s", item.ID)
            _, _, err := handler.Handle(ctx, key, item)
            if err != nil {
                errChan <- err
            }
        }(item)
    }

    wg.Wait()
    close(errChan)

    for err := range errChan {
        if err != nil {
            return err
        }
    }
    return nil
}
```

## Architecture

The package consists of several key components:

- **Handler**: Type-safe wrapper with JSON marshaling
- **Store**: Core idempotency logic with Redis operations
- **Cache**: Atomic Redis operations using compare-and-swap
- **muKey**: Memory-efficient key-based mutex with cleanup

## Thread Safety

All operations are thread-safe and designed for concurrent use:
- Redis operations are atomic using Lua scripts
- Local mutexes prevent race conditions
- Automatic cleanup prevents memory leaks

## License

MIT License - see the LICENSE file for details.
