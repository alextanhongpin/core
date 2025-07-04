package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/debounce"
)

// SearchService demonstrates search input debouncing
type SearchService struct {
	debouncer *debounce.Group
	lastQuery string
	mu        sync.Mutex
}

func NewSearchService() *SearchService {
	return &SearchService{
		debouncer: &debounce.Group{
			Timeout: 500 * time.Millisecond, // Wait 500ms after last input
		},
	}
}

func (s *SearchService) Search(query string) {
	s.mu.Lock()
	s.lastQuery = query
	s.mu.Unlock()

	// Debounce search requests
	s.debouncer.Do(func() {
		s.mu.Lock()
		currentQuery := s.lastQuery
		s.mu.Unlock()

		s.performSearch(currentQuery)
	})
}

func (s *SearchService) performSearch(query string) {
	if query == "" {
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

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
	Service   string
}

// LogAggregator demonstrates log batching with debouncing
type LogAggregator struct {
	debouncer *debounce.Group
	buffer    []LogEntry
	mu        sync.Mutex
}

func NewLogAggregator() *LogAggregator {
	return &LogAggregator{
		debouncer: &debounce.Group{
			Every:   20,              // Flush every 20 logs
			Timeout: 3 * time.Second, // Or every 3 seconds
		},
		buffer: make([]LogEntry, 0),
	}
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
	time.Sleep(50 * time.Millisecond)
	fmt.Println("💾 Logs written to storage")
	fmt.Println()
}

// EventProcessor demonstrates event processing with different debounce strategies
type EventProcessor struct {
	highPriorityDebouncer *debounce.Group
	normalDebouncer       *debounce.Group
	batchDebouncer        *debounce.Group

	highPriorityEvents []string
	normalEvents       []string
	batchEvents        []string

	mu sync.Mutex
}

func NewEventProcessor() *EventProcessor {
	return &EventProcessor{
		// High priority: execute immediately on every call
		highPriorityDebouncer: &debounce.Group{
			Every: 1,
		},
		// Normal priority: execute every 3 events or after 2 seconds
		normalDebouncer: &debounce.Group{
			Every:   3,
			Timeout: 2 * time.Second,
		},
		// Batch processing: execute every 10 events or after 5 seconds
		batchDebouncer: &debounce.Group{
			Every:   10,
			Timeout: 5 * time.Second,
		},

		highPriorityEvents: make([]string, 0),
		normalEvents:       make([]string, 0),
		batchEvents:        make([]string, 0),
	}
}

func (ep *EventProcessor) ProcessHighPriority(event string) {
	ep.mu.Lock()
	ep.highPriorityEvents = append(ep.highPriorityEvents, event)
	ep.mu.Unlock()

	ep.highPriorityDebouncer.Do(func() {
		ep.processHighPriorityBatch()
	})
}

func (ep *EventProcessor) ProcessNormal(event string) {
	ep.mu.Lock()
	ep.normalEvents = append(ep.normalEvents, event)
	ep.mu.Unlock()

	ep.normalDebouncer.Do(func() {
		ep.processNormalBatch()
	})
}

func (ep *EventProcessor) ProcessBatch(event string) {
	ep.mu.Lock()
	ep.batchEvents = append(ep.batchEvents, event)
	ep.mu.Unlock()

	ep.batchDebouncer.Do(func() {
		ep.processBatchEvents()
	})
}

func (ep *EventProcessor) processHighPriorityBatch() {
	ep.mu.Lock()
	if len(ep.highPriorityEvents) == 0 {
		ep.mu.Unlock()
		return
	}

	events := make([]string, len(ep.highPriorityEvents))
	copy(events, ep.highPriorityEvents)
	ep.highPriorityEvents = ep.highPriorityEvents[:0]
	ep.mu.Unlock()

	fmt.Printf("🚨 Processing %d HIGH PRIORITY events: %v\n", len(events), events)
}

func (ep *EventProcessor) processNormalBatch() {
	ep.mu.Lock()
	if len(ep.normalEvents) == 0 {
		ep.mu.Unlock()
		return
	}

	events := make([]string, len(ep.normalEvents))
	copy(events, ep.normalEvents)
	ep.normalEvents = ep.normalEvents[:0]
	ep.mu.Unlock()

	fmt.Printf("⚡ Processing %d NORMAL events: %v\n", len(events), events)
}

func (ep *EventProcessor) processBatchEvents() {
	ep.mu.Lock()
	if len(ep.batchEvents) == 0 {
		ep.mu.Unlock()
		return
	}

	events := make([]string, len(ep.batchEvents))
	copy(events, ep.batchEvents)
	ep.batchEvents = ep.batchEvents[:0]
	ep.mu.Unlock()

	fmt.Printf("📦 Processing %d BATCH events: %v\n", len(events), events)
}

func main() {
	fmt.Println("=== Debounce Package Examples ===")
	fmt.Println()

	// Example 1: Search Input Debouncing
	fmt.Println("1. Search Input Debouncing")
	demonstrateSearchDebouncing()

	fmt.Println()
	fmt.Println("---")
	fmt.Println()

	// Example 2: Log Aggregation
	fmt.Println("2. Log Aggregation with Debouncing")
	demonstrateLogAggregation()

	fmt.Println()
	fmt.Println("---")
	fmt.Println()

	// Example 3: Event Processing with Different Priorities
	fmt.Println("3. Event Processing with Different Debounce Strategies")
	demonstrateEventProcessing()

	fmt.Println()
	fmt.Println("---")
	fmt.Println()

	// Example 4: Simple Count-Based Debouncing
	fmt.Println("4. Simple Count-Based Debouncing")
	demonstrateCountBasedDebouncing()

	fmt.Println()
	fmt.Println("=== All Examples Complete ===")
}

func demonstrateSearchDebouncing() {
	searchService := NewSearchService()

	// Simulate user typing "hello world"
	queries := []string{"h", "he", "hel", "hell", "hello", "hello ", "hello w", "hello wo", "hello wor", "hello worl", "hello world"}

	fmt.Println("User typing 'hello world' character by character...")
	fmt.Println("(Search will only execute after user stops typing for 500ms)")
	fmt.Println()

	for i, query := range queries {
		fmt.Printf("👤 User types: '%s'\n", query)
		searchService.Search(query)

		// Simulate typing delay
		if i < len(queries)-1 {
			time.Sleep(200 * time.Millisecond) // Typing faster than debounce timeout
		}
	}

	// Wait for final search
	fmt.Println("User stops typing...")
	time.Sleep(600 * time.Millisecond)
}

func demonstrateLogAggregation() {
	aggregator := NewLogAggregator()

	services := []string{"auth", "api", "database", "cache", "worker"}
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}

	fmt.Println("Generating log entries...")
	fmt.Printf("(Logs will be flushed every 20 entries or every 3 seconds)\n")
	fmt.Println()

	// Generate 55 log entries with varying intervals
	for i := 0; i < 55; i++ {
		entry := LogEntry{
			Timestamp: time.Now(),
			Level:     levels[i%len(levels)],
			Message:   fmt.Sprintf("Log message %d", i+1),
			Service:   services[i%len(services)],
		}

		aggregator.Log(entry)

		// Print every 10th log for demonstration
		if i%10 == 0 {
			fmt.Printf("📝 Generated %d logs so far...\n", i+1)
		}

		// Simulate varying log generation rates
		if i%25 == 0 {
			time.Sleep(100 * time.Millisecond) // Slower rate occasionally
		} else {
			time.Sleep(20 * time.Millisecond) // Normal rate
		}
	}

	// Wait for final flush
	time.Sleep(4 * time.Second)
}

