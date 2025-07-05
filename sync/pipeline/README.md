# Pipeline Package

A high-performance, production-ready Go pipeline library for building concurrent data processing pipelines with advanced features like metrics, error handling, backpressure, and context cancellation.

## Features

- **Type-safe Generic Functions**: Fully typed pipeline operations using Go generics
- **Concurrent Processing**: Pool-based parallel execution with configurable worker counts
- **Backpressure Control**: Built-in throttling, rate limiting, and semaphore-based concurrency control
- **Context Support**: Full context cancellation support throughout the pipeline
- **Metrics & Monitoring**: Built-in throughput, error rate, and performance metrics
- **Error Handling**: Comprehensive error handling with panic recovery
- **Batching Operations**: Efficient batching with size and timeout controls
- **Channel Management**: Safe channel operations with cleanup and leak prevention
- **Flexible Configuration**: Configurable options for buffer sizes, timeouts, and worker counts

## Installation

```bash
go get github.com/alextanhongpin/core/sync/pipeline
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/alextanhongpin/core/sync/pipeline"
)

func main() {
    ctx := context.Background()
    
    // Create a simple pipeline
    numbers := pipeline.Generator(ctx, 10)
    doubled := pipeline.Transform(numbers, func(n int) int { return n * 2 })
    
    // Process with metrics
    processed := pipeline.Throughput(doubled, func(info pipeline.ThroughputInfo) {
        fmt.Printf("Processed: %d items at %.2f/s\n", info.Total, info.Rate)
    })
    
    // Collect results
    results := pipeline.ToSlice(processed)
    fmt.Printf("Results: %v\n", results)
}
```

## Core Components

### Sources

Data sources generate items for the pipeline:

```go
// Generate sequence of numbers
numbers := pipeline.Generator(ctx, 100)

// Generate from function
timestamps := pipeline.GeneratorFunc(ctx, func() time.Time { 
    return time.Now() 
})

// Repeat values
values := pipeline.Repeat(ctx, 1, 2, 3)
```

### Transformations

Transform data as it flows through the pipeline:

```go
// Transform/Map - change item type
strings := pipeline.Transform(numbers, func(n int) string { 
    return fmt.Sprintf("item-%d", n) 
})

// Filter - keep only matching items
evens := pipeline.Filter(numbers, func(n int) bool { 
    return n%2 == 0 
})

// Take/Skip - limit items
first10 := pipeline.Take(10, numbers)
after5 := pipeline.Skip(5, numbers)

// Distinct - remove duplicates
unique := pipeline.Distinct(numbers)
```

### Parallel Processing

Process items concurrently with configurable worker pools:

```go
// Pool processing with 4 workers
processed := pipeline.Pool(4, numbers, func(n int) string {
    // Simulate work
    time.Sleep(100 * time.Millisecond)
    return fmt.Sprintf("processed-%d", n)
})

// Context-aware pool processing
processed := pipeline.PoolWithContext(ctx, 4, numbers, 
    func(ctx context.Context, n int) string {
        select {
        case <-ctx.Done():
            return "cancelled"
        default:
            return fmt.Sprintf("processed-%d", n)
        }
    })

// Semaphore-based concurrency control
limited := pipeline.Semaphore(2, numbers, func(n int) int {
    return n * 2
})
```

### Flow Control

Control the flow of data through the pipeline:

```go
// Buffer - add buffering between stages
buffered := pipeline.Buffer(100, numbers)

// Throttle - limit processing rate
throttled := pipeline.Throttle(100*time.Millisecond, numbers)

// Rate limit - items per second
limited := pipeline.RateLimit(10, numbers)

// Debounce - minimum time between items
debounced := pipeline.Debounce(1*time.Second, numbers)
```

### Batching

Collect items into batches for efficient processing:

```go
// Batch by size and timeout
batches := pipeline.Batch(10, 1*time.Second, numbers)

// Batch unique items only
uniqueBatches := pipeline.BatchDistinct(5, 500*time.Millisecond, numbers)
```

### Context & Timeout

Add cancellation and timeout support:

```go
// With context cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

cancellable := pipeline.WithContext(ctx, numbers)

// With timeout
timedOut := pipeline.WithTimeout(5*time.Second, numbers)

// Or done pattern
done := make(chan struct{})
controlled := pipeline.OrDone(done, numbers)
```

