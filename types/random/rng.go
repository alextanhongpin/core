package random

import (
	"math/rand/v2"
	"sync"
	"time"
	"unicode/utf8"
)

// RNG is a pseudo-random generator backed by a PCG source. It is safe for
// concurrent use. It is not cryptographically secure.
type RNG struct {
	mu   sync.Mutex
	rand *rand.Rand
}

// New returns an RNG with a random seed.
func New() *RNG {
	return new(RNG).WithSeeds(rand.Uint64(), rand.Uint64())
}

// WithSeed returns a new RNG seeded with seed; the receiver is unchanged.
// RNGs with the same seed produce the same sequence.
func (r *RNG) WithSeed(seed uint64) *RNG {
	return r.WithSeeds(seed, seed)
}

// WithSeeds returns a new RNG seeded with seed1 and seed2; the receiver is
// unchanged. RNGs with the same seeds produce the same sequence.
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
	r.mu.Lock()
	defer r.mu.Unlock()
	return time.Duration(r.rand.Int64N(int64(max)))
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
	return min + time.Duration(r.uint64N(uint64(max)-uint64(min)))
}

// uint64N returns a value in [0, n). Using unsigned arithmetic lets the
// Between methods handle spans wider than the maximum signed value.
func (r *RNG) uint64N(n uint64) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rand.Uint64N(n)
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
	r.mu.Lock()
	defer r.mu.Unlock()
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
	return min + int(r.uint64N(uint64(max)-uint64(min)))
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
	r.mu.Lock()
	defer r.mu.Unlock()
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
	r.mu.Lock()
	defer r.mu.Unlock()
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
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rand.Float64() < probability
}

// String returns length characters drawn uniformly from charset, or from
// [a-zA-Z0-9] if charset is empty. Multi-byte characters in charset are
// treated as single characters. Do not use it for secrets; see package safe.
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
		charset = alphaNumeric
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if !isASCII(charset) {
		runes := []rune(charset)
		out := make([]rune, length)
		for i := range out {
			out[i] = runes[r.rand.IntN(len(runes))]
		}
		return string(out)
	}
	out := make([]byte, length)
	for i := range out {
		out[i] = charset[r.rand.IntN(len(charset))]
	}
	return string(out)
}

const alphaNumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// shuffle shuffles n elements with swap under the RNG's lock.
func (r *RNG) shuffle(n int, swap func(i, j int)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rand.Shuffle(n, swap)
}

// AlphaNumeric generates a random alphanumeric string of the given length.
//
// Example:
//
//	sessionID := random.AlphaNumeric(16) // "a1B2c3D4e5F6g7H8"
func (r *RNG) AlphaNumeric(length int) string {
	return r.String(length, alphaNumeric)
}

// Hex generates a random hexadecimal string of the given length.
//
// Example:
//
//	color := "#" + random.Hex(6) // "#a1b2c3"
func (r *RNG) Hex(length int) string {
	return r.String(length, "0123456789abcdef")
}
