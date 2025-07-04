package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/dataloader"
)

// User represents a user entity
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UserService simulates a database service
type UserService struct {
	// Simulate database with map
	users map[int]*User
	// Track database calls for demonstration
	callCount int
	mu        sync.Mutex
}

func NewUserService() *UserService {
	users := make(map[int]*User)
	for i := 1; i <= 1000; i++ {
		users[i] = &User{
			ID:    i,
			Name:  fmt.Sprintf("User %d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
		}
	}
	return &UserService{users: users}
}

func (s *UserService) GetUsersBatch(ctx context.Context, ids []int) (map[int]*User, error) {
	s.mu.Lock()
	s.callCount++
	call := s.callCount
	s.mu.Unlock()

	log.Printf("Database call #%d - Loading %d users: %v", call, len(ids), ids)

	// Simulate database query time
	time.Sleep(10 * time.Millisecond)

	result := make(map[int]*User)
	for _, id := range ids {
		if user, exists := s.users[id]; exists {
			result[id] = user
		}
	}

	log.Printf("Database call #%d - Returning %d users", call, len(result))
	return result, nil
}

func (s *UserService) GetCallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.callCount
}

// GraphQL-like resolver example
type UserResolver struct {
	userLoader *dataloader.DataLoader[int, *User]
}

func NewUserResolver(userService *UserService) *UserResolver {
	// Create dataloader with comprehensive options
	dl := dataloader.New(context.Background(), &dataloader.Options[int, *User]{
		BatchFn: func(ctx context.Context, keys []int) (map[int]*User, error) {
			return userService.GetUsersBatch(ctx, keys)
		},
		BatchMaxKeys:   100,
		BatchTimeout:   16 * time.Millisecond,
		LoadTimeout:    5 * time.Second,
		BatchQueueSize: 10,

		// Observability callbacks
		OnBatchStart: func(keys []int) {
			log.Printf("🚀 Starting batch for %d keys", len(keys))
		},
		OnBatchComplete: func(keys []int, duration time.Duration, err error) {
			if err != nil {
				log.Printf("❌ Batch completed with error in %v: %v", duration, err)
			} else {
				log.Printf("✅ Batch completed successfully in %v for %d keys", duration, len(keys))
			}
		},
		OnCacheHit: func(key int) {
			log.Printf("🎯 Cache hit for user %d", key)
		},
		OnCacheMiss: func(key int) {
			log.Printf("💫 Cache miss for user %d", key)
		},
		OnError: func(key int, err error) {
			log.Printf("⚠️  Error loading user %d: %v", key, err)
		},
	})

	return &UserResolver{userLoader: dl}
}

func (r *UserResolver) GetUser(ctx context.Context, id int) (*User, error) {
	return r.userLoader.Load(id)
}

func (r *UserResolver) GetUsers(ctx context.Context, ids []int) ([]*User, error) {
	results, err := r.userLoader.LoadMany(ids)
	if err != nil {
		return nil, err
	}

	users := make([]*User, 0, len(results))
	for _, result := range results {
		if result.Err != nil {
			log.Printf("Error loading user: %v", result.Err)
			continue // Skip users that couldn't be loaded
		}
		if result.Data != nil {
			users = append(users, result.Data)
		}
	}
	return users, nil
}

func (r *UserResolver) GetMetrics() dataloader.Metrics {
	return r.userLoader.Metrics()
}

func (r *UserResolver) Stop() {
	r.userLoader.Stop()
}

