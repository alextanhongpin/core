# fstring

A `text/template` source tagged with the type of data it expects.

**Use it for:** prompt and message templates kept in files or constants,
where the type parameter documents what the template is executed with.

**Don't use it for:** HTML (use `html/template`), or hot paths: the
template is parsed on every call.

## Installation

```sh
go get github.com/alextanhongpin/core/types/fstring
```

The package is part of the `github.com/alextanhongpin/core/types` module;
check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/types/fstring"
)

type user struct{ Name string }

func main() {
	s, err := fstring.FString[user]("Hello {{ .Name }}!").Format(user{"Alice"})
	if err != nil {
		panic(err)
	}
	fmt.Println(s)

	j, _ := fstring.FString[any]("{{ json . }}").FormatFunc(map[string]int{"n": 1}, fstring.FuncMap)
	fmt.Println(j)
}
```

It prints `Hello Alice!` and `{"n":1}`.

## API

```go
type FString[T any] string

func (f FString[T]) Format(v T) (string, error)
func (f FString[T]) FormatFunc(v T, fm template.FuncMap) (string, error)
func (f FString[T]) String() string
func (f FString[T]) Bytes() []byte

func ReadFile[T any](path string) (FString[T], error)

var FuncMap template.FuncMap // "json" and "parse_json"
```

## Behavior and guarantees

- Parse and execution errors are returned, with an empty string.
- Missing map keys and nil data render as `<no value>`, as in
  `text/template`; missing struct fields are execution errors.
- `FuncMap` is used only when passed to `FormatFunc`: `json` marshals a
  value to a string and `parse_json` unmarshals a string into `any`.
- Output is not escaped.
- `ReadFile` reads the template source; it is parsed when formatted.

## Testing

```sh
go vet ./fstring
go test -race -count=1 ./fstring
```
