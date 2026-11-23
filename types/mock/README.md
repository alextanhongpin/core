# mock

Minimal test doubles: return canned string options per method and record each
call's arguments.

**Use it for:** hand-written fakes where a method's behavior is selected by a
string (`"ok"`, `"timeout"`, `"error"`) and you want to assert on the
arguments it received.

**Don't use it for:** typed return values, expectation matching, or generated
mocks; use a hand-written fake or a mocking library.

## Installation

```sh
go get github.com/alextanhongpin/core/types/mock
```

This package is part of the `types` module; check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/types/mock"
)

type Service struct {
	*mock.Mock
}

func (s *Service) Fetch(id int) string { return s.Call(id) }

func main() {
	s := &Service{}
	s.Mock = mock.New(s, mock.Options{}.With("Fetch", "ok", "timeout"))

	fmt.Println(s.Fetch(1), s.Fetch(2), s.Fetch(3))
	fmt.Println(s.Calls().Values("Fetch"))
}
```

It prints `ok timeout ok`, then `[[1] [2] [3]]`.

## API

```go
type Map[T any] map[string][]T
type Options = Map[string] // method name -> options returned by Call, in order
type Calls = Map[[]any]    // method name -> arguments of each call

func (m Map[T]) Values(key string) []T
func (m Map[T]) With(key string, vals ...T) Map[T] // copy with vals appended
func (m Map[T]) Del(key string)

func New(v any, options Options) *Mock

func (m *Mock) Call(args ...any) string
func (m *Mock) Calls() Calls
```

## Behavior and guarantees

- `New` panics if `options` names a method that `v` (a struct or pointer to
  one) does not export. Methods are found with
  [`structs.GetMethodNames`](../structs).
- `Call` uses the runtime caller name, so it must be called directly from the
  exported method. Calling it from a closure or helper records the wrong name.
- The nth call to a method returns `options[n % len(options)]`, cycling. A
  method without options returns `""` and is still recorded.
- `New` copies `options`; `With` and `Calls` return copies, so changing them
  does not affect the mock.
- `Mock` is safe for concurrent use.

## Testing

```sh
go vet ./mock/
go test -race -count=1 ./mock/
```

Run from the `types` directory.