// Simulate a GraphQL query execution
func simulateGraphQLQuery(resolver *UserResolver) {
	ctx := context.Background()

	log.Println("🔍 Simulating GraphQL query execution...")

	// Simulate multiple concurrent requests asking for users
	// This would typically come from GraphQL field resolvers
	var wg sync.WaitGroup

	// Query 1: Get user posts (each post has an author)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("📝 Resolving post authors...")

		// Simulate posts with author IDs
		authorIDs := []int{1, 2, 3, 1, 2, 4, 5, 1} // Note: duplicates

		for _, authorID := range authorIDs {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				user, err := resolver.GetUser(ctx, id)
				if err != nil {
					log.Printf("Error getting user %d: %v", id, err)
					return
				}
				log.Printf("📄 Post author: %s", user.Name)
			}(authorID)
		}
	}()

	// Query 2: Get user comments (each comment has an author)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("💬 Resolving comment authors...")

		// Simulate comments with author IDs
		authorIDs := []int{3, 4, 5, 6, 3, 4, 7, 8}

		for _, authorID := range authorIDs {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				user, err := resolver.GetUser(ctx, id)
				if err != nil {
					log.Printf("Error getting user %d: %v", id, err)
					return
				}
				log.Printf("💬 Comment author: %s", user.Name)
			}(authorID)
		}
	}()

	// Query 3: Get user followers
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("👥 Resolving user followers...")

		followers := []int{9, 10, 11, 12, 13}
		users, err := resolver.GetUsers(ctx, followers)
		if err != nil {
			log.Printf("Error getting followers: %v", err)
			return
		}

		for _, user := range users {
			log.Printf("👤 Follower: %s", user.Name)
		}
	}()

	wg.Wait()
	log.Println("✅ GraphQL query execution completed")
}

func main() {
	fmt.Println("🚀 Advanced DataLoader Example")
	fmt.Println("==============================")

	// Create services
	userService := NewUserService()
	resolver := NewUserResolver(userService)
	defer resolver.Stop()

	// Simulate GraphQL queries
	simulateGraphQLQuery(resolver)

	// Wait a bit for all batches to complete
	time.Sleep(100 * time.Millisecond)

	// Show metrics
	fmt.Println("\n📊 DataLoader Metrics:")
	fmt.Println("===================")
	metrics := resolver.GetMetrics()
	fmt.Printf("Total Requests: %d\n", metrics.TotalRequests)
	fmt.Printf("Keys Requested: %d\n", metrics.KeysRequested)
	fmt.Printf("Cache Hits: %d\n", metrics.CacheHits)
	fmt.Printf("Cache Misses: %d\n", metrics.CacheMisses)
	fmt.Printf("Batch Calls: %d\n", metrics.BatchCalls)
	fmt.Printf("Error Count: %d\n", metrics.ErrorCount)
	fmt.Printf("No Result Count: %d\n", metrics.NoResultCount)

	fmt.Printf("\n🗄️  Database Calls: %d\n", userService.GetCallCount())

	// Demonstrate cache effectiveness
	fmt.Println("\n🎯 Cache Effectiveness:")
	fmt.Println("====================")
	cacheHitRate := float64(metrics.CacheHits) / float64(metrics.KeysRequested) * 100
	fmt.Printf("Cache Hit Rate: %.2f%%\n", cacheHitRate)

	// Show the power of batching
	fmt.Println("\n⚡ Batching Effectiveness:")
	fmt.Println("========================")
	if metrics.KeysRequested > 0 {
		batchingEfficiency := float64(metrics.KeysRequested) / float64(userService.GetCallCount())
		fmt.Printf("Keys per Database Call: %.2f\n", batchingEfficiency)
		fmt.Printf("Efficiency Gain: %.2fx\n", batchingEfficiency)
	}

	// Demonstrate timeout handling
	fmt.Println("\n⏱️  Timeout Handling:")
	fmt.Println("==================")

	// Create a dataloader with very short timeout
	shortTimeoutDL := dataloader.New(context.Background(), &dataloader.Options[int, *User]{
		BatchFn: func(ctx context.Context, keys []int) (map[int]*User, error) {
			// Simulate slow operation
			time.Sleep(100 * time.Millisecond)
			return userService.GetUsersBatch(ctx, keys)
		},
		LoadTimeout: 50 * time.Millisecond,
	})
	defer shortTimeoutDL.Stop()

	start := time.Now()
	_, err := shortTimeoutDL.LoadWithTimeout(context.Background(), 1)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("✅ Timeout handled correctly: %v (took %v)\n", err, duration)
	} else {
		fmt.Printf("❌ Expected timeout but got result\n")
	}

	fmt.Println("\n🎉 Advanced DataLoader Example Complete!")
}
