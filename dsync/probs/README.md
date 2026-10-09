# probs

Query approximate membership, cardinality, frequency, and ranking structures in Redis.

## Installation and documentation

```sh
go get github.com/alextanhongpin/core/dsync/probs
go doc github.com/alextanhongpin/core/dsync/probs
```

[Go API reference](https://pkg.go.dev/github.com/alextanhongpin/core/dsync/probs) · [Module requirements](go.mod)

## Typical use

Use HyperLogLog for unique visitor estimates, Bloom filters for likely membership, and TopK for frequent items.

Examples assume a caller-owned Redis client and a context. Create the client once
and close it after all operations finish:

```go
client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
defer client.Close()
ctx := context.Background()
```

Import `context`, `github.com/redis/go-redis/v9`, and this package's import path.

The usage example below shows the main operation. Application types and
callbacks such as `User` and `fetchUsers` belong to your application; import this
package and the standard packages referenced in the snippet.

## Behavior and configuration

Redis wrappers for Bloom and Cuckoo filters, HyperLogLog, count-min sketches,
t-digests, and TopK. Approximate membership/count/rank behavior follows the Redis
commands; these wrappers do not turn estimates into exact results.

```go
hll, err := probs.NewHyperLogLog(client)
if err != nil { return err }
_, err = hll.Add(ctx, "visitors", "alice", "bob")
count, err := hll.Count(ctx, "visitors")
```

Constructors return an error for a nil client and retain it privately. Clients are
borrowed and must remain open until all calls finish; wrappers do not close them.
There is no optional constructor configuration to wrap in a Config. Must helpers
such as `MustNewHyperLogLog` are available for startup wiring. Instances support
concurrent calls with a concurrency-safe Redis client.

Bloom/Cuckoo, CMS, t-digest, and TopK require Redis probabilistic commands/modules;
HyperLogLog uses core Redis commands. Explicit Reserve/Create methods configure
backend structures. Automatic creation on a missing structure retries the command
once, so missing merge sources or unsupported commands return errors rather than
retrying indefinitely. A failed command may have backend-specific partial effects.

Tests use Redis Stack via Docker. Run `go test -race -timeout 120s ./...` and
`go vet ./...`.

## Expected errors

Nil clients fail at construction. Unsupported commands/modules and invalid backend parameters return Redis errors. Approximate results are successful results, not error indicators.

## Pitfalls

Bloom membership can have false positives; counts and ranks may be estimates. Install the Redis probabilistic commands required by the chosen structure.
