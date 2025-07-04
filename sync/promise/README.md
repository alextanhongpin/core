# Promise

A Go implementation of JavaScript-style promises with support for async operations, deferred execution, and concurrent programming patterns.

## Features

- **Promise Pattern**: JavaScript-style promises with resolve/reject semantics
- **Deferred Execution**: Create promises that can be resolved or rejected later
- **Concurrent Safe**: Thread-safe operations for concurrent access
- **Generic Support**: Full Go generics support for type safety
- **Async Operations**: Execute functions asynchronously with promise pattern
- **Chaining**: Support for promise-like patterns in Go

## Installation

```bash
go get github.com/alextanhongpin/core/sync/promise
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/alextanhongpin/core/sync/promise"
)

func main() {
    // Create a promise that resolves after some work
    p := promise.New(func() (string, error) {
        time.Sleep(100 * time.Millisecond)
        return "Hello, World!", nil
    })
    
    // Wait for the promise to resolve
    result, err := p.Await()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    fmt.Printf("Result: %s\n", result)
}
```

## API Reference

### Creating Promises

#### `New[T any](fn func() (T, error)) *Promise[T]`
Creates a new promise that executes the given function asynchronously.

#### `Deferred[T any]() *Promise[T]`
Creates a deferred promise that can be resolved or rejected manually.

#### `Resolve[T any](value T) *Promise[T]`
Creates a promise that immediately resolves with the given value.

#### `Reject[T any](err error) *Promise[T]`
Creates a promise that immediately rejects with the given error.

### Promise Methods

#### `Await() (T, error)`
Waits for the promise to complete and returns the result or error.

#### `Resolve(value T) *Promise[T]`
Resolves a deferred promise with the given value.

#### `Reject(err error) *Promise[T]`
Rejects a deferred promise with the given error.

## Real-World Examples

### Async HTTP Client

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    
    "github.com/alextanhongpin/core/sync/promise"
)

type APIResponse struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
    Data string `json:"data"`
}

type HTTPClient struct {
    client *http.Client
}

func NewHTTPClient() *HTTPClient {
    return &HTTPClient{
        client: &http.Client{Timeout: 30 * time.Second},
    }
}

func (c *HTTPClient) GetAsync(ctx context.Context, url string) *promise.Promise[APIResponse] {
    return promise.New(func() (APIResponse, error) {
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
    })
}

func (c *HTTPClient) GetMultipleAsync(ctx context.Context, urls []string) []*promise.Promise[APIResponse] {
    promises := make([]*promise.Promise[APIResponse], len(urls))
    for i, url := range urls {
        promises[i] = c.GetAsync(ctx, url)
    }
    return promises
}

