# lock

Keyed locks for one process: `KeyLock` blocks callers that share a string key,
and `TryLock` is its nonblocking counterpart that refuses instead of waiting.

**Use it for:** "only one goroutine updates customer 42 at a time", skipping a
job that is already running in this process.

**Don't use it for:** coordinating several processes or replicas (use
[`dsync/lock`](../../dsync/lock), which holds a Redis lease), or for coalescing
callers that all need the same result (use
[`dsync/singleflight`](../../dsync/singleflight)).

## Installation

```sh
go get github.com/alextanhongpin/core/sync/lock
```

Check the module's `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"errors"
	"fmt"

	"github.com/alextanhongpin/core/sync/lock"
)

func main() {
	// Blocking: callers with the same key take turns.
	locks := lock.New()
	held := locks.Lock("customer:42")
	// Update this customer while holding the lock.
	held.Unlock()

	// Nonblocking: refuse instead of waiting.
	var jobs lock.TryLock
	unlock, ok := jobs.TryAcquire("report")
	if !ok {
		return
	}
	defer unlock()

	_, ok = jobs.TryAcquire("report") // busy
	fmt.Println(ok)

	err := jobs.RunInLock("report", func() error { return nil })
	fmt.Println(errors.Is(err, lock.ErrLocked))
}
```

It prints `false` then `true`.

## API

```go
func New() *KeyLock // zero value of KeyLock is also ready to use
func (*KeyLock) Lock(key string) unlocker // blocks; call Unlock() on the result
func (*KeyLock) Has(key string) bool      // entry exists; NOT "is held"
func (*KeyLock) Size() int                // map entries; NOT the number of held locks

func NewTryLock() *TryLock // zero value of TryLock is also ready to use
func (*TryLock) TryLock(key string) bool                      // acquire if free
func (*TryLock) TryAcquire(key string) (unlock func(), ok bool) // acquire if free; unlock is bound to this acquisition
func (*TryLock) Unlock(key string)                            // release whoever holds key
func (*TryLock) RunInLock(key string, fn func() error) error  // ErrLocked if busy

var ErrLocked = errors.New("already locked")
var ErrNotFound, ErrExpired error // declared, currently unused by this package
```

## Behavior and guarantees

- Different keys never block each other. Both types are safe for concurrent use.
- `KeyLock` is not reentrant: locking a key you already hold deadlocks.
- `KeyLock` keeps entries weakly and the garbage collector reclaims idle ones.
  `Has` and `Size` therefore count idle entries until that happens; `Has(key)`
  true does not mean the key is held. Keep the returned unlocker referenced
  until you call `Unlock`.
- `TryLock.TryLock` and `TryAcquire` return immediately. `Unlock(key)` releases
  the key no matter who holds it, so unlock only what you acquired.
- The func from `TryAcquire` releases only its own acquisition. It is
  idempotent, and calling it after the key was released and acquired by someone
  else leaves their lock held.
- `RunInLock` runs `fn` under the key and releases only its own acquisition
  afterwards, even if `fn` called `Unlock(key)`. A panic in `fn` still releases.
  Busy keys return `ErrLocked` without calling `fn`.
- Nothing here coordinates separate processes and nothing has a timeout or
  fairness guarantee.

## Testing

```sh
go vet ./...
go test -race -count=1 ./...
```
