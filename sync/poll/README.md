# Poll

A Go package for building robust polling systems with configurable concurrency, backoff strategies, and failure handling. Perfect for queue processing, message consumption, and periodic task execution.

## Features

- **Configurable Concurrency**: Set maximum number of concurrent workers
- **Multiple Backoff Strategies**: Exponential, linear, constant, and custom backoff
- **Failure Handling**: Configurable failure thresholds with circuit breaker behavior
- **Event Monitoring**: Real-time monitoring of poll events and metrics
- **Graceful Shutdown**: Clean shutdown with proper resource cleanup
- **Context Support**: Full context cancellation support
- **Flexible Configuration**: Rich options for batch size, timeouts, callbacks, and more
- **Runtime Metrics**: Detailed metrics for monitoring poll performance
- **Event Callbacks**: Custom callbacks for handling poll events

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
            fmt.Printf("Event: %s\n", event)
        }
    }()
    
    // Run for 10 seconds
    time.Sleep(10 * time.Second)
    stop()
}
```

## Advanced Configuration

```go
// Create a poller with custom options
poller := poll.NewWithOptions(poll.PollOptions{
    BatchSize:        100,
    FailureThreshold: 10,
    BackOff:          poll.ExponentialBackOff,
    MaxConcurrency:   5,
    Timeout:          30 * time.Second,
    EventBuffer:      50,
    OnBatchStart: func(ctx context.Context, batchID int64) {
        fmt.Printf("Starting batch %d\n", batchID)
    },
    OnBatchEnd: func(ctx context.Context, batchID int64, metrics poll.BatchMetrics) {
        fmt.Printf("Batch %d completed: %d items processed\n", batchID, metrics.ItemsProcessed)
    },
    OnError: func(ctx context.Context, err error) {
        fmt.Printf("Poll error: %v\n", err)
    },
})

events, stop := poller.Poll(processingFunc)
```
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
type PollOptions struct {
    BatchSize        int                                                    // Number of items to process per batch (default: 1000)
    FailureThreshold int                                                    // Number of failures before backing off (default: 25)
    BackOff          func(idle int) time.Duration                          // Backoff strategy function (default: ExponentialBackOff)
    MaxConcurrency   int                                                    // Maximum concurrent workers (default: runtime.NumCPU())
    Timeout          time.Duration                                          // Timeout for individual poll operations (default: 30s)
    EventBuffer      int                                                    // Buffer size for event channel (default: 100)
    OnBatchStart     func(ctx context.Context, batchID int64)               // Called when a batch starts
    OnBatchEnd       func(ctx context.Context, batchID int64, metrics BatchMetrics) // Called when a batch ends
    OnError          func(ctx context.Context, err error)                   // Called when an error occurs
}

type BatchMetrics struct {
    BatchID         int64         // Unique batch identifier
    StartTime       time.Time     // When the batch started
    EndTime         time.Time     // When the batch ended
    Duration        time.Duration // How long the batch took
    ItemsProcessed  int           // Number of items processed in this batch
    Success         bool          // Whether the batch completed successfully
    ErrorCount      int           // Number of errors encountered
}

type PollMetrics struct {
    TotalBatches     int64         // Total number of batches processed
    TotalItems       int64         // Total number of items processed
    TotalErrors      int64         // Total number of errors encountered
    TotalDuration    time.Duration // Total time spent processing
    AverageItems     float64       // Average items per batch
    AverageLatency   time.Duration // Average processing time per batch
    ErrorRate        float64       // Error rate (errors/total)
    IsRunning        bool          // Whether the poller is currently running
    CurrentBatchID   int64         // ID of the current batch being processed
    StartTime        time.Time     // When polling started
    LastActivity     time.Time     // Last time work was processed
    ConsecutiveIdles int           // Number of consecutive idle cycles
}
```

### Methods

#### `New() *Poll`
Creates a new poller with default settings.

#### `NewWithOptions(options PollOptions) *Poll`
Creates a new poller with custom configuration.

#### `Poll(fn func(context.Context) error) (<-chan Event, func())`
Starts polling with the given function and returns an event channel and stop function.

#### `PollWithContext(ctx context.Context, fn func(context.Context) error) (<-chan Event, func())`
Starts polling with the given function and context, returns an event channel and stop function.

#### `Metrics() PollMetrics`
Returns current runtime metrics for the poller.

### Built-in Backoff Strategies

#### `ExponentialBackOff(idle int) time.Duration`
Exponential backoff with jitter, starting at 100ms and capping at 30 seconds.

#### `LinearBackOff(idle int) time.Duration`
Linear backoff, increasing by 100ms each time up to 10 seconds.

#### `ConstantBackOff(idle int) time.Duration`
Constant 1-second backoff regardless of idle count.

#### `CustomExponentialBackOff(base, max time.Duration, factor float64) func(int) time.Duration`
Creates a custom exponential backoff with configurable parameters.

### Event Types

```go
type Event struct {
    Type      string            // Event type (e.g., "start", "stop", "error", "batch_start", "batch_end")
    Message   string            // Human-readable message
    Timestamp time.Time         // When the event occurred
    Metadata  map[string]any    // Additional event data
}
```

#### Event Methods

```go
func (e Event) String() string                    // Human-readable string representation
func (e Event) IsError() bool                     // True if this is an error event
func (e Event) IsBatch() bool                     // True if this is a batch-related event
func (e Event) GetBatchID() (int64, bool)         // Get batch ID if this is a batch event
func (e Event) GetError() (error, bool)           // Get error if this is an error event
func (e Event) GetMetrics() (BatchMetrics, bool)  // Get metrics if this is a batch end event
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
    events, stop := poller.PollWithContext(ctx, mq.ProcessMessages)
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
    events, stop := poller.PollWithContext(ctx, processor.ProcessFiles)
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
    events, stop := poller.PollWithContext(ctx, client.PollAPI)
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

// Use custom exponential backoff
customBackOff := poll.CustomExponentialBackOff(
    500*time.Millisecond, // base delay
    1*time.Minute,        // max delay
    2.0,                  // factor
)

poller := poll.NewWithOptions(poll.PollOptions{
    BackOff: customBackOff,
    // ... other settings
})
```

