# Throttle

A Go package for implementing throttling mechanisms to control the rate of operations and prevent system overload. Features backlog management, timeout handling, and graceful degradation.

## Features

- **Rate Limiting**: Control the maximum number of concurrent operations
- **Backlog Management**: Queue operations when at capacity with configurable limits
- **Timeout Handling**: Configurable timeouts for backlog operations
- **Graceful Degradation**: Reject operations when backlog is full
- **Context Support**: Proper context cancellation and timeout handling
- **Resource Protection**: Prevent system overload and resource exhaustion

## Installation

```bash
go get github.com/alextanhongpin/core/sync/throttle
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/throttle"
)

func main() {
    // Create throttler with default options
    opts := throttle.NewOptions()
    opts.Limit = 3              // Allow 3 concurrent operations
    opts.BacklogLimit = 5       // Queue up to 5 operations
    opts.BacklogTimeout = 5 * time.Second // Timeout for queued operations
    
    throttler := throttle.New(opts)
    defer throttler.Close()
    
    // Function to throttle
    work := func(id int) {
        fmt.Printf("Starting work %d\n", id)
        time.Sleep(2 * time.Second) // Simulate work
        fmt.Printf("Completed work %d\n", id)
    }
    
    // Start multiple goroutines
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            ctx := context.Background()
            err := throttler.Do(ctx, func() error {
                work(id)
                return nil
            })
            
            if err != nil {
                fmt.Printf("Work %d failed: %v\n", id, err)
            }
        }(i)
    }
    
    wg.Wait()
}
```

## API Reference

### Options

```go
type Options struct {
    Limit          int           // Maximum concurrent operations
    BacklogLimit   int           // Maximum queued operations
    BacklogTimeout time.Duration // Timeout for queued operations
}
```

### Methods

#### `New(opts *Options) *Throttler`
Creates a new throttler with the specified options.

#### `Do(ctx context.Context, fn func() error) error`
Executes the function with throttling applied.

#### `Close()`
Closes the throttler and releases resources.

### Error Types

- `ErrTimeout`: Returned when operation times out in backlog
- `ErrCapacityExceeded`: Returned when backlog is full

## Real-World Examples

### HTTP Client with Request Throttling

```go
package main

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/throttle"
)

type ThrottledHTTPClient struct {
    client    *http.Client
    throttler *throttle.Throttler
}

func NewThrottledHTTPClient(maxConcurrent int) *ThrottledHTTPClient {
    opts := throttle.NewOptions()
    opts.Limit = maxConcurrent
    opts.BacklogLimit = maxConcurrent * 2
    opts.BacklogTimeout = 30 * time.Second
    
    return &ThrottledHTTPClient{
        client:    &http.Client{Timeout: 30 * time.Second},
        throttler: throttle.New(opts),
    }
}

func (tc *ThrottledHTTPClient) Close() {
    tc.throttler.Close()
}

func (tc *ThrottledHTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
    var resp *http.Response
    var err error
    
    throttleErr := tc.throttler.Do(ctx, func() error {
        req, reqErr := http.NewRequestWithContext(ctx, "GET", url, nil)
        if reqErr != nil {
            return reqErr
        }
        
        resp, err = tc.client.Do(req)
        return err
    })
    
    if throttleErr != nil {
        return nil, throttleErr
    }
    
    return resp, err
}

func (tc *ThrottledHTTPClient) Post(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
    var resp *http.Response
    var err error
    
    throttleErr := tc.throttler.Do(ctx, func() error {
        req, reqErr := http.NewRequestWithContext(ctx, "POST", url, body)
        if reqErr != nil {
            return reqErr
        }
        
        resp, err = tc.client.Do(req)
        return err
    })
    
    if throttleErr != nil {
        return nil, throttleErr
    }
    
    return resp, err
}

func main() {
    client := NewThrottledHTTPClient(3) // Max 3 concurrent requests
    defer client.Close()
    
    urls := []string{
        "https://httpbin.org/delay/1",
        "https://httpbin.org/delay/2",
        "https://httpbin.org/delay/1",
        "https://httpbin.org/delay/3",
        "https://httpbin.org/delay/1",
        "https://httpbin.org/delay/2",
        "https://httpbin.org/delay/1",
        "https://httpbin.org/delay/1",
    }
    
    var wg sync.WaitGroup
    for i, url := range urls {
        wg.Add(1)
        go func(id int, url string) {
            defer wg.Done()
            
            fmt.Printf("Request %d: Starting %s\n", id, url)
            start := time.Now()
            
            ctx := context.Background()
            resp, err := client.Get(ctx, url)
            if err != nil {
                fmt.Printf("Request %d: Error: %v\n", id, err)
                return
            }
            defer resp.Body.Close()
            
            fmt.Printf("Request %d: Completed in %v, Status: %d\n", 
                id, time.Since(start), resp.StatusCode)
        }(i, url)
    }
    
    wg.Wait()
}
```