### Error Handling

Handle errors gracefully throughout the pipeline:

```go
// Transform with error handling
results := pipeline.Transform(numbers, func(n int) pipeline.Result[string] {
    if n < 0 {
        return pipeline.MakeErrorResult[string](errors.New("negative number"))
    }
    return pipeline.MakeSuccessResult(fmt.Sprintf("item-%d", n))
})

// Filter out errors
success := pipeline.FilterErrors(results, func(err error) {
    log.Printf("Error: %v", err)
})

// Or get just the successful data
data := pipeline.FlatMap(results)
```

### Monitoring & Metrics

Monitor pipeline performance:

```go
// Throughput monitoring
monitored := pipeline.Throughput(results, func(info pipeline.ThroughputInfo) {
    fmt.Printf("Total: %d, Rate: %.2f/s, Errors: %d (%.1f%%)\n",
        info.Total, info.Rate, info.TotalFailures, info.ErrorRate*100)
})

// Rate monitoring
rateMonitored := pipeline.Rate(numbers, func(info pipeline.RateInfo) {
    fmt.Printf("Processed: %d items at %.2f/s\n", info.Total, info.Rate)
})

// Custom monitoring
customMonitored := pipeline.Monitor(numbers, 
    func(item int) { fmt.Printf("Processing: %d\n", item) },
    func(err error) { fmt.Printf("Error: %v\n", err) },
)
```

### Fan-out & Fan-in

Distribute and merge pipeline flows:

```go
// Fan-out to multiple channels
channels := pipeline.FanOut(3, numbers)

// Process each channel differently
processed := make([]<-chan string, len(channels))
for i, ch := range channels {
    processed[i] = pipeline.Transform(ch, func(n int) string {
        return fmt.Sprintf("worker-%d: %d", i, n)
    })
}

// Fan-in to merge results
merged := pipeline.FanIn(processed...)
```

### Utilities

Useful utilities for pipeline operations:

```go
// Tee - split into two identical streams
stream1, stream2 := pipeline.Tee(numbers)

// First/Last - get single items
first, ok := pipeline.First(numbers)
last, ok := pipeline.Last(numbers)

// Collect all items
all := pipeline.ToSlice(numbers)

// Process all items
pipeline.ForEach(numbers, func(n int) {
    fmt.Printf("Item: %d\n", n)
})
```

## Advanced Examples

### Complex Processing Pipeline

```go
func processData(ctx context.Context) {
    // Create data source
    source := pipeline.Generator(ctx, 1000)
    
    // Add buffering
    buffered := pipeline.Buffer(50, source)
    
    // Filter valid items
    valid := pipeline.Filter(buffered, func(n int) bool { 
        return n > 0 
    })
    
    // Transform with error handling
    transformed := pipeline.Pool(4, valid, func(n int) pipeline.Result[string] {
        if n%10 == 0 {
            return pipeline.MakeErrorResult[string](fmt.Errorf("divisible by 10"))
        }
        return pipeline.MakeSuccessResult(fmt.Sprintf("item-%d", n))
    })
    
    // Add metrics
    withMetrics := pipeline.Throughput(transformed, func(info pipeline.ThroughputInfo) {
        if info.Total%100 == 0 {
            fmt.Printf("Processed: %d, Rate: %.2f/s, Errors: %.1f%%\n",
                info.Total, info.Rate, info.ErrorRate*100)
        }
    })
    
    // Handle errors
    success := pipeline.FilterErrors(withMetrics, func(err error) {
        log.Printf("Processing error: %v", err)
    })
    
    // Batch results
    batches := pipeline.Batch(10, 1*time.Second, success)
    
    // Process batches
    for batch := range batches {
        fmt.Printf("Batch: %v\n", batch)
    }
}
```

### Rate-Limited API Processing

