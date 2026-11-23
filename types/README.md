# types

Small, dependency-light Go packages for values, collections, validation, and
text handling. All packages live in one Go module,
`github.com/alextanhongpin/core/types`; each directory is an importable package
with its own README.

## Packages

### Validation and configuration

| Package | Purpose |
| --- | --- |
| [assert](./assert/) | Build field-level validation messages from composable checks. |
| [email](./email/) | Validate, normalize, and split email addresses. |
| [env](./env/) | Parse typed values, durations, times, and slices from environment variables. |
| [must](./must/) | Panic on an error during startup wiring or in tests. |

### Collections and data structures

| Package | Purpose |
| --- | --- |
| [list](./list/) | Slice helpers the standard library lacks: map, filter, reduce, group, partition. |
| [sets](./sets/) | Generic set with union, intersection, difference, and subset checks. |
| [queue](./queue/) | Generic priority queue. |
| [dags](./dags/) | Directed graph with topological sort. |
| [states](./states/) | Boolean condition counting, ordered step sequences, and a finite state machine. |

### Numbers, statistics, and randomness

| Package | Purpose |
| --- | --- |
| [number](./number/) | Clamp, range-check, interpolate, and rescale numbers. |
| [vector](./vector/) | Dot product, magnitude, and cosine similarity. |
| [welford](./welford/) | Online mean and variance with z-score outlier detection. |
| [reservoir](./reservoir/) | Uniform fixed-size sample from a stream of unknown length. |
| [random](./random/) | Non-cryptographic random values, durations, choices, and shuffles; seedable for tests. |
| [safe](./safe/) | AES-GCM encryption, HMAC signatures, and cryptographically secure secrets. |

### Text, encoding, and files

| Package | Purpose |
| --- | --- |
| [stringcase](./stringcase/) | Convert between camel, pascal, snake, kebab, and title case, with initialisms. |
| [fstring](./fstring/) | Typed `text/template` strings. |
| [ahocorasick](./ahocorasick/) | Multi-pattern byte matching with the Aho-Corasick automaton. |
| [jsonb](./jsonb/) | Get, set, delete, and transform values in decoded JSON by path. |
| [jsonl](./jsonl/) | Read and write typed JSON Lines files. |
| [markdown](./markdown/) | Read and write Markdown files with YAML frontmatter. |
| [urls](./urls/) | Normalize, compare, scope, hash, and extract URLs. |
| [useragent](./useragent/) | Fetch and store browser user-agent strings. |

### Testing, debugging, and lifecycle

| Package | Purpose |
| --- | --- |
| [mock](./mock/) | Per-method options and call recording for hand-written test doubles. |
| [structs](./structs/) | Reflection helpers: zero fields by name or path, list method names. |
| [dump](./dump/) | Deterministic pretty-printing of Go values. |
| [tracker](./tracker/) | Timed operation logging on top of `log/slog`. |
| [shutdown](./shutdown/) | Run cleanup callbacks concurrently and collect their errors. |

## Installation

```sh
go get github.com/alextanhongpin/core/types
```

Then import the packages you need, for example
`github.com/alextanhongpin/core/types/sets`. Check `go.mod` for the required Go
version.

## Testing

```sh
cd types
go vet ./...
go test -race -count=1 ./...
```
