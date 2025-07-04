package random

import (
	"math/rand/v2"
	"time"
)

type RNG struct {
	rand *rand.Rand
}

func New() *RNG {
	return new(RNG).WithSeeds(rand.Uint64(), rand.Uint64())
}

func (r *RNG) WithSeed(seed uint64) *RNG {
	return r.WithSeeds(seed, seed)
}

func (r *RNG) WithSeeds(seed1, seed2 uint64) *RNG {
	return &RNG{
		rand: rand.New(rand.NewPCG(seed1, seed2)),
	}
}

// Duration generates a random duration between 0 and the given maximum duration.
// The result is always less than the input duration.
//
// Example:
//
//	randomDelay := random.Duration(5 * time.Second) // 0 to 5 seconds
func (r *RNG) Duration(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	return time.Duration(r.rand.Int64N(max.Nanoseconds())) * time.Nanosecond
}

// DurationBetween generates a random duration between the given minimum and maximum durations (inclusive of min, exclusive of max).
//
// Example:
//
//	randomDelay := random.DurationBetween(1*time.Second, 5*time.Second) // 1 to 5 seconds
func (r *RNG) DurationBetween(min, max time.Duration) time.Duration {
	if max <= min {
		return min
	}
	return r.Duration(max-min) + min
}

// Int generates a random integer between 0 and max (exclusive).
// Returns 0 if max <= 0.
//
// Example:
//
//	dice := random.Int(6) + 1 // 1 to 6
func (r *RNG) Int(max int) int {
	if max <= 0 {
		return 0
	}
	return r.rand.IntN(max)
}

// IntBetween generates a random integer between min (inclusive) and max (exclusive).
//
// Example:
//
//	score := random.IntBetween(80, 100) // 80 to 99
func (r *RNG) IntBetween(min, max int) int {
	if max <= min {
		return min
	}
	return r.Int(max-min) + min
}

// Float generates a random float64 between 0.0 and max (exclusive).
//
// Example:
//
//	percentage := random.Float(100.0) // 0.0 to 100.0
func (r *RNG) Float(max float64) float64 {
	if max <= 0 {
		return 0
	}
	return r.rand.Float64() * max
}

// FloatBetween generates a random float64 between min (inclusive) and max (exclusive).
//
// Example:
//
//	temperature := random.FloatBetween(20.0, 30.0) // 20.0 to 30.0
func (r *RNG) FloatBetween(min, max float64) float64 {
	if max <= min {
		return min
	}
	return r.Float(max-min) + min
}

// Bool generates a random boolean value with 50% probability for each.
//
// Example:
//
//	coinFlip := random.Bool() // true or false
func (r *RNG) Bool() bool {
	return r.rand.IntN(2) == 1
}

// BoolWithProbability generates a random boolean with the given probability of being true.
// Probability should be between 0.0 and 1.0.
//
// Example:
//
//	// 70% chance of being true
//	likely := random.BoolWithProbability(0.7)
func (r *RNG) BoolWithProbability(probability float64) bool {
	if probability <= 0 {
		return false
	}
	if probability >= 1 {
		return true
	}
	return r.rand.Float64() < probability
}

// String generates a random string of the given length using the provided character set.
// If charset is empty, uses alphanumeric characters.
//
// Example:
//
//	id := random.String(8, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
//	token := random.String(32, "") // Uses default alphanumeric
func (r *RNG) String(length int, charset string) string {
	if length <= 0 {
		return ""
	}

	if charset == "" {
		charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	}

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = charset[r.rand.IntN(len(charset))]
	}
	return string(result)
}

// AlphaNumeric generates a random alphanumeric string of the given length.
//
// Example:
//
//	sessionID := random.AlphaNumeric(16) // "a1B2c3D4e5F6g7H8"
func (r *RNG) AlphaNumeric(length int) string {
	return r.String(length, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
}

// Hex generates a random hexadecimal string of the given length.
//
// Example:
//
//	color := "#" + random.Hex(6) // "#a1b2c3"
func (r *RNG) Hex(length int) string {
	return r.String(length, "0123456789abcdef")
}
