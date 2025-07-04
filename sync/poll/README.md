# Poll

A Go package for building robust polling systems with configurable concurrency, backoff strategies, and failure handling. Perfect for queue processing, message consumption, and periodic task execution.

## Features

- **Configurable Concurrency**: Set maximum number of concurrent workers
- **Backoff Strategies**: Exponential backoff when no work is available
- **Failure Handling**: Configurable failure thresholds with circuit breaker behavior
- **Event Monitoring**: Real-time monitoring of poll events and metrics
- **Graceful Shutdown**: Clean shutdown with proper resource cleanup
- **Context Support**: Full context cancellation support

## Installation

```bash
go get github.com/alextanhongpin/core/sync/poll
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/alextanhongpin/core/sync/poll"
)

func main() {
    ctx := context.Background()
    
    // Create a poller with default settings
    poller := poll.New()
    
    // Start polling
    events, stop := poller.Poll(func(ctx context.Context) error {
        // Your polling logic here
        // Return poll.EOQ when no more work is available
        // Return poll.Empty when temporarily no work
        // Return other errors for actual failures
        
        fmt.Println("Processing work...")
        time.Sleep(100 * time.Millisecond)
        
        // Simulate no work available
        return poll.Empty
    })
    
    // Monitor events
    go func() {
        for event := range events {
            fmt.Printf("Event: %+v\n", event)
        }
    }()
    
    // Run for 10 seconds
    time.Sleep(10 * time.Second)
    stop()
}
```

## API Reference

### Poll Configuration

```go
type Poll struct {
    BatchSize        int                              // Number of items to process per batch
    FailureThreshold int                              // Number of failures before backing off
    BackOff          func(idle int) time.Duration     // Backoff strategy function
    MaxConcurrency   int                              // Maximum concurrent workers
}
```

### Methods

#### `New() *Poll`
Creates a new poller with default settings:
- BatchSize: 1000
- FailureThreshold: 25
- BackOff: ExponentialBackOff
- MaxConcurrency: Number of CPU cores

#### `Poll(fn func(context.Context) error) (<-chan Event, func())`
Starts polling with the given function and returns an event channel and stop function.

### Built-in Backoff Strategies

#### `ExponentialBackOff(idle int) time.Duration`
Exponential backoff with jitter, starting at 100ms and capping at 30 seconds.

#### `LinearBackOff(idle int) time.Duration`
Linear backoff, increasing by 100ms each time up to 10 seconds.

### Event Types

```go
type Event struct {
    Type      string        // Event type (e.g., "start", "stop", "error")
    Message   string        // Human-readable message
    Timestamp time.Time     // When the event occurred
    Metadata  map[string]any // Additional event data
}
```

### Error Types

- `EOQ`: End of queue - no more work will be available
- `Empty`: Queue is temporarily empty - more work may be available later
- Other errors: Actual failures in processing

## Real-World Examples

### Message Queue Consumer

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    
    "github.com/alextanhongpin/core/sync/poll"
)

type MessageQueue struct {
    db *sql.DB
}

type Message struct {
    ID      int64
    Payload string
    Retry   int
}

func NewMessageQueue(db *sql.DB) *MessageQueue {
    return &MessageQueue{db: db}
}

func (mq *MessageQueue) ProcessMessages(ctx context.Context) error {
    // Get messages from database
    messages, err := mq.getMessages(ctx, 10)
    if err != nil {
        return err
    }
    
    if len(messages) == 0 {
        return poll.Empty // No messages available
    }
    
    // Process each message
    for _, msg := range messages {
        if err := mq.processMessage(ctx, msg); err != nil {
            // Mark message as failed and increment retry count
            if err := mq.markFailed(ctx, msg.ID); err != nil {
                return err
            }
            continue
        }
        
        // Mark message as processed
        if err := mq.markProcessed(ctx, msg.ID); err != nil {
            return err
        }
    }
    
    return nil
}

