# dsync

Redis-backed coordination and storage packages. Each directory is an independent
Go module with its own release tag.

| Package | Purpose |
| --- | --- |
| [cache](cache/) | Typed storage, file persistence, and cache-fill leases |
| [channel](channel/) | Redis stream broadcast reads with caller-owned cursors |
| [circuitbreaker](circuitbreaker/) | Atomic breaker transitions and generation-aware accounting |
| [idempotent](idempotent/) | Retained request/results and renewable in-flight leases |
| [lock](lock/) | Token-checked single-primary leases with synchronous callbacks |
| [probs](probs/) | Redis probabilistic structure wrappers |
| [ratelimit](ratelimit/) | Atomic fixed-window and GCRA admission |
| [singleflight](singleflight/) | Local coalescing and Redis-coordinated cache fills |

Configuration is passed by value, defaulted and validated before use, and held
privately. Redis clients and callback dependencies are shared; callers retain
client cleanup ownership. See package documentation for constructor errors, Must
helpers, zero semantics, and lifecycle contracts.

Redis coordination does not promise exactly-once external effects across lease
expiry, failover, partitions, or data loss. A local fallback would coordinate only
one process and cannot preserve a distributed guarantee. Cache publication scripts
using separate lease/data keys target one Redis primary, not Redis Cluster.

Circuit breaker and rate limiter deployments must install their Lua functions
using the package Setup method. Updating client code alone does not update Redis
functions.

Run tests and vet from each module. Integration tests use Docker-backed Redis;
probabilistic structure tests use Redis Stack. Race tests cover local lifecycle
and concurrent behavior, while Redis supplies atomic backend operations.
