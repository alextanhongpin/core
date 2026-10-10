# specification

Compose typed predicates into reusable specifications.

## Installation

```sh
go get github.com/alextanhongpin/core/types/specification
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/specification"
)

func main() {
	positive := specification.Func[int](func(n int) bool { return n > 0 })
	even := specification.Func[int](func(n int) bool { return n%2 == 0 })
	fmt.Println(specification.And[int](positive, even).IsSatisfiedBy(4))
}
```

## Behavior and limits

Func adapts a function to IsSatisfiedBy(T) bool. And, Or, Not, and Xor compose specifications. All and Any accept variadic specifications and short circuit; All with no arguments is true, while Any with no arguments is false.