```go
func processAPIRequests(ctx context.Context, urls []string) {
    // Create URL source
    urlSource := pipeline.From(ctx, urls...)
    
    // Rate limit to 10 requests/second
    rateLimited := pipeline.RateLimit(10, urlSource)
    
    // Process with retry logic
    results := pipeline.Pool(3, rateLimited, func(url string) pipeline.Result[Response] {
        resp, err := http.Get(url)
        if err != nil {
            return pipeline.MakeErrorResult[Response](err)
        }
        return pipeline.MakeSuccessResult(Response{URL: url, Status: resp.StatusCode})
    })
    
    // Retry failed requests
    withRetry := pipeline.Retry(3, 1*time.Second, results, func(url string) pipeline.Result[Response] {
        resp, err := http.Get(url)
        if err != nil {
            return pipeline.MakeErrorResult[Response](err)
        }
        return pipeline.MakeSuccessResult(Response{URL: url, Status: resp.StatusCode})
    })
    
    // Filter successful responses
    successful := pipeline.FilterErrors(withRetry, func(err error) {
        log.Printf("Request failed: %v", err)
    })
    
    // Collect results
    responses := pipeline.ToSlice(successful)
    fmt.Printf("Processed %d responses\n", len(responses))
}
```

## Error Handling Best Practices

1. **Use Result Types**: Wrap operations that can fail with `Result[T]`
2. **Handle Panics**: The pipeline automatically recovers from panics
3. **Monitor Errors**: Use `FilterErrors` or `Throughput` to track error rates
4. **Implement Retries**: Use `Retry` for transient failures
5. **Timeout Operations**: Use `WithTimeout` for long-running operations

## Performance Considerations

1. **Buffer Sizing**: Use appropriate buffer sizes to balance memory usage and throughput
2. **Worker Counts**: Match worker count to CPU cores and I/O characteristics
3. **Batching**: Use batching for operations that benefit from amortization
4. **Memory Usage**: Monitor memory usage with large datasets
5. **Goroutine Leaks**: The pipeline handles cleanup automatically

## Thread Safety

All pipeline operations are thread-safe and designed for concurrent use. The pipeline handles:

- Channel lifecycle management
- Goroutine cleanup
- Panic recovery
- Context cancellation
- Resource cleanup

## API Reference

### Sources
- `Generator(ctx, n)` - Generate sequence of numbers
- `GeneratorFunc(ctx, fn)` - Generate using function
- `Repeat(ctx, values...)` - Repeat values

### Transformations
- `Transform(in, fn)` - Transform items (alias: `Map`)
- `Filter(in, predicate)` - Filter items
- `Take(n, in)` - Take first n items
- `Skip(n, in)` - Skip first n items
- `Distinct(in)` - Remove duplicates

### Parallel Processing
- `Pool(n, in, fn)` - Process with n workers
- `PoolWithContext(ctx, n, in, fn)` - Context-aware pool
- `Semaphore(n, in, fn)` - Semaphore-limited processing

### Flow Control
- `Buffer(size, in)` - Add buffering
- `Throttle(interval, in)` - Rate throttling
- `RateLimit(rps, in)` - Rate limiting
- `Debounce(duration, in)` - Debounce items

### Batching
- `Batch(size, timeout, in)` - Batch items
- `BatchDistinct(size, timeout, in)` - Batch unique items

### Context & Timeout
- `WithContext(ctx, in)` - Add context
- `WithTimeout(timeout, in)` - Add timeout
- `OrDone(done, in)` - Done channel pattern

### Error Handling
- `MakeResult(data, err)` - Create result
- `MakeSuccessResult(data)` - Success result
- `MakeErrorResult(err)` - Error result
- `FilterErrors(in, onError)` - Filter errors
- `FlatMap(in)` - Extract successful results
- `Retry(maxRetries, backoff, in, retryFn)` - Retry failed operations

### Monitoring
- `Throughput(in, fn)` - Monitor throughput
- `Rate(in, fn)` - Monitor rate
- `Monitor(in, onItem, onError)` - Custom monitoring

### Fan-out & Fan-in
- `FanOut(n, in)` - Distribute to n channels
- `FanIn(channels...)` - Merge channels
- `Tee(in)` - Split into two channels

### Utilities
- `First(in)` - Get first item
- `Last(in)` - Get last item
- `ToSlice(in)` - Collect to slice
- `ForEach(in, fn)` - Process each item

## License

