# Background - Worker Pool Management

[![Go Reference](https://pkg.go.dev/badge/github.com/alextanhongpin/core/sync/background.svg)](https://pkg.go.dev/github.com/alextanhongpin/core/sync/background)

A simple yet powerful worker pool implementation for concurrent background task processing in Go. This package provides a clean abstraction for managing goroutine pools with graceful shutdown capabilities.

## ✨ Features

- **🔄 Configurable Worker Pools**: Set custom worker count or use CPU-based defaults
- **� Advanced Configuration**: Buffer size, timeouts, error handling, and metrics
- **�🛡️ Graceful Shutdown**: Clean termination with proper resource cleanup
- **📋 Context-Aware**: Full support for context cancellation and timeouts
- **🔒 Thread-Safe**: Concurrent-safe operations with proper synchronization
- **⚡ High Performance**: Minimal overhead with efficient channel-based communication
- **🎯 Generic Support**: Type-safe worker functions with Go generics
- **📈 Built-in Metrics**: Task counts, processing rates, and worker status
- **🚨 Error Recovery**: Panic recovery and error callbacks
- **⏱️ Worker Timeouts**: Configurable per-task timeout limits
- **🔄 Non-blocking Operations**: TrySend for non-blocking task submission

## 📦 Installation

```bash
go get github.com/alextanhongpin/core/sync/background
```

## 🚀 Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/alextanhongpin/core/sync/background"
)

type Task struct {
    ID   string
    Data string
}

func main() {
    ctx := context.Background()
    
    // Create worker pool with 4 workers
    worker, stop := background.New(ctx, 4, func(ctx context.Context, task Task) {
        log.Printf("Processing task %s: %s", task.ID, task.Data)
        time.Sleep(100 * time.Millisecond) // Simulate work
    })
    defer stop()
    
    // Send tasks to worker pool
    for i := 0; i < 10; i++ {
        task := Task{
            ID:   fmt.Sprintf("task-%d", i),
            Data: fmt.Sprintf("data-%d", i),
        }
        
        if err := worker.Send(task); err != nil {
            log.Printf("Failed to send task: %v", err)
        }
    }
    
    // Let workers process tasks
    time.Sleep(2 * time.Second)
}
```

### Advanced Configuration

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/alextanhongpin/core/sync/background"
)

func main() {
    ctx := context.Background()
    
    // Configure advanced options
    opts := background.Options{
        WorkerCount:   8,                    // 8 workers
        BufferSize:    100,                  // Buffered channel
        WorkerTimeout: 30 * time.Second,     // Max task duration
        OnError: func(task interface{}, recovered interface{}) {
            log.Printf("Task panic: %v (task: %v)", recovered, task)
        },
        OnTaskComplete: func(task interface{}, duration time.Duration) {
            fmt.Printf("Task completed in %v\n", duration)
        },
    }
    
    worker, stop := background.NewWithOptions(ctx, opts, func(ctx context.Context, task string) {
        // Process task with potential timeout
        select {
        case <-time.After(1 * time.Second):
            fmt.Printf("Processed: %s\n", task)
        case <-ctx.Done():
            fmt.Printf("Task cancelled: %s\n", task)
        }
    })
    defer stop()
    
    // Send tasks
    tasks := []string{"task1", "task2", "task3"}
    for _, task := range tasks {
        if !worker.TrySend(task) {
            log.Printf("Failed to send task: %s", task)
        }
    }
    
    // Monitor metrics
    time.Sleep(100 * time.Millisecond)
    metrics := worker.Metrics()
    fmt.Printf("Metrics: Queued=%d, Processed=%d, Active=%d\n",
        metrics.TasksQueued, metrics.TasksProcessed, metrics.ActiveWorkers)
    
    time.Sleep(2 * time.Second)
}
```

### Auto-Configured Worker Pool

```go
// Use CPU count as worker count (default behavior when n <= 0)
worker, stop := background.New(ctx, 0, processTask)
defer stop()

// Or explicitly use CPU count
worker, stop := background.New(ctx, runtime.GOMAXPROCS(0), processTask)
defer stop()
```

## 🏗️ API Reference

### Types

```go
type Worker[T any] struct {
    // Contains filtered or unexported fields
}

type Options struct {
    WorkerCount    int                                           // Number of workers
    BufferSize     int                                           // Channel buffer size
    WorkerTimeout  time.Duration                                 // Per-task timeout
    OnError        func(task interface{}, recovered interface{}) // Panic handler
    OnTaskComplete func(task interface{}, duration time.Duration) // Completion callback
}

type Metrics struct {
    TasksQueued    int64 // Total tasks queued
    TasksProcessed int64 // Total tasks processed
    TasksRejected  int64 // Total tasks rejected
    ActiveWorkers  int64 // Current active workers
}
```

### Functions

#### New

```go
func New[T any](ctx context.Context, n int, fn func(context.Context, T)) (*Worker[T], func())
```

Creates a new worker pool with `n` workers. If `n <= 0`, uses `runtime.GOMAXPROCS(0)`.

#### NewWithOptions

```go
func NewWithOptions[T any](ctx context.Context, opts Options, fn func(context.Context, T)) (*Worker[T], func())
```

Creates a new worker pool with advanced configuration options.

### Methods

#### Send

```go
func (w *Worker[T]) Send(vs ...T) error
```

Sends one or more tasks to the worker pool. Blocks if the channel is full.

#### TrySend

```go
func (w *Worker[T]) TrySend(v T) bool
```

Attempts to send a task without blocking. Returns `true` if successful, `false` if the channel is full.

#### Metrics

```go
func (w *Worker[T]) Metrics() Metrics
```

Returns current metrics about the worker pool performance.

**Parameters:**
- `ctx`: Context for the worker pool lifecycle
- `n`: Number of workers (0 for CPU count)
- `fn`: Function to process each task

**Returns:**
- `*Worker[T]`: The worker pool instance
- `func()`: Cleanup function to stop the worker pool

#### Send

```go
func (w *Worker[T]) Send(vs ...T) error
```

Sends one or more tasks to the worker pool for processing.

**Parameters:**
- `vs`: Tasks to send to the worker pool

**Returns:**
- `error`: `ErrTerminated` if the worker pool has been stopped

## 🔧 Configuration Patterns

### High-Throughput Processing

```go
// Use more workers for I/O-bound tasks
worker, stop := background.New(ctx, runtime.GOMAXPROCS(0)*2, func(ctx context.Context, task IOTask) {
    // I/O-bound operation
    result, err := httpClient.Get(task.URL)
    if err != nil {
        log.Printf("HTTP request failed: %v", err)
        return
    }
    defer result.Body.Close()
    
    // Process response
    processResponse(result)
})
defer stop()
```

### CPU-Intensive Tasks

```go
// Use CPU count for CPU-bound tasks
worker, stop := background.New(ctx, runtime.GOMAXPROCS(0), func(ctx context.Context, task CPUTask) {
    // CPU-intensive operation
    result := performComplexCalculation(task.Data)
    
    // Store result
    saveResult(task.ID, result)
})
defer stop()
```

### Error Handling

```go
type TaskWithCallback struct {
    ID       string
    Data     interface{}
    Callback func(error)
}

worker, stop := background.New(ctx, 4, func(ctx context.Context, task TaskWithCallback) {
    var err error
    defer func() {
        if task.Callback != nil {
            task.Callback(err)
        }
    }()
    
    // Process task
    err = processTask(task.Data)
    if err != nil {
        log.Printf("Task %s failed: %v", task.ID, err)
    }
})
defer stop()
```

## 🌟 Real-World Examples

### Image Processing Service

```go
package main

import (
    "context"
    "fmt"
    "image"
    "image/jpeg"
    "log"
    "os"
    "path/filepath"
    "runtime"
    "sync"
    "time"

    "github.com/alextanhongpin/core/sync/background"
)

type ImageTask struct {
    InputPath  string
    OutputPath string
    Quality    int
}

type ImageProcessor struct {
    worker *background.Worker[ImageTask]
    stop   func()
    stats  struct {
        processed int
        failed    int
        mu        sync.RWMutex
    }
}

func NewImageProcessor(ctx context.Context) *ImageProcessor {
    p := &ImageProcessor{}
    
    // Use CPU count for CPU-intensive image processing
    p.worker, p.stop = background.New(ctx, runtime.GOMAXPROCS(0), p.processImage)
    
    return p
}

func (p *ImageProcessor) processImage(ctx context.Context, task ImageTask) {
    start := time.Now()
    
    // Open input image
    inputFile, err := os.Open(task.InputPath)
    if err != nil {
        log.Printf("Failed to open input image %s: %v", task.InputPath, err)
        p.incrementFailed()
        return
    }
    defer inputFile.Close()
    
    // Decode image
    img, _, err := image.Decode(inputFile)
    if err != nil {
        log.Printf("Failed to decode image %s: %v", task.InputPath, err)
        p.incrementFailed()
        return
    }
    
    // Create output directory
    outputDir := filepath.Dir(task.OutputPath)
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        log.Printf("Failed to create output directory %s: %v", outputDir, err)
        p.incrementFailed()
        return
    }
    
    // Create output file
    outputFile, err := os.Create(task.OutputPath)
    if err != nil {
        log.Printf("Failed to create output file %s: %v", task.OutputPath, err)
        p.incrementFailed()
        return
    }
    defer outputFile.Close()
    
    // Encode with specified quality
    options := &jpeg.Options{Quality: task.Quality}
    if err := jpeg.Encode(outputFile, img, options); err != nil {
        log.Printf("Failed to encode image %s: %v", task.OutputPath, err)
        p.incrementFailed()
        return
    }
    
    duration := time.Since(start)
    log.Printf("Processed image %s -> %s in %v", task.InputPath, task.OutputPath, duration)
    p.incrementProcessed()
}

func (p *ImageProcessor) ProcessBatch(tasks []ImageTask) error {
    for _, task := range tasks {
        if err := p.worker.Send(task); err != nil {
            return fmt.Errorf("failed to send task: %w", err)
        }
    }
    return nil
}

func (p *ImageProcessor) Stats() (processed, failed int) {
    p.stats.mu.RLock()
    defer p.stats.mu.RUnlock()
    return p.stats.processed, p.stats.failed
}

func (p *ImageProcessor) incrementProcessed() {
    p.stats.mu.Lock()
    p.stats.processed++
    p.stats.mu.Unlock()
}

func (p *ImageProcessor) incrementFailed() {
    p.stats.mu.Lock()
    p.stats.failed++
    p.stats.mu.Unlock()
}

func (p *ImageProcessor) Close() {
    p.stop()
}

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    processor := NewImageProcessor(ctx)
    defer processor.Close()
    
    // Example tasks
    tasks := []ImageTask{
        {InputPath: "input/image1.jpg", OutputPath: "output/image1_compressed.jpg", Quality: 80},
        {InputPath: "input/image2.jpg", OutputPath: "output/image2_compressed.jpg", Quality: 80},
        {InputPath: "input/image3.jpg", OutputPath: "output/image3_compressed.jpg", Quality: 80},
    }
    
    // Process batch
    if err := processor.ProcessBatch(tasks); err != nil {
        log.Fatalf("Failed to process batch: %v", err)
    }
    
    // Wait for processing to complete
    time.Sleep(5 * time.Second)
    
    // Print statistics
    processed, failed := processor.Stats()
    log.Printf("Processing complete. Processed: %d, Failed: %d", processed, failed)
}
```

### Web Crawler

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "net/url"
    "sync"
    "time"

    "github.com/alextanhongpin/core/sync/background"
)

