# Mock Package

A minimalistic mock utility for Go, designed to help with method-based option injection and runtime method validation. Useful for building test doubles and flexible test helpers.

## Features
- Register all exported methods of a struct or interface
- Attach options to specific methods
- Validate method names at runtime
- Panic on unknown methods or invalid option formats

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/alextanhongpin/core/types/mock"
)

type Service struct{}
func (Service) Foo() {}
func (Service) Bar() {}

func (s Service) FooWithMock(m *mock.Mock) string {
    return m.Option()
}

func main() {
    m := mock.New(Service{}, "Foo=fast", "Bar=slow")
    fmt.Println(Service{}.FooWithMock(m)) // prints "fast"
}
```

## API

### func New(v any, options ...string) *Mock
Registers all exported method names of the given value and attaches options. Panics on error.

### func (Mock) Option() string
Returns the option for the calling method. Panics if called outside a registered method.

---
MIT License
