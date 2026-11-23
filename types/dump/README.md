# dump

Pretty-print Go values as indented, Go-like literals with sorted map keys and
cycle detection.

**Use it for:** debugging, readable golden files, and test failure output where
the result should be stable from run to run.

**Don't use it for:** serialization (the output is not parseable), or printing
unexported fields.

## Installation

```sh
go get github.com/alextanhongpin/core/types
```

`dump` is a package in the `types` module; import it as
`github.com/alextanhongpin/core/types/dump`. Check the module's `go.mod` for the
required Go version.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/types/dump"
)

type Password string

func (Password) Dump() string { return `"***"` }

type User struct {
	Name     string
	Password Password
	Tags     map[string]int
	Manager  *User
}

func main() {
	u := User{Name: "alice", Password: "hunter2", Tags: map[string]int{"b": 2, "a": 1}}
	fmt.Print(dump.SDump(u))
}
```

It prints:

```
main.User{
  Name: "alice",
  Password: "***",
  Tags: map[string]int{
    "a": 1,
    "b": 2,
  },
  Manager: nil,
}
```

## API

```go
type Dumper interface{ Dump() string }

func Dump(v any) []byte
func SDump(v any) string
func FPrettyPrint(w io.Writer, v any)
func PrettyPrint(v any) // to os.Stdout
```

## Behavior and guarantees

- Output ends with a newline.
- Structs print their exported fields in declaration order; unexported fields
  are skipped.
- Map keys of string, integer, unsigned, and float kinds are sorted; other key
  kinds (structs, pointers, interfaces) print in Go's random map order.
- Non-nil pointers print as `&` followed by the value. Nil pointers and
  interfaces print `nil`; nil and empty slices and maps both print `T{}`.
- A pointer, map, or slice that contains itself prints `<cycle detected: T>`.
  The same pointer appearing twice side by side is not a cycle.
- A value (or its address) implementing `Dumper` prints its `Dump()` result
  verbatim, which is useful for redacting secrets. `Dump` is not called on nil
  pointers.
- Scalars use `%#v`, so strings are quoted.

## Testing

```sh
cd types
go vet ./dump
go test -race -count=1 ./dump
```
