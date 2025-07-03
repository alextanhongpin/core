// Package random provides cryptographically secure random value generation utilities.
// It leverages Go's math/rand/v2 package which uses ChaCha8 as its PRNG for security.
//
// This package is designed for generating random values in testing, simulations,
// load testing, and other scenarios where controlled randomness is needed.
package random

import (
	"math/rand/v2"
	"time"
)

// Duration generates a random duration between 0 and the given maximum duration.
// The result is always less than the input duration.
//
// Example:
//
//	randomDelay := random.Duration(5 * time.Second) // 0 to 5 seconds
func Duration(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(max.Nanoseconds())) * time.Nanosecond
}

// DurationBetween generates a random duration between the given minimum and maximum durations (inclusive of min, exclusive of max).
//
// Example:
//
//	randomDelay := random.DurationBetween(1*time.Second, 5*time.Second) // 1 to 5 seconds
func DurationBetween(min, max time.Duration) time.Duration {
	if max <= min {
		return min
	}
	return Duration(max-min) + min
}

// Int generates a random integer between 0 and max (exclusive).
// Returns 0 if max <= 0.
//
// Example:
//
//	dice := random.Int(6) + 1 // 1 to 6
func Int(max int) int {
	if max <= 0 {
		return 0
	}
	return rand.IntN(max)
}

// IntBetween generates a random integer between min (inclusive) and max (exclusive).
//
// Example:
//
//	score := random.IntBetween(80, 100) // 80 to 99
func IntBetween(min, max int) int {
	if max <= min {
		return min
	}
	return Int(max-min) + min
}

// Float generates a random float64 between 0.0 and max (exclusive).
//
// Example:
//
//	percentage := random.Float(100.0) // 0.0 to 100.0
func Float(max float64) float64 {
	if max <= 0 {
		return 0
	}
	return rand.Float64() * max
}

// FloatBetween generates a random float64 between min (inclusive) and max (exclusive).
//
// Example:
//
//	temperature := random.FloatBetween(20.0, 30.0) // 20.0 to 30.0
func FloatBetween(min, max float64) float64 {
	if max <= min {
		return min
	}
	return Float(max-min) + min
}

// Bool generates a random boolean value with 50% probability for each.
//
// Example:
//
//	coinFlip := random.Bool() // true or false
func Bool() bool {
	return rand.IntN(2) == 1
}

// BoolWithProbability generates a random boolean with the given probability of being true.
// Probability should be between 0.0 and 1.0.
//
// Example:
//
//	// 70% chance of being true
//	likely := random.BoolWithProbability(0.7)
func BoolWithProbability(probability float64) bool {
	if probability <= 0 {
		return false
	}
	if probability >= 1 {
		return true
	}
	return rand.Float64() < probability
}

// Choice returns a random element from the given slice.
// Returns the zero value if the slice is empty.
//
// Example:
//
//	colors := []string{"red", "green", "blue"}
//	color := random.Choice(colors)
func Choice[T any](items []T) T {
	var zero T
	if len(items) == 0 {
		return zero
	}
	return items[rand.IntN(len(items))]
}

// Choices returns n random elements from the given slice with replacement.
// If n > len(items), some elements may be repeated.
//
// Example:
//
//	names := []string{"Alice", "Bob", "Charlie"}
//	selected := random.Choices(names, 5) // May contain duplicates
func Choices[T any](items []T, n int) []T {
	if len(items) == 0 || n <= 0 {
		return []T{}
	}

	result := make([]T, n)
	for i := 0; i < n; i++ {
		result[i] = items[rand.IntN(len(items))]
	}
	return result
}

// Sample returns n random elements from the given slice without replacement.
// If n > len(items), returns all items in random order.
//
// Example:
//
//	deck := []string{"A", "K", "Q", "J", "10"}
//	hand := random.Sample(deck, 3) // 3 unique cards
func Sample[T any](items []T, n int) []T {
	if len(items) == 0 || n <= 0 {
		return []T{}
	}

	if n >= len(items) {
		// Return all items in random order
		result := make([]T, len(items))
		copy(result, items)
		Shuffle(result)
		return result
	}

	// Create a copy and shuffle, then take first n elements
	shuffled := make([]T, len(items))
	copy(shuffled, items)
	Shuffle(shuffled)
	return shuffled[:n]
}

// Shuffle randomly shuffles the elements in the given slice in place.
//
// Example:
//
//	playlist := []string{"song1", "song2", "song3"}
//	random.Shuffle(playlist) // playlist is now shuffled
func Shuffle[T any](items []T) {
	rand.Shuffle(len(items), func(i, j int) {
		items[i], items[j] = items[j], items[i]
	})
}

// String generates a random string of the given length using the provided character set.
// If charset is empty, uses alphanumeric characters.
//
// Example:
//
//	id := random.String(8, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
//	token := random.String(32, "") // Uses default alphanumeric
func String(length int, charset string) string {
	if length <= 0 {
		return ""
	}

	if charset == "" {
		charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	}

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = charset[rand.IntN(len(charset))]
	}
	return string(result)
}

// AlphaNumeric generates a random alphanumeric string of the given length.
//
// Example:
//
//	sessionID := random.AlphaNumeric(16) // "a1B2c3D4e5F6g7H8"
func AlphaNumeric(length int) string {
	return String(length, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
}

// Hex generates a random hexadecimal string of the given length.
//
// Example:
//
//	color := "#" + random.Hex(6) // "#a1b2c3"
func Hex(length int) string {
	return String(length, "0123456789abcdef")
}
