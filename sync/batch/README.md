# Batch - Batch Processing and Queuing

[![Go Reference](https://pkg.go.dev/badge/github.com/alextanhongpin/core/sync/batch.svg)](https://pkg.go.dev/github.com/alextanhongpin/core/sync/batch)

A comprehensive batch processing library that provides efficient batching, queuing, and data loading utilities for Go applications. This package helps solve the N+1 query problem and optimizes bulk operations through intelligent request batching and caching.

## ✨ Features

- **📦 Automatic Batching**: Automatically batches individual requests into bulk operations
- **⚡ Request Deduplication**: Prevents duplicate requests for the same data
- **🔄 Intelligent Caching**: Built-in caching with TTL and size limits
- **⏱️ Configurable Timing**: Flexible batch timing and size controls
- **🔒 Thread-Safe**: Concurrent-safe operations with proper synchronization
- **🎯 Generic Support**: Type-safe operations with Go generics
- **📊 Built-in Metrics**: Comprehensive metrics tracking for performance monitoring
- **🔔 Callback Support**: Configurable callbacks for cache hits, misses, and batch calls
- **🚫 Error Handling**: Comprehensive error handling and propagation
- **🧹 Cache Management**: Automatic cleanup of expired entries

## 📦 Installation

```bash
go get github.com/alextanhongpin/core/sync/batch
```

## 🚀 Quick Start

### Basic Batching

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/alextanhongpin/core/sync/batch"
)

// User represents a user entity
type User struct {
    ID   int64  `json:"id"`
    Name string `json:"name"`
}

func main() {
    ctx := context.Background()
    
    // Create a batch loader with options
    opts := batch.LoaderOptions[string, User]{
        BatchFn: func(keys []string) (map[string]User, error) {
            fmt.Printf("Batch loading users: %v\n", keys)
            
            // Simulate database query
            users := make(map[string]User)
            for i, key := range keys {
                users[key] = User{
                    ID:   int64(i + 1),
                    Name: fmt.Sprintf("User %s", key),
                }
            }
            
            return users, nil
        },
        TTL: time.Hour,
        MaxBatchSize: 100,
    }
    
    loader := batch.NewLoader(opts)
    
    // Load individual users (will be batched automatically)
    user, err := loader.Load(ctx, "user1")
    if err != nil {
        log.Fatalf("Failed to load user: %v", err)
    }
    
    fmt.Printf("Loaded user: %+v\n", user)
    
    // Load multiple users
    users, err := loader.LoadMany(ctx, []string{"user2", "user3"})
    if err != nil {
        log.Fatalf("Failed to load users: %v", err)
    }
    
    fmt.Printf("Loaded users: %+v\n", users)
    
    // Check metrics
    metrics := loader.Metrics()
    fmt.Printf("Cache hits: %d, Cache misses: %d, Batch calls: %d\n", 
        metrics.CacheHits, metrics.CacheMisses, metrics.BatchCalls)
}
```

### Queue-Based Processing

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/alextanhongpin/core/sync/batch"
)

func main() {
    ctx := context.Background()
    
    // Create a queue with batch loader
    queue := batch.NewQueue(func(keys []string) ([]User, error) {
        fmt.Printf("Processing batch: %v\n", keys)
        
        // Simulate batch processing
        users := make([]User, len(keys))
        for i, key := range keys {
            users[i] = User{
                ID:   int64(i + 1),
                Name: fmt.Sprintf("User %s", key),
            }
        }
        
        return users, nil
    })
    
    // Add keys to queue
    queue.Add("user1", "user2", "user3")
    
    // Process the queue
    if err := queue.Process(ctx); err != nil {
        log.Fatalf("Failed to process queue: %v", err)
    }
    
    // Load results
    user, err := queue.Load("user1")
    if err != nil {
        log.Fatalf("Failed to load user: %v", err)
    }
    
    fmt.Printf("Loaded user: %+v\n", user)
}
```

## 🏗️ API Reference

### Types

#### Loader

```go
type Loader[K comparable, V any] struct {
    // Contains filtered or unexported fields
}
```

A batch loader that accumulates individual requests and executes them in batches with built-in caching and metrics.

#### LoaderOptions

```go
type LoaderOptions[K comparable, V any] struct {
    BatchFn      func([]K) (map[K]V, error)
    Cache        cache[K, *Result[V]]
    TTL          time.Duration
    MaxBatchSize int
    OnBatchCall  func([]K, time.Duration, error)
    OnCacheHit   func([]K)
    OnCacheMiss  func([]K)
}
```

Configuration options for the batch loader.

#### Cache

```go
type Cache[K comparable, V any] struct {
    // Contains filtered or unexported fields
}
```

A thread-safe cache with TTL support and built-in metrics.

#### Metrics

```go
type Metrics struct {
    CacheHits   int64
    CacheMisses int64
    BatchCalls  int64
    TotalKeys   int64
    ErrorCount  int64
}
```

Metrics for tracking loader performance.

#### CacheMetrics

```go
type CacheMetrics struct {
    Gets      int64
    Sets      int64
    Hits      int64
    Misses    int64
    Evictions int64
    Size      int64
}
```

Metrics for tracking cache performance.

### Functions

#### NewLoader

```go
func NewLoader[K comparable, V any](opts LoaderOptions[K, V]) *Loader[K, V]
```

Creates a new batch loader with the specified options.

#### NewCache

```go
func NewCache[K comparable, V any]() *Cache[K, V]
```

Creates a new cache instance.

### Methods

#### Loader Methods

```go
func (l *Loader[K, V]) Load(ctx context.Context, key K) (V, error)
func (l *Loader[K, V]) LoadMany(ctx context.Context, keys []K) ([]V, error)
func (l *Loader[K, V]) LoadManyResult(ctx context.Context, keys []K) (map[K]*Result[V], error)
func (l *Loader[K, V]) Metrics() Metrics
```

#### Cache Methods

```go
func (c *Cache[K, V]) StoreMany(ctx context.Context, kv map[K]V, ttl time.Duration) error
func (c *Cache[K, V]) LoadMany(ctx context.Context, keys ...K) (map[K]V, error)
func (c *Cache[K, V]) Clear()
func (c *Cache[K, V]) CleanupExpired()
func (c *Cache[K, V]) Metrics() CacheMetrics
```
func (q *Queue[K, V]) Add(keys ...K)
func (q *Queue[K, V]) Load(key K) (V, error)
func (q *Queue[K, V]) LoadMany(keys []K) ([]V, error)
func (q *Queue[K, V]) Process(ctx context.Context) error
```

## 🌟 Real-World Examples

### Database Batch Loading

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "strings"
    "time"

    "github.com/alextanhongpin/core/sync/batch"
    _ "github.com/lib/pq"
)

type User struct {
    ID        int64     `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

type UserService struct {
    db     *sql.DB
    loader *batch.Loader[int64, User]
}

func NewUserService(db *sql.DB) *UserService {
    service := &UserService{db: db}
    
    // Create batch loader for users
    service.loader = batch.NewLoader(service.loadUsersBatch)
    
    return service
}

func (s *UserService) loadUsersBatch(userIDs []int64) ([]User, error) {
    if len(userIDs) == 0 {
        return nil, nil
    }
    
    // Build query with placeholders
    placeholders := make([]string, len(userIDs))
    args := make([]interface{}, len(userIDs))
    
    for i, id := range userIDs {
        placeholders[i] = fmt.Sprintf("$%d", i+1)
        args[i] = id
    }
    
    query := fmt.Sprintf(`
        SELECT id, name, email, created_at 
        FROM users 
        WHERE id IN (%s)
        ORDER BY id
    `, strings.Join(placeholders, ","))
    
    rows, err := s.db.Query(query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to query users: %w", err)
    }
    defer rows.Close()
    
    var users []User
    for rows.Next() {
        var user User
        if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt); err != nil {
            return nil, fmt.Errorf("failed to scan user: %w", err)
        }
        users = append(users, user)
    }
    
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("rows error: %w", err)
    }
    
    log.Printf("Loaded %d users in batch: %v", len(users), userIDs)
    return users, nil
}