type CrawlTask struct {
    URL   string
    Depth int
}

type CrawlResult struct {
    URL     string
    Status  int
    Error   error
    Content string
}

type WebCrawler struct {
    worker    *background.Worker[CrawlTask]
    stop      func()
    client    *http.Client
    results   chan CrawlResult
    visited   map[string]bool
    visitedMu sync.RWMutex
    maxDepth  int
}

func NewWebCrawler(ctx context.Context, maxDepth int, numWorkers int) *WebCrawler {
    if numWorkers <= 0 {
        numWorkers = 10 // Default for I/O-bound tasks
    }
    
    c := &WebCrawler{
        client: &http.Client{
            Timeout: 10 * time.Second,
        },
        results:  make(chan CrawlResult, 100),
        visited:  make(map[string]bool),
        maxDepth: maxDepth,
    }
    
    c.worker, c.stop = background.New(ctx, numWorkers, c.crawlURL)
    
    return c
}

func (c *WebCrawler) crawlURL(ctx context.Context, task CrawlTask) {
    // Check if already visited
    c.visitedMu.RLock()
    if c.visited[task.URL] {
        c.visitedMu.RUnlock()
        return
    }
    c.visitedMu.RUnlock()
    
    // Mark as visited
    c.visitedMu.Lock()
    c.visited[task.URL] = true
    c.visitedMu.Unlock()
    
    log.Printf("Crawling URL: %s (depth: %d)", task.URL, task.Depth)
    
    // Create request with context
    req, err := http.NewRequestWithContext(ctx, "GET", task.URL, nil)
    if err != nil {
        c.results <- CrawlResult{URL: task.URL, Error: err}
        return
    }
    
    // Perform request
    resp, err := c.client.Do(req)
    if err != nil {
        c.results <- CrawlResult{URL: task.URL, Error: err}
        return
    }
    defer resp.Body.Close()
    
    // Read response (simplified - in real implementation, you'd parse HTML)
    content := fmt.Sprintf("Status: %d, Content-Length: %d", 
        resp.StatusCode, resp.ContentLength)
    
    result := CrawlResult{
        URL:     task.URL,
        Status:  resp.StatusCode,
        Content: content,
    }
    
    c.results <- result
    
    // Extract and queue more URLs if within depth limit
    if task.Depth < c.maxDepth && resp.StatusCode == 200 {
        // In a real implementation, you'd parse HTML and extract links
        // For demo purposes, we'll just add some example URLs
        baseURL, _ := url.Parse(task.URL)
        examplePaths := []string{"/about", "/contact", "/products"}
        
        for _, path := range examplePaths {
            newURL := baseURL.ResolveReference(&url.URL{Path: path}).String()
            
            // Check if not already visited
            c.visitedMu.RLock()
            if !c.visited[newURL] {
                c.visitedMu.RUnlock()
                c.worker.Send(CrawlTask{URL: newURL, Depth: task.Depth + 1})
            } else {
                c.visitedMu.RUnlock()
            }
        }
    }
}

