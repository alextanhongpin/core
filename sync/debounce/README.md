# Debounce - Event Throttling and Rate Limiting

[![Go Reference](https://pkg.go.dev/badge/github.com/alextanhongpin/core/sync/debounce.svg)](https://pkg.go.dev/github.com/alextanhongpin/core/sync/debounce)

A simple yet effective debouncing library for Go that provides event throttling and rate limiting capabilities. This package helps prevent excessive function calls by implementing both count-based and time-based debouncing mechanisms.

## ✨ Features

- **📊 Count-Based Debouncing**: Execute function every N calls
- **⏰ Time-Based Debouncing**: Execute function after timeout period
- **🔄 Hybrid Debouncing**: Combine count and time-based triggering
- **🔒 Thread-Safe**: Concurrent-safe operations with proper synchronization
- **⚡ Lightweight**: Minimal overhead with efficient implementation
- **🎯 Flexible**: Configurable parameters for different use cases

## 📦 Installation

```bash
go get github.com/alextanhongpin/core/sync/debounce
```

## 🚀 Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/alextanhongpin/core/sync/debounce"
)

func main() {
    // Create debouncer that executes every 5 calls or after 2 seconds
    debouncer := &debounce.Group{
        Every:   5,                // Execute every 5 calls
        Timeout: 2 * time.Second,  // Or after 2 seconds
    }
    
    // Function to be debounced
    counter := 0
    debouncedFunc := func() {
        counter++
        fmt.Printf("Debounced execution #%d\n", counter)
    }
    
    // Call the debounced function multiple times
    for i := 0; i < 12; i++ {
        debouncer.Do(debouncedFunc)
        time.Sleep(100 * time.Millisecond)
    }
    
    // Wait for final timeout
    time.Sleep(3 * time.Second)
}
```

### Time-Based Debouncing

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/alextanhongpin/core/sync/debounce"
)

func main() {
    // Create debouncer that executes after 1 second of inactivity
    debouncer := &debounce.Group{
        Timeout: 1 * time.Second,  // Execute after 1 second
    }
    
    // Simulate rapid user input
    debouncer.Do(func() {
        fmt.Println("Search query executed")
    })
    
    // Multiple rapid calls - only the last one will execute
    for i := 0; i < 5; i++ {
        time.Sleep(200 * time.Millisecond)
        debouncer.Do(func() {
            fmt.Printf("Processing user input #%d\n", i+1)
        })
    }
    
    // Wait for debounced execution
    time.Sleep(2 * time.Second)
}
```

### Count-Based Debouncing

```go
package main

import (
    "fmt"
    "time"

    "github.com/alextanhongpin/core/sync/debounce"
)

func main() {
    // Create debouncer that executes every 3 calls
    debouncer := &debounce.Group{
        Every: 3,  // Execute every 3 calls
    }
    
    // Track executions
    executions := 0
    
    // Call 10 times
    for i := 0; i < 10; i++ {
        debouncer.Do(func() {
            executions++
            fmt.Printf("Execution #%d (call #%d)\n", executions, i+1)
        })
    }
    
    fmt.Printf("Total executions: %d\n", executions)
}
```

## 🏗️ API Reference

### Types

#### Group

```go
type Group struct {
    Every   int           // Execute every N calls (0 disables count-based)
    Timeout time.Duration // Execute after timeout (0 disables time-based)
    // contains filtered or unexported fields
}
```

The main debounce group that manages debouncing behavior.

### Methods

#### Do

```go
func (g *Group) Do(fn func())
```

Executes the provided function according to the debouncing rules.

**Parameters:**
- `fn`: Function to execute when debounce conditions are met

**Behavior:**
- If `Every` is set and the call count reaches the threshold, executes immediately
- If `Timeout` is set, starts/restarts the timeout timer
- If both are set, executes on whichever condition is met first

## 🌟 Real-World Examples

### Search Input Debouncing

```go
package main

import (
    "fmt"
    "log"
    "strings"
    "time"

    "github.com/alextanhongpin/core/sync/debounce"
)

type SearchService struct {
    debouncer *debounce.Group
    lastQuery string
}

func NewSearchService() *SearchService {
    return &SearchService{
        debouncer: &debounce.Group{
            Timeout: 500 * time.Millisecond, // Wait 500ms after last input
        },
    }
}

func (s *SearchService) Search(query string) {
    s.lastQuery = query
    
    // Debounce search requests
    s.debouncer.Do(func() {
        s.performSearch(s.lastQuery)
    })
}

func (s *SearchService) performSearch(query string) {
    if strings.TrimSpace(query) == "" {
        return
    }
    
    fmt.Printf("🔍 Searching for: '%s'\n", query)
    
    // Simulate search API call
    time.Sleep(100 * time.Millisecond)
    
    // Mock search results
    results := []string{
        fmt.Sprintf("Result 1 for '%s'", query),
        fmt.Sprintf("Result 2 for '%s'", query),
        fmt.Sprintf("Result 3 for '%s'", query),
    }
    
    fmt.Printf("📋 Found %d results:\n", len(results))
    for _, result := range results {
        fmt.Printf("  - %s\n", result)
    }
    fmt.Println()
}

func main() {
    searchService := NewSearchService()
    
    // Simulate user typing
    queries := []string{"h", "he", "hel", "hell", "hello", "hello w", "hello wo", "hello wor", "hello worl", "hello world"}
    
    fmt.Println("=== Search Input Debouncing Demo ===")
    fmt.Println("User typing 'hello world' character by character...")
    fmt.Println()
    
    for i, query := range queries {
        fmt.Printf("👤 User types: '%s'\n", query)
        searchService.Search(query)
        
        // Simulate typing delay
        if i < len(queries)-1 {
            time.Sleep(200 * time.Millisecond)
        }
    }
    
    // Wait for final search
    time.Sleep(1 * time.Second)
}
```

### Log Aggregation

```go
package main

import (
    "fmt"
    "log"
    "sync"
    "time"

    "github.com/alextanhongpin/core/sync/debounce"
)

type LogEntry struct {
    Timestamp time.Time
    Level     string
    Message   string
    Service   string
}

type LogAggregator struct {
    debouncer *debounce.Group
    buffer    []LogEntry
    mu        sync.Mutex
}

func NewLogAggregator() *LogAggregator {
    aggregator := &LogAggregator{
        debouncer: &debounce.Group{
            Every:   100,               // Flush every 100 logs
            Timeout: 5 * time.Second,   // Or every 5 seconds
        },
        buffer: make([]LogEntry, 0),
    }
    
    return aggregator
}

func (la *LogAggregator) Log(entry LogEntry) {
    la.mu.Lock()
    la.buffer = append(la.buffer, entry)
    la.mu.Unlock()
    
    // Debounce the flush operation
    la.debouncer.Do(func() {
        la.flush()
    })
}

func (la *LogAggregator) flush() {
    la.mu.Lock()
    if len(la.buffer) == 0 {
        la.mu.Unlock()
        return
    }
    
    // Copy buffer for processing
    entries := make([]LogEntry, len(la.buffer))
    copy(entries, la.buffer)
    
    // Clear buffer
    la.buffer = la.buffer[:0]
    la.mu.Unlock()
    
    // Process entries
    fmt.Printf("🔄 Flushing %d log entries\n", len(entries))
    
    // Group by service and level
    stats := make(map[string]map[string]int)
    for _, entry := range entries {
        if stats[entry.Service] == nil {
            stats[entry.Service] = make(map[string]int)
        }
        stats[entry.Service][entry.Level]++
    }
    
    // Print statistics
    fmt.Println("📊 Log Statistics:")
    for service, levels := range stats {
        fmt.Printf("  %s:\n", service)
        for level, count := range levels {
            fmt.Printf("    %s: %d\n", level, count)
        }
    }
    
    // Simulate writing to persistent storage
    time.Sleep(100 * time.Millisecond)
    fmt.Println("💾 Logs written to storage")
    fmt.Println()
}

func main() {
    aggregator := NewLogAggregator()
    
    fmt.Println("=== Log Aggregation Demo ===")
    fmt.Println("Generating log entries...")
    fmt.Println()
    
    // Generate log entries
    services := []string{"auth", "api", "database", "cache", "worker"}
    levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
    
    // Generate 250 log entries
    for i := 0; i < 250; i++ {
        entry := LogEntry{
            Timestamp: time.Now(),
            Level:     levels[i%len(levels)],
            Message:   fmt.Sprintf("Log message %d", i+1),
            Service:   services[i%len(services)],
        }
        
        aggregator.Log(entry)
        
        // Simulate varying log rates
        if i%50 == 0 {
            time.Sleep(100 * time.Millisecond)
        } else {
            time.Sleep(10 * time.Millisecond)
        }
    }
    
    // Wait for final flush
    time.Sleep(6 * time.Second)
    
    fmt.Println("=== Log Aggregation Complete ===")
}
```

### Rate-Limited API Client

```go
package main

import (
    "fmt"
    "log"
    "sync"
    "time"

    "github.com/alextanhongpin/core/sync/debounce"
)

type APIRequest struct {
    ID       string
    Endpoint string
    Data     interface{}
}

type APIResponse struct {
    RequestID string
    Status    int
    Data      interface{}
}

type RateLimitedAPIClient struct {
    debouncer *debounce.Group
    requests  []APIRequest
    mu        sync.Mutex
    responses chan APIResponse
}

func NewRateLimitedAPIClient() *RateLimitedAPIClient {
    client := &RateLimitedAPIClient{
        debouncer: &debounce.Group{
            Every:   10,               // Batch every 10 requests
            Timeout: 2 * time.Second,  // Or every 2 seconds
        },
        requests:  make([]APIRequest, 0),
        responses: make(chan APIResponse, 100),
    }
    
    return client
}

func (c *RateLimitedAPIClient) MakeRequest(req APIRequest) <-chan APIResponse {
    c.mu.Lock()
    c.requests = append(c.requests, req)
    c.mu.Unlock()
    
    // Debounce batch processing
    c.debouncer.Do(func() {
        c.processBatch()
    })
    
    return c.responses
}

func (c *RateLimitedAPIClient) processBatch() {
    c.mu.Lock()
    if len(c.requests) == 0 {
        c.mu.Unlock()
        return
    }
    
    // Copy requests for processing
    batch := make([]APIRequest, len(c.requests))
    copy(batch, c.requests)
    
    // Clear requests
    c.requests = c.requests[:0]
    c.mu.Unlock()
    
    fmt.Printf("🚀 Processing batch of %d API requests\n", len(batch))
    
    // Process batch
    for _, req := range batch {
        // Simulate API call
        time.Sleep(50 * time.Millisecond)
        
        response := APIResponse{
            RequestID: req.ID,
            Status:    200,
            Data:      fmt.Sprintf("Response for %s", req.Endpoint),
        }
        
        select {
        case c.responses <- response:
        default:
            log.Printf("Response channel full, dropping response for %s", req.ID)
        }
    }
    
    fmt.Printf("✅ Batch processing complete\n")
}

func main() {
    client := NewRateLimitedAPIClient()
    
    fmt.Println("=== Rate-Limited API Client Demo ===")
    fmt.Println("Making API requests...")
    fmt.Println()
    
    // Make multiple API requests
    var wg sync.WaitGroup
    
    for i := 0; i < 25; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            
            req := APIRequest{
                ID:       fmt.Sprintf("req-%d", i),
                Endpoint: fmt.Sprintf("/api/resource/%d", i),
                Data:     map[string]interface{}{"id": i},
            }
            
            responseChan := client.MakeRequest(req)
            
            // Wait for response
            select {
            case response := <-responseChan:
                fmt.Printf("📤 Request %s: %d - %v\n", response.RequestID, response.Status, response.Data)
            case <-time.After(10 * time.Second):
                fmt.Printf("⏰ Request %s timed out\n", req.ID)
            }
        }(i)
        
        // Stagger requests
        time.Sleep(100 * time.Millisecond)
    }
    
    wg.Wait()
    fmt.Println()
    fmt.Println("=== All API Requests Complete ===")
}
```

### Event Bus with Debouncing

```go
package main

import (
    "fmt"
    "sync"
    "time"

    "github.com/alextanhongpin/core/sync/debounce"
)

type Event struct {
    Type      string
    Data      interface{}
    Timestamp time.Time
}

type EventBus struct {
    subscribers map[string][]func([]Event)
    debouncers  map[string]*debounce.Group
    buffers     map[string][]Event
    mu          sync.RWMutex
}

func NewEventBus() *EventBus {
    return &EventBus{
        subscribers: make(map[string][]func([]Event)),
        debouncers:  make(map[string]*debounce.Group),
        buffers:     make(map[string][]Event),
    }
}

func (eb *EventBus) Subscribe(eventType string, handler func([]Event), config debounce.Group) {
    eb.mu.Lock()
    defer eb.mu.Unlock()
    
    // Add subscriber
    eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
    
    // Create debouncer if not exists
    if eb.debouncers[eventType] == nil {
        eb.debouncers[eventType] = &debounce.Group{
            Every:   config.Every,
            Timeout: config.Timeout,
        }
        eb.buffers[eventType] = make([]Event, 0)
    }
}

func (eb *EventBus) Publish(event Event) {
    eb.mu.Lock()
    defer eb.mu.Unlock()
    
    // Add to buffer
    eb.buffers[event.Type] = append(eb.buffers[event.Type], event)
    
    // Debounce notification
    if debouncer := eb.debouncers[event.Type]; debouncer != nil {
        debouncer.Do(func() {
            eb.notifySubscribers(event.Type)
        })
    }
}

func (eb *EventBus) notifySubscribers(eventType string) {
    eb.mu.Lock()
    buffer := eb.buffers[eventType]
    if len(buffer) == 0 {
        eb.mu.Unlock()
        return
    }
    
    // Copy events
    events := make([]Event, len(buffer))
    copy(events, buffer)
    
    // Clear buffer
    eb.buffers[eventType] = eb.buffers[eventType][:0]
    
    // Get subscribers
    subscribers := eb.subscribers[eventType]
    eb.mu.Unlock()
    
    // Notify subscribers
    for _, handler := range subscribers {
        go handler(events)
    }
}

func main() {
    bus := NewEventBus()
    
    fmt.Println("=== Event Bus with Debouncing Demo ===")
    fmt.Println()
    
    // Subscribe to user events (batch every 5 events or every 2 seconds)
    bus.Subscribe("user.action", func(events []Event) {
        fmt.Printf("👤 Processing %d user actions:\n", len(events))
        for _, event := range events {
            fmt.Printf("  - %v\n", event.Data)
        }
        fmt.Println()
    }, debounce.Group{
        Every:   5,
        Timeout: 2 * time.Second,
    })
    
    // Subscribe to system events (batch every 3 events or every 1 second)
    bus.Subscribe("system.event", func(events []Event) {
        fmt.Printf("🖥️ Processing %d system events:\n", len(events))
        for _, event := range events {
            fmt.Printf("  - %v\n", event.Data)
        }
        fmt.Println()
    }, debounce.Group{
        Every:   3,
        Timeout: 1 * time.Second,
    })
    
    // Publish events
    userActions := []string{"login", "view_profile", "edit_profile", "logout", "login", "view_dashboard", "create_post"}
    systemEvents := []string{"startup", "memory_check", "disk_check", "health_check", "cleanup"}
    
    // Publish user actions
    for i, action := range userActions {
        bus.Publish(Event{
            Type:      "user.action",
            Data:      fmt.Sprintf("User %s at %v", action, time.Now().Format("15:04:05")),
            Timestamp: time.Now(),
        })
        
        // Stagger publishing
        if i%2 == 0 {
            time.Sleep(300 * time.Millisecond)
        } else {
            time.Sleep(100 * time.Millisecond)
        }
    }
    
    // Publish system events
    for i, event := range systemEvents {
        bus.Publish(Event{
            Type:      "system.event",
            Data:      fmt.Sprintf("System %s at %v", event, time.Now().Format("15:04:05")),
            Timestamp: time.Now(),
        })
        
        time.Sleep(400 * time.Millisecond)
    }
    
    // Wait for final debounced events
    time.Sleep(3 * time.Second)
    
    fmt.Println("=== Event Bus Demo Complete ===")
}
```

## 📊 Performance Considerations

### Memory Usage

The debounce package has minimal memory overhead:
- Single timer per debouncer group
- Thread-safe counter with mutex
- No memory leaks with proper timer management

### Benchmarks

```go
func BenchmarkDebounce(b *testing.B) {
    debouncer := &debounce.Group{
        Every:   100,
        Timeout: 100 * time.Millisecond,
    }
    
    var executions int
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        debouncer.Do(func() {
            executions++
        })
    }
}
```

### Tuning Parameters

```go
// High-frequency events (user input)
inputDebouncer := &debounce.Group{
    Timeout: 300 * time.Millisecond,
}

// Batch processing
batchDebouncer := &debounce.Group{
    Every:   50,
    Timeout: 5 * time.Second,
}

// Rate limiting
rateLimitDebouncer := &debounce.Group{
    Every:   10,
    Timeout: 1 * time.Second,
}
```

## 🔧 Best Practices

### 1. Choose Appropriate Parameters

```go
// For user input debouncing
userInputDebouncer := &debounce.Group{
    Timeout: 300 * time.Millisecond, // Common UI debounce delay
}

// For batch processing
batchDebouncer := &debounce.Group{
    Every:   100,              // Batch size
    Timeout: 5 * time.Second,  // Maximum wait time
}
```

### 2. Handle Cleanup

```go
type Service struct {
    debouncer *debounce.Group
    done      chan struct{}
}

func (s *Service) Close() {
    close(s.done)
    // Debouncer will be garbage collected
}
```

### 3. Error Handling

```go
debouncer.Do(func() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Debounced function panic: %v", r)
        }
    }()
    
    // Your function logic
})
```

### 4. Testing

```go
func TestDebounce(t *testing.T) {
    var executions int
    debouncer := &debounce.Group{
        Every: 3,
    }
    
    // Call 5 times, should execute twice (at 3 and 6 calls)
    for i := 0; i < 5; i++ {
        debouncer.Do(func() {
            executions++
        })
    }
    
    if executions != 1 {
        t.Errorf("Expected 1 execution, got %d", executions)
    }
}
```

## 🧪 Testing

### Unit Tests

```go
func TestCountBasedDebounce(t *testing.T) {
    var executions int
    debouncer := &debounce.Group{Every: 3}
    
    // First 3 calls should trigger execution
    for i := 0; i < 3; i++ {
        debouncer.Do(func() {
            executions++
        })
    }
    
    if executions != 1 {
        t.Errorf("Expected 1 execution after 3 calls, got %d", executions)
    }
}

func TestTimeBasedDebounce(t *testing.T) {
    var executions int
    debouncer := &debounce.Group{Timeout: 100 * time.Millisecond}
    
    // Call function
    debouncer.Do(func() {
        executions++
    })
    
    // Should not execute immediately
    if executions != 0 {
        t.Errorf("Expected 0 executions immediately, got %d", executions)
    }
    
    // Wait for timeout
    time.Sleep(150 * time.Millisecond)
    
    if executions != 1 {
        t.Errorf("Expected 1 execution after timeout, got %d", executions)
    }
}
```

## 🔗 Related Packages

- [`throttle`](../throttle/) - Adaptive throttling and load control
- [`rate`](../rate/) - Rate limiting utilities
- [`background`](../background/) - Background task processing

## 📄 License

This package is part of the `github.com/alextanhongpin/core/sync` module and is licensed under the MIT License.

---

**Built with ❤️ for efficient event debouncing in Go**