func (mq *MessageQueue) getMessages(ctx context.Context, limit int) ([]Message, error) {
    query := `
        SELECT id, payload, retry_count 
        FROM messages 
        WHERE status = 'pending' 
        AND retry_count < 5 
        ORDER BY created_at ASC 
        LIMIT ?
    `
    
    rows, err := mq.db.QueryContext(ctx, query, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var messages []Message
    for rows.Next() {
        var msg Message
        if err := rows.Scan(&msg.ID, &msg.Payload, &msg.Retry); err != nil {
            return nil, err
        }
        messages = append(messages, msg)
    }
    
    return messages, nil
}

func (mq *MessageQueue) processMessage(ctx context.Context, msg Message) error {
    // Simulate message processing
    fmt.Printf("Processing message %d: %s\n", msg.ID, msg.Payload)
    time.Sleep(100 * time.Millisecond)
    
    // Simulate occasional failures
    if msg.ID%10 == 0 {
        return fmt.Errorf("processing failed for message %d", msg.ID)
    }
    
    return nil
}

func (mq *MessageQueue) markProcessed(ctx context.Context, id int64) error {
    _, err := mq.db.ExecContext(ctx, "UPDATE messages SET status = 'processed' WHERE id = ?", id)
    return err
}

func (mq *MessageQueue) markFailed(ctx context.Context, id int64) error {
    _, err := mq.db.ExecContext(ctx, 
        "UPDATE messages SET retry_count = retry_count + 1 WHERE id = ?", id)
    return err
}

func main() {
    ctx := context.Background()
    
    // Initialize database connection
    db, err := sql.Open("sqlite3", "messages.db")
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    mq := NewMessageQueue(db)
    
    // Configure poller
    poller := &poll.Poll{
        BatchSize:        100,
        FailureThreshold: 10,
        BackOff:          poll.ExponentialBackOff,
        MaxConcurrency:   5,
    }
    
    // Start polling
    events, stop := poller.Poll(mq.ProcessMessages)
    defer stop()
    
    // Monitor events
    go func() {
        for event := range events {
            fmt.Printf("Poll Event: %s - %s\n", event.Type, event.Message)
        }
    }()
    
    // Run until interrupted
    select {
    case <-ctx.Done():
        fmt.Println("Shutting down...")
    }
}
```

### File Processing System

```go
package main

import (
    "context"
    "fmt"
    "io/fs"
    "path/filepath"
    "time"
    
    "github.com/alextanhongpin/core/sync/poll"
)

type FileProcessor struct {
    inputDir    string
    outputDir   string
    processedDir string
}

func NewFileProcessor(inputDir, outputDir, processedDir string) *FileProcessor {
    return &FileProcessor{
        inputDir:    inputDir,
        outputDir:   outputDir,
        processedDir: processedDir,
    }
}

func (fp *FileProcessor) ProcessFiles(ctx context.Context) error {
    // Find files to process
    files, err := fp.findFiles()
    if err != nil {
        return err
    }
    
    if len(files) == 0 {
        return poll.Empty // No files to process
    }
    
    // Process each file
    for _, file := range files {
        if err := fp.processFile(ctx, file); err != nil {
            fmt.Printf("Error processing file %s: %v\n", file, err)
            continue
        }
        
        // Move processed file
        if err := fp.moveToProcessed(file); err != nil {
            fmt.Printf("Error moving file %s: %v\n", file, err)
        }
    }
    
    return nil
}

func (fp *FileProcessor) findFiles() ([]string, error) {
    var files []string
    
    err := filepath.WalkDir(fp.inputDir, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        
        if !d.IsDir() && filepath.Ext(path) == ".txt" {
            files = append(files, path)
        }
        
        return nil
    })
    
    return files, err
}

func (fp *FileProcessor) processFile(ctx context.Context, filePath string) error {
    // Simulate file processing
    fmt.Printf("Processing file: %s\n", filePath)
    time.Sleep(500 * time.Millisecond)
    
    // In a real implementation, you would:
    // 1. Read the file
    // 2. Process the content
    // 3. Write to output directory
    
    return nil
}

func (fp *FileProcessor) moveToProcessed(filePath string) error {
    filename := filepath.Base(filePath)
    processedPath := filepath.Join(fp.processedDir, filename)
    
    // In a real implementation, you would move the file
    fmt.Printf("Moving %s to %s\n", filePath, processedPath)
    
    return nil
}

