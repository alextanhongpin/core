# number

Clamp, range-check, interpolate, and rescale values of any integer or
floating-point type.

**Use it for:** keeping user input or computed values inside bounds, and
converting between ranges (sensor readings to volts, scores to percentages).

**Don't use it for:** functions the standard library already has: `math.Abs`,
`math.Round`, `math.Floor`, `math.Ceil`, and the built-in `min` and `max`
(one-sided clamps).

## Installation

```sh
go get github.com/alextanhongpin/core/types
```

All `types` packages ship in the one `github.com/alextanhongpin/core/types`
module. Import this one as `github.com/alextanhongpin/core/types/number`. Check
the module's `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/types/number"
)

func main() {
	volume := number.Clip(0, 100, 130)
	volts := number.Map(512.0, 0, 1023, 0, 5)
	progress := number.Normalize(30.0, 0, 120)

	fmt.Println(volume, number.InRange(0, 100, 42))
	fmt.Printf("%.2f %.2f\n", volts, progress)
}
```

It prints:

```
100 true
2.50 0.25
```

## API

```go
type Number interface{ constraints.Integer | constraints.Float }

func Clip[T Number](lo, hi, v T) T
func InRange[T Number](lo, hi, v T) bool
func Sign[T Number](v T) int
func Lerp[T constraints.Float](a, b, t T) T
func Map[T constraints.Float](v, inMin, inMax, outMin, outMax T) T
func Normalize[T constraints.Float](v, lo, hi T) T
```

## Behavior and guarantees

- `Clip` and `InRange` accept their bounds in either order, and both bounds are
  inclusive.
- NaN: `Clip` returns a NaN input unchanged, `InRange` is false, and `Sign` is 0.
- `Lerp(a, b, t)` does not clamp `t`; values outside `[0, 1]` extrapolate.
- `Map` does not clamp its result; wrap it in `Clip` when needed. Reversed
  ranges are allowed (`outMin > outMax` inverts the mapping). When
  `inMin == inMax` the input range is empty and `Map` returns `outMin` instead
  of dividing by zero.
- `Normalize(v, lo, hi)` is `Map(v, lo, hi, 0, 1)`, so it returns 0 when
  `lo == hi`.
- All functions are pure and safe for concurrent use.

## Testing

```sh
go vet ./number/...
go test -race -count=1 ./number/...
```

Run from the `types` directory.
