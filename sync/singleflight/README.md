# Singleflight

A Go package implementing the singleflight pattern to suppress duplicate function calls and share results among concurrent callers. Perfect for caching, database queries, and expensive operations that should only run once per key.

## Features

- **Duplicate Suppression**: Prevents duplicate function calls for the same key
- **Result Sharing**: Shares results among all concurrent callers
- **Generic Support**: Full Go generics support for type safety
- **Context Support**: Proper context propagation and cancellation
- **Thread Safe**: Concurrent access safe with proper synchronization
- **Shared Flag**: Indicates whether the result was computed or shared

## Installation

```bash
go get github.com/alextanhongpin/core/sync/singleflight
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/singleflight"
)

func main() {
    ctx := context.Background()
    group := singleflight.New[string]()
    
    // Function that simulates expensive work
    expensiveWork := func(ctx context.Context) (string, error) {
        fmt.Println("Doing expensive work...")
        time.Sleep(2 * time.Second)
        return "result", nil
    }
    
    // Start multiple concurrent calls with the same key
    var wg sync.WaitGroup
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            result, shared, err := group.Do(ctx, "key1", expensiveWork)
            if err != nil {
                fmt.Printf("Worker %d: Error: %v\n", id, err)
                return
            }
            
            fmt.Printf("Worker %d: Result: %s, Shared: %t\n", id, result, shared)
        }(i)
    }
    
    wg.Wait()
    // Output: Only one "Doing expensive work..." message
}
```

## API Reference

### Group

```go
type Group[T any] struct {
    // internal fields
}
```

#### `New[T any]() *Group[T]`
Creates a new singleflight group for type T.

#### `Do(ctx context.Context, key string, fn func(ctx context.Context) (T, error)) (T, bool, error)`
Executes the function for the given key. If another call with the same key is in progress, waits for its result.

Returns:
- `T`: The result of the function call
- `bool`: `true` if the result was shared from another call, `false` if computed by this call
- `error`: Any error that occurred during execution

## Real-World Examples

### Database Query Caching

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/singleflight"
)

type User struct {
    ID    int64
    Name  string
    Email string
}

type UserService struct {
    db    *sql.DB
    group *singleflight.Group[User]
}

func NewUserService(db *sql.DB) *UserService {
    return &UserService{
        db:    db,
        group: singleflight.New[User](),
    }
}

func (s *UserService) GetUser(ctx context.Context, userID int64) (User, error) {
    key := fmt.Sprintf("user:%d", userID)
    
    user, shared, err := s.group.Do(ctx, key, func(ctx context.Context) (User, error) {
        fmt.Printf("Fetching user %d from database...\n", userID)
        return s.fetchUserFromDB(ctx, userID)
    })
    
    if err != nil {
        return User{}, err
    }
    
    if shared {
        fmt.Printf("User %d result shared from concurrent call\n", userID)
    } else {
        fmt.Printf("User %d result computed by this call\n", userID)
    }
    
    return user, nil
}

func (s *UserService) fetchUserFromDB(ctx context.Context, userID int64) (User, error) {
    // Simulate database query
    time.Sleep(500 * time.Millisecond)
    
    var user User
    query := "SELECT id, name, email FROM users WHERE id = ?"
    err := s.db.QueryRowContext(ctx, query, userID).Scan(&user.ID, &user.Name, &user.Email)
    if err != nil {
        return User{}, err
    }
    
    return user, nil
}