func (s *UserService) GetUser(ctx context.Context, userID int64) (User, error) {
    // Add to batch and load
    s.loader.Add(userID)
    return s.loader.Load(userID)
}

func (s *UserService) GetUsers(ctx context.Context, userIDs []int64) ([]User, error) {
    // Add all IDs to batch
    s.loader.Add(userIDs...)
    
    // Load all users
    return s.loader.LoadMany(userIDs)
}

func (s *UserService) ProcessBatch(ctx context.Context) error {
    return s.loader.Process(ctx)
}

func main() {
    // Initialize database connection
    db, err := sql.Open("postgres", "user=postgres dbname=test sslmode=disable")
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()
    
    // Create user service
    userService := NewUserService(db)
    ctx := context.Background()
    
    // Example: Get individual users (will be batched)
    user1, err := userService.GetUser(ctx, 1)
    if err != nil {
        log.Printf("Error getting user 1: %v", err)
    } else {
        fmt.Printf("User 1: %+v\n", user1)
    }
    
    user2, err := userService.GetUser(ctx, 2)
    if err != nil {
        log.Printf("Error getting user 2: %v", err)
    } else {
        fmt.Printf("User 2: %+v\n", user2)
    }
    
    // Example: Get multiple users at once
    users, err := userService.GetUsers(ctx, []int64{3, 4, 5})
    if err != nil {
        log.Printf("Error getting users: %v", err)
    } else {
        fmt.Printf("Users 3-5: %+v\n", users)
    }
    
    // Process any remaining batches
    if err := userService.ProcessBatch(ctx); err != nil {
        log.Printf("Error processing batch: %v", err)
    }
}
```

### GraphQL Data Loader

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/alextanhongpin/core/sync/batch"
)

type Post struct {
    ID       int64  `json:"id"`
    Title    string `json:"title"`
    Content  string `json:"content"`
    AuthorID int64  `json:"author_id"`
}

type Comment struct {
    ID     int64  `json:"id"`
    PostID int64  `json:"post_id"`
    Text   string `json:"text"`
    Author string `json:"author"`
}

type GraphQLDataLoader struct {
    postLoader       *batch.Loader[int64, Post]
    commentLoader    *batch.Loader[int64, []Comment]
    userLoader       *batch.Loader[int64, User]
}

func NewGraphQLDataLoader() *GraphQLDataLoader {
    return &GraphQLDataLoader{
        postLoader:    batch.NewLoader(loadPostsBatch),
        commentLoader: batch.NewLoader(loadCommentsBatch),
        userLoader:    batch.NewLoader(loadUsersBatch),
    }
}

func loadPostsBatch(postIDs []int64) ([]Post, error) {
    log.Printf("Loading posts batch: %v", postIDs)
    
    // Simulate database query
    posts := make([]Post, len(postIDs))
    for i, id := range postIDs {
        posts[i] = Post{
            ID:       id,
            Title:    fmt.Sprintf("Post %d", id),
            Content:  fmt.Sprintf("Content for post %d", id),
            AuthorID: id % 5 + 1, // Simulate author assignment
        }
    }
    
    return posts, nil
}

func loadCommentsBatch(postIDs []int64) ([][]Comment, error) {
    log.Printf("Loading comments batch for posts: %v", postIDs)
    
    // Simulate database query
    results := make([][]Comment, len(postIDs))
    for i, postID := range postIDs {
        comments := make([]Comment, 2) // 2 comments per post
        for j := 0; j < 2; j++ {
            comments[j] = Comment{
                ID:     postID*10 + int64(j+1),
                PostID: postID,
                Text:   fmt.Sprintf("Comment %d on post %d", j+1, postID),
                Author: fmt.Sprintf("User %d", j+1),
            }
        }
        results[i] = comments
    }
    
    return results, nil
}

func loadUsersBatch(userIDs []int64) ([]User, error) {
    log.Printf("Loading users batch: %v", userIDs)
    
    // Simulate database query
    users := make([]User, len(userIDs))
    for i, id := range userIDs {
        users[i] = User{
            ID:   id,
            Name: fmt.Sprintf("User %d", id),
        }
    }
    
    return users, nil
}

func (dl *GraphQLDataLoader) GetPost(ctx context.Context, postID int64) (Post, error) {
    dl.postLoader.Add(postID)
    return dl.postLoader.Load(postID)
}

func (dl *GraphQLDataLoader) GetCommentsForPost(ctx context.Context, postID int64) ([]Comment, error) {
    dl.commentLoader.Add(postID)
    return dl.commentLoader.Load(postID)
}

func (dl *GraphQLDataLoader) GetUser(ctx context.Context, userID int64) (User, error) {
    dl.userLoader.Add(userID)
    return dl.userLoader.Load(userID)
}

func (dl *GraphQLDataLoader) ProcessBatches(ctx context.Context) error {
    // Process all pending batches
    if err := dl.postLoader.Process(ctx); err != nil {
        return fmt.Errorf("failed to process posts: %w", err)
    }
    
    if err := dl.commentLoader.Process(ctx); err != nil {
        return fmt.Errorf("failed to process comments: %w", err)
    }
    
    if err := dl.userLoader.Process(ctx); err != nil {
        return fmt.Errorf("failed to process users: %w", err)
    }
    
    return nil
}

// GraphQL resolver example
func (dl *GraphQLDataLoader) ResolvePostWithDetails(ctx context.Context, postID int64) (map[string]interface{}, error) {
    // Get post
    post, err := dl.GetPost(ctx, postID)
    if err != nil {
        return nil, fmt.Errorf("failed to get post: %w", err)
    }
    
    // Get post author
    author, err := dl.GetUser(ctx, post.AuthorID)
    if err != nil {
        return nil, fmt.Errorf("failed to get author: %w", err)
    }
    
    // Get post comments
    comments, err := dl.GetCommentsForPost(ctx, postID)
    if err != nil {
        return nil, fmt.Errorf("failed to get comments: %w", err)
    }
    
    // Process all batches
    if err := dl.ProcessBatches(ctx); err != nil {
        return nil, fmt.Errorf("failed to process batches: %w", err)
    }
    
    return map[string]interface{}{
        "post":     post,
        "author":   author,
        "comments": comments,
    }, nil
}

func main() {
    ctx := context.Background()
    loader := NewGraphQLDataLoader()
    
    // Simulate GraphQL query resolving multiple posts
    postIDs := []int64{1, 2, 3, 4, 5}
    
    var results []map[string]interface{}
    for _, postID := range postIDs {
        result, err := loader.ResolvePostWithDetails(ctx, postID)
        if err != nil {
            log.Printf("Error resolving post %d: %v", postID, err)
            continue
        }
        results = append(results, result)
    }
    
    fmt.Printf("Resolved %d posts with details\n", len(results))
    for i, result := range results {
        fmt.Printf("Post %d: %+v\n", i+1, result)
    }
}
```

