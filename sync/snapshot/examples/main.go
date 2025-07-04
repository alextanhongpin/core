package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/snapshot"
)

func main() {
	// Example 1: Basic snapshot usage
	fmt.Println("=== Basic Snapshot Example ===")
	basicSnapshotExample()

	// Example 2: In-memory database with snapshots
	fmt.Println("\n=== In-Memory Database Example ===")
	inMemoryDBExample()

	// Example 3: Configuration manager with snapshots
	fmt.Println("\n=== Configuration Manager Example ===")
	configManagerExample()

	// Example 4: Custom snapshot policies
	fmt.Println("\n=== Custom Snapshot Policies Example ===")
	customPoliciesExample()
}

func basicSnapshotExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create snapshot handler
	snapshotCount := 0
	handler := func(ctx context.Context, evt snapshot.Event) {
		snapshotCount++
		fmt.Printf("Snapshot %d: Count=%d, Policy=%+v\n",
			snapshotCount, evt.Count, evt.Policy)
	}

	// Create snapshot manager with default policies
	snap, stop := snapshot.New(ctx, handler)
	defer stop()

	// Simulate data changes
	fmt.Println("Simulating 1500 data changes...")
	for i := 0; i < 1500; i++ {
		snap.Inc(1) // Notify of a change

		// Vary the speed of changes
		if i%100 == 0 {
			time.Sleep(100 * time.Millisecond)
		} else {
			time.Sleep(5 * time.Millisecond)
		}
	}

	// Wait for potential snapshots
	time.Sleep(3 * time.Second)
	fmt.Printf("Total snapshots taken: %d\n", snapshotCount)
}

func inMemoryDBExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Create an in-memory database
	db := NewInMemoryDB(ctx)
	defer db.Close()

	// Simulate database operations
	fmt.Println("Simulating database operations...")

	// Insert operations
	for i := 0; i < 150; i++ {
		key := fmt.Sprintf("user:%d", i)
		value := fmt.Sprintf("User %d", i)
		db.Set(key, value)

		// Occasionally delete some keys
		if i%20 == 0 && i > 0 {
			deleteKey := fmt.Sprintf("user:%d", i-10)
			db.Delete(deleteKey)
		}

		time.Sleep(50 * time.Millisecond)
	}

	// Show final statistics
	fmt.Printf("Database size: %d entries\n", db.Size())
	fmt.Printf("Total snapshots: %d\n", db.SnapshotCount())

	// Wait for final snapshots
	time.Sleep(2 * time.Second)
}

func configManagerExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create configuration manager
	cm := NewConfigManager(ctx)
	defer cm.Close()

	// Simulate configuration changes
	configs := []struct {
		key   string
		value string
	}{
		{"database.host", "localhost"},
		{"database.port", "5432"},
		{"database.name", "myapp"},
		{"redis.host", "localhost"},
		{"redis.port", "6379"},
		{"api.timeout", "30s"},
		{"log.level", "info"},
		{"debug.enabled", "false"},
		{"cache.ttl", "1h"},
		{"max.connections", "100"},
	}

	fmt.Println("Simulating configuration changes...")
	for i, cfg := range configs {
		cm.Set(cfg.key, cfg.value)

		// Occasionally update existing configs
		if i%3 == 0 && i > 0 {
			cm.Set("database.host", "updated-host")
		}

		time.Sleep(1 * time.Second)
	}

	// Show final config
	fmt.Printf("Final configuration size: %d entries\n", cm.Size())

	// Wait for final snapshot
	time.Sleep(3 * time.Second)
}

func customPoliciesExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	// Create snapshot handler
	handler := func(ctx context.Context, evt snapshot.Event) {
		fmt.Printf("Custom snapshot: Count=%d, Policy=%+v\n",
			evt.Count, evt.Policy)
	}

	// Create snapshot manager with custom policies
	customPolicies := []snapshot.Policy{
		{Every: 5, Interval: 2 * time.Second}, // After 5 changes, wait 2s
		{Every: 1, Interval: 8 * time.Second}, // After 1 change, wait 8s
	}

	snap, stop := snapshot.New(ctx, handler, customPolicies...)
	defer stop()

	// Simulate changes with different patterns
	fmt.Println("Pattern 1: Rapid changes")
	for i := 0; i < 10; i++ {
		snap.Inc(1)
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(3 * time.Second)

	fmt.Println("Pattern 2: Slow changes")
	for i := 0; i < 3; i++ {
		snap.Inc(1)
		time.Sleep(3 * time.Second)
	}

	// Wait for final snapshots
	time.Sleep(5 * time.Second)
}

// InMemoryDB demonstrates snapshot usage with an in-memory database
type InMemoryDB struct {
	data          map[string]string
	mu            sync.RWMutex
	snapshot      *snapshot.Background
	stop          func()
	snapshotCount int
}

