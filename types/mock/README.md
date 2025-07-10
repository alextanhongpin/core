# Mock Package

A minimalistic mock utility for Go, designed to help with method-based option injection and runtime method validation. Useful for building test doubles and flexible test helpers.

## Features
- Register all exported methods of a struct or interface
- Attach options to specific methods (type-safe)
- Validate method names at construction (panic on unknown)
- Nil-safe options map and fluent API

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/alextanhongpin/core/types/mock"
)

type Service struct {
    *mock.Mock
}

func (s *Service) WithOptions(options mock.Options) *Service {
    s.Mock = mock.New(s, options)
    return s
}

func (s *Service) Foo() string { return s.Option() }
func (s *Service) Bar() string { return s.Option() }

func main() {
    opts := mock.Options{}.
        With("Foo", "fast").
        With("Bar", "slow")
    s := new(Service).WithOptions(opts)
    fmt.Println(s.Foo()) // fast
    fmt.Println(s.Bar()) // slow
}
```

## API

### type Options
A type-safe map for method options. Use `Options{}.With("Method", "option")` to build options fluently.

### func New(v any, options Options) *Mock
Creates a new Mock for the exported methods of v, with the given options. Panics if any option is for an unknown method.

### func (*Mock) Option() string
Returns the option for the calling method, or an empty string if not set.

---
MIT License
