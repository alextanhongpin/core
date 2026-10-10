# urls

Normalize, compare, hash, and extract URLs.

## Installation

```sh
go get github.com/alextanhongpin/core/types/urls
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/urls"
)

func main() {
	u, err := urls.Normalize("https://example.com/page/?q=1#section")
	if err != nil {
		panic(err)
	}
	fmt.Println(u.String(), urls.Hash(u))
}
```

## Behavior and limits

Normalize trims one trailing slash from the input before parsing, then removes query and fragment. IsEqual compares normalized strings and returns false on parse errors. Hash returns a SHA-256 hex digest of the normalized URL. IsScopedDomain compares scheme and host exactly. IsScopedPrefix performs a string prefix check and is not a general URL containment validator. Extract returns anchor href values from HTML, without resolving relative links.
