package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/background"
)

// EmailTask represents an email to be sent
type EmailTask struct {
	ID        string    `json:"id"`
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Priority  int       `json:"priority"` // 1 = high, 2 = medium, 3 = low
	CreatedAt time.Time `json:"created_at"`
}

// EmailService handles email sending with worker pools
type EmailService struct {
	highPriorityWorker   *background.Worker[EmailTask]
	mediumPriorityWorker *background.Worker[EmailTask]
	lowPriorityWorker    *background.Worker[EmailTask]
	stop                 func()
	stats                struct {
		sent   int
		failed int
		mu     sync.RWMutex
	}
}

// NewEmailService creates a new email service with priority-based worker pools
func NewEmailService(ctx context.Context) *EmailService {
	service := &EmailService{}

	// Create separate worker pools for different priorities
	highWorker, stopHigh := background.New(ctx, 4, service.sendEmail)
	mediumWorker, stopMedium := background.New(ctx, 2, service.sendEmail)
	lowWorker, stopLow := background.New(ctx, 1, service.sendEmail)

	service.highPriorityWorker = highWorker
	service.mediumPriorityWorker = mediumWorker
	service.lowPriorityWorker = lowWorker

	// Combine all stop functions
	service.stop = func() {
		stopHigh()
		stopMedium()
		stopLow()
	}

	return service
}

// sendEmail simulates sending an email
func (s *EmailService) sendEmail(ctx context.Context, email EmailTask) {
	start := time.Now()

	// Simulate email sending with random delay and failure
	delay := time.Duration(rand.Intn(1000)) * time.Millisecond
	select {
	case <-ctx.Done():
		log.Printf("Email sending cancelled: %s", email.ID)
		return
	case <-time.After(delay):
		// Continue with sending
	}

	// Simulate random failures (10% failure rate)
	if rand.Float32() < 0.1 {
		log.Printf("Failed to send email %s to %s: simulated failure", email.ID, email.To)
		s.incrementFailed()
		return
	}

	duration := time.Since(start)
	log.Printf("Sent email %s to %s (priority: %d) in %v",
		email.ID, email.To, email.Priority, duration)

	s.incrementSent()
}

// SendEmail queues an email for sending based on priority
func (s *EmailService) SendEmail(email EmailTask) error {
	switch email.Priority {
	case 1: // High priority
		return s.highPriorityWorker.Send(email)
	case 2: // Medium priority
		return s.mediumPriorityWorker.Send(email)
	case 3: // Low priority
		return s.lowPriorityWorker.Send(email)
	default:
		return fmt.Errorf("invalid priority: %d", email.Priority)
	}
}

// SendBatch sends multiple emails
func (s *EmailService) SendBatch(emails []EmailTask) error {
	for _, email := range emails {
		if err := s.SendEmail(email); err != nil {
			return fmt.Errorf("failed to queue email %s: %w", email.ID, err)
		}
	}
	return nil
}

// Stats returns email sending statistics
func (s *EmailService) Stats() (sent, failed int) {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()
	return s.stats.sent, s.stats.failed
}

func (s *EmailService) incrementSent() {
	s.stats.mu.Lock()
	s.stats.sent++
	s.stats.mu.Unlock()
}

func (s *EmailService) incrementFailed() {
	s.stats.mu.Lock()
	s.stats.failed++
	s.stats.mu.Unlock()
}

// Close stops the email service
func (s *EmailService) Close() {
	s.stop()
}

// LogProcessingService demonstrates real-time log processing
type LogProcessingService struct {
	worker *background.Worker[LogEntry]
	stop   func()
	stats  struct {
		processed map[string]int
		mu        sync.RWMutex
	}
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Service   string    `json:"service"`
	Message   string    `json:"message"`
	TraceID   string    `json:"trace_id"`
}

func NewLogProcessingService(ctx context.Context) *LogProcessingService {
	service := &LogProcessingService{
		stats: struct {
			processed map[string]int
			mu        sync.RWMutex
		}{
			processed: make(map[string]int),
		},
	}

	// Use CPU count for log processing
	service.worker, service.stop = background.New(ctx, runtime.GOMAXPROCS(0), service.processLog)

	return service
}

func (s *LogProcessingService) processLog(ctx context.Context, entry LogEntry) {
	// Process log entry based on level
	switch entry.Level {
	case "ERROR":
		s.handleError(entry)
	case "WARN":
		s.handleWarning(entry)
	case "INFO":
		s.handleInfo(entry)
	case "DEBUG":
		s.handleDebug(entry)
	}

	s.incrementStat(entry.Level)
	s.incrementStat("total")
}

func (s *LogProcessingService) handleError(entry LogEntry) {
	// In a real system, this might send alerts, write to error log, etc.
	log.Printf("ERROR ALERT: %s - %s (trace: %s)", entry.Service, entry.Message, entry.TraceID)
}

