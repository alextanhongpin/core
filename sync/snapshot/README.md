# snapshot

A Go package implementing a Redis-style snapshot (`RDB`) mechanism for periodic data persistence and checkpointing based on change thresholds and elapsed time intervals.

Similar to Redis's `save <seconds> <changes>` configuration, `snapshot` triggers persistence operations when a given number of changes have occurred after a specified duration (e.g. *save after 1s if 1,000 changes occurred, or after 10s if 100 changes occurred*).

## Features

- **Redis-Style Save Policies**: Trigger snapshots when both time elapsed (`After`) and change count (`Changes`) conditions are met.
- **Multiple Priority Tiers**: Define multiple policies evaluated in ascending order of `After`.
- **Built-in Broadcast**: Embedded pub/sub (`broadcast.Broadcast[Policy]`) allows multiple listeners via channels (`Chan()`) or background workers (`Go()`).
- **Thread-Safe**: Safely record modifications using `Inc()` and `Add(n)`.
- **Configurable Buffering**: Customize channel buffer capacity to decouple writers from the background evaluation loop.
- **Graceful Shutdown**: `stop()` closes the worker, waits for pending work and cleans up resources.

## Installation

```bash
go get github.com/alextanhongpin/core/sync/snapshot
```

## Quick Start

```go
package main

import (
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/snapshot"
)

func main() {
	// Initialize snapshot manager with default configuration
	snap, stop := snapshot.New(snapshot.DefaultConfig())
	defer stop()

	// Register a listener using the embedded broadcast Go method
	snap.Go(func(p snapshot.Policy) {
		fmt.Printf("Snapshot triggered by policy: %d changes after %v\n", p.Changes, p.After)
		// Perform persistence (e.g., save to disk, write checkpoint)
	})

	// Record changes
	for i := 0; i < 1_500; i++ {
		snap.Inc()
	}

	time.Sleep(2 * time.Second)
}
```

Listen using a channel:

```go
snap, stop := snapshot.New(snapshot.DefaultConfig())
defer stop()

ch := snap.Chan()
go func() {
	for p := range ch {
		fmt.Printf("Snapshot triggered: %+v\n", p)
	}
}()
```

## How It Works

`snapshot` evaluates policies in ascending order of their time duration (`After`).

Each policy specifies:
- `After`: Minimum time that must have elapsed since the last snapshot.
- `Changes`: Minimum accumulated changes required to trigger a snapshot.

When `Inc()` or `Add(n)` is called:
1. Changes are accumulated in the background loop.
2. The loop checks policies in order. The first policy with `elapsed >= After` and `count >= Changes` triggers.
3. The counter resets to `0`, the timer resets, and the matched `Policy` is broadcast to all listeners.

The worker uses the smallest `After` interval among policies as its tick interval.

### Default Policies

`snapshot.DefaultPolicies()` provides Redis-like defaults:

| Changes | After | Description |
|---|---|---|
| 1,000 | 1s | High frequency: 1,000 changes in 1 second |
| 100 | 10s | Medium frequency: 100 changes in 10 seconds |
| 10 | 1m | Low frequency: 10 changes in 1 minute |
| 1 | 1h | Inactivity fallback: 1 change in 1 hour |

## API Reference

### `type Policy`

```go
type Policy struct {
	After   time.Duration // Time that must elapse before checking change count
	Changes int           // Number of accumulated changes required
}
```

### `type Config`

```go
type Config struct {
	BufferSize int       // Channel buffer size for change inputs (default: 0)
	Policies   []Policy  // Snapshot trigger policies
}
```

- `snapshot.DefaultConfig() *Config`: Returns configuration with buffer size `0` and default policies.
- `snapshot.DefaultPolicies() []Policy`: Returns the default set of 4 tiered policies.

### `func New(cfg *Config) (*Background, func())`

Creates and starts a snapshot background worker. Returns the `*Background` instance and a `stop` cleanup function. `New` validates the config and sorts policies by `After` ascending.

