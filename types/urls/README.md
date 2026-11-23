# urls

Normalize, compare, scope, and hash URLs, and extract links from HTML. Built
for crawlers that must decide whether two links point to the same page.

**Use it for:** deduplicating crawled pages, restricting a crawl to a site
or a path, and collecting `<a href>` values.

**Don't use it for:** URLs whose query string identifies the resource
(`?id=42`): normalization drops it. It does not resolve relative links, so
use `(*url.URL).ResolveReference` for that.

## Installation

```sh
go get github.com/alextanhongpin/core/types/urls
```

The package is part of the `github.com/alextanhongpin/core/types` module;
check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"fmt"
	"net/url"

	"github.com/alextanhongpin/core/types/urls"
)

func main() {
	u, _ := urls.Normalize("HTTPS://Example.com:443/docs/?page=2#intro")
	fmt.Println(u)

	root, _ := url.Parse("https://example.com/docs")
	child, _ := url.Parse("https://example.com/docs/install")
	sibling, _ := url.Parse("https://example.com/docs-old")
	fmt.Println(urls.IsScopedPrefix(root, child), urls.IsScopedPrefix(root, sibling))
}
```

It prints `https://example.com/docs` and `true false`.

## API

```go
func Normalize(link string) (*url.URL, error)
func IsEqual(a, b string) bool
func Hash(u *url.URL) string
func IsScopedDomain(root, u *url.URL) bool
func IsScopedPrefix(root, u *url.URL) bool
func Extract(r io.Reader) []string
```

## Behavior and guarantees

- `Normalize` removes the query string, fragment, trailing slashes, and the
  default port (`:80` for http, `:443` for https), and lowercases the scheme
  and host. The path's case and escaping are kept.
- `IsEqual` compares normalized forms; it is false if either fails to
  parse. `Hash` is the hex SHA-256 of the normalized form, so equal URLs
  hash alike.
- `IsScopedDomain` compares scheme and host after normalization.
  `IsScopedPrefix` additionally requires `u`'s path to continue `root`'s
  after a `/`: a URL is not under itself, and `/docs-old` is not under
  `/docs`. Both accept unnormalized input and never modify their arguments.
- `Extract` returns `<a>` `href` values in document order exactly as
  written, including relative, fragment-only, and `mailto:` links. It stops
  at the end of input or on a read error, returning what it found.

## Testing

```sh
go vet ./urls
go test -race -count=1 ./urls
```
