# stringcase

Convert identifiers and phrases between snake_case, kebab-case, camelCase,
PascalCase, and Title Case, keeping initialisms such as `ID` and `HTTP`
intact.

**Use it for:** generating Go-style names, column names, or labels from
mixed-case input.

**Don't use it for:** natural-language title casing (articles and small
words are capitalized too) or round-tripping the exact original casing.

## Installation

```sh
go get github.com/alextanhongpin/core/types/stringcase
```

The package is part of the `github.com/alextanhongpin/core/types` module;
check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/types/stringcase"
)

func main() {
	fmt.Println(stringcase.ToSnake("userIDsAPIURL"))
	fmt.Println(stringcase.ToCamel("user_id"))
	fmt.Println(stringcase.ToPascal("api-response"))
	fmt.Println(stringcase.ToTitle("httpServer"))

	t := stringcase.NewTokenizer("SKU")
	fmt.Println(t.Pascal("product_sku"))
}
```

It prints:

```
user_ids_api_url
userID
APIResponse
HTTP Server
ProductSKU
```

## API

```go
var DefaultInitialisms []string

func ToSnake(s string) string
func ToKebab(s string) string
func ToCamel(s string) string
func ToPascal(s string) string
func ToTitle(s string) string
func Tokenize(s string, initialisms []string) []string

func NewTokenizer(initialisms ...string) *Tokenizer
func (t *Tokenizer) Snake(text string) string
func (t *Tokenizer) Kebab(text string) string
func (t *Tokenizer) Camel(text string) string
func (t *Tokenizer) Pascal(text string) string
func (t *Tokenizer) Title(text string) string
func (t *Tokenizer) Tokenize(text string) []string
```

The package-level functions use `DefaultInitialisms`.

## Behavior and guarantees

- Characters other than Unicode letters and digits separate words. Within a
  word, splits happen at lower-to-upper (`apiServer`), digit-to-upper
  (`1990Year`), upper-to-digit (`ID1`), and acronym-to-word
  (`HTTPServer`) boundaries.
- Initialisms are matched case-sensitively in the input. Where several
  splits are possible, the one that matches the most initialisms wins, so
  `XMLHTTPRequest` becomes `xml http request`.
- Words are lowercased, then written back with initialisms in their
  configured form and other words capitalized. Initialisms that differ only
  by case share one form, the first listed: `HTTPs` in the input comes back
  as `HTTPS`.
- Camel keeps the first word lowercase, including an initialism
  (`ids_count` becomes `idsCount`).
- Splitting tries every initialism at each step, so cost grows with the
  number of initialisms and their occurrences in a word; keep lists short.
- Functions and `Tokenizer` methods are safe for concurrent use.

## Testing

```sh
go vet ./stringcase
go test -race -count=1 ./stringcase
```