func main() {
    ctx := context.Background()
    client := NewHTTPClient()
    
    // Single async request
    fmt.Println("Making single async request...")
    promise1 := client.GetAsync(ctx, "https://api.example.com/users/1")
    
    // Do other work while request is in progress
    fmt.Println("Doing other work...")
    time.Sleep(50 * time.Millisecond)
    
    // Wait for result
    result, err := promise1.Await()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("Single result: %+v\n", result)
    }
    
    // Multiple concurrent requests
    fmt.Println("Making multiple concurrent requests...")
    urls := []string{
        "https://api.example.com/users/1",
        "https://api.example.com/users/2",
        "https://api.example.com/users/3",
    }
    
    promises := client.GetMultipleAsync(ctx, urls)
    
    // Wait for all promises to complete
    fmt.Println("Waiting for all requests to complete...")
    for i, p := range promises {
        result, err := p.Await()
        if err != nil {
            fmt.Printf("Request %d error: %v\n", i+1, err)
        } else {
            fmt.Printf("Request %d result: %+v\n", i+1, result)
        }
    }
}
```

### Database Operations with Promises

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    
    "github.com/alextanhongpin/core/sync/promise"
)

type User struct {
    ID    int64
    Name  string
    Email string
}

type UserRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
    return &UserRepository{db: db}
}

func (r *UserRepository) GetByIDAsync(ctx context.Context, id int64) *promise.Promise[User] {
    return promise.New(func() (User, error) {
        var user User
        
        query := "SELECT id, name, email FROM users WHERE id = ?"
        err := r.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Name, &user.Email)
        if err != nil {
            return User{}, err
        }
        
        return user, nil
    })
}

func (r *UserRepository) CreateAsync(ctx context.Context, name, email string) *promise.Promise[User] {
    return promise.New(func() (User, error) {
        query := "INSERT INTO users (name, email) VALUES (?, ?) RETURNING id"
        
        var user User
        err := r.db.QueryRowContext(ctx, query, name, email).Scan(&user.ID)
        if err != nil {
            return User{}, err
        }
        
        user.Name = name
        user.Email = email
        
        return user, nil
    })
}

func (r *UserRepository) UpdateAsync(ctx context.Context, id int64, name, email string) *promise.Promise[User] {
    return promise.New(func() (User, error) {
        query := "UPDATE users SET name = ?, email = ? WHERE id = ?"
        
        _, err := r.db.ExecContext(ctx, query, name, email, id)
        if err != nil {
            return User{}, err
        }
        
        // Return updated user
        return User{ID: id, Name: name, Email: email}, nil
    })
}

func (r *UserRepository) GetMultipleAsync(ctx context.Context, ids []int64) []*promise.Promise[User] {
    promises := make([]*promise.Promise[User], len(ids))
    for i, id := range ids {
        promises[i] = r.GetByIDAsync(ctx, id)
    }
    return promises
}

func main() {
    ctx := context.Background()
    
    // Initialize database connection
    db, err := sql.Open("sqlite3", "users.db")
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    repo := NewUserRepository(db)
    
    // Create a new user asynchronously
    fmt.Println("Creating user...")
    createPromise := repo.CreateAsync(ctx, "John Doe", "john@example.com")
    
    // Do other work while creation is in progress
    fmt.Println("Doing other work...")
    time.Sleep(10 * time.Millisecond)
    
    // Wait for creation to complete
    newUser, err := createPromise.Await()
    if err != nil {
        fmt.Printf("Error creating user: %v\n", err)
        return
    }
    fmt.Printf("Created user: %+v\n", newUser)
    
    // Update user asynchronously
    fmt.Println("Updating user...")
    updatePromise := repo.UpdateAsync(ctx, newUser.ID, "Jane Doe", "jane@example.com")
    
    // Get user by ID asynchronously (concurrent with update)
    fmt.Println("Getting user by ID...")
    getPromise := repo.GetByIDAsync(ctx, newUser.ID)
    
    // Wait for both operations
    updatedUser, err := updatePromise.Await()
    if err != nil {
        fmt.Printf("Error updating user: %v\n", err)
    } else {
        fmt.Printf("Updated user: %+v\n", updatedUser)
    }
    
    fetchedUser, err := getPromise.Await()
    if err != nil {
        fmt.Printf("Error fetching user: %v\n", err)
    } else {
        fmt.Printf("Fetched user: %+v\n", fetchedUser)
    }
    
    // Get multiple users concurrently
    fmt.Println("Getting multiple users...")
    userIDs := []int64{1, 2, 3, 4, 5}
    userPromises := repo.GetMultipleAsync(ctx, userIDs)
    
    // Wait for all users
    for i, p := range userPromises {
        user, err := p.Await()
        if err != nil {
            fmt.Printf("Error getting user %d: %v\n", userIDs[i], err)
        } else {
            fmt.Printf("User %d: %+v\n", userIDs[i], user)
        }
    }
}
```

### File Processing with Promises

