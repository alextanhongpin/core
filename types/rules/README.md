# rules

Evaluate named specifications in registration order.

## Installation

```sh
go get github.com/alextanhongpin/core/types/rules
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/rules"
	"github.com/alextanhongpin/core/types/specification"
)

func main() {
	engine := rules.NewEngine[int]()
	engine.AddRule(rules.NewRule("positive", specification.Func[int](func(n int) bool { return n > 0 })))
	fmt.Println(engine.Evaluate(3))
}
```

## Behavior and limits

Specifications implement IsSatisfiedBy(T) bool. IsSatisfiedBy requires all rules to pass. Evaluate stops at the first failure and reports its name; success joins evaluated names with AND. An empty engine passes. Register rules before sharing an engine across readers.
