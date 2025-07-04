package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/timer"
)

func main() {
	// Example 1: Basic setTimeout usage
	fmt.Println("=== Basic setTimeout Example ===")
	setTimeoutExample()

	// Example 2: Basic setInterval usage
	fmt.Println("\n=== Basic setInterval Example ===")
	setIntervalExample()

	// Example 3: Canceling timeouts
	fmt.Println("\n=== Timeout Cancellation Example ===")
	timeoutCancellationExample()

	// Example 4: Stopping intervals
	fmt.Println("\n=== Interval Stopping Example ===")
	intervalStoppingExample()

	// Example 5: Debouncing with timer
	fmt.Println("\n=== Debouncing Example ===")
	debouncingExample()

	// Example 6: Heartbeat system
	fmt.Println("\n=== Heartbeat System Example ===")
	heartbeatExample()
}

func setTimeoutExample() {
	fmt.Println("Setting timeout for 2 seconds...")

	start := time.Now()
	cancel := timer.SetTimeout(func() {
		fmt.Printf("Timeout executed after %v\n", time.Since(start))
	}, 2*time.Second)

	// Do some work while waiting
	fmt.Println("Doing other work...")
	time.Sleep(1 * time.Second)
	fmt.Println("Still doing work...")

	// Wait for timeout to complete
	time.Sleep(1500 * time.Millisecond)

	// Cancel function is available but not used in this example
	_ = cancel
}

func setIntervalExample() {
	fmt.Println("Starting interval (every 500ms)...")

	counter := 0
	stop := timer.SetInterval(func() {
		counter++
		fmt.Printf("Interval tick %d at %v\n", counter, time.Now().Format("15:04:05.000"))
	}, 500*time.Millisecond)

	// Let it run for 3 seconds
	time.Sleep(3 * time.Second)

	// Stop the interval
	stop()
	fmt.Printf("Interval stopped after %d ticks\n", counter)
}

func timeoutCancellationExample() {
	fmt.Println("Setting timeout that will be cancelled...")

	cancel := timer.SetTimeout(func() {
		fmt.Println("This should not be printed!")
	}, 2*time.Second)

	// Cancel after 1 second
	time.Sleep(1 * time.Second)
	cancel()
	fmt.Println("Timeout cancelled")

	// Wait a bit more to ensure it doesn't execute
	time.Sleep(2 * time.Second)
	fmt.Println("Timeout cancellation confirmed")
}

func intervalStoppingExample() {
	fmt.Println("Starting interval that will be stopped...")

	counter := 0
	stop := timer.SetInterval(func() {
		counter++
		fmt.Printf("Interval tick %d\n", counter)
	}, 300*time.Millisecond)

	// Stop after 5 ticks
	go func() {
		for counter < 5 {
			time.Sleep(100 * time.Millisecond)
		}
		stop()
		fmt.Println("Interval stopped")
	}()

	// Wait for completion
	time.Sleep(2 * time.Second)
	fmt.Printf("Final counter value: %d\n", counter)
}