### API Request Batching

```go
package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "time"

    "github.com/alextanhongpin/core/sync/batch"
)

type APIClient struct {
    baseURL    string
    httpClient *http.Client
    loader     *batch.Loader[string, APIResponse]
}

type APIResponse struct {
    ID   string      `json:"id"`
    Data interface{} `json:"data"`
}

type BatchRequest struct {
    IDs []string `json:"ids"`
}

type BatchResponse struct {
    Results []APIResponse `json:"results"`
}

func NewAPIClient(baseURL string) *APIClient {
    client := &APIClient{
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
    
    // Create batch loader
    client.loader = batch.NewLoader(client.loadBatch)
    
    return client
}

func (c *APIClient) loadBatch(ids []string) ([]APIResponse, error) {
    if len(ids) == 0 {
        return nil, nil
    }
    
    log.Printf("Making batch API request for IDs: %v", ids)
    
    // Create batch request
    batchReq := BatchRequest{IDs: ids}
    
    // Marshal request
    reqData, err := json.Marshal(batchReq)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }
    
    // Make HTTP request
    resp, err := c.httpClient.Post(
        c.baseURL+"/batch",
        "application/json",
        bytes.NewBuffer(reqData),
    )
    if err != nil {
        return nil, fmt.Errorf("HTTP request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
    }
    
    // Read response
    respData, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }
    
    // Unmarshal response
    var batchResp BatchResponse
    if err := json.Unmarshal(respData, &batchResp); err != nil {
        return nil, fmt.Errorf("failed to unmarshal response: %w", err)
    }
    
    return batchResp.Results, nil
}

func (c *APIClient) GetResource(ctx context.Context, id string) (APIResponse, error) {
    // Add to batch
    c.loader.Add(id)
    
    // Load from batch
    return c.loader.Load(id)
}

func (c *APIClient) GetResources(ctx context.Context, ids []string) ([]APIResponse, error) {
    // Add all IDs to batch
    c.loader.Add(ids...)
    
    // Load all resources
    return c.loader.LoadMany(ids)
}

func (c *APIClient) ProcessBatch(ctx context.Context) error {
    return c.loader.Process(ctx)
}

// Mock HTTP server for testing
func createMockServer() *http.Server {
    mux := http.NewServeMux()
    
    mux.HandleFunc("/batch", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }
        
        var req BatchRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid request", http.StatusBadRequest)
            return
        }
        
        // Simulate processing
        results := make([]APIResponse, len(req.IDs))
        for i, id := range req.IDs {
            results[i] = APIResponse{
                ID:   id,
                Data: map[string]interface{}{
                    "name":  fmt.Sprintf("Resource %s", id),
                    "value": i * 10,
                },
            }
        }
        
        resp := BatchResponse{Results: results}
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    })
    
    return &http.Server{
        Addr:    ":8080",
        Handler: mux,
    }
}

func main() {
    // Start mock server
    server := createMockServer()
    go func() {
        log.Println("Starting mock server on :8080")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Printf("Server error: %v", err)
        }
    }()
    
    // Wait for server to start
    time.Sleep(100 * time.Millisecond)
    
    // Create API client
    client := NewAPIClient("http://localhost:8080")
    ctx := context.Background()
    
    // Example: Get individual resources (will be batched)
    resource1, err := client.GetResource(ctx, "resource1")
    if err != nil {
        log.Printf("Error getting resource1: %v", err)
    } else {
        fmt.Printf("Resource 1: %+v\n", resource1)
    }
    
    resource2, err := client.GetResource(ctx, "resource2")
    if err != nil {
        log.Printf("Error getting resource2: %v", err)
    } else {
        fmt.Printf("Resource 2: %+v\n", resource2)
    }
    
    // Example: Get multiple resources at once
    resources, err := client.GetResources(ctx, []string{"resource3", "resource4", "resource5"})
    if err != nil {
        log.Printf("Error getting resources: %v", err)
    } else {
        fmt.Printf("Resources 3-5: %+v\n", resources)
    }
    
    // Process any remaining batches
    if err := client.ProcessBatch(ctx); err != nil {
        log.Printf("Error processing batch: %v", err)
    }
    
    // Shutdown server
    server.Shutdown(context.Background())
}
```

