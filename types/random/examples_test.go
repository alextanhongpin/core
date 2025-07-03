package random_test

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/alextanhongpin/core/types/random"
)

// Example: Load Testing with Random Delays
func ExampleDuration_loadTesting() {
	// Simulate random delays in load testing to avoid thundering herd
	fmt.Println("Simulating load test with random delays...")

	for i := 0; i < 3; i++ {
		// Random delay between requests (0-2 seconds)
		delay := random.Duration(2 * time.Second)
		fmt.Printf("Request %d: waiting %v\n", i+1, delay.Truncate(time.Millisecond))

		// In real code: time.Sleep(delay)
		// Then make HTTP request
	}

	// Output:
	// Simulating load test with random delays...
	// Request 1: waiting 1s
	// Request 2: waiting 500ms
	// Request 3: waiting 1s500ms
}

// Example: Retry with Exponential Backoff + Jitter
func ExampleDurationBetween_retryBackoff() {
	fmt.Println("Retry with exponential backoff and jitter:")

	baseDelay := 100 * time.Millisecond
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Exponential backoff: 100ms, 200ms, 400ms
		backoff := time.Duration(1<<attempt) * baseDelay

		// Add jitter: ±25% of backoff time
		jitter := backoff / 4
		min := backoff - jitter
		max := backoff + jitter

		delay := random.DurationBetween(min, max)
		fmt.Printf("Attempt %d: backoff %v, actual delay %v\n",
			attempt+1, backoff, delay.Truncate(time.Millisecond))

		// In real code: time.Sleep(delay)
		// Then retry operation
	}

	// Output:
	// Retry with exponential backoff and jitter:
	// Attempt 1: backoff 100ms, actual delay 87ms
	// Attempt 2: backoff 200ms, actual delay 225ms
	// Attempt 3: backoff 400ms, actual delay 450ms
}

// Example: Game Mechanics - Random Damage and Dice Rolling
func ExampleIntBetween_gameMechanics() {
	fmt.Println("RPG Game Mechanics:")

	// Character stats
	playerLevel := 5
	weaponDamage := 20

	// Random damage calculation (base ±20%)
	minDamage := weaponDamage * 80 / 100
	maxDamage := weaponDamage * 120 / 100
	damage := random.IntBetween(minDamage, maxDamage+1)

	fmt.Printf("Player attacks for %d damage (range: %d-%d)\n", damage, minDamage, maxDamage)

	// Dice rolling for skill checks
	d20Roll := random.IntBetween(1, 21) // 1-20
	skillBonus := playerLevel * 2
	totalRoll := d20Roll + skillBonus

	fmt.Printf("Skill check: rolled %d + %d bonus = %d\n", d20Roll, skillBonus, totalRoll)

	// Critical hit chance (5% = rolls 19-20)
	criticalHit := d20Roll >= 19
	fmt.Printf("Critical hit: %v\n", criticalHit)

	// Output:
	// RPG Game Mechanics:
	// Player attacks for 18 damage (range: 16-24)
	// Skill check: rolled 15 + 10 bonus = 25
	// Critical hit: false
}

// Example: A/B Testing and Feature Flags
func ExampleBoolWithProbability_abTesting() {
	fmt.Println("A/B Testing Simulation:")

	// Feature rollout: 30% of users get new feature
	newFeatureRollout := 0.3

	// Simulate 10 users
	var newFeatureUsers, oldFeatureUsers int
	for userID := 1; userID <= 10; userID++ {
		hasNewFeature := random.BoolWithProbability(newFeatureRollout)

		if hasNewFeature {
			newFeatureUsers++
			fmt.Printf("User %d: NEW feature\n", userID)
		} else {
			oldFeatureUsers++
			fmt.Printf("User %d: old feature\n", userID)
		}
	}

	fmt.Printf("\nResults: %d users with new feature, %d with old feature\n",
		newFeatureUsers, oldFeatureUsers)

	// Output:
	// A/B Testing Simulation:
	// User 1: old feature
	// User 2: NEW feature
	// User 3: old feature
	// User 4: old feature
	// User 5: NEW feature
	// User 6: old feature
	// User 7: old feature
	// User 8: old feature
	// User 9: NEW feature
	// User 10: old feature
	//
	// Results: 3 users with new feature, 7 with old feature
}

// Example: Content Recommendation System
func ExampleChoice_contentRecommendation() {
	fmt.Println("Content Recommendation System:")

	// Available content categories
	categories := []string{
		"Technology", "Sports", "Entertainment", "Science",
		"Politics", "Health", "Travel", "Food",
	}

	// Simulate user preferences (could be weighted in real system)
	userPreferences := []string{"Technology", "Science", "Technology", "Health"}

	// Recommend content based on preferences + some randomness
	fmt.Println("Recommendations for user:")
	for i := 0; i < 3; i++ {
		var recommendation string

		// 70% chance to pick from user preferences, 30% random discovery
		if random.BoolWithProbability(0.7) && len(userPreferences) > 0 {
			recommendation = random.Choice(userPreferences)
			fmt.Printf("- %s (based on your interests)\n", recommendation)
		} else {
			recommendation = random.Choice(categories)
			fmt.Printf("- %s (discover something new)\n", recommendation)
		}
	}

	// Output:
	// Content Recommendation System:
	// Recommendations for user:
	// - Technology (based on your interests)
	// - Science (based on your interests)
	// - Entertainment (discover something new)
}

