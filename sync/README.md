# Concurrent programming utilities

Each package is an independent Go module. Install and test it from its own directory; this directory is not a root Go module.

| Package | Purpose |
| --- | --- |
| [broadcast](./broadcast/) | Broadcast values to subscribers; slow subscribers apply backpressure. |
| [cache](./cache/) | Weak-reference cache whose values may be reclaimed by garbage collection. |
| [circuitbreaker](./circuitbreaker/) | Failure and latency based circuit breaker, function decorator, and HTTP transport. |
| [dataloader](./dataloader/) | Batch loading and deduplication of concurrent key requests. |
| [goroutine](./goroutine/) | Start, cancel, and wait for a managed background function. |
| [lock](./lock/) | Keyed blocking locks and nonblocking try locks. |
| [pipeline](./pipeline/) | Channel sources, transformations, batching, and sinks. |
| [promise](./promise/) | Context-aware asynchronous results and promise combinators. |
| [rate](./rate/) | Time-normalized counters, error ratios, and probabilistic admission. |
| [ratelimit](./ratelimit/) | Per-key fixed-window and GCRA rate limiting. |
| [retry](./retry/) | Retry policies and HTTP request retries. |
| [snapshot](./snapshot/) | Notifications triggered by count and time policies. |
| [throttle](./throttle/) | Limit concurrent operations and queued admission. |
| [timer](./timer/) | Cancelable interval and timeout callbacks. |

Check the individual package documentation for public APIs, configuration, cancellation, and cleanup contracts. Channel pipelines require consumers to drain their outputs; context cancellation at a source does not make every downstream stage cancelable. Weak caches do not promise permanent retention or exactly one call to a value factory.

## Installation

```sh
go get github.com/alextanhongpin/core/sync/dataloader
```

## Verification

Run checks inside an individual module:

```sh
cd sync/dataloader
go test -race ./...
go vet ./...
```

To check the tracked modules from the repository root:

```sh
for module in sync/*/go.mod; do
    (cd "$(dirname "$module")" && go test -race ./... && go vet ./...) || exit 1
done
```

Modules declare their supported Go version in `go.mod`. Dependencies between packages normally resolve to released module versions; use a temporary Go workspace when testing changes to multiple local modules together.
