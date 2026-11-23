# ahocorasick

Find every occurrence of many byte patterns in one pass over the input with
the Aho-Corasick algorithm.

**Use it for:** keyword spotting, blocklists, and tagging text against a
fixed dictionary, where the dictionary is built once and scanned often.

**Don't use it for:** a single pattern (`bytes.Index`), regular
expressions, case-insensitive matching (normalize the input and patterns
yourself), or very large dictionaries where 1 KiB per trie node is too much.

## Installation

```sh
go get github.com/alextanhongpin/core/types/ahocorasick
```

The package is part of the `github.com/alextanhongpin/core/types` module;
check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/types/ahocorasick"
)

func main() {
	patterns := [][]byte{[]byte("he"), []byte("she"), []byte("his"), []byte("hers")}
	m := ahocorasick.NewMatcher(patterns)

	for _, match := range m.FindAll([]byte("ushers")) {
		fmt.Printf("%s at [%d:%d]\n", patterns[match.PatternIndex], match.Start, match.End)
	}
}
```

It prints:

```
she at [1:4]
he at [2:4]
hers at [2:6]
```

## API

```go
type Match struct {
	Start, End   int // text[Start:End] is the match
	PatternIndex int // index into the patterns passed to NewMatcher
}

func NewMatcher(patterns [][]byte) *Matcher
func (m *Matcher) FindAll(text []byte) []Match
func (m *Matcher) MatchFunc(text []byte, fn func(Match) bool)
```

## Behavior and guarantees

- All matches are reported, including overlapping and nested ones.
- Order: by `End`; for the same `End`, longest pattern first, then lower
  pattern index. Duplicate patterns each produce a match.
- Matching is byte-wise and case-sensitive. Offsets are byte offsets.
- Empty patterns never match. The patterns are copied into the automaton
  and may be modified afterwards.
- `MatchFunc` allocates nothing per match; return false to stop the scan.
- Building costs O(total pattern bytes × 256); memory is 1 KiB per trie
  node (about one node per distinct pattern prefix byte).
- A `Matcher` is immutable after `NewMatcher` and safe for concurrent use.

## Testing

```sh
go vet ./ahocorasick
go test -race -count=1 ./ahocorasick
go test -bench=. ./ahocorasick
```