func main() {
    ctx := context.Background()
    
    // Initialize database connection
    db, err := sql.Open("sqlite3", "users.db")
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    service := NewUserService(db)
    
    // Simulate multiple concurrent requests for the same user
    var wg sync.WaitGroup
    userID := int64(1)
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(requestID int) {
            defer wg.Done()
            
            user, err := service.GetUser(ctx, userID)
            if err != nil {
                fmt.Printf("Request %d error: %v\n", requestID, err)
                return
            }
            
            fmt.Printf("Request %d: User %+v\n", requestID, user)
        }(i)
    }
    
    wg.Wait()
    // Only one database query will be executed
}
```

### HTTP Client with Request Deduplication

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/singleflight"
)

type APIResponse struct {
    Data    string    `json:"data"`
    Message string    `json:"message"`
    Time    time.Time `json:"timestamp"`
}

type HTTPClient struct {
    client *http.Client
    group  *singleflight.Group[APIResponse]
}

func NewHTTPClient() *HTTPClient {
    return &HTTPClient{
        client: &http.Client{Timeout: 30 * time.Second},
        group:  singleflight.New[APIResponse](),
    }
}

func (c *HTTPClient) Get(ctx context.Context, url string) (APIResponse, error) {
    response, shared, err := c.group.Do(ctx, url, func(ctx context.Context) (APIResponse, error) {
        fmt.Printf("Making HTTP request to %s\n", url)
        return c.makeRequest(ctx, url)
    })
    
    if err != nil {
        return APIResponse{}, err
    }
    
    if shared {
        fmt.Printf("Response for %s shared from concurrent call\n", url)
    } else {
        fmt.Printf("Response for %s computed by this call\n", url)
    }
    
    return response, nil
}

func (c *HTTPClient) makeRequest(ctx context.Context, url string) (APIResponse, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return APIResponse{}, err
    }
    
    resp, err := c.client.Do(req)
    if err != nil {
        return APIResponse{}, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return APIResponse{}, fmt.Errorf("HTTP error: %d", resp.StatusCode)
    }
    
    var apiResp APIResponse
    if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
        return APIResponse{}, err
    }
    
    return apiResp, nil
}

func main() {
    ctx := context.Background()
    client := NewHTTPClient()
    
    // Simulate multiple concurrent requests to the same URL
    var wg sync.WaitGroup
    url := "https://api.example.com/data"
    
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(requestID int) {
            defer wg.Done()
            
            response, err := client.Get(ctx, url)
            if err != nil {
                fmt.Printf("Request %d error: %v\n", requestID, err)
                return
            }
            
            fmt.Printf("Request %d: %+v\n", requestID, response)
        }(i)
    }
    
    wg.Wait()
    // Only one HTTP request will be made
}
```

### Cache with Singleflight

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/singleflight"
)

type Cache struct {
    data   map[string]CacheItem
    mu     sync.RWMutex
    group  *singleflight.Group[string]
    loader func(ctx context.Context, key string) (string, error)
}

type CacheItem struct {
    Value     string
    ExpiresAt time.Time
}

func NewCache(loader func(ctx context.Context, key string) (string, error)) *Cache {
    return &Cache{
        data:   make(map[string]CacheItem),
        group:  singleflight.New[string](),
        loader: loader,
    }
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
    // Check cache first
    c.mu.RLock()
    item, exists := c.data[key]
    c.mu.RUnlock()
    
    if exists && time.Now().Before(item.ExpiresAt) {
        fmt.Printf("Cache hit for key: %s\n", key)
        return item.Value, nil
    }
    
    // Cache miss or expired, load with singleflight
    value, shared, err := c.group.Do(ctx, key, func(ctx context.Context) (string, error) {
        fmt.Printf("Loading data for key: %s\n", key)
        return c.loader(ctx, key)
    })
    
    if err != nil {
        return "", err
    }
    
    if shared {
        fmt.Printf("Value for key %s shared from concurrent call\n", key)
    } else {
        fmt.Printf("Value for key %s computed by this call\n", key)
        
        // Update cache
        c.mu.Lock()
        c.data[key] = CacheItem{
            Value:     value,
            ExpiresAt: time.Now().Add(5 * time.Minute),
        }
        c.mu.Unlock()
    }
    
    return value, nil
}

func (c *Cache) Delete(key string) {
    c.mu.Lock()
    delete(c.data, key)
    c.mu.Unlock()
}

func (c *Cache) Clear() {
    c.mu.Lock()
    c.data = make(map[string]CacheItem)
    c.mu.Unlock()
}

// Simulate expensive data loading
func expensiveDataLoader(ctx context.Context, key string) (string, error) {
    // Simulate network call or database query
    time.Sleep(1 * time.Second)
    return fmt.Sprintf("data_for_%s", key), nil
}

func main() {
    ctx := context.Background()
    cache := NewCache(expensiveDataLoader)
    
    // Simulate multiple concurrent requests for the same key
    var wg sync.WaitGroup
    key := "user:123"
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(requestID int) {
            defer wg.Done()
            
            value, err := cache.Get(ctx, key)
            if err != nil {
                fmt.Printf("Request %d error: %v\n", requestID, err)
                return
            }
            
            fmt.Printf("Request %d: %s\n", requestID, value)
        }(i)
    }
    
    wg.Wait()
    
    // Wait a bit then try again (should hit cache)
    time.Sleep(2 * time.Second)
    fmt.Println("\nTrying again (should hit cache):")
    
    for i := 0; i < 3; i++ {
        wg.Add(1)
        go func(requestID int) {
            defer wg.Done()
            
            value, err := cache.Get(ctx, key)
            if err != nil {
                fmt.Printf("Request %d error: %v\n", requestID, err)
                return
            }
            
            fmt.Printf("Request %d: %s\n", requestID, value)
        }(i)
    }
    
    wg.Wait()
}
```

### File System Operations

```go
package main

