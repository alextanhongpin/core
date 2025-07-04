// Package main demonstrates cache patterns without requiring Redis
// This example uses in-memory simulation to show the concepts
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// Simulate the cache interface for demonstration
type MockCache struct {
	mu   sync.RWMutex
	data map[string]cacheItem
}

type cacheItem struct {
	value  []byte
	expiry time.Time
}

func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]cacheItem),
	}
}

var ErrNotExist = errors.New("key does not exist")
var ErrExists = errors.New("key already exists")

func (m *MockCache) Store(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	expiry := time.Now().Add(ttl)
	if ttl == 0 {
		expiry = time.Time{} // No expiry
	}

	m.data[key] = cacheItem{
		value:  value,
		expiry: expiry,
	}
	return nil
}

func (m *MockCache) Load(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.data[key]
	if !exists {
		return nil, ErrNotExist
	}

	// Check expiry
	if !item.expiry.IsZero() && time.Now().After(item.expiry) {
		delete(m.data, key)
		return nil, ErrNotExist
	}

	return item.value, nil
}

func (m *MockCache) StoreOnce(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.data[key]; exists {
		return ErrExists
	}

	expiry := time.Now().Add(ttl)
	if ttl == 0 {
		expiry = time.Time{}
	}

	m.data[key] = cacheItem{
		value:  value,
		expiry: expiry,
	}
	return nil
}

func (m *MockCache) Delete(ctx context.Context, keys ...string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var deleted int64
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			deleted++
		}
	}
	return deleted, nil
}

// User represents a user entity
type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserService demonstrates cache patterns
type UserService struct {
	cache    *MockCache
	database map[int64]*User // Simulated database
	mu       sync.RWMutex    // For database operations
	cacheTTL time.Duration
}

func NewUserService() *UserService {
	return &UserService{
		cache:    NewMockCache(),
		database: make(map[int64]*User),
		cacheTTL: 15 * time.Minute,
	}
}

// GetUser demonstrates cache-aside pattern
func (s *UserService) GetUser(ctx context.Context, id int64) (*User, error) {
	key := fmt.Sprintf("user:%d", id)

	// Try cache first
	data, err := s.cache.Load(ctx, key)
	if err == nil {
		var user User
		if err := json.Unmarshal(data, &user); err != nil {
			return nil, err
		}
		log.Printf("User %d loaded from cache", id)
		return &user, nil
	}

	if !errors.Is(err, ErrNotExist) {
		return nil, err
	}

	// Cache miss - load from database
	log.Printf("Cache miss for user %d, loading from database", id)
	s.mu.RLock()
	user, exists := s.database[id]
	s.mu.RUnlock()

	if !exists {
		return nil, errors.New("user not found")
	}

	// Cache the result
	userData, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}

	if err := s.cache.Store(ctx, key, userData, s.cacheTTL); err != nil {
		log.Printf("Warning: failed to cache user %d: %v", id, err)
	}

	log.Printf("User %d loaded from database and cached", id)
	return user, nil
}

// CreateUser demonstrates write-through pattern
func (s *UserService) CreateUser(ctx context.Context, user *User) error {
	// Generate ID if not set
	if user.ID == 0 {
		s.mu.RLock()
		user.ID = int64(len(s.database) + 1)
		s.mu.RUnlock()
	}

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	// Write to database first
	s.mu.Lock()
	s.database[user.ID] = user
	s.mu.Unlock()

	// Cache the new user
	key := fmt.Sprintf("user:%d", user.ID)
	userData, err := json.Marshal(user)
	if err != nil {
		return err
	}

	if err := s.cache.Store(ctx, key, userData, s.cacheTTL); err != nil {
		log.Printf("Warning: failed to cache new user %d: %v", user.ID, err)
	}

	log.Printf("User %d created and cached", user.ID)
	return nil
}