func main() {
    ctx := context.Background()
    
    processor := NewFileProcessor("./input", "./output", "./processed")
    
    // Configure poller for file processing
    poller := &poll.Poll{
        BatchSize:        50,
        FailureThreshold: 5,
        BackOff: func(idle int) time.Duration {
            // Custom backoff for file processing
            // Check every 5 seconds when no files are available
            return 5 * time.Second
        },
        MaxConcurrency: 2, // Limit concurrent file processing
    }
    
    // Start polling
    events, stop := poller.Poll(processor.ProcessFiles)
    defer stop()
    
    // Monitor events
    go func() {
        for event := range events {
            fmt.Printf("File Processor Event: %s - %s\n", event.Type, event.Message)
        }
    }()
    
    // Run for 1 hour
    time.Sleep(time.Hour)
}
```

### Web API Polling Client

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    
    "github.com/alextanhongpin/core/sync/poll"
)

type APIClient struct {
    client  *http.Client
    baseURL string
    lastID  int64
}

type APIResponse struct {
    Items  []Item `json:"items"`
    NextID int64  `json:"next_id"`
}

type Item struct {
    ID   int64  `json:"id"`
    Data string `json:"data"`
}

func NewAPIClient(baseURL string) *APIClient {
    return &APIClient{
        client:  &http.Client{Timeout: 30 * time.Second},
        baseURL: baseURL,
    }
}

func (c *APIClient) PollAPI(ctx context.Context) error {
    // Make API request
    url := fmt.Sprintf("%s/api/items?since=%d", c.baseURL, c.lastID)
    
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return err
    }
    
    resp, err := c.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == http.StatusNotFound {
        return poll.Empty // No new items
    }
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("API error: %d", resp.StatusCode)
    }
    
    var apiResp APIResponse
    if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
        return err
    }
    
    if len(apiResp.Items) == 0 {
        return poll.Empty // No new items
    }
    
    // Process items
    for _, item := range apiResp.Items {
        if err := c.processItem(ctx, item); err != nil {
            return err
        }
    }
    
    // Update last processed ID
    c.lastID = apiResp.NextID
    
    return nil
}

func (c *APIClient) processItem(ctx context.Context, item Item) error {
    // Process the item
    fmt.Printf("Processing item %d: %s\n", item.ID, item.Data)
    time.Sleep(100 * time.Millisecond)
    return nil
}

func main() {
    ctx := context.Background()
    
    client := NewAPIClient("https://api.example.com")
    
    // Configure poller for API polling
    poller := &poll.Poll{
        BatchSize:        1, // Process one API call at a time
        FailureThreshold: 3, // Allow 3 failures before backing off
        BackOff: func(idle int) time.Duration {
            // Poll every 10 seconds when no new items
            return 10 * time.Second
        },
        MaxConcurrency: 1, // Single API client
    }
    
    // Start polling
    events, stop := poller.Poll(client.PollAPI)
    defer stop()
    
    // Monitor events
    go func() {
        for event := range events {
            fmt.Printf("API Poll Event: %s - %s\n", event.Type, event.Message)
        }
    }()
    
    // Run until interrupted
    select {
    case <-ctx.Done():
        fmt.Println("Shutting down API poller...")
    }
}
```

## Custom Backoff Strategies

You can implement custom backoff strategies:

```go
// Custom backoff that increases delay based on consecutive failures
func CustomBackOff(idle int) time.Duration {
    if idle < 5 {
        return 1 * time.Second
    } else if idle < 10 {
        return 5 * time.Second
    } else {
        return 30 * time.Second
    }
}

poller := &poll.Poll{
    BackOff: CustomBackOff,
    // ... other settings
}
```

## Monitoring and Metrics

Monitor poll performance using the event channel:

```go
events, stop := poller.Poll(processingFunc)

// Track metrics
var (
    totalEvents    int64
    errorCount     int64
    successCount   int64
)

go func() {
    for event := range events {
        totalEvents++
        
        switch event.Type {
        case "error":
            errorCount++
            log.Printf("Poll error: %s", event.Message)
        case "success":
            successCount++
        }
        
        // Send metrics to monitoring system
        metrics.Increment("poll.events.total")
        metrics.Increment(fmt.Sprintf("poll.events.%s", event.Type))
    }
}()
```

## Testing

```go
func TestPoller(t *testing.T) {
    ctx := context.Background()
    
    callCount := 0
    pollingFunc := func(ctx context.Context) error {
        callCount++
        if callCount > 3 {
            return poll.EOQ // Stop after 3 calls
        }
        return nil
    }
    
    poller := poll.New()
    events, stop := poller.Poll(pollingFunc)
    defer stop()
    
    // Collect events
    var eventTypes []string
    for event := range events {
        eventTypes = append(eventTypes, event.Type)
    }
    
    // Verify polling behavior
    assert.Equal(t, 3, callCount)
    assert.Contains(t, eventTypes, "start")
    assert.Contains(t, eventTypes, "stop")
}
```

## Best Practices

1. **Error Handling**: Use `poll.Empty` for temporary unavailability and `poll.EOQ` for permanent completion
2. **Backoff Strategy**: Choose appropriate backoff strategies based on your use case
3. **Failure Threshold**: Set reasonable failure thresholds to avoid excessive retries
4. **Concurrency**: Tune concurrency based on your system's capacity
5. **Monitoring**: Always monitor poll events for operational insights
6. **Graceful Shutdown**: Always call the stop function to clean up resources

## License

MIT License. See [LICENSE](../../LICENSE) for details.