### Database Connection Pool Throttling

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/throttle"
)

type ThrottledDB struct {
    db        *sql.DB
    throttler *throttle.Throttler
}

func NewThrottledDB(db *sql.DB, maxConcurrent int) *ThrottledDB {
    opts := throttle.NewOptions()
    opts.Limit = maxConcurrent
    opts.BacklogLimit = maxConcurrent * 3
    opts.BacklogTimeout = 10 * time.Second
    
    return &ThrottledDB{
        db:        db,
        throttler: throttle.New(opts),
    }
}

func (tdb *ThrottledDB) Close() {
    tdb.throttler.Close()
}

func (tdb *ThrottledDB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
    var rows *sql.Rows
    var err error
    
    throttleErr := tdb.throttler.Do(ctx, func() error {
        rows, err = tdb.db.QueryContext(ctx, query, args...)
        return err
    })
    
    if throttleErr != nil {
        return nil, throttleErr
    }
    
    return rows, err
}

func (tdb *ThrottledDB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
    var result sql.Result
    var err error
    
    throttleErr := tdb.throttler.Do(ctx, func() error {
        result, err = tdb.db.ExecContext(ctx, query, args...)
        return err
    })
    
    if throttleErr != nil {
        return nil, throttleErr
    }
    
    return result, err
}

type User struct {
    ID   int
    Name string
}

func (tdb *ThrottledDB) CreateUser(ctx context.Context, name string) error {
    _, err := tdb.Exec(ctx, "INSERT INTO users (name) VALUES (?)", name)
    return err
}

func (tdb *ThrottledDB) GetUser(ctx context.Context, id int) (*User, error) {
    var user User
    
    err := tdb.throttler.Do(ctx, func() error {
        row := tdb.db.QueryRowContext(ctx, "SELECT id, name FROM users WHERE id = ?", id)
        return row.Scan(&user.ID, &user.Name)
    })
    
    if err != nil {
        return nil, err
    }
    
    return &user, nil
}

func main() {
    // Initialize database connection
    db, err := sql.Open("sqlite3", "test.db")
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    // Create throttled database wrapper
    tdb := NewThrottledDB(db, 5) // Max 5 concurrent database operations
    defer tdb.Close()
    
    // Create table
    _, err = db.Exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)")
    if err != nil {
        panic(err)
    }
    
    // Simulate high concurrent database access
    var wg sync.WaitGroup
    
    // Create users concurrently
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            ctx := context.Background()
            name := fmt.Sprintf("User_%d", id)
            
            start := time.Now()
            err := tdb.CreateUser(ctx, name)
            if err != nil {
                fmt.Printf("Create user %d failed: %v\n", id, err)
                return
            }
            
            fmt.Printf("Created user %d in %v\n", id, time.Since(start))
        }(i)
    }
    
    wg.Wait()
    
    // Read users concurrently
    for i := 1; i <= 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            ctx := context.Background()
            start := time.Now()
            
            user, err := tdb.GetUser(ctx, id)
            if err != nil {
                fmt.Printf("Get user %d failed: %v\n", id, err)
                return
            }
            
            fmt.Printf("Retrieved user %d (%s) in %v\n", id, user.Name, time.Since(start))
        }(i)
    }
    
    wg.Wait()
}
```

### File Processing with Throttling

```go
package main

