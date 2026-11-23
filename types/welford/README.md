# welford

Flag outliers in a stream by z-score, using Welford's online algorithm for the
running mean and sample variance in constant memory.

**Use it for:** simple anomaly alerts on latencies, counts, or sizes where
values are roughly normally distributed.

**Don't use it for:** seasonal or trending data (the statistics never forget
old values), or skewed distributions where z-scores mislead.

## Installation

```sh
go get github.com/alextanhongpin/core/types/welford
```

This package is part of the `types` module; check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/types/welford"
)

func main() {
	d := welford.NewDetector(3) // flag values more than 3 standard deviations out
	for _, ms := range []float64{101, 99, 102, 98, 100, 250, 101} {
		if d.Update(ms) {
			fmt.Println("anomaly:", ms)
		}
	}
	fmt.Printf("n=%d mean=%.1f\n", d.Count(), d.Mean())
}
```

It prints:

```
anomaly: 250
n=7 mean=121.6
```

## API

```go
func NewDetector(threshold float64) *WelfordAnomalyDetector

func (w *WelfordAnomalyDetector) Update(x float64) bool // true if x is an anomaly
func (w *WelfordAnomalyDetector) Count() int
func (w *WelfordAnomalyDetector) Mean() float64
func (w *WelfordAnomalyDetector) Variance() float64 // sample variance (n-1)
func (w *WelfordAnomalyDetector) StdDev() float64
```

## Behavior and guarantees

- `Update` scores `x` against the values seen before it, then adds `x` to the
  statistics. A value is an anomaly when `|x - mean| / stddev > threshold`.
- Nothing is flagged until at least two earlier values exist with a nonzero
  standard deviation.
- Anomalies are added to the statistics too, so a burst of outliers raises the
  mean and variance and later outliers become harder to detect.
- `Mean` is 0 with no values; `Variance` and `StdDev` are 0 with fewer than two.
- The zero value is usable and has threshold 0. Not safe for concurrent use.

## Testing

```sh
go vet ./welford/
go test -race -count=1 ./welford/
```

Run from the `types` directory.