import (
    "context"
    "fmt"
    "os"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/singleflight"
)

type FileReader struct {
    group *singleflight.Group[string]
}

func NewFileReader() *FileReader {
    return &FileReader{
        group: singleflight.New[string](),
    }
}

func (fr *FileReader) ReadFile(ctx context.Context, filename string) (string, error) {
    content, shared, err := fr.group.Do(ctx, filename, func(ctx context.Context) (string, error) {
        fmt.Printf("Reading file: %s\n", filename)
        return fr.readFromDisk(ctx, filename)
    })
    
    if err != nil {
        return "", err
    }
    
    if shared {
        fmt.Printf("File %s content shared from concurrent call\n", filename)
    } else {
        fmt.Printf("File %s content read by this call\n", filename)
    }
    
    return content, nil
}

func (fr *FileReader) readFromDisk(ctx context.Context, filename string) (string, error) {
    // Simulate slow disk I/O
    time.Sleep(500 * time.Millisecond)
    
    data, err := os.ReadFile(filename)
    if err != nil {
        return "", err
    }
    
    return string(data), nil
}

func main() {
    ctx := context.Background()
    reader := NewFileReader()
    
    // Create a test file
    testFile := "test.txt"
    err := os.WriteFile(testFile, []byte("Hello, World!"), 0644)
    if err != nil {
        panic(err)
    }
    defer os.Remove(testFile)
    
    // Simulate multiple concurrent reads of the same file
    var wg sync.WaitGroup
    
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(readerID int) {
            defer wg.Done()
            
            content, err := reader.ReadFile(ctx, testFile)
            if err != nil {
                fmt.Printf("Reader %d error: %v\n", readerID, err)
                return
            }
            
            fmt.Printf("Reader %d: %s\n", readerID, content)
        }(i)
    }
    
    wg.Wait()
    // Only one file read will be performed
}
```

## Context Cancellation

Singleflight properly handles context cancellation:

```go
func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()
    
    group := singleflight.New[string]()
    
    result, shared, err := group.Do(ctx, "key", func(ctx context.Context) (string, error) {
        // This will be cancelled after 1 second
        time.Sleep(2 * time.Second)
        return "result", nil
    })
    
    if err != nil {
        fmt.Printf("Error: %v\n", err) // context deadline exceeded
    }
}
```

## Testing

```go
func TestSingleflight(t *testing.T) {
    group := singleflight.New[int]()
    
    var callCount int32
    fn := func(ctx context.Context) (int, error) {
        atomic.AddInt32(&callCount, 1)
        time.Sleep(100 * time.Millisecond)
        return 42, nil
    }
    
    ctx := context.Background()
    
    // Start multiple concurrent calls
    var wg sync.WaitGroup
    results := make([]int, 5)
    shared := make([]bool, 5)
    
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            result, isShared, err := group.Do(ctx, "test", fn)
            assert.NoError(t, err)
            results[i] = result
            shared[i] = isShared
        }(i)
    }
    
    wg.Wait()
    
    // Verify only one call was made
    assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
    
    // Verify all results are the same
    for i := 0; i < 5; i++ {
        assert.Equal(t, 42, results[i])
    }
    
    // Verify sharing behavior
    sharedCount := 0
    for i := 0; i < 5; i++ {
        if shared[i] {
            sharedCount++
        }
    }
    assert.Equal(t, 4, sharedCount) // 4 out of 5 should be shared
}
```

## Best Practices

1. **Use Meaningful Keys**: Use descriptive, unique keys that properly identify the operation
2. **Handle Errors**: Always check errors returned by `Do()`
3. **Context Propagation**: Always pass context for proper cancellation
4. **Avoid Long-Running Operations**: Be mindful of operations that might run for a long time
5. **Memory Management**: Consider the lifetime of your singleflight groups
6. **Monitoring**: Monitor shared vs. computed ratios for optimization insights

## Performance Considerations

- Singleflight is most effective when you have many concurrent calls for the same key
- The overhead is minimal for operations that aren't duplicated
- Consider using multiple groups for different types of operations
- Be aware of potential memory usage if you have many unique keys

## License

MIT License. See [LICENSE](../../LICENSE) for details.
