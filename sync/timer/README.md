# Timer

A Go package providing JavaScript-style timer functions (`setTimeout` and `setInterval`) with proper resource management and cancellation support.

## Features

- **setTimeout**: Execute a function after a specified delay
- **setInterval**: Execute a function repeatedly at specified intervals
- **Cancellation**: Cancel timers before they execute
- **Resource Management**: Proper cleanup and resource management
- **Goroutine Safe**: Safe for concurrent use
- **Simple API**: Familiar JavaScript-style API

## Installation

```bash
go get github.com/alextanhongpin/core/sync/timer
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/alextanhongpin/core/sync/timer"
)

func main() {
    // setTimeout - execute once after delay
    cancel := timer.SetTimeout(func() {
        fmt.Println("This runs after 2 seconds")
    }, 2*time.Second)
    
    // You can cancel if needed
    _ = cancel
    
    // setInterval - execute repeatedly
    stop := timer.SetInterval(func() {
        fmt.Println("This runs every 1 second")
    }, 1*time.Second)
    
    // Let it run for 5 seconds
    time.Sleep(5 * time.Second)
    
    // Stop the interval
    stop()
    
    fmt.Println("Timer stopped")
}
```

## API Reference

### SetTimeout

```go
func SetTimeout(fn func(), duration time.Duration) func()
```

Executes the function `fn` after the specified `duration`. Returns a cancel function that can be called to prevent execution.

### SetInterval

```go
func SetInterval(fn func(), duration time.Duration) func()
```

Executes the function `fn` repeatedly at the specified `duration` intervals. Returns a stop function that can be called to stop the interval.

## Real-World Examples

### HTTP Server with Cleanup Timer

```go
package main

import (
    "fmt"
    "net/http"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/timer"
)

type Server struct {
    mu        sync.RWMutex
    sessions  map[string]*Session
    cleanupStop func()
}

type Session struct {
    ID        string
    LastAccess time.Time
    Data      map[string]interface{}
}

func NewServer() *Server {
    s := &Server{
        sessions: make(map[string]*Session),
    }
    
    // Start cleanup timer - runs every 5 minutes
    s.cleanupStop = timer.SetInterval(func() {
        s.cleanupExpiredSessions()
    }, 5*time.Minute)
    
    return s
}

func (s *Server) Close() {
    if s.cleanupStop != nil {
        s.cleanupStop()
    }
}

func (s *Server) cleanupExpiredSessions() {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    now := time.Now()
    expired := make([]string, 0)
    
    for id, session := range s.sessions {
        if now.Sub(session.LastAccess) > 30*time.Minute {
            expired = append(expired, id)
        }
    }
    
    for _, id := range expired {
        delete(s.sessions, id)
    }
    
    if len(expired) > 0 {
        fmt.Printf("Cleaned up %d expired sessions\n", len(expired))
    }
}

func (s *Server) getSession(id string) *Session {
    s.mu.RLock()
    session := s.sessions[id]
    s.mu.RUnlock()
    
    if session != nil {
        s.mu.Lock()
        session.LastAccess = time.Now()
        s.mu.Unlock()
    }
    
    return session
}

func (s *Server) createSession(id string) *Session {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    session := &Session{
        ID:         id,
        LastAccess: time.Now(),
        Data:       make(map[string]interface{}),
    }
    s.sessions[id] = session
    
    return session
}

func (s *Server) sessionHandler(w http.ResponseWriter, r *http.Request) {
    sessionID := r.URL.Query().Get("session")
    if sessionID == "" {
        http.Error(w, "Missing session ID", http.StatusBadRequest)
        return
    }
    
    session := s.getSession(sessionID)
    if session == nil {
        session = s.createSession(sessionID)
        fmt.Fprintf(w, "Created new session: %s\n", sessionID)
    } else {
        fmt.Fprintf(w, "Found existing session: %s, last accessed: %v\n", 
            sessionID, session.LastAccess)
    }
}

func main() {
    server := NewServer()
    defer server.Close()
    
    http.HandleFunc("/session", server.sessionHandler)
    
    fmt.Println("Server starting on :8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}
```

### Task Scheduler with Delayed Execution

