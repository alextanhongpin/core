# Pipeline

A Go package for building high-performance data processing pipelines with support for parallel processing, rate limiting, queuing, and various transformation operations.

## Features

- **Parallel Processing**: Pool-based concurrent processing with configurable workers
- **Rate Limiting**: Built-in rate limiting and throughput control
- **Queuing**: Buffered channel queuing for pipeline stages
- **Transformations**: Map, filter, and flatten operations
- **Monitoring**: Rate and throughput monitoring with callbacks
- **Context Support**: Full context cancellation support
- **Generic Types**: Type-safe pipeline operations with Go generics

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
    "strconv"
    "time"
    
    "github.com/alextanhongpin/core/sync/pipeline"
)

func main() {
    ctx := context.Background()
    
    // Create a pipeline that processes integers
    numbers := pipeline.Generator(ctx, 100) // Generate numbers 0-99
    
    // Add rate limiting
    limited := pipeline.RateLimit(60, time.Second, numbers)
    
    // Transform to strings
    strings := pipeline.Map(limited, func(i int) string {
        return strconv.Itoa(i)
    })
    
    // Add parallel processing
    processed := pipeline.Pool(5, strings, func(s string) string {
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
