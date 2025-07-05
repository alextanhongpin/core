package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/poll"
)

func main() {
	// Example 1: Basic polling
	fmt.Println("=== Basic Polling Example ===")
	basicPollingExample()

	// Example 2: Queue processing
	fmt.Println("\n=== Queue Processing Example ===")
	queueProcessingExample()

	// Example 3: Custom backoff strategy
	fmt.Println("\n=== Custom Backoff Example ===")
	customBackoffExample()

	// Example 4: File monitoring
	fmt.Println("\n=== File Monitoring Example ===")
	fileMonitoringExample()
}

func basicPollingExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a simple poller
	poller := poll.New()

	counter := 0
	events, stop := poller.PollWithContext(ctx, func(ctx context.Context) error {
		counter++
		fmt.Printf("Poll iteration %d\n", counter)

		// Simulate work
		time.Sleep(100 * time.Millisecond)

		// Stop after 5 iterations
		if counter >= 5 {
			return poll.EOQ // End of queue
		}

		// Occasionally return empty to test backoff
		if counter%3 == 0 {
			return poll.Empty
		}

		return nil
	})

	// Monitor events
	go func() {
		for event := range events {
			fmt.Printf("Event: %+v\n", event)
		}
	}()

	// Wait for polling to complete
	select {
	case <-ctx.Done():
		fmt.Println("Polling timed out")
	case <-time.After(5 * time.Second):
		fmt.Println("Polling completed")
	}

	stop()
}

func queueProcessingExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Simulate a message queue
	queue := NewMessageQueue()

	// Add some messages
	for i := 0; i < 20; i++ {
		queue.Add(fmt.Sprintf("Message %d", i))
	}

	// Configure poller for queue processing
	poller := &poll.Poll{
		BatchSize:        5,
		FailureThreshold: 3,
		BackOff:          poll.ExponentialBackOff,
		MaxConcurrency:   2,
	}

	events, stop := poller.PollWithContext(ctx, func(ctx context.Context) error {
		messages := queue.GetBatch(3)
		if len(messages) == 0 {
			return poll.Empty
		}

		// Process messages
		for _, msg := range messages {
			fmt.Printf("Processing: %s\n", msg)
			time.Sleep(200 * time.Millisecond)

			// Simulate occasional failures
			if rand.Float32() < 0.1 {
				return fmt.Errorf("processing failed for %s", msg)
			}
		}

		return nil
	})

	// Monitor events
	go func() {
		for event := range events {
			fmt.Printf("Queue Event: %+v\n", event)
		}
	}()

	// Wait for processing to complete
	select {
	case <-ctx.Done():
		fmt.Println("Queue processing timed out")
	case <-time.After(12 * time.Second):
		fmt.Println("Queue processing completed")
	}

	stop()
}

func customBackoffExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// Custom backoff strategy
	customBackoff := func(idle int) time.Duration {
		switch {
		case idle < 3:
			return 100 * time.Millisecond
		case idle < 6:
			return 500 * time.Millisecond
		default:
			return 2 * time.Second
		}
	}

	poller := &poll.Poll{
		BatchSize:        10,
		FailureThreshold: 5,
		BackOff:          customBackoff,
		MaxConcurrency:   1,
	}

	attempts := 0
	events, stop := poller.PollWithContext(ctx, func(ctx context.Context) error {
		attempts++
		fmt.Printf("Attempt %d\n", attempts)

		if attempts >= 8 {
			return poll.EOQ
		}

		// Always return empty to test backoff
		return poll.Empty
	})

	// Monitor events
	go func() {
		for event := range events {
			fmt.Printf("Backoff Event: %+v\n", event)
		}
	}()

	// Wait for completion
	select {
	case <-ctx.Done():
		fmt.Println("Custom backoff example timed out")
	case <-time.After(7 * time.Second):
		fmt.Println("Custom backoff example completed")
	}

	stop()
}

func fileMonitoringExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Simulate file monitoring
	fileMonitor := NewFileMonitor()

	// Add some files to monitor
	files := []string{
		"/tmp/file1.txt",
		"/tmp/file2.txt",
		"/tmp/file3.txt",
	}

	for _, file := range files {
		fileMonitor.AddFile(file)
	}

	// Configure poller for file monitoring
	poller := &poll.Poll{
		BatchSize:        50,
		FailureThreshold: 10,
		BackOff: func(idle int) time.Duration {
			// Check files every 2 seconds when idle
			return 2 * time.Second
		},
		MaxConcurrency: 3,
	}

	events, stop := poller.PollWithContext(ctx, func(ctx context.Context) error {
		changes := fileMonitor.CheckForChanges()
		if len(changes) == 0 {
			return poll.Empty
		}

		// Process file changes
		for _, change := range changes {
			fmt.Printf("File changed: %s\n", change)
			time.Sleep(100 * time.Millisecond)
		}

		return nil
	})

	// Monitor events
	go func() {
		for event := range events {
			fmt.Printf("File Monitor Event: %+v\n", event)
		}
	}()

	// Simulate file changes
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Simulate file change
				file := files[rand.Intn(len(files))]
				fileMonitor.SimulateChange(file)
			}
		}
	}()

	// Wait for monitoring to complete
	select {
	case <-ctx.Done():
		fmt.Println("File monitoring timed out")
	case <-time.After(9 * time.Second):
		fmt.Println("File monitoring completed")
	}

	stop()
}

// MessageQueue simulates a simple message queue
type MessageQueue struct {
	mu       sync.Mutex
	messages []string
}

func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		messages: make([]string, 0),
	}
}

func (mq *MessageQueue) Add(message string) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.messages = append(mq.messages, message)
}

func (mq *MessageQueue) GetBatch(size int) []string {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if len(mq.messages) == 0 {
		return nil
	}

	batchSize := size
	if len(mq.messages) < batchSize {
		batchSize = len(mq.messages)
	}

	batch := mq.messages[:batchSize]
	mq.messages = mq.messages[batchSize:]

	return batch
}

// FileMonitor simulates file monitoring
type FileMonitor struct {
	mu      sync.Mutex
	files   map[string]time.Time
	changes []string
}

func NewFileMonitor() *FileMonitor {
	return &FileMonitor{
		files:   make(map[string]time.Time),
		changes: make([]string, 0),
	}
}

func (fm *FileMonitor) AddFile(filename string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.files[filename] = time.Now()
}

func (fm *FileMonitor) SimulateChange(filename string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if _, exists := fm.files[filename]; exists {
		fm.changes = append(fm.changes, filename)
	}
}

func (fm *FileMonitor) CheckForChanges() []string {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if len(fm.changes) == 0 {
		return nil
	}

	changes := make([]string, len(fm.changes))
	copy(changes, fm.changes)
	fm.changes = fm.changes[:0] // Clear changes

	return changes
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
