# structs

Reflection helpers: clear struct fields by name or by path, list a type's
exported methods, and test for nil-or-zero values.

**Use it for:** redacting fields such as passwords or timestamps before
snapshot tests or logging, and building test helpers (see [`mock`](../mock)).

**Don't use it for:** hot paths (reflection is slow), or general deep copies
and comparisons.

## Installation

```sh
go get github.com/alextanhongpin/core/types/structs
```

This package is part of the `types` module; check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/types/structs"
)

type Account struct {
	Email    string
	Password string
}

func main() {
	accounts := []Account{{"a@example.com", "hunter2"}, {"b@example.com", "secret"}}
	if err := structs.ZeroField(accounts, "Password"); err != nil {
		panic(err)
	}
	fmt.Println(accounts)
}
```

It prints `[{a@example.com } {b@example.com }]`.

## API

```go
func ZeroField(input any, fields ...string) error     // by name, at any depth
func ZeroPath(input any, fieldPaths ...string) error  // by dotted path, e.g. "Profile.Age"
func GetMethodNames(v any) ([]string, error)
func IsNilOrZero(x any) bool
```

## Behavior and guarantees

- `ZeroField` clears every struct field whose name is in `fields`, wherever it
  appears: it follows pointers and interfaces and visits struct fields, slice
  elements, and array elements. Map values are not visited. Pointer cycles are
  visited once.
- `ZeroPath` clears exactly one field per path. Pointers and interfaces on the
  path are followed, slices and arrays apply the rest of the path to every
  element, and a nil pointer ends that branch without error. A missing field,
  or a segment applied to a non-struct, is an error.
- Both need settable fields: pass a pointer (or a slice, whose elements are
  addressable). A matching unexported field, or a struct passed by value, is
  an error. Both reject an empty field list, and changes made before an error
  are kept.
- `GetMethodNames` returns the sorted exported methods of `*T` (value and
  pointer receivers) whether given a `T` or a `*T`. It errors on nil.
- `IsNilOrZero` is `x == nil || reflect.ValueOf(x).IsZero()`: typed nil
  pointers, maps, slices, channels, and funcs are zero; a non-nil empty slice
  or map is not.

## Testing

```sh
go vet ./structs/
go test -race -count=1 ./structs/
```

Run from the `types` directory.
