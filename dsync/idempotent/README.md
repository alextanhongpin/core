# Idempotent Package

A Redis-backed idempotent request execution library for Go that ensures operations identified by a key are executed at most once, returning cached results on subsequent invocations.

This package provides distributed idempotency across multiple application instances using Redis, and eliminates redundant execution on the same instance via an in-memory key-level mutex.

## Features

- **Distributed Idempotency**: Coordinates across multiple nodes via Redis atomic conditional commands.
- **In-Process Single-Flight**: Prevents redundant execution on the same node using a garbage-collected per-key mutex.
- **Type Safety**: Fully generic API (`[K, V any]`) with type-safe request and response handling.
- **Lock Extension (Heartbeat)**: Automatically refreshes the lock TTL in the background for long-running operations.
- **Safe Cancellation & Panic Handling**: Safely cleans up locks on error, cancellation, or handler panics without crashing background goroutines.
- **Semantic Request Validation**: Ensures that repeated calls with the same key have matching request payloads.
- **Flexible Configuration**: Customizable lock acquisition and response retention TTLs.

## Requirements

- Go 1.24+
- Redis 8.4+ (uses `SET ... IFDEQ` and `DELEX ... IFDEQ` conditional digest operations)

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
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer client.Close()

    createUser := func(ctx context.Context, req CreateUserRequest) (*CreateUserResponse, error) {
        // Business logic (e.g. database write, payment, third-party API)
        time.Sleep(100 * time.Millisecond)

        return &CreateUserResponse{
            UserID: 12345,
            Name:   req.Name,
            Email:  req.Email,
        }, nil
    }

    idb := idempotent.NewWithRedis(client)
    handler := idb.HandlerFunc(createUser, nil)

    ctx := context.Background()
    req := CreateUserRequest{
        Name:  "John Doe",
        Email: "john@example.com",
    }

    // First request - executes the function
    resp1, shared1, err := handler.Do(ctx, "create-user-123", req)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("First request: %+v (shared: %v)\n", resp1, shared1)

    // Second request - returns the cached result without calling createUser
    resp2, shared2, err := handler.Do(ctx, "create-user-123", req)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Second request: %+v (shared: %v)\n", resp2, shared2)
}
```

### Custom Configuration

```go
handler := idb.HandlerFunc(businessLogic, &idempotent.HandlerConfig{
    LockTTL: 30 * time.Second, // Max time for in-flight execution (auto-refreshed)
    KeepTTL: 24 * time.Hour,   // How long the completed response is cached
})
```

## How It Works

1. **In-Process Coordination**: Callers on the same instance contend on an in-memory key-level mutex (`cache.Cache` backed by weak pointers and automatic GC finalizers).
2. **Lock Acquisition**: The winner attempts to atomically claim the distributed key in Redis via `SET NX GET` storing the request payload alongside a unique UUIDv7 in-flight token.
3. **Duplicate Detection**:
   - If another process arrives while the operation is in flight, Redis returns the existing uncompleted entry and the caller receives `ErrRequestInFlight`.
   - If the operation already completed, the stored payload is verified against the incoming request. If they match, the cached response is returned (`shared = true`). If payloads differ, `ErrRequestMismatch` is returned.
4. **Lock Extension**: A background goroutine refreshes the Redis lock TTL at 70% of `LockTTL` so that long-running operations do not lose their lock.
5. **Result Caching & Cleanup**:
   - On success, the in-flight token is atomically replaced with the response payload using compare-and-swap (`SET IFDEQ`) for `KeepTTL`.
   - On error or handler panic, the lock is released immediately (`DELEX IFDEQ`) so retries can proceed.

## Error Handling

The package provides standard sentinel errors for predictable error handling:

```go
resp, shared, err := handler.Do(ctx, key, req)
if err != nil {
    switch {
    case errors.Is(err, idempotent.ErrRequestInFlight):
        // Another instance is currently processing this key (e.g. return 409 Conflict)
        log.Println("Request already in flight")
    case errors.Is(err, idempotent.ErrRequestMismatch):
        // Same idempotency key was reused with a different request payload
        log.Println("Request body mismatch for key")
    case errors.Is(err, idempotent.ErrLockConflict):
        // Lock expired or was preempted
        log.Println("Lock expired or conflict occurred")
    case errors.Is(err, idempotent.ErrEmptyKey):
        // Empty key provided
        log.Println("Key cannot be empty")
    default:
        log.Printf("Execution error: %v", err)
    }
}
```

## HTTP Middleware / Handler Example

```go
func (s *Server) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
    key := r.Header.Get("Idempotency-Key")
    if key == "" {
        http.Error(w, "Missing Idempotency-Key header", http.StatusBadRequest)
        return
    }

    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    resp, shared, err := s.createUserHandler.Do(r.Context(), key, req)
    if err != nil {
        if errors.Is(err, idempotent.ErrRequestInFlight) {
            http.Error(w, "Request in progress", http.StatusConflict)
            return
        }
        if errors.Is(err, idempotent.ErrRequestMismatch) {
            http.Error(w, "Idempotency key payload mismatch", http.StatusUnprocessableEntity)
            return
        }
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("X-Idempotent-Replayed", fmt.Sprintf("%t", shared))
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}
```

## License

MIT License

Renewal loss now cancels the task context and prevents publication of a successful result. Tasks run synchronously and cleanup waits for completion. Renewal stops before completed-result replacement to avoid racing the completed entry. Cleanup is bounded and its errors are joined. A Redis lease cannot promise exactly-once external effects across expiry, failures, or data loss; tasks must use application-level idempotency/fencing where required.
