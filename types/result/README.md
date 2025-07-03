# Result Package

[![Go Reference](https://pkg.go.dev/badge/github.com/alextanhongpin/core/types/result.svg)](https://pkg.go.dev/github.com/alextanhongpin/core/types/result)

The `result` package provides a generic Result type for handling operations that can succeed or fail, similar to Rust's Result type. This enables more functional error handling patterns in Go and is particularly useful when passing results through channels, collecting multiple operations, or when you want to defer error handling.

## Features

- **Generic Result Type**: Type-safe wrapper for values and errors
- **Functional Operations**: Map, FlatMap, Filter operations for composability
- **Collection Operations**: Process multiple results with All, Any, Filter, Partition
- **Safe Unwrapping**: Multiple ways to safely extract values with defaults
- **Channel-Friendly**: Designed for passing through channels and concurrent operations
- **Zero Dependencies**: Pure Go implementation with no external dependencies

## Installation

```bash
go get github.com/alextanhongpin/core/types/result
```

## Quick Start

```go
package main

import (
    "errors"
    "fmt"
    
    "github.com/alextanhongpin/core/types/result"
)

func main() {
    // Create successful result
    success := result.OK(42)
    value, err := success.Unwrap()
    fmt.Printf("Success: %d, Error: %v\n", value, err)
    
    // Create error result
    failure := result.Err[int](errors.New("something went wrong"))
    value, err = failure.Unwrap()
    fmt.Printf("Value: %d, Error: %v\n", value, err)
    
    // Use default value for errors
    value = failure.UnwrapOr(100)
    fmt.Printf("With default: %d\n", value)
}
```

## Core Functions

### Creating Results

```go
// Successful result
success := result.OK("hello")

// Error result
failure := result.Err[string](errors.New("failed"))

// From existing Go function
apiResult := result.From(func() (string, error) {
    return makeAPICall()
})

// Empty result
empty := result.New[string]()
```

### Extracting Values

```go
// Basic unwrapping
value, err := res.Unwrap()

// With default value
value := res.UnwrapOr("default")

// With default function
value := res.UnwrapOrElse(func(err error) string {
    return fmt.Sprintf("Error: %v", err)
})

// Check state
if res.IsOK() {
    // Handle success
}
if res.IsErr() {
    // Handle error
}
```

### Transforming Results

```go
// Transform successful values
doubled := result.OK(21).Map(func(x int) int {
    return x * 2
}) // Result[int] with value 42

// Transform errors
enhanced := failure.MapError(func(err error) error {
    return fmt.Errorf("enhanced: %w", err)
})

// Chain operations (FlatMap)
chainResult := result.OK(5).FlatMap(func(x int) *result.Result[int] {
    if x > 0 {
        return result.OK(x * x)
    }
    return result.Err[int](errors.New("negative number"))
})
```

## Collection Operations

### All - Collect All Successful Results

```go
results := []*result.Result[int]{
    result.OK(1),
    result.OK(2),
    result.OK(3),
}

values, err := result.All(results...)
if err != nil {
    // One or more results failed
} else {
    fmt.Printf("All values: %v\n", values) // [1, 2, 3]
}
```

### Any - First Successful Result

```go
results := []*result.Result[string]{
    result.Err[string](errors.New("failed 1")),
    result.Err[string](errors.New("failed 2")),
    result.OK("success"),
    result.OK("another success"),
}

value, err := result.Any(results...)
if err != nil {
    // All results failed
} else {
    fmt.Printf("First success: %s\n", value) // "success"
}
```

### Filter - Extract Only Successful Values

```go
results := []*result.Result[int]{
    result.OK(1),
    result.Err[int](errors.New("failed")),
    result.OK(3),
}

values := result.Filter(results...) // [1, 3]
```

### Partition - Separate Success and Errors

```go
results := []*result.Result[int]{
    result.OK(1),
    result.Err[int](errors.New("error 1")),
    result.OK(3),
    result.Err[int](errors.New("error 2")),
}

values, errors := result.Partition(results...)
// values: [1, 3]
// errors: [error 1, error 2]
```

## Real-World Use Cases

### Channel Communication

Perfect for passing results through channels:

```go
func worker(input <-chan string, output chan<- *result.Result[ProcessedData]) {
    defer close(output)
    
    for data := range input {
        result := processData(data)
        output <- result
    }
}
```

### Concurrent Operations

Collect results from multiple goroutines:

```go
func fetchFromMultipleSources(id string) ([]Data, error) {
    var wg sync.WaitGroup
    results := make([]*result.Result[Data], 3)
    
    sources := []func(string) *result.Result[Data]{
        fetchFromDB, fetchFromCache, fetchFromAPI,
    }
    
    for i, fetch := range sources {
        wg.Add(1)
        go func(index int, fn func(string) *result.Result[Data]) {
            defer wg.Done()
            results[index] = fn(id)
        }(i, fetch)
    }
    wg.Wait()
    
    // Get all successful results
    return result.All(results...)
}
```

### Error Recovery

Implement fallback strategies:

```go
func getDataWithFallback(id string) Data {
    // Try multiple sources, return first successful result
    sources := []*result.Result[Data]{
        fetchFromPrimary(id),
        fetchFromSecondary(id),
        fetchFromCache(id),
    }
    
    data, err := result.Any(sources...)
    if err != nil {
        // All sources failed, return default
        return getDefaultData(id)
    }
    
    return data
}
```

### Batch Processing

Process collections with mixed success/failure:

```go
func processBatch(items []string) BatchResult {
    results := make([]*result.Result[ProcessedItem], len(items))
    
    for i, item := range items {
        results[i] = processItem(item)
    }
    
    successful, failed := result.Partition(results...)
    
    return BatchResult{
        Successful: successful,
        Failed:     failed,
        Total:      len(items),
    }
}
```

## Best Practices

1. **Use for Error-Prone Operations**: Ideal for operations that commonly fail
2. **Chain Operations**: Use Map/FlatMap for composing operations
3. **Batch Processing**: Use collection functions for processing multiple results
4. **Channel Communication**: Perfect for passing results through channels
5. **Graceful Degradation**: Use UnwrapOr for providing fallback values

## When to Use Result vs Traditional Go Error Handling

**Use Result when:**
- Working with channels and concurrent operations
- Processing collections of potentially failing operations
- Building functional pipelines
- You want to defer error handling
- Working with external APIs or unreliable services

**Use traditional error handling when:**
- Writing idiomatic Go libraries for public consumption
- Performance is absolutely critical
- Working with existing codebases that expect (T, error)
- Simple, linear error handling is sufficient

## Performance

- **Zero Allocation**: OK/Err creation only allocates the Result struct
- **Memory Efficient**: No boxing of primitive types
- **Concurrent Safe**: Results are immutable and safe for concurrent access

## Contributing

1. Follow Go conventions and the existing code style
2. Add comprehensive tests for new functionality
3. Include real-world examples in documentation
4. Ensure backward compatibility
5. Update benchmarks for performance-critical changes

## License

This package is part of the core types library and follows the same license terms.