func (c *WebCrawler) Crawl(startURL string) <-chan CrawlResult {
    // Start crawling
    c.worker.Send(CrawlTask{URL: startURL, Depth: 0})
    return c.results
}

func (c *WebCrawler) Close() {
    c.stop()
    close(c.results)
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    crawler := NewWebCrawler(ctx, 2, 5) // max depth 2, 5 workers
    defer crawler.Close()
    
    // Start crawling
    results := crawler.Crawl("https://example.com")
    
    // Process results
    var totalCrawled int
    for result := range results {
        if result.Error != nil {
            log.Printf("Error crawling %s: %v", result.URL, result.Error)
        } else {
            log.Printf("Successfully crawled %s: %s", result.URL, result.Content)
        }
        totalCrawled++
        
        // Stop after processing some results for demo
        if totalCrawled >= 10 {
            break
        }
    }
    
    log.Printf("Crawling complete. Total URLs crawled: %d", totalCrawled)
}
```

### Log Processing Pipeline

```go
package main

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "runtime"
    "strings"
    "sync"
    "time"

    "github.com/alextanhongpin/core/sync/background"
)

type LogEntry struct {
    Timestamp time.Time `json:"timestamp"`
    Level     string    `json:"level"`
    Message   string    `json:"message"`
    Service   string    `json:"service"`
    TraceID   string    `json:"trace_id"`
}