```go
package main

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
    "time"
    
    "github.com/alextanhongpin/core/sync/promise"
)

type FileProcessor struct {
    inputDir  string
    outputDir string
}

type ProcessResult struct {
    InputFile   string
    OutputFile  string
    LinesCount  int
    ProcessTime time.Duration
}

func NewFileProcessor(inputDir, outputDir string) *FileProcessor {
    return &FileProcessor{
        inputDir:  inputDir,
        outputDir: outputDir,
    }
}

func (fp *FileProcessor) ProcessFileAsync(inputFile string) *promise.Promise[ProcessResult] {
    return promise.New(func() (ProcessResult, error) {
        start := time.Now()
        
        // Read input file
        content, err := os.ReadFile(inputFile)
        if err != nil {
            return ProcessResult{}, err
        }
        
        // Process content (e.g., convert to uppercase)
        processedContent := strings.ToUpper(string(content))
        lines := strings.Split(processedContent, "\n")
        
        // Simulate processing time
        time.Sleep(100 * time.Millisecond)
        
        // Write output file
        filename := filepath.Base(inputFile)
        outputFile := filepath.Join(fp.outputDir, "processed_"+filename)
        
        err = os.WriteFile(outputFile, []byte(processedContent), 0644)
        if err != nil {
            return ProcessResult{}, err
        }
        
        return ProcessResult{
            InputFile:   inputFile,
            OutputFile:  outputFile,
            LinesCount:  len(lines),
            ProcessTime: time.Since(start),
        }, nil
    })
}

func (fp *FileProcessor) ProcessDirectoryAsync(pattern string) *promise.Promise[[]ProcessResult] {
    return promise.New(func() ([]ProcessResult, error) {
        // Find all files matching pattern
        files, err := filepath.Glob(filepath.Join(fp.inputDir, pattern))
        if err != nil {
            return nil, err
        }
        
        // Process all files concurrently
        promises := make([]*promise.Promise[ProcessResult], len(files))
        for i, file := range files {
            promises[i] = fp.ProcessFileAsync(file)
        }
        
        // Wait for all promises to complete
        results := make([]ProcessResult, len(promises))
        for i, p := range promises {
            result, err := p.Await()
            if err != nil {
                return nil, err
            }
            results[i] = result
        }
        
        return results, nil
    })
}

func (fp *FileProcessor) CopyFileAsync(src, dst string) *promise.Promise[int64] {
    return promise.New(func() (int64, error) {
        srcFile, err := os.Open(src)
        if err != nil {
            return 0, err
        }
        defer srcFile.Close()
        
        dstFile, err := os.Create(dst)
        if err != nil {
            return 0, err
        }
        defer dstFile.Close()
        
        return io.Copy(dstFile, srcFile)
    })
}

func main() {
    processor := NewFileProcessor("./input", "./output")
    
    // Process a single file
    fmt.Println("Processing single file...")
    singlePromise := processor.ProcessFileAsync("./input/example.txt")
    
    // Do other work while processing
    fmt.Println("Doing other work...")
    time.Sleep(50 * time.Millisecond)
    
    // Wait for single file processing
    result, err := singlePromise.Await()
    if err != nil {
        fmt.Printf("Error processing file: %v\n", err)
    } else {
        fmt.Printf("Single file result: %+v\n", result)
    }
    
    // Process entire directory
    fmt.Println("Processing entire directory...")
    dirPromise := processor.ProcessDirectoryAsync("*.txt")
    
    // Copy files while directory is being processed
    fmt.Println("Copying files...")
    copyPromises := []*promise.Promise[int64]{
        processor.CopyFileAsync("./input/file1.txt", "./backup/file1.txt"),
        processor.CopyFileAsync("./input/file2.txt", "./backup/file2.txt"),
    }
    
    // Wait for directory processing
    dirResults, err := dirPromise.Await()
    if err != nil {
        fmt.Printf("Error processing directory: %v\n", err)
    } else {
        fmt.Printf("Directory processing completed: %d files\n", len(dirResults))
        for _, result := range dirResults {
            fmt.Printf("  %s -> %s (%d lines, %v)\n", 
                result.InputFile, result.OutputFile, result.LinesCount, result.ProcessTime)
        }
    }
    
    // Wait for copy operations
    fmt.Println("Waiting for copy operations...")
    for i, p := range copyPromises {
        bytes, err := p.Await()
        if err != nil {
            fmt.Printf("Error copying file %d: %v\n", i+1, err)
        } else {
            fmt.Printf("Copied file %d: %d bytes\n", i+1, bytes)
        }
    }
}
```

### Deferred Promises for Event-Driven Programming

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/alextanhongpin/core/sync/promise"
)

type EventManager struct {
    events chan Event
    stop   chan struct{}
}

type Event struct {
    Type string
    Data interface{}
}

func NewEventManager() *EventManager {
    return &EventManager{
        events: make(chan Event, 100),
        stop:   make(chan struct{}),
    }
}

func (em *EventManager) Start() {
    go em.eventLoop()
}

func (em *EventManager) Stop() {
    close(em.stop)
}

func (em *EventManager) Emit(event Event) {
    select {
    case em.events <- event:
    case <-em.stop:
    }
}