// UpdateUser demonstrates cache invalidation
func (s *UserService) UpdateUser(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now()

	// Update database first
	s.mu.Lock()
	if _, exists := s.database[user.ID]; !exists {
		s.mu.Unlock()
		return errors.New("user not found")
	}
	s.database[user.ID] = user
	s.mu.Unlock()

	// Invalidate cache
	keys := []string{
		fmt.Sprintf("user:%d", user.ID),
		fmt.Sprintf("user:email:%s", user.Email),
	}

	deleted, err := s.cache.Delete(ctx, keys...)
	if err != nil {
		log.Printf("Warning: failed to invalidate cache for user %d: %v", user.ID, err)
	} else {
		log.Printf("Invalidated %d cache entries for user %d", deleted, user.ID)
	}

	return nil
}

// ExpensiveComputation demonstrates distributed locking pattern (simulated)
func (s *UserService) ExpensiveComputation(ctx context.Context, key string) (string, error) {
	lockKey := "lock:" + key
	lockValue := []byte(fmt.Sprintf("locked-%d", time.Now().UnixNano()))

	// Try to acquire lock
	err := s.cache.StoreOnce(ctx, lockKey, lockValue, 30*time.Second)
	if errors.Is(err, ErrExists) {
		log.Printf("Lock exists for %s, someone else is computing...", key)
		// In real implementation, you'd wait for the result
		return "result-from-other-process", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to acquire lock: %w", err)
	}

	log.Printf("Acquired lock for %s, computing...", key)

	// Ensure we release the lock
	defer func() {
		s.cache.Delete(ctx, lockKey)
		log.Printf("Released lock for %s", key)
	}()

	// Simulate expensive computation
	time.Sleep(500 * time.Millisecond)
	result := fmt.Sprintf("computed-result-for-%s", key)

	// Store the result
	if err := s.cache.Store(ctx, key, []byte(result), 5*time.Minute); err != nil {
		return "", fmt.Errorf("failed to store result: %w", err)
	}

	log.Printf("Stored computation result for %s", key)
	return result, nil
}

func main() {
	fmt.Println("=== Cache Patterns Demo (In-Memory Simulation) ===\n")

	service := NewUserService()
	ctx := context.Background()

	// 1. Create a user
	fmt.Println("1. Creating user...")
	user := &User{
		Email: "john@example.com",
		Name:  "John Doe",
	}

	if err := service.CreateUser(ctx, user); err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}
	fmt.Printf("Created user: %+v\n\n", user)

	// 2. Get user (should hit cache)
	fmt.Println("2. Getting user by ID (cache hit)...")
	retrieved, err := service.GetUser(ctx, user.ID)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}
	fmt.Printf("Retrieved user: %+v\n\n", retrieved)

	// 3. Get user again (should hit cache)
	fmt.Println("3. Getting user by ID again (cache hit)...")
	retrieved, err = service.GetUser(ctx, user.ID)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}
	fmt.Printf("Retrieved user: %+v\n\n", retrieved)

	// 4. Update user (should invalidate cache)
	fmt.Println("4. Updating user (cache invalidation)...")
	user.Name = "John Smith"
	user.Email = "johnsmith@example.com"
	if err := service.UpdateUser(ctx, user); err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}
	fmt.Printf("Updated user: %+v\n\n", user)

	// 5. Get user again (should miss cache and reload from DB)
	fmt.Println("5. Getting user after update (cache miss)...")
	retrieved, err = service.GetUser(ctx, user.ID)
	if err != nil {
		log.Fatalf("Failed to get updated user: %v", err)
	}
	fmt.Printf("Retrieved updated user: %+v\n\n", retrieved)

	// 6. Demonstrate expensive computation with locking
	fmt.Println("6. Expensive computation with locking...")

	var wg sync.WaitGroup
	key := "expensive-task"

	// Simulate 3 concurrent requests
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			log.Printf("Worker %d: Starting expensive computation", id)
			result, err := service.ExpensiveComputation(ctx, key)
			if err != nil {
				log.Printf("Worker %d: Error: %v", id, err)
			} else {
				log.Printf("Worker %d: Got result: %s", id, result)
			}
		}(i)
	}

	wg.Wait()

	fmt.Println("\n=== Demo Complete ===")
}