MIT License - see LICENSE file for details.
        // Simulate some work
        time.Sleep(10 * time.Millisecond)
        return "processed: " + s
    })
    
    // Consume results
    for result := range processed {
        fmt.Println(result)
    }
}
```

## API Reference

### Core Functions

#### `Generator(ctx context.Context, n int) <-chan int`
Creates a channel that generates sequential numbers from 0 to n-1.

#### `Map[T, U any](in <-chan T, fn func(T) U) <-chan U`
Transforms each item in the input channel using the provided function.

#### `Filter[T any](in <-chan T, fn func(T) bool) <-chan T`
Filters items based on the provided predicate function.

#### `Pool[T, U any](n int, in <-chan T, fn func(T) U) <-chan U`
Processes items concurrently using n workers.

#### `Queue[T any](n int, in <-chan T) <-chan T`
Adds buffering to the pipeline with a buffer size of n.

#### `RateLimit[T any](rate int, per time.Duration, in <-chan T) <-chan T`
Limits the rate of items passing through the pipeline.

### Monitoring Functions

#### `Rate[T any](in <-chan T, fn func(RateInfo)) <-chan T`
Monitors the rate of items passing through and calls the callback function.

#### `Throughput[T any](in <-chan T, fn func(ThroughputInfo)) <-chan T`
Monitors throughput and calls the callback function with metrics.

### Utility Functions

#### `Context[T any](ctx context.Context, in <-chan T) <-chan T`
Adds context cancellation to a pipeline stage.

#### `FlatMap[T any](in <-chan Result[T]) <-chan T`
Flattens Result types, extracting successful values and discarding errors.

## Real-World Examples

### Log Processing Pipeline

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "time"
    
    "github.com/alextanhongpin/core/sync/pipeline"
)

type LogEntry struct {
    Timestamp time.Time `json:"timestamp"`
    Level     string    `json:"level"`
    Message   string    `json:"message"`
    UserID    string    `json:"user_id,omitempty"`
}

type ProcessedLog struct {
    LogEntry
    Processed time.Time `json:"processed"`
    Hash      string    `json:"hash"`
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Simulate log entries
    rawLogs := make(chan string, 100)
    go func() {
        defer close(rawLogs)
        for i := 0; i < 1000; i++ {
            log := LogEntry{
                Timestamp: time.Now(),
                Level:     []string{"INFO", "WARN", "ERROR"}[i%3],
                Message:   fmt.Sprintf("Log message %d", i),
                UserID:    fmt.Sprintf("user_%d", i%10),
            }
            data, _ := json.Marshal(log)
            rawLogs <- string(data)
        }
    }()
    
    // Parse JSON logs
    parsedLogs := pipeline.Map(rawLogs, func(raw string) pipeline.Result[LogEntry] {
        var log LogEntry
        if err := json.Unmarshal([]byte(raw), &log); err != nil {
            return pipeline.MakeResult(LogEntry{}, err)
        }
        return pipeline.MakeResult(log, nil)
    })
    
    // Filter out parsing errors
    validLogs := pipeline.FlatMap(parsedLogs)
    
    // Filter only ERROR level logs
    errorLogs := pipeline.Filter(validLogs, func(log LogEntry) bool {
        return log.Level == "ERROR"
    })
    
    // Process logs in parallel (e.g., enrich with additional data)
    processedLogs := pipeline.Pool(5, errorLogs, func(log LogEntry) ProcessedLog {
        // Simulate processing time
        time.Sleep(50 * time.Millisecond)
        
        return ProcessedLog{
            LogEntry:  log,
            Processed: time.Now(),
            Hash:      fmt.Sprintf("hash_%s", log.UserID),
        }
    })
    
    // Add rate limiting for downstream systems
    rateLimited := pipeline.RateLimit(10, time.Second, processedLogs)
    
    // Add monitoring
    monitored := pipeline.Throughput(rateLimited, func(info pipeline.ThroughputInfo) {
        fmt.Printf("Processed: %d, Rate: %.2f/s\n", info.Total, info.Rate)
    })
    
    // Consume results
    for processed := range monitored {
        fmt.Printf("Processed log: %s - %s\n", processed.UserID, processed.Message)
    }
}
```

### Image Processing Pipeline