func demonstrateEventProcessing() {
	processor := NewEventProcessor()

	fmt.Println("Processing events with different priorities:")
	fmt.Println("- High Priority: Execute immediately (every 1 event)")
	fmt.Println("- Normal: Execute every 3 events or after 2 seconds")
	fmt.Println("- Batch: Execute every 10 events or after 5 seconds")
	fmt.Println()

	// Generate various events
	go func() {
		// High priority events (should execute immediately)
		highPriorityEvents := []string{"system_error", "security_alert", "service_down"}
		for i, event := range highPriorityEvents {
			processor.ProcessHighPriority(fmt.Sprintf("%s_%d", event, i+1))
			time.Sleep(500 * time.Millisecond)
		}
	}()

	go func() {
		// Normal priority events
		for i := 0; i < 7; i++ {
			processor.ProcessNormal(fmt.Sprintf("user_action_%d", i+1))
			time.Sleep(300 * time.Millisecond)
		}
	}()

	go func() {
		// Batch events
		for i := 0; i < 25; i++ {
			processor.ProcessBatch(fmt.Sprintf("analytics_event_%d", i+1))
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Wait for all processing to complete
	time.Sleep(8 * time.Second)
}

func demonstrateCountBasedDebouncing() {
	fmt.Println("Count-based debouncing example:")
	fmt.Println("Function will execute every 5 calls")
	fmt.Println()

	var executions int
	debouncer := &debounce.Group{
		Every: 5, // Execute every 5 calls
	}

	// Make 12 calls
	for i := 1; i <= 12; i++ {
		debouncer.Do(func() {
			executions++
			fmt.Printf("🎯 Execution #%d (triggered by call #%d)\n", executions, i)
		})

		fmt.Printf("📞 Call #%d\n", i)
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Printf("\nTotal function executions: %d (out of 12 calls)\n", executions)
}

// Additional utility functions for more examples

// SaveDataDebouncer demonstrates file saving with debouncing
type SaveDataDebouncer struct {
	debouncer *debounce.Group
	data      map[string]interface{}
	mu        sync.RWMutex
}

func NewSaveDataDebouncer() *SaveDataDebouncer {
	return &SaveDataDebouncer{
		debouncer: &debounce.Group{
			Timeout: 1 * time.Second, // Save after 1 second of inactivity
		},
		data: make(map[string]interface{}),
	}
}

func (sdd *SaveDataDebouncer) UpdateData(key string, value interface{}) {
	sdd.mu.Lock()
	sdd.data[key] = value
	sdd.mu.Unlock()

	// Debounce save operation
	sdd.debouncer.Do(func() {
		sdd.saveData()
	})
}

func (sdd *SaveDataDebouncer) saveData() {
	sdd.mu.RLock()
	dataCopy := make(map[string]interface{})
	for k, v := range sdd.data {
		dataCopy[k] = v
	}
	sdd.mu.RUnlock()

	fmt.Printf("💾 Saving data: %v\n", dataCopy)
	// Simulate save operation
	time.Sleep(100 * time.Millisecond)
}

// NetworkRequestBatcher demonstrates API request batching
type NetworkRequestBatcher struct {
	debouncer *debounce.Group
	requests  []string
	mu        sync.Mutex
}

func NewNetworkRequestBatcher() *NetworkRequestBatcher {
	return &NetworkRequestBatcher{
		debouncer: &debounce.Group{
			Every:   5,               // Batch every 5 requests
			Timeout: 2 * time.Second, // Or every 2 seconds
		},
		requests: make([]string, 0),
	}
}

func (nrb *NetworkRequestBatcher) AddRequest(request string) {
	nrb.mu.Lock()
	nrb.requests = append(nrb.requests, request)
	nrb.mu.Unlock()

	nrb.debouncer.Do(func() {
		nrb.processBatch()
	})
}

func (nrb *NetworkRequestBatcher) processBatch() {
	nrb.mu.Lock()
	if len(nrb.requests) == 0 {
		nrb.mu.Unlock()
		return
	}

	batch := make([]string, len(nrb.requests))
	copy(batch, nrb.requests)
	nrb.requests = nrb.requests[:0]
	nrb.mu.Unlock()

	fmt.Printf("🌐 Processing batch of %d network requests: %v\n", len(batch), batch)
	// Simulate network request
	time.Sleep(200 * time.Millisecond)
}
