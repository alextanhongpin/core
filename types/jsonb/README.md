# jsonb

Find, replace, and delete values inside decoded JSON (`map[string]any` and
`[]any`) using dot-separated path patterns.

**Use it for:** editing JSON whose shape you don't want to model as structs,
such as redacting fields before logging or patching documents from a store.

**Don't use it for:** typed data (decode into a struct), JSON Pointer or
JSONPath compatibility, or keys that contain dots.

## Installation

```sh
go get github.com/alextanhongpin/core/types/jsonb
```

The package is part of the `github.com/alextanhongpin/core/types` module;
check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/alextanhongpin/core/types/jsonb"
)

func main() {
	var data any
	_ = json.Unmarshal([]byte(`{"users":[{"name":"Alice"},{"name":"Bob"}]}`), &data)

	names, _ := jsonb.Get(data, "users[].name")
	fmt.Println(names)

	_ = jsonb.Set(data, "users[].name", "redacted")
	b, _ := json.Marshal(data)
	fmt.Println(string(b))
}
```

It prints:

```
[Alice Bob]
{"users":[{"name":"redacted"},{"name":"redacted"}]}
```

## API

```go
var ErrSkip   error // returned by a Reviver callback: stop, no error
var ErrDelete error // returned by a Reviver callback: remove this value
var ErrType   error // Set: existing value has a different Go type

func Get(a any, pattern string) ([]any, error)
func Set(a any, pattern string, value any) error
func Delete(a any, pattern string) error
func Reviver(a any, fn func(path Path, value any) (any, error)) error

type Path []string
func (p Path) Match(pattern string) bool
func (p Path) String() string
func (p Path) Base() (string, bool)
```

## Behavior and guarantees

- Paths have one segment per map key. Array elements append their index to
  the last segment: `items[0]`, `matrix[1][2]`, or `[0]` for an element of a
  root array.
- In patterns, `*` matches any one segment and `[]` matches any index, so
  `items[].name` matches `items[3].name`. Patterns must have the same number
  of segments as the path.
- `Reviver` visits every value once, children before parents and the root
  last, and replaces each value with what `fn` returns. Map keys are visited
  in random order, so `Get` results from maps are unordered.
- `ErrDelete` removes a map entry or an array element (nested arrays are
  replaced with a shorter copy). Deleting the root or an element of a root
  array returns an error wrapping `ErrDelete` and leaves that array unchanged.
- `ErrSkip` stops the traversal and `Reviver` returns nil. Any other error
  stops the traversal and is returned. Changes made before a stop are kept.
- `Set` only replaces existing values, and only with a value of the same Go
  type; otherwise it returns an error wrapping `ErrType`. Decoded JSON
  numbers are `float64`, so set `2.0`, not `2`.
- JSON `null` values are preserved unless deleted.
- Values are mutated in place; not safe for concurrent use on the same data.

## Testing

```sh
go vet ./jsonb
go test -race -count=1 ./jsonb
```
