# ahocorasick

Compile multiple byte patterns and find their matches in one scan.

## Installation

```sh
go get github.com/alextanhongpin/core/types/ahocorasick
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/ahocorasick"
)

func main() {
	m := ahocorasick.NewMatcher([][]byte{[]byte("he"), []byte("she")})
	for _, match := range m.FindAll([]byte("she")) {
		fmt.Println(match)
	}
}
```

## Behavior and limits

Matches contain byte offsets Start and End (exclusive), and PatternIndex into the original patterns. Empty patterns are ignored. MatchFunc streams matches without collecting a slice; return false to stop scanning. Matching is case sensitive and byte based.