## Metrics and Monitoring

### Real-time Metrics

```go
poller := poll.New()
events, stop := poller.Poll(processingFunc)

// Monitor metrics in real-time
go func() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            metrics := poller.Metrics()
            fmt.Printf("Poll Metrics: %+v\n", metrics)
            
            // Send to monitoring system
            sendToMonitoring(metrics)
        case <-ctx.Done():
            return
        }
    }
}()
```

### Event-based Monitoring

Monitor poll performance using the event channel:

```go
events, stop := poller.Poll(processingFunc)

// Track metrics
var (
    totalEvents    int64
    errorCount     int64
    successCount   int64
    batchCount     int64
)

go func() {
    for event := range events {
        totalEvents++
        
        switch event.Type {
        case "error":
            errorCount++
            if err, ok := event.GetError(); ok {
                log.Printf("Poll error: %v", err)
            }
        case "batch_end":
            batchCount++
            if metrics, ok := event.GetMetrics(); ok {
                successCount += int64(metrics.ItemsProcessed)
                log.Printf("Batch %d: processed %d items in %v", 
                    metrics.BatchID, metrics.ItemsProcessed, metrics.Duration)
            }
        }
        
        // Send metrics to monitoring system
        sendMetric("poll.events.total", totalEvents)
        sendMetric("poll.events.errors", errorCount)
        sendMetric("poll.events.success", successCount)
        sendMetric("poll.batches.total", batchCount)
    }
}()
```

### Callback-based Monitoring

```go
poller := poll.NewWithOptions(poll.PollOptions{
    OnBatchStart: func(ctx context.Context, batchID int64) {
        log.Printf("Starting batch %d", batchID)
        startTime := time.Now()
        ctx = context.WithValue(ctx, "batch_start_time", startTime)
    },
    OnBatchEnd: func(ctx context.Context, batchID int64, metrics poll.BatchMetrics) {
        log.Printf("Batch %d completed: %d items in %v", 
            batchID, metrics.ItemsProcessed, metrics.Duration)
        
        // Send metrics to monitoring system
        sendBatchMetrics(metrics)
    },
    OnError: func(ctx context.Context, err error) {
        log.Printf("Poll error: %v", err)
        
        // Send error to monitoring system
        sendError(err)
        
        // Optionally trigger alerts
        if isCriticalError(err) {
            triggerAlert(err)
        }
    },
})
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
    events, stop := poller.PollWithContext(ctx, pollingFunc)
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

func TestPollerWithOptions(t *testing.T) {
    ctx := context.Background()
    
    var batchStarts, batchEnds int
    poller := poll.NewWithOptions(poll.PollOptions{
        BatchSize: 10,
        OnBatchStart: func(ctx context.Context, batchID int64) {
            batchStarts++
        },
        OnBatchEnd: func(ctx context.Context, batchID int64, metrics poll.BatchMetrics) {
            batchEnds++
            assert.True(t, metrics.Success)
            assert.Equal(t, batchID, metrics.BatchID)
        },
    })
    
    callCount := 0
    pollingFunc := func(ctx context.Context) error {
        callCount++
        if callCount > 2 {
            return poll.EOQ
        }
        return nil
    }
    
    events, stop := poller.PollWithContext(ctx, pollingFunc)
    defer stop()
    
    // Wait for completion
    for range events {
        // Consume events
    }
    
    assert.Equal(t, 2, callCount)
    assert.Equal(t, 1, batchStarts)
    assert.Equal(t, 1, batchEnds)
}
```

## Best Practices

1. **Error Handling**: Use `poll.Empty` for temporary unavailability and `poll.EOQ` for permanent completion
2. **Backoff Strategy**: Choose appropriate backoff strategies based on your use case
3. **Failure Threshold**: Set reasonable failure thresholds to avoid excessive retries
4. **Concurrency**: Tune concurrency based on your system's capacity and resource constraints
5. **Monitoring**: Always monitor poll events and metrics for operational insights
6. **Graceful Shutdown**: Always call the stop function to clean up resources properly
7. **Context Management**: Use context for cancellation and timeout control
8. **Callback Usage**: Use callbacks for real-time monitoring and alerting
9. **Batch Size**: Configure batch sizes based on your workload characteristics
10. **Timeout Configuration**: Set appropriate timeouts to prevent hanging operations

### Performance Tuning

- **Batch Size**: Larger batches reduce overhead but increase memory usage
- **Concurrency**: More workers increase throughput but consume more resources
- **Backoff Strategy**: Aggressive backoff saves resources but increases latency
- **Event Buffer**: Larger buffers prevent blocking but use more memory
- **Timeout Values**: Balance between responsiveness and stability

### Error Handling Patterns

```go
func processingFunc(ctx context.Context) error {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Get work items
    items, err := getWorkItems(ctx)
    if err != nil {
        // Temporary error - log and retry
        log.Printf("Failed to get work items: %v", err)
        return err
    }
    
    if len(items) == 0 {
        // No work available - use Empty to trigger backoff
        return poll.Empty
    }
    
    // Process items
    for _, item := range items {
        if err := processItem(ctx, item); err != nil {
            // Item-specific error - log but continue
            log.Printf("Failed to process item %v: %v", item, err)
            continue
        }
    }
    
    return nil
}
```

## License

MIT License. See [LICENSE](../../LICENSE) for details.
