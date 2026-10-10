# policy

Evaluate exact string allow and deny lists.

## Installation

```sh
go get github.com/alextanhongpin/core/types/policy
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/policy"
)

func main() {
	p := policy.Policy{AllowList: []string{"read", "write"}, DenyList: []string{"write"}}
	fmt.Println(p.Allow("read"), p.Allow("write"))
}
```

## Behavior and limits

DenyList takes precedence. An empty AllowList permits values absent from DenyList; a nonempty AllowList denies unlisted values. The zero value allows everything. Matching is exact and case sensitive.