type LogProcessor struct {
    worker      *background.Worker[string]
    stop        func()
    stats       map[string]int
    statsMu     sync.RWMutex
    outputFile  *os.File
    outputMu    sync.Mutex
}

func NewLogProcessor(ctx context.Context, outputPath string) (*LogProcessor, error) {
    outputFile, err := os.Create(outputPath)
    if err != nil {
        return nil, fmt.Errorf("failed to create output file: %w", err)
    }
    
    p := &LogProcessor{
        stats:      make(map[string]int),
        outputFile: outputFile,
    }
    
    // Use CPU count for log processing
    p.worker, p.stop = background.New(ctx, runtime.GOMAXPROCS(0), p.processLogLine)
    
    return p, nil
}

func (p *LogProcessor) processLogLine(ctx context.Context, line string) {
    line = strings.TrimSpace(line)
    if line == "" {
        return
    }
    
    // Parse log entry
    var entry LogEntry
    if err := json.Unmarshal([]byte(line), &entry); err != nil {
        log.Printf("Failed to parse log line: %v", err)
        p.incrementStat("parse_errors")
        return
    }
    
    // Process based on log level
    switch entry.Level {
    case "ERROR":
        p.processError(entry)
    case "WARN":
        p.processWarning(entry)
    case "INFO":
        p.processInfo(entry)
    case "DEBUG":
        p.processDebug(entry)
    }
    
    p.incrementStat("total_processed")
    p.incrementStat(fmt.Sprintf("level_%s", strings.ToLower(entry.Level)))
}

