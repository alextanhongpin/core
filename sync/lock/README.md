# lock

Coordinate work by string key within one process.

## Installation

```sh
go get github.com/alextanhongpin/core/sync/lock
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"github.com/alextanhongpin/core/sync/lock"
)

func main() {
	locks := lock.New()
	held := locks.Lock("customer:42")
	defer held.Unlock()
	// Update this customer while holding the lock.
}
```

## Behavior and limits

KeyLock blocks callers sharing a key; different keys proceed independently. Keep the returned unlocker alive until Unlock. Both KeyLock and TryLock support their zero values. KeyLock entries are reclaimed through garbage collection, so Size is not a count of currently held locks. TryLock.TryLock returns immediately; RunInLock returns ErrLocked when busy and releases the key after the callback. Unlock only a key you acquired. These locks do not coordinate separate processes.