import (
    "context"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/throttle"
)

type FileProcessor struct {
    throttler *throttle.Throttler
    outputDir string
}

func NewFileProcessor(maxConcurrent int, outputDir string) *FileProcessor {
    opts := throttle.NewOptions()
    opts.Limit = maxConcurrent
    opts.BacklogLimit = maxConcurrent * 2
    opts.BacklogTimeout = 2 * time.Minute
    
    return &FileProcessor{
        throttler: throttle.New(opts),
        outputDir: outputDir,
    }
}

func (fp *FileProcessor) Close() {
    fp.throttler.Close()
}

func (fp *FileProcessor) ProcessFile(ctx context.Context, inputPath string) error {
    return fp.throttler.Do(ctx, func() error {
        return fp.processFile(ctx, inputPath)
    })
}

func (fp *FileProcessor) processFile(ctx context.Context, inputPath string) error {
    // Simulate file processing
    time.Sleep(500 * time.Millisecond)
    
    // Read input file
    inputFile, err := os.Open(inputPath)
    if err != nil {
        return fmt.Errorf("failed to open input file: %w", err)
    }
    defer inputFile.Close()
    
    // Create output file
    filename := filepath.Base(inputPath)
    outputPath := filepath.Join(fp.outputDir, "processed_"+filename)
    outputFile, err := os.Create(outputPath)
    if err != nil {
        return fmt.Errorf("failed to create output file: %w", err)
    }
    defer outputFile.Close()
    
    // Copy and process content
    _, err = io.Copy(outputFile, inputFile)
    if err != nil {
        return fmt.Errorf("failed to process file: %w", err)
    }
    
    return nil
}

func main() {
    // Create processor with throttling
    processor := NewFileProcessor(3, "./output") // Max 3 concurrent file operations
    defer processor.Close()
    
    // Create output directory
    os.MkdirAll("./output", 0755)
    
    // Create sample input files
    inputFiles := []string{"file1.txt", "file2.txt", "file3.txt", "file4.txt", "file5.txt"}
    for _, filename := range inputFiles {
        content := fmt.Sprintf("Content of %s\n", filename)
        err := os.WriteFile(filename, []byte(content), 0644)
        if err != nil {
            panic(err)
        }
    }
    
    // Process files concurrently
    var wg sync.WaitGroup
    for i, filename := range inputFiles {
        wg.Add(1)
        go func(id int, filename string) {
            defer wg.Done()
            
            fmt.Printf("Processing file %d: %s\n", id, filename)
            start := time.Now()
            
            ctx := context.Background()
            err := processor.ProcessFile(ctx, filename)
            if err != nil {
                fmt.Printf("File %d processing failed: %v\n", id, err)
                return
            }
            
            fmt.Printf("File %d processed in %v\n", id, time.Since(start))
        }(i, filename)
    }
    
    wg.Wait()
    
    // Clean up input files
    for _, filename := range inputFiles {
        os.Remove(filename)
    }
    
    fmt.Println("All files processed")
}
```

### API Rate Limiting

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/throttle"
)

type APIHandler struct {
    throttler *throttle.Throttler
}

func NewAPIHandler(maxConcurrent int) *APIHandler {
    opts := throttle.NewOptions()
    opts.Limit = maxConcurrent
    opts.BacklogLimit = maxConcurrent * 5
    opts.BacklogTimeout = 10 * time.Second
    
    return &APIHandler{
        throttler: throttle.New(opts),
    }
}

func (ah *APIHandler) Close() {
    ah.throttler.Close()
}

func (ah *APIHandler) ThrottledHandler(handler http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        err := ah.throttler.Do(ctx, func() error {
            handler(w, r)
            return nil
        })
        
        if err != nil {
            switch err {
            case throttle.ErrTimeout:
                http.Error(w, "Request timeout", http.StatusRequestTimeout)
            case throttle.ErrCapacityExceeded:
                http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
            default:
                http.Error(w, "Internal server error", http.StatusInternalServerError)
            }
        }
    }
}

func expensiveHandler(w http.ResponseWriter, r *http.Request) {
    // Simulate expensive operation
    time.Sleep(2 * time.Second)
    
    fmt.Fprintf(w, "Response from expensive operation at %v", time.Now())
}

func main() {
    handler := NewAPIHandler(3) // Max 3 concurrent requests
    defer handler.Close()
    
    // Set up HTTP server
    mux := http.NewServeMux()
    mux.HandleFunc("/expensive", handler.ThrottledHandler(expensiveHandler))
    
    server := &http.Server{
        Addr:    ":8080",
        Handler: mux,
    }
    
    // Start server in background
    go func() {
        fmt.Println("Server starting on :8080")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            fmt.Printf("Server error: %v\n", err)
        }
    }()
    
    // Wait for server to start
    time.Sleep(100 * time.Millisecond)
    
    // Simulate concurrent requests
    var wg sync.WaitGroup
    client := &http.Client{Timeout: 15 * time.Second}
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            start := time.Now()
            resp, err := client.Get("http://localhost:8080/expensive")
            if err != nil {
                fmt.Printf("Request %d failed: %v\n", id, err)
                return
            }
            defer resp.Body.Close()
            
            fmt.Printf("Request %d completed in %v, Status: %d\n", 
                id, time.Since(start), resp.StatusCode)
        }(i)
    }
    
    wg.Wait()
    
    // Shutdown server
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    server.Shutdown(ctx)
}
```