## 📊 Performance Considerations

### Batch Size Optimization

```go
// Configure optimal batch sizes
const (
    MaxBatchSize = 100    // Maximum items per batch
    BatchTimeout = 10 * time.Millisecond // Maximum wait time
)

loader := batch.NewLoader(func(keys []string) ([]User, error) {
    // Process in smaller chunks if needed
    if len(keys) > MaxBatchSize {
        return processBatchInChunks(keys, MaxBatchSize)
    }
    return processBatch(keys)
})
```

### Memory Management

```go
// Use caching to reduce memory usage
cache := batch.NewCache[string, User](1000, 5*time.Minute)

loader := batch.NewLoaderWithCache(loadUsersBatch, cache)
```

### Error Handling

```go
loader := batch.NewLoader(func(keys []string) ([]User, error) {
    results := make([]User, len(keys))
    
    for i, key := range keys {
        user, err := loadUser(key)
        if err != nil {
            // Handle individual errors
            log.Printf("Failed to load user %s: %v", key, err)
            continue
        }
        results[i] = user
    }
    
    return results, nil
})
```

### Benchmarks

The batch package includes comprehensive benchmarks to track performance:

```bash
go test -bench=. -benchmem
```

Example benchmark results:
```
BenchmarkLoader/basic_loader-11         	 4262727	       293.5 ns/op	     408 B/op	       7 allocs/op
BenchmarkLoader/batch_loader-11         	 1000000	      1034 ns/op	    1440 B/op	      14 allocs/op
BenchmarkLoader/with_callbacks-11       	 4092427	       277.2 ns/op	     408 B/op	       7 allocs/op
BenchmarkCache/cache_operations-11      	 4967122	       285.0 ns/op	     256 B/op	       2 allocs/op
BenchmarkCache/cache_metrics-11         	1000000000	         0.2170 ns/op	       0 B/op	       0 allocs/op
BenchmarkLoaderMetrics-11               	1000000000	         0.2260 ns/op	       0 B/op	       0 allocs/op
```