```go
package main

import (
    "fmt"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/timer"
)

type TaskScheduler struct {
    mu      sync.RWMutex
    tasks   map[string]*ScheduledTask
    counter int
}

type ScheduledTask struct {
    ID     string
    Name   string
    Fn     func()
    Cancel func()
}

func NewTaskScheduler() *TaskScheduler {
    return &TaskScheduler{
        tasks: make(map[string]*ScheduledTask),
    }
}

func (ts *TaskScheduler) ScheduleOnce(name string, fn func(), delay time.Duration) string {
    ts.mu.Lock()
    defer ts.mu.Unlock()
    
    ts.counter++
    id := fmt.Sprintf("task_%d", ts.counter)
    
    cancel := timer.SetTimeout(func() {
        fmt.Printf("Executing scheduled task: %s\n", name)
        fn()
        
        // Remove task after execution
        ts.mu.Lock()
        delete(ts.tasks, id)
        ts.mu.Unlock()
    }, delay)
    
    task := &ScheduledTask{
        ID:     id,
        Name:   name,
        Fn:     fn,
        Cancel: cancel,
    }
    
    ts.tasks[id] = task
    
    fmt.Printf("Scheduled task '%s' (ID: %s) to run in %v\n", name, id, delay)
    return id
}

func (ts *TaskScheduler) ScheduleRecurring(name string, fn func(), interval time.Duration) string {
    ts.mu.Lock()
    defer ts.mu.Unlock()
    
    ts.counter++
    id := fmt.Sprintf("recurring_%d", ts.counter)
    
    stop := timer.SetInterval(func() {
        fmt.Printf("Executing recurring task: %s\n", name)
        fn()
    }, interval)
    
    task := &ScheduledTask{
        ID:     id,
        Name:   name,
        Fn:     fn,
        Cancel: stop,
    }
    
    ts.tasks[id] = task
    
    fmt.Printf("Scheduled recurring task '%s' (ID: %s) to run every %v\n", name, id, interval)
    return id
}

func (ts *TaskScheduler) Cancel(id string) bool {
    ts.mu.Lock()
    defer ts.mu.Unlock()
    
    task, exists := ts.tasks[id]
    if !exists {
        return false
    }
    
    task.Cancel()
    delete(ts.tasks, id)
    
    fmt.Printf("Cancelled task '%s' (ID: %s)\n", task.Name, id)
    return true
}

func (ts *TaskScheduler) CancelAll() {
    ts.mu.Lock()
    defer ts.mu.Unlock()
    
    for id, task := range ts.tasks {
        task.Cancel()
        fmt.Printf("Cancelled task '%s' (ID: %s)\n", task.Name, id)
    }
    
    ts.tasks = make(map[string]*ScheduledTask)
}

func (ts *TaskScheduler) ListTasks() []string {
    ts.mu.RLock()
    defer ts.mu.RUnlock()
    
    var tasks []string
    for id, task := range ts.tasks {
        tasks = append(tasks, fmt.Sprintf("%s: %s", id, task.Name))
    }
    
    return tasks
}

func main() {
    scheduler := NewTaskScheduler()
    defer scheduler.CancelAll()
    
    // Schedule some one-time tasks
    scheduler.ScheduleOnce("Send Email", func() {
        fmt.Println("📧 Sending email...")
    }, 2*time.Second)
    
    scheduler.ScheduleOnce("Generate Report", func() {
        fmt.Println("📊 Generating report...")
    }, 5*time.Second)
    
    // Schedule recurring tasks
    heartbeatID := scheduler.ScheduleRecurring("Heartbeat", func() {
        fmt.Println("💓 Heartbeat")
    }, 3*time.Second)
    
    backupID := scheduler.ScheduleRecurring("Backup", func() {
        fmt.Println("💾 Running backup...")
    }, 10*time.Second)
    
    // List current tasks
    fmt.Println("\nCurrent tasks:")
    for _, task := range scheduler.ListTasks() {
        fmt.Printf("  %s\n", task)
    }
    
    // Let tasks run for a while
    time.Sleep(15 * time.Second)
    
    // Cancel the heartbeat
    scheduler.Cancel(heartbeatID)
    
    // Let backup run a bit more
    time.Sleep(10 * time.Second)
    
    // Cancel backup
    scheduler.Cancel(backupID)
    
    fmt.Println("All tasks completed or cancelled")
}
```

### Rate Limiting with Timeout

