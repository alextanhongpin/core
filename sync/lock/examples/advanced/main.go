package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/lock"
)

// UserService demonstrates a real-world service using named locks
type UserService struct {
	locks *lock.Lock
	users map[string]*User
	mu    sync.RWMutex
}

type User struct {
	ID      string
	Name    string
	Balance int64
}

func NewUserService() *UserService {
	opts := lock.Options{
		DefaultTimeout:  5 * time.Second,
		CleanupInterval: 1 * time.Minute,
		LockType:        lock.Mutex,
		MaxLocks:        10000,
		OnLockAcquired: func(key string, waitTime time.Duration) {
			if waitTime > 100*time.Millisecond {
				log.Printf("Lock acquired for %s after %v (potential contention)", key, waitTime)
			}
		},
		OnLockReleased: func(key string, holdTime time.Duration) {
			if holdTime > 1*time.Second {
				log.Printf("Lock held for %s for %v (long operation)", key, holdTime)
			}
		},
		OnLockContention: func(key string, waitingGoroutines int) {
			log.Printf("Lock contention detected for %s: %d goroutines waiting", key, waitingGoroutines)
		},
	}

	return &UserService{
		locks: lock.NewWithOptions(opts),
		users: make(map[string]*User),
	}
}

func (s *UserService) Stop() {
	s.locks.Stop()
}

// GetUser retrieves a user (no locking needed for reads)
func (s *UserService) GetUser(id string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.users[id]
}

// CreateUser creates a new user with per-user locking
func (s *UserService) CreateUser(id, name string) error {
	lockKey := fmt.Sprintf("user:%s", id)

	unlock, err := s.locks.LockWithTimeout(lockKey, 2*time.Second)
	if err != nil {
		return fmt.Errorf("failed to acquire lock for user %s: %w", id, err)
	}
	defer unlock()

	// Check if user already exists
	if s.GetUser(id) != nil {
		return fmt.Errorf("user %s already exists", id)
	}

	// Create user
	user := &User{
		ID:      id,
		Name:    name,
		Balance: 0,
	}

	s.mu.Lock()
	s.users[id] = user
	s.mu.Unlock()

	log.Printf("Created user %s", id)
	return nil
}

// UpdateBalance updates user balance with per-user locking
func (s *UserService) UpdateBalance(id string, amount int64) error {
	lockKey := fmt.Sprintf("user:%s", id)

	unlock, err := s.locks.LockWithTimeout(lockKey, 2*time.Second)
	if err != nil {
		return fmt.Errorf("failed to acquire lock for user %s: %w", id, err)
	}
	defer unlock()

	user := s.GetUser(id)
	if user == nil {
		return fmt.Errorf("user %s not found", id)
	}

	user.Balance += amount
	log.Printf("Updated balance for user %s: %d", id, user.Balance)
	return nil
}

// TransferBalance transfers balance between users with ordered locking to avoid deadlocks
func (s *UserService) TransferBalance(fromID, toID string, amount int64) error {
	if fromID == toID {
		return fmt.Errorf("cannot transfer to same user")
	}

	// Order locks by ID to prevent deadlocks
	firstID, secondID := fromID, toID
	if fromID > toID {
		firstID, secondID = toID, fromID
	}

	firstLockKey := fmt.Sprintf("user:%s", firstID)
	secondLockKey := fmt.Sprintf("user:%s", secondID)

	// Acquire first lock
	firstUnlock, err := s.locks.LockWithTimeout(firstLockKey, 2*time.Second)
	if err != nil {
		return fmt.Errorf("failed to acquire first lock: %w", err)
	}
	defer firstUnlock()

	// Acquire second lock
	secondUnlock, err := s.locks.LockWithTimeout(secondLockKey, 2*time.Second)
	if err != nil {
		return fmt.Errorf("failed to acquire second lock: %w", err)
	}
	defer secondUnlock()

	fromUser := s.GetUser(fromID)
	toUser := s.GetUser(toID)

	if fromUser == nil {
		return fmt.Errorf("from user %s not found", fromID)
	}
	if toUser == nil {
		return fmt.Errorf("to user %s not found", toID)
	}

	if fromUser.Balance < amount {
		return fmt.Errorf("insufficient balance for user %s", fromID)
	}

	fromUser.Balance -= amount
	toUser.Balance += amount

	log.Printf("Transferred %d from %s to %s", amount, fromID, toID)
	return nil
}

