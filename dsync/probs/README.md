# probs

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