func (s *LogProcessingService) handleWarning(entry LogEntry) {
	// Process warnings
	log.Printf("WARNING: %s - %s", entry.Service, entry.Message)
}

func (s *LogProcessingService) handleInfo(entry LogEntry) {
	// Process info logs
	if entry.Service == "auth" && entry.Message == "user_login" {
		s.incrementStat("user_logins")
	}
}

func (s *LogProcessingService) handleDebug(entry LogEntry) {
	// Debug logs might be ignored in production
	s.incrementStat("debug_ignored")
}

func (s *LogProcessingService) ProcessLog(entry LogEntry) error {
	return s.worker.Send(entry)
}

func (s *LogProcessingService) incrementStat(key string) {
	s.stats.mu.Lock()
	s.stats.processed[key]++
	s.stats.mu.Unlock()
}

func (s *LogProcessingService) Stats() map[string]int {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	stats := make(map[string]int)
	for k, v := range s.stats.processed {
		stats[k] = v
	}
	return stats
}

func (s *LogProcessingService) Close() {
	s.stop()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("=== Background Worker Pool Examples ===")
	fmt.Println()

	// Example 1: Email Service with Priority Queues
	fmt.Println("1. Email Service with Priority-Based Worker Pools")
	demonstrateEmailService(ctx)

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	// Example 2: Log Processing Service
	fmt.Println("2. Real-Time Log Processing Service")
	demonstrateLogProcessing(ctx)

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	// Example 3: File Processing Pipeline
	fmt.Println("3. File Processing Pipeline")
	demonstrateFileProcessing(ctx)

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	// Example 4: Enhanced Features Demo
	fmt.Println("4. Enhanced Features Demo")
	enhancedFeaturesExample()

	fmt.Println()
	fmt.Println("=== All Examples Complete ===")
}

func demonstrateEmailService(ctx context.Context) {
	emailService := NewEmailService(ctx)
	defer emailService.Close()

	// Create sample emails with different priorities
	emails := []EmailTask{
		{
			ID:        "email-001",
			To:        "user1@example.com",
			Subject:   "Critical Alert",
			Body:      "System error detected",
			Priority:  1, // High priority
			CreatedAt: time.Now(),
		},
		{
			ID:        "email-002",
			To:        "user2@example.com",
			Subject:   "Newsletter",
			Body:      "Monthly newsletter",
			Priority:  3, // Low priority
			CreatedAt: time.Now(),
		},
		{
			ID:        "email-003",
			To:        "user3@example.com",
			Subject:   "Password Reset",
			Body:      "Password reset request",
			Priority:  2, // Medium priority
			CreatedAt: time.Now(),
		},
	}

	// Send batch of emails
	if err := emailService.SendBatch(emails); err != nil {
		log.Printf("Failed to send email batch: %v", err)
		return
	}

	// Send more emails to demonstrate load
	for i := 0; i < 20; i++ {
		email := EmailTask{
			ID:        fmt.Sprintf("email-%03d", i+4),
			To:        fmt.Sprintf("user%d@example.com", i+4),
			Subject:   fmt.Sprintf("Test Email %d", i+1),
			Body:      fmt.Sprintf("This is test email number %d", i+1),
			Priority:  rand.Intn(3) + 1, // Random priority 1-3
			CreatedAt: time.Now(),
		}

		if err := emailService.SendEmail(email); err != nil {
			log.Printf("Failed to queue email: %v", err)
		}
	}

	// Wait for processing
	time.Sleep(3 * time.Second)

	// Print statistics
	sent, failed := emailService.Stats()
	fmt.Printf("Email Statistics: Sent: %d, Failed: %d\n", sent, failed)
}

func demonstrateLogProcessing(ctx context.Context) {
	logService := NewLogProcessingService(ctx)
	defer logService.Close()

	// Simulate real-time log entries
	logEntries := []LogEntry{
		{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Service:   "database",
			Message:   "connection timeout",
			TraceID:   "trace-001",
		},
		{
			Timestamp: time.Now(),
			Level:     "INFO",
			Service:   "auth",
			Message:   "user_login",
			TraceID:   "trace-002",
		},
		{
			Timestamp: time.Now(),
			Level:     "WARN",
			Service:   "api",
			Message:   "deprecated endpoint used",
			TraceID:   "trace-003",
		},
		{
			Timestamp: time.Now(),
			Level:     "DEBUG",
			Service:   "cache",
			Message:   "cache miss",
			TraceID:   "trace-004",
		},
	}

	// Process initial logs
	for _, entry := range logEntries {
		if err := logService.ProcessLog(entry); err != nil {
			log.Printf("Failed to process log: %v", err)
		}
	}

	// Simulate continuous log stream
	go func() {
		for i := 0; i < 50; i++ {
			levels := []string{"ERROR", "WARN", "INFO", "DEBUG"}
			services := []string{"auth", "database", "api", "cache", "worker"}

			entry := LogEntry{
				Timestamp: time.Now(),
				Level:     levels[rand.Intn(len(levels))],
				Service:   services[rand.Intn(len(services))],
				Message:   fmt.Sprintf("log message %d", i+1),
				TraceID:   fmt.Sprintf("trace-%03d", i+5),
			}

			logService.ProcessLog(entry)
			time.Sleep(50 * time.Millisecond)
		}
	}()

	// Wait for processing
	time.Sleep(5 * time.Second)

	// Print statistics
	stats := logService.Stats()
	fmt.Printf("Log Processing Statistics:\n")
	for key, value := range stats {
		fmt.Printf("  %s: %d\n", key, value)
	}
}