func (p *LogProcessor) processError(entry LogEntry) {
    // Write critical errors to output file
    p.outputMu.Lock()
    defer p.outputMu.Unlock()
    
    errorData := map[string]interface{}{
        "timestamp": entry.Timestamp,
        "service":   entry.Service,
        "trace_id":  entry.TraceID,
        "message":   entry.Message,
        "severity":  "ERROR",
    }
    
    if data, err := json.Marshal(errorData); err == nil {
        p.outputFile.WriteString(string(data) + "\n")
    }
    
    p.incrementStat("errors_written")
}

func (p *LogProcessor) processWarning(entry LogEntry) {
    // Process warnings (could aggregate, filter, etc.)
    if strings.Contains(entry.Message, "deprecated") {
        p.incrementStat("deprecation_warnings")
    }
}

func (p *LogProcessor) processInfo(entry LogEntry) {
    // Process info logs
    if strings.Contains(entry.Message, "user_login") {
        p.incrementStat("user_logins")
    }
}

func (p *LogProcessor) processDebug(entry LogEntry) {
    // Skip debug logs in production mode
    p.incrementStat("debug_skipped")
}

func (p *LogProcessor) incrementStat(key string) {
    p.statsMu.Lock()
    p.stats[key]++
    p.statsMu.Unlock()
}

func (p *LogProcessor) ProcessFile(filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("failed to open file: %w", err)
    }
    defer file.Close()
    
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        if err := p.worker.Send(line); err != nil {
            return fmt.Errorf("failed to send line to worker: %w", err)
        }
    }
    
    return scanner.Err()
}

func (p *LogProcessor) Stats() map[string]int {
    p.statsMu.RLock()
    defer p.statsMu.RUnlock()
    
    stats := make(map[string]int)
    for k, v := range p.stats {
        stats[k] = v
    }
    return stats
}

func (p *LogProcessor) Close() {
    p.stop()
    p.outputFile.Close()
}

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    processor, err := NewLogProcessor(ctx, "processed_logs.json")
    if err != nil {
        log.Fatalf("Failed to create log processor: %v", err)
    }
    defer processor.Close()
    
    // Create sample log file
    if err := createSampleLogFile("sample.log"); err != nil {
        log.Fatalf("Failed to create sample log file: %v", err)
    }
    
    // Process log file
    start := time.Now()
    if err := processor.ProcessFile("sample.log"); err != nil {
        log.Fatalf("Failed to process log file: %v", err)
    }
    
    // Wait for processing to complete
    time.Sleep(2 * time.Second)
    
    duration := time.Since(start)
    stats := processor.Stats()
    
    log.Printf("Processing complete in %v", duration)
    log.Printf("Statistics:")
    for key, value := range stats {
        log.Printf("  %s: %d", key, value)
    }
}

func createSampleLogFile(filename string) error {
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()
    
    sampleLogs := []LogEntry{
        {Timestamp: time.Now(), Level: "INFO", Message: "user_login successful", Service: "auth", TraceID: "trace-1"},
        {Timestamp: time.Now(), Level: "ERROR", Message: "database connection failed", Service: "db", TraceID: "trace-2"},
        {Timestamp: time.Now(), Level: "WARN", Message: "deprecated API used", Service: "api", TraceID: "trace-3"},
        {Timestamp: time.Now(), Level: "DEBUG", Message: "debug information", Service: "debug", TraceID: "trace-4"},
        {Timestamp: time.Now(), Level: "ERROR", Message: "authentication failed", Service: "auth", TraceID: "trace-5"},
    }
    
    for _, entry := range sampleLogs {
        if data, err := json.Marshal(entry); err == nil {
            file.WriteString(string(data) + "\n")
        }
    }
    
    return nil
}
```

## 📊 Performance Considerations

## 📊 Performance Benchmarks

### Benchmark Results

```
BenchmarkWorkerPool/unbuffered-11    2612962    453.5 ns/op    0 B/op    0 allocs/op
BenchmarkWorkerPool/buffered-11      3968424    313.6 ns/op    0 B/op    0 allocs/op
BenchmarkWorkerPool/try_send-11     19121017     61.59 ns/op    0 B/op    0 allocs/op
BenchmarkMetrics-11                1000000000     0.2063 ns/op   0 B/op    0 allocs/op
```

### Performance Characteristics

- **Zero Allocations**: All operations are allocation-free after initialization
- **High Throughput**: Buffered channels provide ~30% better throughput
- **Fast Non-blocking**: `TrySend` is ~7x faster than blocking `Send`
- **Efficient Metrics**: Metric collection has minimal overhead

### Memory Usage

- **Minimal Overhead**: Only allocates channels and worker goroutines
- **Configurable Buffer**: Trade memory for throughput with buffer size
- **No Memory Leaks**: Proper cleanup with graceful shutdown

### Scalability

- **Linear Scaling**: Performance scales with worker count up to CPU limits
- **Efficient Context**: Fast context cancellation propagation
- **Low Contention**: Minimal lock contention with atomic metrics

### Throughput Optimization

```go
// For high-throughput scenarios
worker, stop := background.New(ctx, numWorkers, func(ctx context.Context, batch []Task) {
    // Process tasks in batches
    for _, task := range batch {
        processTask(task)
    }
})
```

## 🔧 Best Practices

### 1. Choose Appropriate Worker Count

```go
// CPU-bound tasks
workerCount := runtime.GOMAXPROCS(0)