### `Background` Methods

- `Inc()`: Increments the change counter by 1. Calls `Add(1)`.
- `Add(n int)`: Increments the change counter by `n`.
- `Go(fn func(Policy))`: Spawns a background listener for snapshot trigger events. Equivalent to subscribing and running `fn` for each event.
- `Chan() <-chan Policy`: Subscribes and returns a receive-only channel for snapshot trigger events. Close the background with `stop()` to close the channel.

## Real-World Examples

### In-Memory Store with Snapshots

```go
type Store struct {
	mu   sync.RWMutex
	data map[string]string
	snap *snapshot.Background
	stop func()
}

func NewStore() *Store {
	s := &Store{data: make(map[string]string)}
	cfg := snapshot.DefaultConfig()
	// Custom policies for this store
	cfg.Policies = []snapshot.Policy{
		{Changes: 500, After: time.Second},
		{Changes: 50, After: 10 * time.Second},
	}
	s.snap, s.stop = snapshot.New(cfg)
	s.snap.Go(func(p snapshot.Policy) {
		s.save(p)
	})
	return s
}

func (s *Store) Set(k, v string) {
	s.mu.Lock()
	s.data[k] = v
	s.mu.Unlock()
	s.snap.Inc()
}

func (s *Store) save(p snapshot.Policy) {
	s.mu.RLock()
	// copy data for persistence
	_ = make(map[string]string, len(s.data))
	for k := range s.data { _ = k }
	s.mu.RUnlock()
	fmt.Printf("saved snapshot by %+v\n", p)
}

func (s *Store) Close() { s.stop() }
```

### Custom Snapshot Policies

```go
// High-frequency snapshots for critical data
critical := []snapshot.Policy{
	{Changes: 100, After: 5 * time.Second},
	{Changes: 10, After: 30 * time.Second},
	{Changes: 1, After: 2 * time.Minute},
}

// Low-frequency snapshots for background data
background := []snapshot.Policy{
	{Changes: 10_000, After: 1 * time.Minute},
	{Changes: 1_000, After: 10 * time.Minute},
	{Changes: 100, After: 1 * time.Hour},
}
```

## Testing

```go
func TestSnapshot(t *testing.T) {
	policies := []snapshot.Policy{
		{Changes: 10_000, After: 10 * time.Millisecond},
		{Changes: 1_000, After: 20 * time.Millisecond},
		{Changes: 100, After: 30 * time.Millisecond},
	}

	synctest.Test(t, func(t *testing.T) {
		s, stop := snapshot.New(&snapshot.Config{Policies: policies})
		defer stop()

		var logs []snapshot.Policy
		s.Go(func(p snapshot.Policy) { logs = append(logs, p) })
		ch := s.Chan()

		time.Sleep(11 * time.Millisecond)
		s.Add(10_000)
		<-ch // policies[0]

		time.Sleep(21 * time.Millisecond)
		s.Add(1_000)
		<-ch // policies[1]

		time.Sleep(31 * time.Millisecond)
		s.Add(100)
		<-ch // policies[2]
	})
}
```

## Best Practices

1. **Choose Appropriate Policies**: Balance data safety and performance.
2. **Handle Errors**: Always handle errors in snapshot handlers; the library only broadcasts policies.
3. **Atomic Operations**: Ensure snapshot data is consistent when copying state.
4. **Monitor Performance**: Track snapshot frequency and duration.
5. **Graceful Shutdown**: Always call `stop()` to ensure clean termination.
6. **Buffer Size**: Use a buffered channel for high write throughput to avoid blocking on `Inc()`/`Add()`.

## Performance Considerations

- Snapshot handlers should be non-blocking. Offload heavy I/O to a separate goroutine.
- The worker ticks at the smallest `After` interval. Keep policies reasonable to avoid busy loops.
- `Inc()`/`Add()` are non-blocking when the buffer has capacity; otherwise they may block.

## License

MIT License. See [LICENSE](../../LICENSE) for details.