func demonstrateFileProcessing(ctx context.Context) {
	// File processing task
	type FileTask struct {
		FileName string
		Size     int64
		Data     []byte
	}

	var processedFiles int
	var totalSize int64
	var mu sync.Mutex

	// Create worker pool for file processing
	worker, stop := background.New(ctx, 3, func(ctx context.Context, task FileTask) {
		// Simulate file processing
		processingTime := time.Duration(task.Size/1000) * time.Millisecond
		time.Sleep(processingTime)

		mu.Lock()
		processedFiles++
		totalSize += task.Size
		mu.Unlock()

		log.Printf("Processed file: %s (size: %d bytes)", task.FileName, task.Size)
	})
	defer stop()

	// Create sample files
	files := []FileTask{
		{FileName: "document1.pdf", Size: 1024 * 100, Data: make([]byte, 1024*100)},
		{FileName: "image1.jpg", Size: 1024 * 50, Data: make([]byte, 1024*50)},
		{FileName: "video1.mp4", Size: 1024 * 1024, Data: make([]byte, 1024*1024)},
		{FileName: "archive1.zip", Size: 1024 * 200, Data: make([]byte, 1024*200)},
	}

	// Process files
	for _, file := range files {
		if err := worker.Send(file); err != nil {
			log.Printf("Failed to queue file: %v", err)
		}
	}

	// Wait for processing
	time.Sleep(3 * time.Second)

	// Print statistics
	mu.Lock()
	fmt.Printf("File Processing Statistics:\n")
	fmt.Printf("  Files processed: %d\n", processedFiles)
	fmt.Printf("  Total size: %.2f MB\n", float64(totalSize)/(1024*1024))
	mu.Unlock()
}

// Example 4: Enhanced Features Demo
func enhancedFeaturesExample() {
	fmt.Println("=== Enhanced Features Demo ===")

	ctx := context.Background()

	// Create worker with advanced options
	opts := background.Options{
		WorkerCount:   4,
		BufferSize:    50,
		WorkerTimeout: 2 * time.Second,
		OnError: func(task interface{}, recovered interface{}) {
			fmt.Printf("Task panic recovered: %v (task: %v)\n", recovered, task)
		},
		OnTaskComplete: func(task interface{}, duration time.Duration) {
			fmt.Printf("Task completed in %v\n", duration)
		},
	}

	worker, stop := background.NewWithOptions(ctx, opts, func(ctx context.Context, task string) {
		switch task {
		case "panic":
			panic("intentional panic for demo")
		case "timeout":
			time.Sleep(3 * time.Second) // Will timeout
		case "slow":
			time.Sleep(100 * time.Millisecond)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	})
	defer stop()

	// Send various types of tasks
	tasks := []string{"normal", "slow", "panic", "timeout", "normal"}

	for _, task := range tasks {
		if task == "panic" {
			// Use TrySend for potentially problematic tasks
			if worker.TrySend(task) {
				fmt.Printf("Sent task: %s\n", task)
			} else {
				fmt.Printf("Failed to send task: %s\n", task)
			}
		} else {
			worker.Send(task)
			fmt.Printf("Sent task: %s\n", task)
		}
	}

	// Show metrics
	time.Sleep(1 * time.Second)
	metrics := worker.Metrics()
	fmt.Printf("Metrics: Queued=%d, Processed=%d, Rejected=%d, Active=%d\n",
		metrics.TasksQueued, metrics.TasksProcessed, metrics.TasksRejected, metrics.ActiveWorkers)

	// Wait for remaining tasks to complete
	time.Sleep(2 * time.Second)

	// Final metrics
	metrics = worker.Metrics()
	fmt.Printf("Final Metrics: Queued=%d, Processed=%d, Rejected=%d, Active=%d\n",
		metrics.TasksQueued, metrics.TasksProcessed, metrics.TasksRejected, metrics.ActiveWorkers)
}