```go
package main

import (
    "context"
    "fmt"
    "image"
    "image/jpeg"
    "os"
    "path/filepath"
    "strings"
    "time"
    
    "github.com/alextanhongpin/core/sync/pipeline"
)

type ImageJob struct {
    Path string
    Size string // "small", "medium", "large"
}

type ProcessedImage struct {
    OriginalPath string
    OutputPath   string
    Size         string
    ProcessTime  time.Duration
}

func main() {
    ctx := context.Background()
    
    // Create image processing jobs
    jobs := make(chan ImageJob, 100)
    go func() {
        defer close(jobs)
        
        files, _ := filepath.Glob("./images/*.jpg")
        for _, file := range files {
            for _, size := range []string{"small", "medium", "large"} {
                jobs <- ImageJob{Path: file, Size: size}
            }
        }
    }()
    
    // Add queue for buffering
    queuedJobs := pipeline.Queue(50, jobs)
    
    // Process images in parallel
    processedImages := pipeline.Pool(3, queuedJobs, func(job ImageJob) pipeline.Result[ProcessedImage] {
        start := time.Now()
        
        // Simulate image processing
        outputPath, err := processImage(job.Path, job.Size)
        if err != nil {
            return pipeline.MakeResult(ProcessedImage{}, err)
        }
        
        return pipeline.MakeResult(ProcessedImage{
            OriginalPath: job.Path,
            OutputPath:   outputPath,
            Size:         job.Size,
            ProcessTime:  time.Since(start),
        }, nil)
    })
    
    // Filter successful results
    successfulResults := pipeline.FlatMap(processedImages)
    
    // Add rate limiting to avoid overwhelming storage
    rateLimited := pipeline.RateLimit(5, time.Second, successfulResults)
    
    // Monitor processing rate
    monitored := pipeline.Rate(rateLimited, func(info pipeline.RateInfo) {
        if info.Total%10 == 0 {
            fmt.Printf("Images processed: %d\n", info.Total)
        }
    })
    
    // Consume results
    for result := range monitored {
        fmt.Printf("Processed: %s -> %s (took %v)\n", 
            result.OriginalPath, result.OutputPath, result.ProcessTime)
    }
}

func processImage(inputPath, size string) (string, error) {
    // Simulate image processing
    time.Sleep(100 * time.Millisecond)
    
    dir := filepath.Dir(inputPath)
    filename := filepath.Base(inputPath)
    ext := filepath.Ext(filename)
    name := strings.TrimSuffix(filename, ext)
    
    outputPath := filepath.Join(dir, fmt.Sprintf("%s_%s%s", name, size, ext))
    
    // In a real implementation, you would resize the image here
    // For simulation, we'll just copy the file
    return outputPath, nil
}
```

### Data ETL Pipeline

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    
    "github.com/alextanhongpin/core/sync/pipeline"
)

type RawData struct {
    ID        int
    Data      string
    Timestamp time.Time
}

type TransformedData struct {
    ID            int
    ProcessedData string
    Category      string
    Timestamp     time.Time
}

type ETLPipeline struct {
    sourceDB *sql.DB
    targetDB *sql.DB
}

func NewETLPipeline(sourceDB, targetDB *sql.DB) *ETLPipeline {
    return &ETLPipeline{
        sourceDB: sourceDB,
        targetDB: targetDB,
    }
}

func (e *ETLPipeline) Run(ctx context.Context) error {
    // Extract: Read from source database
    rawData := e.extract(ctx)
    
    // Transform: Process data in parallel
    transformedData := pipeline.Pool(10, rawData, func(raw RawData) pipeline.Result[TransformedData] {
        transformed, err := e.transform(raw)
        return pipeline.MakeResult(transformed, err)
    })
    
    // Filter successful transformations
    validData := pipeline.FlatMap(transformedData)
    
    // Add rate limiting for database writes
    rateLimited := pipeline.RateLimit(100, time.Second, validData)
    
    // Batch for efficient loading
    batched := e.batch(rateLimited, 50)
    
    // Load: Write to target database
    results := pipeline.Map(batched, func(batch []TransformedData) pipeline.Result[int] {
        count, err := e.load(ctx, batch)
        return pipeline.MakeResult(count, err)
    })
    
    // Monitor progress
    monitored := pipeline.Throughput(results, func(info pipeline.ThroughputInfo) {
        fmt.Printf("Batches processed: %d, Records/s: %.2f\n", info.Total, info.Rate*50)
    })
    
    // Consume final results
    totalRecords := 0
    for result := range monitored {
        if result.Err == nil {
            totalRecords += result.Data
        } else {
            fmt.Printf("Error processing batch: %v\n", result.Err)
        }
    }
    
    fmt.Printf("Total records processed: %d\n", totalRecords)
    return nil
}

