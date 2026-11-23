# useragent

Fetch lists of real browser user-agent strings from useragentstring.com and
keep them in memory, optionally cached on disk.

**Use it for:** rotating realistic `User-Agent` headers in scrapers and load
tests.

**Don't use it for:** parsing or classifying user-agent strings, or
anything that needs the list to be current or available: it depends on a
third-party site and its HTML layout.

## Installation

```sh
go get github.com/alextanhongpin/core/types/useragent
```

The package is part of the `github.com/alextanhongpin/core/types` module;
check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/types/useragent"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 10 * time.Second}
	uas, err := useragent.Fetch(ctx, client, useragent.Firefox)
	if err != nil {
		panic(err)
	}
	fmt.Println(len(uas) > 0)
}
```

To cache the list in a file and refresh it daily, pair a `Loader` with
[`markdown.Loader`](../markdown):

```go
uas := useragent.NewLoader([]string{useragent.Firefox, useragent.Chrome})
cache := markdown.NewLoader("cache/useragents.md", nil, 24*time.Hour, uas)
if err := cache.Load(); err != nil { // fetches only when missing or stale
	panic(err)
}
fmt.Println(len(uas.Load()))
```

## API

```go
const Chrome, Edge, Firefox, InternetExplorer, Mozilla, Opera, OperaMini, Safari string
var Browsers []string

func Fetch(ctx context.Context, client *http.Client, browser string) ([]string, error)
func Parse(r io.Reader) ([]string, error)

type Loader struct {
	Client *http.Client // used by WriteTo; nil means http.DefaultClient
	// unexported fields
}

func NewLoader(browsers []string) *Loader
func (l *Loader) WriteTo(w io.Writer) (int64, error) // fetch, one string per line
func (l *Loader) ReadFrom(r io.Reader) (int64, error) // load lines
func (l *Loader) Load() []string
```

## Behavior and guarantees

- `Fetch` requests
  `http://www.useragentstring.com/pages/useragentstring.php?name=<browser>`
  with the browser name query-escaped. Non-200 responses are errors. A nil
  client uses `http.DefaultClient`, which has no timeout; pass a context
  or client with one.
- `Parse` returns the trimmed text of the links inside the element with id
  `liste`, skipping empty links and the `Google` and `Chrome` headings. A
  page without that element yields an empty list.
- `WriteTo` fetches each browser in order without a deadline, writes one
  string per line, and stops at the first error.
- `ReadFrom` replaces the list with the non-empty trimmed lines of the
  input; on error the list is unchanged.
- `Load` returns a copy of the list, or nil before the first successful
  `ReadFrom`. `Load` and `ReadFrom` are safe for concurrent use.

## Testing

Tests run offline against a stubbed `http.Client`:

```sh
go vet ./useragent
go test -race -count=1 ./useragent
```
