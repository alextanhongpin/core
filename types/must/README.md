# must

Turn `(value, error)` results into values by panicking on error.

**Use it for:** tests, examples, package-level variables, and startup code where
an error means the program cannot continue.

**Don't use it for:** request handling or any path where the error should be
returned or recovered from.

## Installation

```sh
go get github.com/alextanhongpin/core/types
```

`must` is a package in the `types` module; import it as
`github.com/alextanhongpin/core/types/must`. Check the module's `go.mod` for the
required Go version.

## Quick start

```go
package main

import (
	"fmt"
	"net/url"
	"os"

	"github.com/alextanhongpin/core/types/must"
)

var base = must.Value(url.Parse("https://api.example.com/v1"))

func main() {
	fmt.Println(base.Host)
	must.NoError(os.Setenv("MODE", "test"))
	fmt.Println(os.Getenv("MODE"))
}
```

It prints:

```
api.example.com
test
```

## API

```go
func Value[T any](v T, err error) T // returns v, or panics if err != nil
func NoError(err error)             // panics if err != nil
```

## Behavior and guarantees

- The panic value is an `error` wrapping the original, with the message
  prefixed by `must: `; `errors.Is` and `errors.As` work on a recovered value.
- `Value` accepts a function call that returns exactly two values directly:
  `must.Value(strconv.Atoi(s))`.

## Testing

```sh
cd types
go vet ./must
go test -race -count=1 ./must
```