func (em *EventManager) WaitForEvent(eventType string) *promise.Promise[Event] {
    p := promise.Deferred[Event]()
    
    go func() {
        for {
            select {
            case event := <-em.events:
                if event.Type == eventType {
                    p.Resolve(event)
                    return
                }
            case <-em.stop:
                p.Reject(fmt.Errorf("event manager stopped"))
                return
            }
        }
    }()
    
    return p
}

func (em *EventManager) WaitForEvents(eventTypes []string, timeout time.Duration) *promise.Promise[[]Event] {
    p := promise.Deferred[[]Event]()
    
    go func() {
        var collectedEvents []Event
        eventSet := make(map[string]bool)
        for _, eventType := range eventTypes {
            eventSet[eventType] = true
        }
        
        timer := time.NewTimer(timeout)
        defer timer.Stop()
        
        for {
            select {
            case event := <-em.events:
                if eventSet[event.Type] {
                    collectedEvents = append(collectedEvents, event)
                    delete(eventSet, event.Type)
                    
                    if len(eventSet) == 0 {
                        p.Resolve(collectedEvents)
                        return
                    }
                }
            case <-timer.C:
                p.Reject(fmt.Errorf("timeout waiting for events"))
                return
            case <-em.stop:
                p.Reject(fmt.Errorf("event manager stopped"))
                return
            }
        }
    }()
    
    return p
}

func (em *EventManager) eventLoop() {
    // This would be where you handle events
    // For demo purposes, we'll just simulate some events
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    counter := 0
    for {
        select {
        case <-ticker.C:
            counter++
            em.Emit(Event{
                Type: fmt.Sprintf("event_%d", counter%3),
                Data: counter,
            })
        case <-em.stop:
            return
        }
    }
}

func main() {
    manager := NewEventManager()
    manager.Start()
    defer manager.Stop()
    
    // Wait for a specific event
    fmt.Println("Waiting for 'event_1'...")
    eventPromise := manager.WaitForEvent("event_1")
    
    // Do other work while waiting
    fmt.Println("Doing other work...")
    time.Sleep(2 * time.Second)
    
    // Check if event arrived
    event, err := eventPromise.Await()
    if err != nil {
        fmt.Printf("Error waiting for event: %v\n", err)
    } else {
        fmt.Printf("Received event: %+v\n", event)
    }
    
    // Wait for multiple events with timeout
    fmt.Println("Waiting for multiple events...")
    eventsPromise := manager.WaitForEvents([]string{"event_0", "event_2"}, 5*time.Second)
    
    events, err := eventsPromise.Await()
    if err != nil {
        fmt.Printf("Error waiting for events: %v\n", err)
    } else {
        fmt.Printf("Received events: %+v\n", events)
    }
}
```

## Testing

```go
func TestPromise(t *testing.T) {
    // Test successful promise
    p := promise.New(func() (int, error) {
        return 42, nil
    })
    
    result, err := p.Await()
    assert.NoError(t, err)
    assert.Equal(t, 42, result)
    
    // Test failed promise
    p2 := promise.New(func() (int, error) {
        return 0, errors.New("test error")
    })
    
    result2, err2 := p2.Await()
    assert.Error(t, err2)
    assert.Equal(t, 0, result2)
    
    // Test deferred promise
    p3 := promise.Deferred[string]()
    
    go func() {
        time.Sleep(10 * time.Millisecond)
        p3.Resolve("hello")
    }()
    
    result3, err3 := p3.Await()
    assert.NoError(t, err3)
    assert.Equal(t, "hello", result3)
}
```

## Best Practices

1. **Error Handling**: Always handle errors returned by `Await()`
2. **Avoid Blocking**: Don't call `Await()` on the same goroutine that resolves the promise
3. **Resource Management**: Be mindful of goroutines created by promises
4. **Timeouts**: Consider implementing timeouts for long-running operations
5. **Context**: Use context for cancellation in long-running promise operations
6. **Memory**: Be careful with large data structures in promises to avoid memory leaks

## Performance Considerations

- Each promise creates a goroutine, so be mindful of the number of concurrent promises
- Use promise pools or worker patterns for high-throughput scenarios
- Consider using buffered channels for better performance in high-concurrency situations

## License

MIT License. See [LICENSE](../../LICENSE) for details.