// BatchUpdateBalances updates multiple users' balances concurrently
func (s *UserService) BatchUpdateBalances(updates map[string]int64) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(updates))

	for userID, amount := range updates {
		wg.Add(1)
		go func(id string, amt int64) {
			defer wg.Done()
			if err := s.UpdateBalance(id, amt); err != nil {
				errChan <- err
			}
		}(userID, amount)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// GetMetrics returns lock metrics
func (s *UserService) GetMetrics() lock.Metrics {
	return s.locks.Metrics()
}

// CacheService demonstrates using RWMutex locks for cache operations
type CacheService struct {
	locks *lock.Lock
	cache map[string]interface{}
	mu    sync.RWMutex
}

func NewCacheService() *CacheService {
	opts := lock.Options{
		LockType:        lock.RWMutex,
		CleanupInterval: 30 * time.Second,
		MaxLocks:        1000,
	}

	return &CacheService{
		locks: lock.NewWithOptions(opts),
		cache: make(map[string]interface{}),
	}
}

func (c *CacheService) Stop() {
	c.locks.Stop()
}

func (c *CacheService) Get(key string) interface{} {
	lockKey := fmt.Sprintf("cache:%s", key)
	rwMutex := c.locks.GetRW(lockKey)

	rwMutex.RLock()
	defer rwMutex.RUnlock()

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cache[key]
}

func (c *CacheService) Set(key string, value interface{}) {
	lockKey := fmt.Sprintf("cache:%s", key)
	rwMutex := c.locks.GetRW(lockKey)

	rwMutex.Lock()
	defer rwMutex.Unlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = value
}

func main() {
	// Example 1: User Service with per-user locking
	fmt.Println("=== User Service Example ===")
	userService := NewUserService()
	defer userService.Stop()

	// Create users
	users := []string{"alice", "bob", "charlie", "david"}
	for _, user := range users {
		if err := userService.CreateUser(user, fmt.Sprintf("User %s", user)); err != nil {
			log.Printf("Error creating user %s: %v", user, err)
		}
	}

	// Update balances concurrently
	updates := map[string]int64{
		"alice":   1000,
		"bob":     2000,
		"charlie": 1500,
		"david":   500,
	}

	if err := userService.BatchUpdateBalances(updates); err != nil {
		log.Printf("Error updating balances: %v", err)
	}

	// Perform transfers
	transfers := []struct {
		from, to string
		amount   int64
	}{
		{"alice", "bob", 200},
		{"charlie", "david", 300},
		{"bob", "alice", 100},
	}

	for _, transfer := range transfers {
		if err := userService.TransferBalance(transfer.from, transfer.to, transfer.amount); err != nil {
			log.Printf("Error transferring from %s to %s: %v", transfer.from, transfer.to, err)
		}
	}

	// Print final balances
	fmt.Println("\nFinal balances:")
	for _, userID := range users {
		user := userService.GetUser(userID)
		if user != nil {
			fmt.Printf("  %s: %d\n", user.Name, user.Balance)
		}
	}

	// Print metrics
	metrics := userService.GetMetrics()
	fmt.Printf("\nUser Service Metrics:\n")
	fmt.Printf("  Active Locks: %d\n", metrics.ActiveLocks)
	fmt.Printf("  Total Locks: %d\n", metrics.TotalLocks)
	fmt.Printf("  Lock Acquisitions: %d\n", metrics.LockAcquisitions)
	fmt.Printf("  Lock Contentions: %d\n", metrics.LockContentions)
	fmt.Printf("  Average Wait Time: %v\n", metrics.AverageWaitTime)
	fmt.Printf("  Max Wait Time: %v\n", metrics.MaxWaitTime)

	// Example 2: Cache Service with RWMutex
	fmt.Println("\n=== Cache Service Example ===")
	cacheService := NewCacheService()
	defer cacheService.Stop()

	// Simulate cache operations
	var wg sync.WaitGroup

	// Writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id%5)
			value := fmt.Sprintf("value-%d", id)
			cacheService.Set(key, value)
		}(i)
	}

	// Readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id%5)
			value := cacheService.Get(key)
			if value != nil {
				fmt.Printf("Read %s: %v\n", key, value)
			}
		}(i)
	}

	wg.Wait()

	// Example 3: Context-aware locking
	fmt.Println("\n=== Context-Aware Locking Example ===")
	l := lock.New()
	defer l.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Try to acquire a lock that's already held
	locker := l.Get("busy-key")
	locker.Lock()

	go func() {
		time.Sleep(200 * time.Millisecond)
		locker.Unlock()
	}()

	unlock, err := l.LockWithContext(ctx, "busy-key")
	if err != nil {
		fmt.Printf("Context-aware lock failed as expected: %v\n", err)
	} else {
		unlock()
		fmt.Printf("Context-aware lock acquired unexpectedly\n")
	}

	fmt.Println("\n=== Example Complete ===")
}