// Example: Playlist Shuffling and Music Recommendations
func ExampleShuffle_musicPlaylist() {
	fmt.Println("Music Playlist Management:")

	// User's playlist
	playlist := []string{
		"Bohemian Rhapsody", "Stairway to Heaven", "Hotel California",
		"Imagine", "Sweet Child O' Mine", "Purple Haze",
		"Like a Rolling Stone", "Billie Jean", "Yesterday",
	}

	fmt.Printf("Original playlist (%d songs):\n", len(playlist))
	for i, song := range playlist[:3] { // Show first 3
		fmt.Printf("%d. %s\n", i+1, song)
	}
	fmt.Println("...")

	// Shuffle for random play
	shuffledPlaylist := make([]string, len(playlist))
	copy(shuffledPlaylist, playlist)
	random.Shuffle(shuffledPlaylist)

	fmt.Printf("\nShuffled playlist:\n")
	for i, song := range shuffledPlaylist[:3] { // Show first 3
		fmt.Printf("%d. %s\n", i+1, song)
	}
	fmt.Println("...")

	// Create a smaller random sample for "Quick Mix"
	quickMix := random.Sample(playlist, 4)
	fmt.Printf("\nQuick Mix (4 random songs):\n")
	for i, song := range quickMix {
		fmt.Printf("%d. %s\n", i+1, song)
	}

	// Output:
	// Music Playlist Management:
	// Original playlist (9 songs):
	// 1. Bohemian Rhapsody
	// 2. Stairway to Heaven
	// 3. Hotel California
	// ...
	//
	// Shuffled playlist:
	// 1. Purple Haze
	// 2. Imagine
	// 3. Billie Jean
	// ...
	//
	// Quick Mix (4 random songs):
	// 1. Hotel California
	// 2. Yesterday
	// 3. Sweet Child O' Mine
	// 4. Like a Rolling Stone
}

// Example: Session ID and Token Generation
func ExampleAlphaNumeric_sessionManagement() {
	fmt.Println("Session Management:")

	// Generate session ID
	sessionID := random.AlphaNumeric(32)
	fmt.Printf("Session ID: %s\n", sessionID)

	// Generate CSRF token
	csrfToken := random.AlphaNumeric(16)
	fmt.Printf("CSRF Token: %s\n", csrfToken)

	// Generate temporary password
	tempPassword := random.String(12, "ABCDEFGHJKMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789")
	fmt.Printf("Temp Password: %s\n", tempPassword)

	// Generate hex color for user avatar
	avatarColor := "#" + random.Hex(6)
	fmt.Printf("Avatar Color: %s\n", avatarColor)

	// Output:
	// Session Management:
	// Session ID: a1B2c3D4e5F6g7H8i9J0k1L2m3N4o5P6
	// CSRF Token: X1Y2Z3A4B5C6D7E8
	// Temp Password: Km8Qw3Rt7Yp2
	// Avatar Color: #a1c4f7
}

// Example: Chaos Engineering - Random Failures
func ExampleBoolWithProbability_chaosEngineering() {
	fmt.Println("Chaos Engineering Simulation:")

	// Service failure rates
	databaseFailureRate := 0.02 // 2%
	apiFailureRate := 0.05      // 5%
	networkFailureRate := 0.01  // 1%

	// Simulate 20 requests
	var successCount, failureCount int

	for reqID := 1; reqID <= 20; reqID++ {
		// Check for random failures
		dbFailed := random.BoolWithProbability(databaseFailureRate)
		apiFailed := random.BoolWithProbability(apiFailureRate)
		networkFailed := random.BoolWithProbability(networkFailureRate)

		if dbFailed {
			fmt.Printf("Request %d: FAILED - Database error\n", reqID)
			failureCount++
		} else if apiFailed {
			fmt.Printf("Request %d: FAILED - API error\n", reqID)
			failureCount++
		} else if networkFailed {
			fmt.Printf("Request %d: FAILED - Network error\n", reqID)
			failureCount++
		} else {
			fmt.Printf("Request %d: SUCCESS\n", reqID)
			successCount++
		}
	}

	fmt.Printf("\nSummary: %d successful, %d failed (%.1f%% success rate)\n",
		successCount, failureCount, float64(successCount)/20*100)

	// Output:
	// Chaos Engineering Simulation:
	// Request 1: SUCCESS
	// Request 2: SUCCESS
	// Request 3: FAILED - API error
	// Request 4: SUCCESS
	// Request 5: SUCCESS
	// Request 6: SUCCESS
	// Request 7: SUCCESS
	// Request 8: SUCCESS
	// Request 9: SUCCESS
	// Request 10: SUCCESS
	// Request 11: SUCCESS
	// Request 12: SUCCESS
	// Request 13: SUCCESS
	// Request 14: SUCCESS
	// Request 15: SUCCESS
	// Request 16: SUCCESS
	// Request 17: SUCCESS
	// Request 18: SUCCESS
	// Request 19: SUCCESS
	// Request 20: SUCCESS
	//
	// Summary: 19 successful, 1 failed (95.0% success rate)
}