```go
package main

import (
    "fmt"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/timer"
)

type RateLimiter struct {
    mu         sync.Mutex
    tokens     int
    maxTokens  int
    refillRate time.Duration
    refillStop func()
}

func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
    rl := &RateLimiter{
        tokens:     maxTokens,
        maxTokens:  maxTokens,
        refillRate: refillRate,
    }
    
    // Start token refill timer
    rl.refillStop = timer.SetInterval(func() {
        rl.refillTokens()
    }, refillRate)
    
    return rl
}

func (rl *RateLimiter) Close() {
    if rl.refillStop != nil {
        rl.refillStop()
    }
}

func (rl *RateLimiter) refillTokens() {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    if rl.tokens < rl.maxTokens {
        rl.tokens++
        fmt.Printf("Token refilled. Available: %d/%d\n", rl.tokens, rl.maxTokens)
    }
}

func (rl *RateLimiter) TryAcquire() bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    if rl.tokens > 0 {
        rl.tokens--
        fmt.Printf("Token acquired. Remaining: %d/%d\n", rl.tokens, rl.maxTokens)
        return true
    }
    
    return false
}

func (rl *RateLimiter) AcquireWithTimeout(timeout time.Duration) bool {
    // Try immediate acquisition
    if rl.TryAcquire() {
        return true
    }
    
    // Setup timeout
    acquired := make(chan bool, 1)
    var cancel func()
    
    cancel = timer.SetTimeout(func() {
        acquired <- false
    }, timeout)
    
    // Keep trying until timeout
    go func() {
        ticker := time.NewTicker(10 * time.Millisecond)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                if rl.TryAcquire() {
                    cancel() // Cancel timeout
                    acquired <- true
                    return
                }
            case <-time.After(timeout):
                return
            }
        }
    }()
    
    return <-acquired
}

func main() {
    // Create rate limiter: 3 tokens, refill every 2 seconds
    limiter := NewRateLimiter(3, 2*time.Second)
    defer limiter.Close()
    
    // Simulate multiple requests
    var wg sync.WaitGroup
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            fmt.Printf("Request %d: Trying to acquire token...\n", id)
            start := time.Now()
            
            if limiter.AcquireWithTimeout(5 * time.Second) {
                fmt.Printf("Request %d: Token acquired after %v\n", id, time.Since(start))
                
                // Simulate work
                time.Sleep(500 * time.Millisecond)
                fmt.Printf("Request %d: Work completed\n", id)
            } else {
                fmt.Printf("Request %d: Timeout after %v\n", id, time.Since(start))
            }
        }(i)
        
        // Stagger requests slightly
        time.Sleep(100 * time.Millisecond)
    }
    
    wg.Wait()
    fmt.Println("All requests completed")
}
```

### Debounced Function Calls

```go
package main

import (
    "fmt"
    "sync"
    "time"
    
    "github.com/alextanhongpin/core/sync/timer"
)

type Debouncer struct {
    mu     sync.Mutex
    cancel func()
}

func NewDebouncer() *Debouncer {
    return &Debouncer{}
}

func (d *Debouncer) Debounce(fn func(), delay time.Duration) {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    // Cancel previous timer if it exists
    if d.cancel != nil {
        d.cancel()
    }
    
    // Set new timer
    d.cancel = timer.SetTimeout(fn, delay)
}

func (d *Debouncer) Cancel() {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    if d.cancel != nil {
        d.cancel()
        d.cancel = nil
    }
}

type SearchService struct {
    debouncer *Debouncer
}

func NewSearchService() *SearchService {
    return &SearchService{
        debouncer: NewDebouncer(),
    }
}

func (s *SearchService) Search(query string) {
    fmt.Printf("Search input: '%s'\n", query)
    
    // Debounce the actual search to avoid too many API calls
    s.debouncer.Debounce(func() {
        s.performSearch(query)
    }, 300*time.Millisecond)
}

func (s *SearchService) performSearch(query string) {
    fmt.Printf("🔍 Performing search for: '%s'\n", query)
    // Simulate API call
    time.Sleep(200 * time.Millisecond)
    fmt.Printf("✅ Search completed for: '%s'\n", query)
}

func main() {
    service := NewSearchService()
    
    // Simulate rapid typing
    searches := []string{
        "a",
        "ap",
        "app",
        "appl",
        "apple",
    }
    
    fmt.Println("Simulating rapid typing...")
    for _, search := range searches {
        service.Search(search)
        time.Sleep(100 * time.Millisecond) // Fast typing
    }
    
    // Wait for debounced search
    time.Sleep(1 * time.Second)
    
    fmt.Println("\nSimulating slower typing...")
    service.Search("goo")
    time.Sleep(500 * time.Millisecond) // Slower typing
    service.Search("google")
    
    // Wait for final search
    time.Sleep(1 * time.Second)
    
    fmt.Println("Search simulation completed")
}
```

### Retry with Exponential Backoff