func (e *ETLPipeline) extract(ctx context.Context) <-chan RawData {
    ch := make(chan RawData)
    
    go func() {
        defer close(ch)
        
        rows, err := e.sourceDB.QueryContext(ctx, "SELECT id, data, timestamp FROM raw_data WHERE processed = false")
        if err != nil {
            return
        }
        defer rows.Close()
        
        for rows.Next() {
            var raw RawData
            if err := rows.Scan(&raw.ID, &raw.Data, &raw.Timestamp); err != nil {
                continue
            }
            
            select {
            case ch <- raw:
            case <-ctx.Done():
                return
            }
        }
    }()
    
    return ch
}

func (e *ETLPipeline) transform(raw RawData) (TransformedData, error) {
    // Simulate transformation logic
    time.Sleep(10 * time.Millisecond)
    
    category := "default"
    if len(raw.Data) > 100 {
        category = "large"
    } else if len(raw.Data) > 50 {
        category = "medium"
    } else {
        category = "small"
    }
    
    return TransformedData{
        ID:            raw.ID,
        ProcessedData: strings.ToUpper(raw.Data),
        Category:      category,
        Timestamp:     raw.Timestamp,
    }, nil
}

func (e *ETLPipeline) batch(in <-chan TransformedData, size int) <-chan []TransformedData {
    out := make(chan []TransformedData)
    
    go func() {
        defer close(out)
        
        var batch []TransformedData
        for data := range in {
            batch = append(batch, data)
            
            if len(batch) >= size {
                out <- batch
                batch = nil
            }
        }
        
        if len(batch) > 0 {
            out <- batch
        }
    }()
    
    return out
}

func (e *ETLPipeline) load(ctx context.Context, batch []TransformedData) (int, error) {
    // Simulate batch insert
    time.Sleep(50 * time.Millisecond)
    
    // In a real implementation, you would use a prepared statement
    // with batch insert for efficiency
    return len(batch), nil
}
```

## Performance Monitoring

The pipeline package provides built-in monitoring capabilities:

```go
// Rate monitoring
rateLimited := pipeline.Rate(input, func(info pipeline.RateInfo) {
    fmt.Printf("Rate: %.2f items/s, Total: %d\n", info.Rate, info.Total)
})

// Throughput monitoring
monitored := pipeline.Throughput(rateLimited, func(info pipeline.ThroughputInfo) {
    fmt.Printf("Throughput: %.2f items/s, Total: %d, Duration: %v\n", 
        info.Rate, info.Total, info.Duration)
})
```

## Error Handling

Use the Result type for operations that might fail:

```go
results := pipeline.Map(input, func(item string) pipeline.Result[int] {
    value, err := strconv.Atoi(item)
    return pipeline.MakeResult(value, err)
})

// Filter out errors and extract successful values
successfulResults := pipeline.FlatMap(results)
```

## Best Practices

1. **Use Context**: Always pass and respect context for cancellation
2. **Buffer Appropriately**: Use `Queue()` to buffer between stages with different processing speeds
3. **Monitor Performance**: Use rate and throughput monitoring in production
4. **Handle Errors**: Use Result types for operations that might fail
5. **Optimize Parallelism**: Tune worker pool sizes based on your workload
6. **Rate Limiting**: Protect downstream systems with appropriate rate limiting

## Testing

```go
func TestPipeline(t *testing.T) {
    ctx := context.Background()
    
    // Create test data
    input := make(chan int, 10)
    for i := 0; i < 10; i++ {
        input <- i
    }
    close(input)
    
    // Process through pipeline
    output := pipeline.Map(input, func(i int) int {
        return i * 2
    })
    
    // Collect results
    var results []int
    for result := range output {
        results = append(results, result)
    }
    
    // Verify
    assert.Len(t, results, 10)
    assert.Equal(t, 0, results[0])
    assert.Equal(t, 18, results[9])
}
```

## License

MIT License. See [LICENSE](../../LICENSE) for details.