Key insights:
- **Basic loader**: ~294ns per operation with 408B allocated
- **Batch loader**: ~1034ns per operation with 1440B allocated (handles larger batches)
- **Cache operations**: ~285ns per operation with 256B allocated
- **Metrics access**: ~0.22ns per operation with zero allocations (atomic operations)

## 🔧 Best Practices

### 1. Batch Size Configuration

```go
// Configure appropriate batch sizes based on your use case
const (
    DatabaseBatchSize = 100  // Database queries
    APIBatchSize      = 50   // API requests
    MemoryBatchSize   = 1000 // In-memory operations
)
```

### 2. Timeout Management

```go
// Set appropriate timeouts for batch operations
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := loader.Process(ctx); err != nil {
    // Handle timeout or other errors
}
```

### 3. Error Handling

```go
// Implement comprehensive error handling
loader := batch.NewLoader(func(keys []string) ([]User, error) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Batch loading panic: %v", r)
        }
    }()
    
    return loadUsersWithRetry(keys)
})
```

### 4. Built-in Metrics and Monitoring

The batch package now includes built-in metrics tracking:

```go
// Built-in metrics are automatically tracked
opts := batch.LoaderOptions[string, User]{
    BatchFn: func(keys []string) (map[string]User, error) {
        return loadUsers(keys)
    },
    TTL: time.Hour,
    MaxBatchSize: 100,
    // Optional callbacks for monitoring
    OnBatchCall: func(keys []string, duration time.Duration, err error) {
        log.Printf("Batch call: %d keys, took %v, error: %v", len(keys), duration, err)
    },
    OnCacheHit: func(keys []string) {
        log.Printf("Cache hit for %d keys", len(keys))
    },
    OnCacheMiss: func(keys []string) {
        log.Printf("Cache miss for %d keys", len(keys))
    },
}

loader := batch.NewLoader(opts)

// Get metrics
metrics := loader.Metrics()
fmt.Printf("Cache hits: %d, Cache misses: %d, Batch calls: %d", 
    metrics.CacheHits, metrics.CacheMisses, metrics.BatchCalls)

// Cache metrics
cache := batch.NewCache[string, User]()
cacheMetrics := cache.Metrics()
fmt.Printf("Cache: %d gets, %d sets, %d hits, %d misses", 
    cacheMetrics.Gets, cacheMetrics.Sets, cacheMetrics.Hits, cacheMetrics.Misses)
```