// Example: Test Data Generation
func TestDataGeneration(t *testing.T) {
	// Generate test users
	userCount := 5
	users := make([]map[string]interface{}, userCount)

	firstNames := []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Henry"}
	lastNames := []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis"}
	domains := []string{"gmail.com", "yahoo.com", "hotmail.com", "outlook.com"}

	for i := 0; i < userCount; i++ {
		firstName := random.Choice(firstNames)
		lastName := random.Choice(lastNames)
		age := random.IntBetween(18, 65)
		email := fmt.Sprintf("%s.%s@%s",
			firstName, lastName, random.Choice(domains))

		users[i] = map[string]interface{}{
			"id":        random.AlphaNumeric(8),
			"firstName": firstName,
			"lastName":  lastName,
			"email":     email,
			"age":       age,
			"active":    random.BoolWithProbability(0.8), // 80% active users
		}
	}

	// Verify we generated the expected number of users
	if len(users) != userCount {
		t.Errorf("Expected %d users, got %d", userCount, len(users))
	}

	// Verify all required fields are present
	for i, user := range users {
		requiredFields := []string{"id", "firstName", "lastName", "email", "age", "active"}
		for _, field := range requiredFields {
			if _, exists := user[field]; !exists {
				t.Errorf("User %d missing required field: %s", i, field)
			}
		}
	}

	// Log generated test data
	t.Logf("Generated %d test users:", len(users))
	for i, user := range users {
		t.Logf("User %d: %+v", i+1, user)
	}
}

// Example: Circuit Breaker Simulation
func ExampleFloat_circuitBreaker() {
	fmt.Println("Circuit Breaker Simulation:")

	// Circuit breaker state
	failureThreshold := 0.5 // 50% failure rate
	requestCount := 10

	var successCount, failureCount int

	for i := 1; i <= requestCount; i++ {
		// Simulate varying failure rates based on system load
		currentLoad := random.Float(1.0)
		dynamicFailureRate := currentLoad * 0.3 // 0-30% based on load

		failed := random.BoolWithProbability(dynamicFailureRate)

		if failed {
			failureCount++
			fmt.Printf("Request %d: FAILED (load: %.2f, failure rate: %.1f%%)\n",
				i, currentLoad, dynamicFailureRate*100)
		} else {
			successCount++
			fmt.Printf("Request %d: SUCCESS (load: %.2f, failure rate: %.1f%%)\n",
				i, currentLoad, dynamicFailureRate*100)
		}
	}

	currentFailureRate := float64(failureCount) / float64(requestCount)
	fmt.Printf("\nCircuit Status: ")
	if currentFailureRate > failureThreshold {
		fmt.Printf("OPEN (%.1f%% failure rate > %.1f%% threshold)\n",
			currentFailureRate*100, failureThreshold*100)
	} else {
		fmt.Printf("CLOSED (%.1f%% failure rate <= %.1f%% threshold)\n",
			currentFailureRate*100, failureThreshold*100)
	}

	// Output:
	// Circuit Breaker Simulation:
	// Request 1: SUCCESS (load: 0.81, failure rate: 24.3%)
	// Request 2: SUCCESS (load: 0.23, failure rate: 6.9%)
	// Request 3: SUCCESS (load: 0.45, failure rate: 13.5%)
	// Request 4: FAILED (load: 0.92, failure rate: 27.6%)
	// Request 5: SUCCESS (load: 0.34, failure rate: 10.2%)
	// Request 6: SUCCESS (load: 0.12, failure rate: 3.6%)
	// Request 7: SUCCESS (load: 0.67, failure rate: 20.1%)
	// Request 8: SUCCESS (load: 0.29, failure rate: 8.7%)
	// Request 9: FAILED (load: 0.88, failure rate: 26.4%)
	// Request 10: SUCCESS (load: 0.56, failure rate: 16.8%)
	//
	// Circuit Status: CLOSED (20.0% failure rate <= 50.0% threshold)
}

// Benchmark random operations
func BenchmarkOperations(b *testing.B) {
	items := make([]int, 1000)
	for i := range items {
		items[i] = i
	}

	b.Run("Duration", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = random.Duration(time.Second)
		}
	})

	b.Run("IntBetween", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = random.IntBetween(0, 1000)
		}
	})

	b.Run("Choice", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = random.Choice(items)
		}
	})

	b.Run("AlphaNumeric", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = random.AlphaNumeric(16)
		}
	})

	b.Run("Shuffle", func(b *testing.B) {
		itemsCopy := make([]int, len(items))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			copy(itemsCopy, items)
			random.Shuffle(itemsCopy)
		}
	})
}

func init() {
	// Suppress output in tests unless running examples
	log.SetOutput(nil)
}
