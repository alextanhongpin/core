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
	fmt.Println("Simulating load test with random delays...")
	rng := random.New().WithSeed(42)
	for i := range 3 {
		delay := rng.Duration(2 * time.Second)
		fmt.Printf("Request %d: waiting %v\n", i+1, delay.Truncate(time.Millisecond))
	}
	// Output:
	// Simulating load test with random delays...
	// Request 1: waiting 1.238s
	// Request 2: waiting 752ms
	// Request 3: waiting 1.276s
}

// Example: Retry with Exponential Backoff + Jitter
func ExampleDurationBetween_retryBackoff() {
	fmt.Println("Retry with exponential backoff and jitter:")
	rng := random.New().WithSeed(42)
	baseDelay := 100 * time.Millisecond
	maxRetries := 3
	for attempt := range maxRetries {
		backoff := time.Duration(1<<attempt) * baseDelay
		jitter := backoff / 4
		min := backoff - jitter
		max := backoff + jitter
		delay := rng.DurationBetween(min, max)
		fmt.Printf("Attempt %d: backoff %v, actual delay %v\n", attempt+1, backoff, delay.Truncate(time.Millisecond))
	}
	// Output:
	// Retry with exponential backoff and jitter:
	// Attempt 1: backoff 100ms, actual delay 105ms
	// Attempt 2: backoff 200ms, actual delay 187ms
	// Attempt 3: backoff 400ms, actual delay 427ms
}

// Example: Game Mechanics - Random Damage and Dice Rolling
func ExampleIntBetween_gameMechanics() {
	fmt.Println("RPG Game Mechanics:")
	rng := random.New().WithSeed(42)
	playerLevel := 5
	weaponDamage := 20
	minDamage := weaponDamage * 80 / 100
	maxDamage := weaponDamage * 120 / 100
	damage := rng.IntBetween(minDamage, maxDamage+1)
	fmt.Printf("Player attacks for %d damage (range: %d-%d)\n", damage, minDamage, maxDamage)
	d20Roll := rng.IntBetween(1, 21)
	skillBonus := playerLevel * 2
	totalRoll := d20Roll + skillBonus
	fmt.Printf("Skill check: rolled %d + %d bonus = %d\n", d20Roll, skillBonus, totalRoll)
	criticalHit := d20Roll >= 19
	fmt.Printf("Critical hit: %v\n", criticalHit)
	// Output:
	// RPG Game Mechanics:
	// Player attacks for 21 damage (range: 16-24)
	// Skill check: rolled 8 + 10 bonus = 18
	// Critical hit: false
}

// Example: A/B Testing and Feature Flags
func ExampleBoolWithProbability_abTesting() {
	fmt.Println("A/B Testing Simulation:")
	newFeatureRollout := 0.3
	var newFeatureUsers, oldFeatureUsers int
	rng := random.New().WithSeed(42)
	for userID := 1; userID <= 10; userID++ {
		hasNewFeature := rng.BoolWithProbability(newFeatureRollout)
		if hasNewFeature {
			newFeatureUsers++
			fmt.Printf("User %d: NEW feature\n", userID)
		} else {
			oldFeatureUsers++
			fmt.Printf("User %d: old feature\n", userID)
		}
	}
	fmt.Printf("\nResults: %d users with new feature, %d with old feature\n", newFeatureUsers, oldFeatureUsers)
	// Output:
	// A/B Testing Simulation:
	// User 1: old feature
	// User 2: old feature
	// User 3: old feature
	// User 4: old feature
	// User 5: NEW feature
	// User 6: NEW feature
	// User 7: old feature
	// User 8: NEW feature
	// User 9: old feature
	// User 10: old feature
	//
	// Results: 3 users with new feature, 7 with old feature
}

// Example: Content Recommendation System
func ExampleChoice_contentRecommendation() {
	fmt.Println("Content Recommendation System:")
	categories := []string{"Technology", "Sports", "Entertainment", "Science", "Politics", "Health", "Travel", "Food"}
	userPreferences := []string{"Technology", "Science", "Technology", "Health"}
	fmt.Println("Recommendations for user:")
	rng := random.New().WithSeed(42)
	for range 3 {
		var recommendation string
		if rng.BoolWithProbability(0.7) && len(userPreferences) > 0 {
			recommendation = random.ChoiceWithRNG(rng, userPreferences)
			fmt.Printf("- %s (based on your interests)\n", recommendation)
		} else {
			recommendation = random.ChoiceWithRNG(rng, categories)
			fmt.Printf("- %s (discover something new)\n", recommendation)
		}
	}
	// Output:
	// Content Recommendation System:
	// Recommendations for user:
	// - Health (based on your interests)
	// - Entertainment (discover something new)
	// - Science (based on your interests)
}