// I/O-bound tasks
workerCount := runtime.GOMAXPROCS(0) * 2

// Memory-intensive tasks
workerCount := runtime.GOMAXPROCS(0) / 2
```

### 2. Handle Context Cancellation

```go
worker, stop := background.New(ctx, 4, func(ctx context.Context, task Task) {
    select {
    case <-ctx.Done():
        log.Println("Task cancelled")
        return
    default:
        // Process task
    }
})
```

### 3. Implement Proper Error Handling

```go
worker, stop := background.New(ctx, 4, func(ctx context.Context, task Task) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Worker panic: %v", r)
        }
    }()
    
    if err := processTask(task); err != nil {
        log.Printf("Task failed: %v", err)
        // Handle error appropriately
    }
})
```

### 4. Monitor Performance

```go
type TaskWithMetrics struct {
    Task      Task
    StartTime time.Time
}

worker, stop := background.New(ctx, 4, func(ctx context.Context, task TaskWithMetrics) {
    defer func() {
        duration := time.Since(task.StartTime)
        metrics.RecordProcessingTime(duration)
    }()
    
    processTask(task.Task)
})
```

## 🚨 Error Handling

### Common Errors

- `ErrTerminated`: Worker pool has been stopped
- Context cancellation: Task cancelled before completion
- Channel blocking: Worker pool overwhelmed

### Error Recovery

```go
worker, stop := background.New(ctx, 4, func(ctx context.Context, task Task) {
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        if err := processTask(task); err == nil {
            return
        }
        
        if i < maxRetries-1 {
            time.Sleep(time.Duration(i+1) * time.Second)
        }
    }
    
    log.Printf("Task failed after %d retries", maxRetries)
})
```

## 🧪 Testing

### Unit Tests

```go
func TestWorkerPool(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    var processed int
    var mu sync.Mutex
    
    worker, stop := background.New(ctx, 2, func(ctx context.Context, task int) {
        mu.Lock()
        processed++
        mu.Unlock()
    })
    defer stop()
    
    // Send tasks
    for i := 0; i < 10; i++ {
        if err := worker.Send(i); err != nil {
            t.Errorf("Send failed: %v", err)
        }
    }
    
    // Wait for processing
    time.Sleep(100 * time.Millisecond)
    
    mu.Lock()
    if processed != 10 {
        t.Errorf("Expected 10 tasks processed, got %d", processed)
    }
    mu.Unlock()
}
```

### Integration Tests

```go
func TestWorkerPoolShutdown(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    
    worker, stop := background.New(ctx, 2, func(ctx context.Context, task int) {
        time.Sleep(100 * time.Millisecond)
    })
    
    // Send tasks
    for i := 0; i < 5; i++ {
        worker.Send(i)
    }
    
    // Cancel context
    cancel()
    
    // Stop worker pool
    stop()
    
    // Verify sending fails after shutdown
    if err := worker.Send(999); err != background.ErrTerminated {
        t.Errorf("Expected ErrTerminated, got %v", err)
    }
}
```

## 🔗 Related Packages

- [`batch`](../batch/) - Batch processing utilities
- [`pipeline`](../pipeline/) - Stream processing pipelines
- [`throttle`](../throttle/) - Adaptive load control

## 📄 License

This package is part of the `github.com/alextanhongpin/core/sync` module and is licensed under the MIT License.

---

**Built with ❤️ for concurrent Go applications**