## 🧪 Testing

### Unit Tests

```go
func TestBatchLoader(t *testing.T) {
    var batchCalls int
    var loadedKeys []string
    
    loader := batch.NewLoader(func(keys []string) ([]string, error) {
        batchCalls++
        loadedKeys = append(loadedKeys, keys...)
        
        results := make([]string, len(keys))
        for i, key := range keys {
            results[i] = fmt.Sprintf("result-%s", key)
        }
        
        return results, nil
    })
    
    // Add keys
    loader.Add("key1", "key2", "key3")
    
    // Load results
    result, err := loader.Load("key1")
    if err != nil {
        t.Errorf("Load failed: %v", err)
    }
    
    if result != "result-key1" {
        t.Errorf("Expected 'result-key1', got '%s'", result)
    }
    
    if batchCalls != 1 {
        t.Errorf("Expected 1 batch call, got %d", batchCalls)
    }
}
```

### Integration Tests

```go
func TestBatchLoaderIntegration(t *testing.T) {
    // Test with real database or API
    db := setupTestDB(t)
    defer db.Close()
    
    loader := batch.NewLoader(func(keys []int64) ([]User, error) {
        return loadUsersFromDB(db, keys)
    })
    
    // Test batch loading
    loader.Add(1, 2, 3)
    
    users, err := loader.LoadMany([]int64{1, 2, 3})
    if err != nil {
        t.Fatalf("LoadMany failed: %v", err)
    }
    
    if len(users) != 3 {
        t.Errorf("Expected 3 users, got %d", len(users))
    }
}
```

## 🔗 Related Packages

- [`dataloader`](../dataloader/) - GraphQL-style data loading
- [`singleflight`](../singleflight/) - Request deduplication
- [`background`](../background/) - Background processing
- [`promise`](../promise/) - Async operations

## 📄 License

This package is part of the `github.com/alextanhongpin/core/sync` module and is licensed under the MIT License.

---

**Built with ❤️ for efficient batch processing in Go**