// Example: Playlist Shuffling and Music Recommendations
func ExampleShuffle_musicPlaylist() {
	fmt.Println("Music Playlist Management:")

	rng := random.New().WithSeed(42)
	playlist := []string{"Bohemian Rhapsody", "Stairway to Heaven", "Hotel California", "Imagine", "Sweet Child O' Mine", "Purple Haze", "Like a Rolling Stone", "Billie Jean", "Yesterday"}
	fmt.Printf("Original playlist (%d songs):\n", len(playlist))
	for i, song := range playlist[:3] {
		fmt.Printf("%d. %s\n", i+1, song)
	}
	fmt.Println("...")
	shuffledPlaylist := make([]string, len(playlist))
	copy(shuffledPlaylist, playlist)
	random.ShuffleWithRNG(rng, shuffledPlaylist)
	fmt.Printf("\nShuffled playlist:\n")
	for i, song := range shuffledPlaylist[:3] {
		fmt.Printf("%d. %s\n", i+1, song)
	}
	fmt.Println("...")
	quickMix := random.SampleWithRNG(rng, playlist, 4)
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
	// 1. Hotel California
	// 2. Yesterday
	// 3. Bohemian Rhapsody
	// ...
	//
	// Quick Mix (4 random songs):
	// 1. Sweet Child O' Mine
	// 2. Yesterday
	// 3. Purple Haze
	// 4. Hotel California
}

// Example: Session ID and Token Generation
func ExampleAlphaNumeric_sessionManagement() {
	fmt.Println("Session Management:")
	rng := random.New().WithSeed(42)
	sessionID := rng.AlphaNumeric(32)
	fmt.Printf("Session ID: %s\n", sessionID)
	csrfToken := rng.AlphaNumeric(16)
	fmt.Printf("CSRF Token: %s\n", csrfToken)
	tempPassword := rng.String(12, "ABCDEFGHJKMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789")
	fmt.Printf("Temp Password: %s\n", tempPassword)
	avatarColor := "#" + rng.Hex(6)
	fmt.Printf("Avatar Color: %s\n", avatarColor)
	// Output:
	// Session Management:
	// Session ID: MxNF7qpUYEedN5my4P4Crvbmvw0bxFDO
	// CSRF Token: 2YXC9L9CaxYJwgLO
	// Temp Password: tJwehPkt7fXg
	// Avatar Color: #b515a6
}