func debouncingExample() {
	fmt.Println("Debouncing example - simulating rapid key presses...")

	debouncer := NewDebouncer(500 * time.Millisecond)

	// Simulate rapid key presses
	keys := []string{"h", "e", "l", "l", "o"}
	query := ""

	for _, key := range keys {
		query += key
		fmt.Printf("Typed: %s\n", query)

		// Debounce the search
		currentQuery := query
		debouncer.Debounce(func() {
			fmt.Printf("🔍 Searching for: %s\n", currentQuery)
		})

		// Simulate typing delay
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for debounced search
	time.Sleep(1 * time.Second)

	// Now type something else after a delay
	time.Sleep(600 * time.Millisecond)
	query += " world"
	fmt.Printf("Typed: %s\n", query)

	finalQuery := query
	debouncer.Debounce(func() {
		fmt.Printf("🔍 Searching for: %s\n", finalQuery)
	})

	// Wait for final search
	time.Sleep(1 * time.Second)
}

func heartbeatExample() {
	fmt.Println("Starting heartbeat system...")

	// Main heartbeat
	heartbeatCount := 0
	heartbeat := timer.SetInterval(func() {
		heartbeatCount++
		fmt.Printf("💓 Heartbeat %d at %v\n", heartbeatCount, time.Now().Format("15:04:05"))
	}, 1*time.Second)

	// Health check
	healthCheckCount := 0
	healthCheck := timer.SetInterval(func() {
		healthCheckCount++
		fmt.Printf("🏥 Health check %d - System OK\n", healthCheckCount)
	}, 3*time.Second)

	// Backup process
	backupCount := 0
	backup := timer.SetInterval(func() {
		backupCount++
		fmt.Printf("💾 Backup %d - Data saved\n", backupCount)
	}, 5*time.Second)

	// Status timeout
	timer.SetTimeout(func() {
		fmt.Printf("📊 Status Report: Heartbeats=%d, Health Checks=%d, Backups=%d\n",
			heartbeatCount, healthCheckCount, backupCount)
	}, 7*time.Second)

	// Run for 10 seconds
	time.Sleep(10 * time.Second)

	// Stop all intervals
	heartbeat()
	healthCheck()
	backup()

	fmt.Println("Heartbeat system stopped")
}

// Debouncer demonstrates debouncing using timers
type Debouncer struct {
	mu     sync.Mutex
	delay  time.Duration
	cancel func()
}

func NewDebouncer(delay time.Duration) *Debouncer {
	return &Debouncer{
		delay: delay,
	}
}

func (d *Debouncer) Debounce(fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Cancel previous timer if it exists
	if d.cancel != nil {
		d.cancel()
	}

	// Set new timer
	d.cancel = timer.SetTimeout(fn, d.delay)
}

// RetryWithTimer demonstrates retry logic using timers
type RetryConfig struct {
	maxRetries int
	delay      time.Duration
}

func NewRetryConfig(maxRetries int, delay time.Duration) *RetryConfig {
	return &RetryConfig{
		maxRetries: maxRetries,
		delay:      delay,
	}
}

func (rc *RetryConfig) Retry(fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < rc.maxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("Retrying in %v (attempt %d/%d)\n", rc.delay, attempt+1, rc.maxRetries)

			// Use timer for delay
			done := make(chan struct{})
			timer.SetTimeout(func() {
				close(done)
			}, rc.delay)
			<-done
		}

		if err := fn(); err != nil {
			lastErr = err
			fmt.Printf("Attempt %d failed: %v\n", attempt+1, err)
			continue
		}

		return nil
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// TaskScheduler demonstrates task scheduling using timers
type TaskScheduler struct {
	mu    sync.Mutex
	tasks map[string]func()
}

func NewTaskScheduler() *TaskScheduler {
	return &TaskScheduler{
		tasks: make(map[string]func()),
	}
}

func (ts *TaskScheduler) ScheduleOnce(name string, fn func(), delay time.Duration) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	cancel := timer.SetTimeout(func() {
		fmt.Printf("Executing scheduled task: %s\n", name)
		fn()

		// Remove from tasks map
		ts.mu.Lock()
		delete(ts.tasks, name)
		ts.mu.Unlock()
	}, delay)

	ts.tasks[name] = cancel
}

func (ts *TaskScheduler) ScheduleRecurring(name string, fn func(), interval time.Duration) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	stop := timer.SetInterval(func() {
		fmt.Printf("Executing recurring task: %s\n", name)
		fn()
	}, interval)

	ts.tasks[name] = stop
}

func (ts *TaskScheduler) Cancel(name string) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if cancel, exists := ts.tasks[name]; exists {
		cancel()
		delete(ts.tasks, name)
		return true
	}

	return false
}

func (ts *TaskScheduler) CancelAll() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	for name, cancel := range ts.tasks {
		cancel()
		delete(ts.tasks, name)
	}
}
