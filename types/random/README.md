# random

Pseudo-random durations, numbers in ranges, strings, and slice samples on top
of `math/rand/v2`, with seedable generators for reproducible runs.

**Use it for:** jitter, test fixtures, simulations, load tests, sampling, and
A/B bucketing where predictability is not a security risk.

**Don't use it for:** tokens, passwords, keys, or anything an attacker must not
guess. It is **not cryptographically secure**; use [`safe`](../safe) instead.

## Installation

```sh
go get github.com/alextanhongpin/core/types
```

`random` is a package in the `types` module; import it as
`github.com/alextanhongpin/core/types/random`. Check the module's `go.mod` for the
required Go version.

## Quick start

```go
package main

import (
	"fmt"
	"time"

	"github.com/alextanhongpin/core/types/random"
)

func main() {
	// Package-level functions use a shared, randomly seeded RNG.
	delay := random.DurationBetween(100*time.Millisecond, time.Second)
	fmt.Println(delay >= 100*time.Millisecond && delay < time.Second)

	// A seeded RNG repeats the same sequence.
	rng := random.New().WithSeed(42)
	fmt.Println(rng.IntBetween(1, 7))
	fmt.Println(random.ChoiceWithRNG(rng, []string{"red", "green", "blue"}))
	fmt.Println(rng.AlphaNumeric(8))
}
```

It prints:

```
true
4
green
NF7qpUYE
```

The last three lines are the same on every run because the RNG is seeded.

## API

```go
// Package-level functions use a shared, randomly seeded RNG.
func Duration(max time.Duration) time.Duration
func DurationBetween(min, max time.Duration) time.Duration
func IntBetween(min, max int) int
func FloatBetween(min, max float64) float64
func BoolWithProbability(p float64) bool
func Choice[T any](items []T) T
func Choices[T any](items []T, n int) []T  // with replacement
func Sample[T any](items []T, n int) []T   // without replacement
func Shuffle[T any](items []T)
func String(length int, charset string) string
func AlphaNumeric(length int) string
func Hex(length int) string

// The same generic helpers with an explicit RNG.
func ChoiceWithRNG[T any](r *RNG, items []T) T
func ChoicesWithRNG[T any](r *RNG, items []T, n int) []T
func SampleWithRNG[T any](r *RNG, items []T, n int) []T
func ShuffleWithRNG[T any](r *RNG, items []T)

type RNG struct{ /* ... */ }
func New() *RNG                              // randomly seeded
func (r *RNG) WithSeed(seed uint64) *RNG     // new RNG; receiver unchanged
func (r *RNG) WithSeeds(s1, s2 uint64) *RNG
// Methods: Duration, DurationBetween, Int, IntBetween, Float, FloatBetween,
// Bool, BoolWithProbability, String, AlphaNumeric, Hex.
```

For a plain integer, float, or bool, call `math/rand/v2` directly
(`rand.IntN`, `rand.Float64`); this package does not wrap them at package level.

## Behavior and guarantees

- Ranges are `[min, max)`. When `max <= min`, the `Between` functions return
  `min`; `Duration`, `Int`, and `Float` return 0 for a nonpositive max. Spans
  wider than the maximum signed value (e.g. `IntBetween(math.MinInt, math.MaxInt)`)
  are handled correctly.
- `BoolWithProbability` clamps: `p <= 0` is always false, `p >= 1` always true.
- `Choice` of an empty slice returns the zero value. `Choices` and `Sample`
  return an empty slice for empty input or `n <= 0`. `Sample` with
  `n >= len(items)` returns a shuffled copy of all items. Inputs are not modified
  except by `Shuffle`.
- `String` uses `[a-zA-Z0-9]` when `charset` is empty and treats multi-byte
  characters in `charset` as single characters.
- `RNG` uses a PCG source and is safe for concurrent use (calls are serialized
  by a mutex). The same seeds produce the same sequence, but sequences may
  change between versions of this package or of Go.

## Testing

```sh
cd types
go vet ./random
go test -race -count=1 ./random
```
