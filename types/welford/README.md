# welford

Detect outliers using running mean and sample variance.

## Installation

```sh
go get github.com/alextanhongpin/core/types/welford
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/welford"
)

func main() {
	detector := welford.NewDetector(2)
	for _, value := range []float64{10, 11, 9, 10, 100} {
		fmt.Println(detector.Update(value))
	}
}
```

## Behavior and limits

Update includes the new value in the running statistics before comparing its absolute Z-score with the threshold. It returns false for fewer than two samples or zero standard deviation. State is cumulative with no reset or window API; synchronize concurrent access externally.