## Error Handling

Handle throttling errors appropriately:

```go
err := throttler.Do(ctx, func() error {
    // Your operation here
    return nil
})

switch err {
case throttle.ErrTimeout:
    // Handle timeout - operation was queued but timed out
    log.Printf("Operation timed out in backlog")
case throttle.ErrCapacityExceeded:
    // Handle capacity exceeded - backlog is full
    log.Printf("System at capacity, rejecting request")
case context.DeadlineExceeded:
    // Handle context timeout
    log.Printf("Context deadline exceeded")
default:
    if err != nil {
        log.Printf("Operation failed: %v", err)
    }
}
```

## Testing

```go
func TestThrottler(t *testing.T) {
    opts := throttle.NewOptions()
    opts.Limit = 2
    opts.BacklogLimit = 1
    opts.BacklogTimeout = 100 * time.Millisecond
    
    throttler := throttle.New(opts)
    defer throttler.Close()
    
    var counter int32
    operation := func() error {
        atomic.AddInt32(&counter, 1)
        time.Sleep(50 * time.Millisecond)
        return nil
    }
    
    ctx := context.Background()
    var wg sync.WaitGroup
    
    // Start operations
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            err := throttler.Do(ctx, operation)
            if err != nil {
                t.Logf("Operation failed: %v", err)
            }
        }()
    }
    
    wg.Wait()
    
    // Verify throttling occurred
    finalCount := atomic.LoadInt32(&counter)
    assert.True(t, finalCount <= 3) // Only 2 concurrent + 1 backlog should succeed
}
```

## Best Practices

1. **Choose Appropriate Limits**: Balance between throughput and resource protection
2. **Handle Errors Gracefully**: Implement proper error handling for throttling scenarios
3. **Monitor Metrics**: Track throttling rates and adjust limits accordingly
4. **Use Context**: Always use context for proper cancellation
5. **Backlog Management**: Set reasonable backlog limits and timeouts
6. **Resource Cleanup**: Always call `Close()` to release resources

## Performance Considerations

- Throttling adds minimal overhead when not at capacity
- Backlog operations consume memory, so set appropriate limits
- Consider using multiple throttlers for different operation types
- Monitor queue depths and processing times

## License

MIT License. See [LICENSE](../../LICENSE) for details.