func NewInMemoryDB(ctx context.Context) *InMemoryDB {
	db := &InMemoryDB{
		data: make(map[string]string),
	}

	// Create snapshot manager
	db.snapshot, db.stop = snapshot.New(ctx, db.saveSnapshot,
		snapshot.Policy{Every: 100, Interval: 2 * time.Second}, // High frequency
		snapshot.Policy{Every: 10, Interval: 5 * time.Second},  // Medium frequency
		snapshot.Policy{Every: 1, Interval: 10 * time.Second},  // Low frequency
	)

	return db
}

func (db *InMemoryDB) Close() {
	db.stop()
}

func (db *InMemoryDB) Set(key, value string) {
	db.mu.Lock()
	db.data[key] = value
	db.mu.Unlock()

	// Notify snapshot manager
	db.snapshot.Touch()
}

func (db *InMemoryDB) Get(key string) (string, bool) {
	db.mu.RLock()
	value, exists := db.data[key]
	db.mu.RUnlock()
	return value, exists
}

func (db *InMemoryDB) Delete(key string) {
	db.mu.Lock()
	delete(db.data, key)
	db.mu.Unlock()

	// Notify snapshot manager
	db.snapshot.Touch()
}

func (db *InMemoryDB) Size() int {
	db.mu.RLock()
	size := len(db.data)
	db.mu.RUnlock()
	return size
}

func (db *InMemoryDB) SnapshotCount() int {
	db.mu.RLock()
	count := db.snapshotCount
	db.mu.RUnlock()
	return count
}

func (db *InMemoryDB) saveSnapshot(ctx context.Context, evt snapshot.Event) {
	db.mu.Lock()
	db.snapshotCount++
	count := db.snapshotCount
	size := len(db.data)
	db.mu.Unlock()

	fmt.Printf("📸 Database snapshot %d: %d entries, %d changes, policy=%+v\n",
		count, size, evt.Count, evt.Policy)
}

// ConfigManager demonstrates snapshot usage with configuration management
type ConfigManager struct {
	config   map[string]string
	mu       sync.RWMutex
	snapshot *snapshot.Background
	stop     func()
}

func NewConfigManager(ctx context.Context) *ConfigManager {
	cm := &ConfigManager{
		config: make(map[string]string),
	}

	// Create snapshot manager with conservative policies for config
	cm.snapshot, cm.stop = snapshot.New(ctx, cm.saveConfig,
		snapshot.Policy{Every: 5, Interval: 3 * time.Second},  // Save after 5 changes
		snapshot.Policy{Every: 1, Interval: 10 * time.Second}, // Save after 1 change, wait 10s
	)

	return cm
}

func (cm *ConfigManager) Close() {
	cm.stop()
}

func (cm *ConfigManager) Set(key, value string) {
	cm.mu.Lock()
	cm.config[key] = value
	cm.mu.Unlock()

	fmt.Printf("Config updated: %s = %s\n", key, value)
	cm.snapshot.Touch()
}

func (cm *ConfigManager) Get(key string) (string, bool) {
	cm.mu.RLock()
	value, exists := cm.config[key]
	cm.mu.RUnlock()
	return value, exists
}

func (cm *ConfigManager) Size() int {
	cm.mu.RLock()
	size := len(cm.config)
	cm.mu.RUnlock()
	return size
}

func (cm *ConfigManager) saveConfig(ctx context.Context, evt snapshot.Event) {
	cm.mu.RLock()
	size := len(cm.config)
	cm.mu.RUnlock()

	fmt.Printf("💾 Config snapshot: %d entries, %d changes, policy=%+v\n",
		size, evt.Count, evt.Policy)
}

// EventCounter demonstrates snapshot usage with event counting
type EventCounter struct {
	events   map[string]int
	mu       sync.RWMutex
	snapshot *snapshot.Background
	stop     func()
}

func NewEventCounter(ctx context.Context) *EventCounter {
	ec := &EventCounter{
		events: make(map[string]int),
	}

	// High-frequency snapshots for event counting
	ec.snapshot, ec.stop = snapshot.New(ctx, ec.saveSnapshot,
		snapshot.Policy{Every: 50, Interval: 1 * time.Second},
		snapshot.Policy{Every: 10, Interval: 5 * time.Second},
	)

	return ec
}

func (ec *EventCounter) Close() {
	ec.stop()
}

func (ec *EventCounter) Increment(eventType string) {
	ec.mu.Lock()
	ec.events[eventType]++
	ec.mu.Unlock()

	ec.snapshot.Touch()
}

func (ec *EventCounter) Get(eventType string) int {
	ec.mu.RLock()
	count := ec.events[eventType]
	ec.mu.RUnlock()
	return count
}

func (ec *EventCounter) saveSnapshot(ctx context.Context, evt snapshot.Event) {
	ec.mu.RLock()
	totalEvents := 0
	for _, count := range ec.events {
		totalEvents += count
	}
	ec.mu.RUnlock()

	fmt.Printf("📊 Event snapshot: %d total events, %d changes, policy=%+v\n",
		totalEvents, evt.Count, evt.Policy)
}
