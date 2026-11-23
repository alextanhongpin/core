# assert

Build per-field validation messages without a validation framework. Every check
returns `""` when it passes and a human-readable message when it fails, so a
request's errors are a plain `map[string]string`.

**Use it for:** validating request structs and configuration in a `Validate`
method, and returning field errors to API clients.

**Don't use it for:** test assertions (use `testing` or a test library), or
struct-tag driven validation.

## Installation

```sh
go get github.com/alextanhongpin/core/types
```

`assert` is a package in the `types` module; import it as
`github.com/alextanhongpin/core/types/assert`. Check the module's `go.mod` for the
required Go version.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/types/assert"
)

type SignUp struct {
	Name  string
	Email string
	Age   int
	Plan  string
}

func main() {
	req := SignUp{Name: "A", Email: "not-an-email", Age: 15}

	errs := assert.Map(map[string]string{
		"name":  assert.Required(req.Name, assert.MinLength(req.Name, 2)),
		"email": assert.Required(req.Email, assert.Email(req.Email)),
		"age":   assert.Required(req.Age, assert.Range(req.Age, 18, 120)),
		"plan":  assert.Optional(req.Plan, assert.OneOf(req.Plan, "free", "pro")),
	})
	fmt.Println(len(errs))
	fmt.Println(errs["name"])
	fmt.Println(errs["email"])
	fmt.Println(errs["age"])
}
```

It prints:

```
3
must be at least 2 characters
must be a valid email address
must be between 18 and 120
```

`plan` is empty and optional, so it is skipped and dropped from the map.

## API

```go
func Is(ok bool, msg string, args ...any) string   // "" if ok, else fmt.Sprintf(msg, args...)
func Required(v any, checks ...string) string      // "required" if v is zero, plus failed checks
func Optional(v any, checks ...string) string      // "" if v is zero, else like Required
func Map(fields map[string]string) map[string]string // drops fields with ""
func IsZero(v any) bool

func MinLength(s string, min int) string
func MaxLength(s string, max int) string
func Range[T cmp.Ordered](v, lo, hi T) string
func OneOf[T comparable](v T, allowed ...T) string
func Email(s string) string                        // uses email.IsValid
```

## Behavior and guarantees

- `Required` and `Optional` join messages with `", "`; empty messages are skipped.
  A zero value is reported as `required` together with any checks that also
  failed, e.g. `"required, must be a valid email address"`.
- Checks are evaluated eagerly by Go before `Required`/`Optional` run; they only
  decide which messages are kept.
- `IsZero` is true for `nil`, zero values (including zero structs and arrays),
  empty or nil maps and slices, nil pointers, and pointers to zero values.
- `MinLength` and `MaxLength` count runes, not bytes.
- `Range` is inclusive at both ends.
- `Map` returns a new map; the input is not modified.

## Testing

```sh
cd types
go vet ./assert
go test -race -count=1 ./assert
```