```go
package main

import (
    "fmt"
    "math"
    "math/rand"
    "time"
    
    "github.com/alextanhongpin/core/sync/timer"
)

type RetryConfig struct {
    MaxRetries  int
    BaseDelay   time.Duration
    MaxDelay    time.Duration
    Multiplier  float64
    Jitter      bool
}

func NewRetryConfig() *RetryConfig {
    return &RetryConfig{
        MaxRetries: 5,
        BaseDelay:  100 * time.Millisecond,
        MaxDelay:   30 * time.Second,
        Multiplier: 2.0,
        Jitter:     true,
    }
}

func RetryWithBackoff(fn func() error, config *RetryConfig) error {
    var lastErr error
    
    for attempt := 0; attempt <= config.MaxRetries; attempt++ {
        if attempt > 0 {
            delay := config.calculateDelay(attempt)
            fmt.Printf("Retry attempt %d after %v\n", attempt, delay)
            
            // Use timer for delay
            done := make(chan struct{})
            timer.SetTimeout(func() {
                close(done)
            }, delay)
            <-done
        }
        
        fmt.Printf("Executing attempt %d\n", attempt+1)
        if err := fn(); err != nil {
            lastErr = err
            fmt.Printf("Attempt %d failed: %v\n", attempt+1, err)
            
            if attempt == config.MaxRetries {
                return fmt.Errorf("max retries exceeded, last error: %w", lastErr)
            }
            continue
        }
        
        fmt.Printf("Attempt %d succeeded\n", attempt+1)
        return nil
    }
    
    return lastErr
}

func (c *RetryConfig) calculateDelay(attempt int) time.Duration {
    delay := float64(c.BaseDelay) * math.Pow(c.Multiplier, float64(attempt-1))
    
    if c.Jitter {
        // Add random jitter (±25%)
        jitter := 0.25 * delay * (2*rand.Float64() - 1)
        delay += jitter
    }
    
    if delay > float64(c.MaxDelay) {
        delay = float64(c.MaxDelay)
    }
    
    return time.Duration(delay)
}

func main() {
    config := NewRetryConfig()
    
    // Simulate a flaky service
    attempts := 0
    flakyService := func() error {
        attempts++
        
        // Fail first 3 attempts, then succeed
        if attempts < 4 {
            return fmt.Errorf("service temporarily unavailable")
        }
        
        return nil
    }
    
    fmt.Println("Attempting to call flaky service...")
    start := time.Now()
    
    err := RetryWithBackoff(flakyService, config)
    if err != nil {
        fmt.Printf("Final error: %v\n", err)
    } else {
        fmt.Printf("Service call succeeded after %v\n", time.Since(start))
    }
}
```

## Error Handling and Cancellation

```go
func main() {
    // Cancel a timeout before it executes
    cancel := timer.SetTimeout(func() {
        fmt.Println("This will not print")
    }, 2*time.Second)
    
    // Cancel after 1 second
    time.Sleep(1 * time.Second)
    cancel()
    
    // Stop an interval
    stop := timer.SetInterval(func() {
        fmt.Println("Interval tick")
    }, 500*time.Millisecond)
    
    // Let it run for 2 seconds
    time.Sleep(2 * time.Second)
    stop()
    
    fmt.Println("All timers cancelled")
}
```

## Testing

```go
func TestSetTimeout(t *testing.T) {
    executed := false
    
    timer.SetTimeout(func() {
        executed = true
    }, 100*time.Millisecond)
    
    // Check it hasn't executed yet
    assert.False(t, executed)
    
    // Wait for execution
    time.Sleep(150 * time.Millisecond)
    assert.True(t, executed)
}

func TestSetInterval(t *testing.T) {
    count := 0
    
    stop := timer.SetInterval(func() {
        count++
    }, 50*time.Millisecond)
    
    // Let it run for ~125ms (should execute 2-3 times)
    time.Sleep(125 * time.Millisecond)
    stop()
    
    assert.True(t, count >= 2)
    assert.True(t, count <= 3)
}

func TestCancellation(t *testing.T) {
    executed := false
    
    cancel := timer.SetTimeout(func() {
        executed = true
    }, 100*time.Millisecond)
    
    // Cancel before execution
    cancel()
    
    // Wait past execution time
    time.Sleep(150 * time.Millisecond)
    assert.False(t, executed)
}
```

## Best Practices

1. **Always Cancel**: Always call the cancel/stop function to prevent resource leaks
2. **Use Reasonable Delays**: Avoid very short intervals that might impact performance
3. **Handle Panics**: Consider wrapping timer functions with panic recovery
4. **Resource Management**: Keep track of active timers in long-running applications
5. **Context Integration**: Consider integrating with context for cancellation
6. **Testing**: Use deterministic delays in tests or mock time

## Performance Considerations

- Each timer creates a goroutine, so be mindful of the number of active timers
- Very short intervals (< 1ms) may not be accurate due to Go's scheduler
- Consider using time.Ticker directly for high-frequency operations
- Cancel unused timers to prevent goroutine leaks

## License

MIT License. See [LICENSE](../../LICENSE) for details.