// Example: Chaos Engineering - Random Failures
func ExampleBoolWithProbability_chaosEngineering() {
	fmt.Println("Chaos Engineering Simulation:")
	databaseFailureRate := 0.02
	apiFailureRate := 0.05
	networkFailureRate := 0.01
	rng := random.New().WithSeed(42)
	var successCount, failureCount int
	for reqID := 1; reqID <= 20; reqID++ {
		dbFailed := rng.BoolWithProbability(databaseFailureRate)
		apiFailed := rng.BoolWithProbability(apiFailureRate)
		networkFailed := rng.BoolWithProbability(networkFailureRate)
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
	fmt.Printf("\nSummary: %d successful, %d failed (%.1f%% success rate)\n", successCount, failureCount, float64(successCount)/20*100)
	// Output:
	// Chaos Engineering Simulation:
	// Request 1: SUCCESS
	// Request 2: SUCCESS
	// Request 3: SUCCESS
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
	// Request 14: FAILED - API error
	// Request 15: SUCCESS
	// Request 16: SUCCESS
	// Request 17: SUCCESS
	// Request 18: SUCCESS
	// Request 19: SUCCESS
	// Request 20: SUCCESS
	//
	// Summary: 19 successful, 1 failed (95.0% success rate)
}

// Example: Circuit Breaker Simulation
func ExampleFloat_circuitBreaker() {
	fmt.Println("Circuit Breaker Simulation:")
	rng := random.New().WithSeed(42)
	failureThreshold := 0.5
	requestCount := 10
	var successCount, failureCount int
	for i := 1; i <= requestCount; i++ {
		currentLoad := rng.Float(1.0)
		dynamicFailureRate := currentLoad * 0.3
		failed := rng.BoolWithProbability(dynamicFailureRate)
		if failed {
			failureCount++
			fmt.Printf("Request %d: FAILED (load: %.2f, failure rate: %.1f%%)\n", i, currentLoad, dynamicFailureRate*100)
		} else {
			successCount++
			fmt.Printf("Request %d: SUCCESS (load: %.2f, failure rate: %.1f%%)\n", i, currentLoad, dynamicFailureRate*100)
		}
	}
	currentFailureRate := float64(failureCount) / float64(requestCount)
	fmt.Printf("\nCircuit Status: ")
	if currentFailureRate > failureThreshold {
		fmt.Printf("OPEN (%.1f%% failure rate > %.1f%% threshold)\n", currentFailureRate*100, failureThreshold*100)
	} else {
		fmt.Printf("CLOSED (%.1f%% failure rate <= %.1f%% threshold)\n", currentFailureRate*100, failureThreshold*100)
	}
	// Output:
	// Circuit Breaker Simulation:
	// Request 1: SUCCESS (load: 0.31, failure rate: 9.2%)
	// Request 2: SUCCESS (load: 0.86, failure rate: 25.9%)
	// Request 3: FAILED (load: 0.27, failure rate: 8.0%)
	// Request 4: SUCCESS (load: 0.64, failure rate: 19.3%)
	// Request 5: SUCCESS (load: 0.87, failure rate: 26.1%)
	// Request 6: SUCCESS (load: 0.74, failure rate: 22.1%)
	// Request 7: SUCCESS (load: 0.97, failure rate: 29.0%)
	// Request 8: SUCCESS (load: 0.02, failure rate: 0.5%)
	// Request 9: SUCCESS (load: 0.69, failure rate: 20.7%)
	// Request 10: SUCCESS (load: 0.63, failure rate: 19.0%)
	//
	// Circuit Status: CLOSED (10.0% failure rate <= 50.0% threshold)
}

// Benchmark random operations
func BenchmarkOperations(b *testing.B) {
	items := make([]int, 1000)
	for i := range items {
		items[i] = i
	}

	b.Run("Duration", func(b *testing.B) {
		for b.Loop() {
			_ = random.Duration(time.Second)
		}
	})

	b.Run("IntBetween", func(b *testing.B) {
		for b.Loop() {
			_ = random.IntBetween(0, 1000)
		}
	})

	b.Run("Choice", func(b *testing.B) {
		for b.Loop() {
			_ = random.Choice(items)
		}
	})

	b.Run("AlphaNumeric", func(b *testing.B) {
		for b.Loop() {
			_ = random.AlphaNumeric(16)
		}
	})

	b.Run("Shuffle", func(b *testing.B) {
		itemsCopy := make([]int, len(items))
		b.ResetTimer()
		for b.Loop() {
			copy(itemsCopy, items)
			random.Shuffle(itemsCopy)
		}
	})
}

func init() {
	// Suppress output in tests unless running examples
	log.SetOutput(nil)
}

// Example: Deterministic Choice, Choices, Sample, Shuffle with WithRNG and WithSeed
func TestDeterministicRandomOps(t *testing.T) {
	seed := uint64(12345)
	customRNG := random.New().WithSeed(seed)
	items := []string{"a", "b", "c", "d", "e"}

	// ChoiceWithRNG
	choice := random.ChoiceWithRNG(customRNG, items)
	if choice != "e" { // expected value may differ based on RNG
		t.Errorf("ChoiceWithRNG: got %q, want %q", choice, "e")
	}

	// ChoicesWithRNG
	choices := random.ChoicesWithRNG(customRNG, items, 3)
	if len(choices) != 3 {
		t.Errorf("ChoicesWithRNG: got %d items, want 3", len(choices))
	}

	// SampleWithRNG
	sample := random.SampleWithRNG(customRNG, items, 3)
	if len(sample) != 3 {
		t.Errorf("SampleWithRNG: got %d items, want 3", len(sample))
	}

	// ShuffleWithRNG
	shuffled := make([]string, len(items))
	copy(shuffled, items)
	random.ShuffleWithRNG(customRNG, shuffled)
	if len(shuffled) != len(items) {
		t.Errorf("ShuffleWithRNG: got %d items, want %d", len(shuffled), len(items))
	}
}

func TestDeterministicWithSeed(t *testing.T) {
	seed := uint64(42)
	items := []int{1, 2, 3, 4, 5}
	customRNG := random.New().WithSeed(seed)

	choice := random.ChoiceWithRNG(customRNG, items)
	if choice < 1 || choice > 5 {
		t.Errorf("ChoiceWithRNG: got %v, want value in 1..5", choice)
	}

	choices := random.ChoicesWithRNG(customRNG, items, 3)
	sample := random.SampleWithRNG(customRNG, items, 3)
	shuffled := make([]int, len(items))
	copy(shuffled, items)
	random.ShuffleWithRNG(customRNG, shuffled)

	if len(choices) != 3 {
		t.Errorf("ChoicesWithRNG: got %d items, want 3", len(choices))
	}
	if len(sample) != 3 {
		t.Errorf("SampleWithRNG: got %d items, want 3", len(sample))
	}
	if len(shuffled) != len(items) {
		t.Errorf("ShuffleWithRNG: got %d items, want %d", len(shuffled), len(items))
	}
}
