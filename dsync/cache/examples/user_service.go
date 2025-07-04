// Package main demonstrates a real-world user service with caching.
// This example shows how to implement a user service that caches user data
// with proper error handling, cache invalidation, and distributed locking.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/redis/go-redis/v9"
)

// User represents a user entity
type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserNotFoundError is returned when a user is not found
var UserNotFoundError = errors.New("user not found")

// UserService provides user operations with caching
type UserService struct {
	cache     *cache.JSON
	db        UserRepository // Simulated database
	cacheTTL  time.Duration
	keyPrefix string
}

// UserRepository simulates a database interface
type UserRepository interface {
	FindByID(ctx context.Context, id int64) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id int64) error
}

// NewUserService creates a new user service with caching
func NewUserService(client *redis.Client, db UserRepository) *UserService {
	return &UserService{
		cache:     cache.NewJSON(client),
		db:        db,
		cacheTTL:  15 * time.Minute,
		keyPrefix: "user:",
	}
}

// GetUser retrieves a user by ID with caching
func (s *UserService) GetUser(ctx context.Context, id int64) (*User, error) {
	key := fmt.Sprintf("%s%d", s.keyPrefix, id)

	var user *User
	loaded, err := s.cache.LoadOrStore(ctx, key, &user, func() (any, error) {
		log.Printf("Cache miss for user %d, loading from database", id)
		return s.db.FindByID(ctx, id)
	}, s.cacheTTL)

	if err != nil {
		return nil, fmt.Errorf("failed to get user %d: %w", id, err)
	}

	if loaded {
		log.Printf("User %d loaded from cache", id)
	} else {
		log.Printf("User %d loaded from database and cached", id)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email with caching
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	key := fmt.Sprintf("%semail:%s", s.keyPrefix, email)

	var user *User
	loaded, err := s.cache.LoadOrStore(ctx, key, &user, func() (any, error) {
		log.Printf("Cache miss for email %s, loading from database", email)
		return s.db.FindByEmail(ctx, email)
	}, s.cacheTTL)

	if err != nil {
		return nil, fmt.Errorf("failed to get user by email %s: %w", email, err)
	}

	if loaded {
		log.Printf("User with email %s loaded from cache", email)
	} else {
		log.Printf("User with email %s loaded from database and cached", email)
	}

	return user, nil
}

// CreateUser creates a new user and handles cache consistency
func (s *UserService) CreateUser(ctx context.Context, user *User) error {
	// Create in database first
	if err := s.db.Create(ctx, user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Cache the new user
	key := fmt.Sprintf("%s%d", s.keyPrefix, user.ID)
	if err := s.cache.Store(ctx, key, user, s.cacheTTL); err != nil {
		log.Printf("Warning: failed to cache new user %d: %v", user.ID, err)
	}

	// Cache by email as well
	emailKey := fmt.Sprintf("%semail:%s", s.keyPrefix, user.Email)
	if err := s.cache.Store(ctx, emailKey, user, s.cacheTTL); err != nil {
		log.Printf("Warning: failed to cache user by email %s: %v", user.Email, err)
	}

	log.Printf("User %d created and cached", user.ID)
	return nil
}

// UpdateUser updates a user and invalidates cache
func (s *UserService) UpdateUser(ctx context.Context, user *User) error {
	// Get the current user to know the old email for cache invalidation
	currentUser, err := s.db.FindByID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Update in database first
	if err := s.db.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Invalidate caches
	keys := []string{
		fmt.Sprintf("%s%d", s.keyPrefix, user.ID),
		fmt.Sprintf("%semail:%s", s.keyPrefix, currentUser.Email),
	}

	// If email changed, also invalidate the new email key
	if currentUser.Email != user.Email {
		keys = append(keys, fmt.Sprintf("%semail:%s", s.keyPrefix, user.Email))
	}

	deleted, err := s.cache.Delete(ctx, keys...)
	if err != nil {
		log.Printf("Warning: failed to invalidate cache for user %d: %v", user.ID, err)
	} else {
		log.Printf("Invalidated %d cache entries for user %d", deleted, user.ID)
	}

	return nil
}

// DeleteUser deletes a user and invalidates cache
func (s *UserService) DeleteUser(ctx context.Context, id int64) error {
	// Get user details for cache invalidation
	user, err := s.db.FindByID(ctx, id)
	if err != nil && !errors.Is(err, UserNotFoundError) {
		return fmt.Errorf("failed to get user for deletion: %w", err)
	}

	// Delete from database first
	if err := s.db.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Invalidate caches
	keys := []string{fmt.Sprintf("%s%d", s.keyPrefix, id)}
	if user != nil {
		keys = append(keys, fmt.Sprintf("%semail:%s", s.keyPrefix, user.Email))
	}

	deleted, err := s.cache.Delete(ctx, keys...)
	if err != nil {
		log.Printf("Warning: failed to invalidate cache for deleted user %d: %v", id, err)
	} else {
		log.Printf("Invalidated %d cache entries for deleted user %d", deleted, id)
	}

	return nil
}

// GetUserStats returns cached user statistics
func (s *UserService) GetUserStats(ctx context.Context) (*UserStats, error) {
	key := "user:stats"

	var stats *UserStats
	loaded, err := s.cache.LoadOrStore(ctx, key, &stats, func() (any, error) {
		log.Println("Computing user statistics...")
		// Simulate expensive computation
		time.Sleep(100 * time.Millisecond)
		return &UserStats{
			TotalUsers:  1000,
			ActiveUsers: 850,
			ComputedAt:  time.Now(),
		}, nil
	}, 5*time.Minute) // Cache stats for 5 minutes

	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	if loaded {
		log.Println("User stats loaded from cache")
	} else {
		log.Println("User stats computed and cached")
	}

	return stats, nil
}

// UserStats represents user statistics
type UserStats struct {
	TotalUsers  int       `json:"total_users"`
	ActiveUsers int       `json:"active_users"`
	ComputedAt  time.Time `json:"computed_at"`
}

// Example usage and testing
func main() {
	// Initialize Redis client
	client := redis.NewClient(&redis.Options{
		Addr: ":6379",
	})
	defer client.Close()

	// Create a mock database
	db := &MockUserRepository{
		users: make(map[int64]*User),
	}

	// Create user service
	service := NewUserService(client, db)
	ctx := context.Background()

	// Example usage
	demonstrateUserService(ctx, service)
}

func demonstrateUserService(ctx context.Context, service *UserService) {
	// Create a test user
	user := &User{
		ID:        1,
		Email:     "john@example.com",
		Name:      "John Doe",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create user
	if err := service.CreateUser(ctx, user); err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	// Get user by ID (should hit cache)
	retrieved, err := service.GetUser(ctx, 1)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}
	fmt.Printf("Retrieved user: %+v\n", retrieved)

	// Get user by email (should hit cache)
	retrieved, err = service.GetUserByEmail(ctx, "john@example.com")
	if err != nil {
		log.Fatalf("Failed to get user by email: %v", err)
	}
	fmt.Printf("Retrieved user by email: %+v\n", retrieved)

	// Get user stats
	stats, err := service.GetUserStats(ctx)
	if err != nil {
		log.Fatalf("Failed to get user stats: %v", err)
	}
	fmt.Printf("User stats: %+v\n", stats)

	// Update user (should invalidate cache)
	user.Name = "John Smith"
	user.UpdatedAt = time.Now()
	if err := service.UpdateUser(ctx, user); err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}

	// Get user again (should miss cache and reload from DB)
	retrieved, err = service.GetUser(ctx, 1)
	if err != nil {
		log.Fatalf("Failed to get updated user: %v", err)
	}
	fmt.Printf("Updated user: %+v\n", retrieved)
}

// MockUserRepository is a simple in-memory implementation for testing
type MockUserRepository struct {
	users map[int64]*User
}

func (m *MockUserRepository) FindByID(ctx context.Context, id int64) (*User, error) {
	user, exists := m.users[id]
	if !exists {
		return nil, UserNotFoundError
	}
	return user, nil
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, UserNotFoundError
}

func (m *MockUserRepository) Create(ctx context.Context, user *User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) Update(ctx context.Context, user *User) error {
	if _, exists := m.users[user.ID]; !exists {
		return UserNotFoundError
	}
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id int64) error {
	if _, exists := m.users[id]; !exists {
		return UserNotFoundError
	}
	delete(m.users, id)
	return nil
}
