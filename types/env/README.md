# env

Load environment variables into typed values with one of three failure modes:
return an error, fall back to a default, or panic.

**Use it for:** reading configuration at startup.

**Don't use it for:** config files, struct-tag binding, or reloading
configuration at runtime.

## Installation

```sh
go get github.com/alextanhongpin/core/types
```

`env` is a package in the `types` module; import it as
`github.com/alextanhongpin/core/types/env`. Check the module's `go.mod` for the
required Go version.

## Quick start

```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/alextanhongpin/core/types/env"
)

func main() {
	os.Setenv("PORT", "8080")
	os.Setenv("TIMEOUT", "5s")
	os.Setenv("HOSTS", "a.internal, b.internal")

	port := env.MustLoad[int]("PORT")
	timeout := env.LoadDurationOr("TIMEOUT", 30*time.Second)
	hosts := env.MustLoadSlice[string]("HOSTS", ",")
	debug := env.LoadOr("DEBUG", false)

	fmt.Println(port, timeout, hosts, debug)

	_, err := env.Load[int]("MISSING")
	fmt.Println(err)
}
```

It prints:

```
8080 5s [a.internal b.internal] false
env: variable not set: "MISSING"
```

## API

```go
var ErrNotSet, ErrParseFailed error

type Parseable interface { ~string | ~bool | ~int... | ~uint... | ~float32 | ~float64 | ~complex64 | ~complex128 }

func Parse[T Parseable](s string) (T, error)

func Load[T Parseable](name string) (T, error)
func LoadOr[T Parseable](name string, def T) T
func MustLoad[T Parseable](name string) T

func LoadDuration(name string) (time.Duration, error)          // also LoadDurationOr, MustLoadDuration
func LoadTime(name, layout string) (time.Time, error)          // also LoadTimeOr, MustLoadTime
func LoadSlice[T Parseable](name, sep string) ([]T, error)     // also LoadSliceOr, MustLoadSlice

func MustExist(names ...string) // panics listing every missing name
func Exists(name string) bool   // set, possibly empty
func IsSet(name string) bool    // set and non-empty
```

## Behavior and guarantees

- `Load*` errors wrap `ErrNotSet` when the variable is unset and
  `ErrParseFailed` when the value cannot be parsed; test with `errors.Is`.
- A variable set to the empty string is set: `Load[string]` returns `""`,
  `Load[int]` fails to parse, and `LoadSlice` returns an empty slice.
- Values are trimmed of surrounding whitespace, and so is each slice element.
- `Parse` uses `strconv` and must consume the whole string: `"12abc"` and
  out-of-range values such as `"300"` for `uint8` are errors. Integers accept
  base prefixes (`0x`, `0o`, `0b`) and `_` separators. Booleans accept
  `1, t, T, TRUE, true, True` and their false counterparts.
- Named types work: `env.Load[Port]("PORT")` for `type Port int`.
- `*Or` variants return the default on any error, including a malformed value.
  Use `Load` when a typo should not be silently ignored.
- `LoadDuration` uses `time.ParseDuration`; `LoadTime` uses `time.Parse`.

## Testing

```sh
cd types
go vet ./env
go test -race -count=1 ./env
```
