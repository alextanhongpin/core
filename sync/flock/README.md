# flock

Generate a file once, under an advisory OS file lock, and share the result with
every process that asks for it.

**Use it for:** cache files, downloaded or compiled artifacts, and other
"compute it once on this machine" work shared between processes.

**Don't use it for:** coordinating goroutines inside one process
(use [`sync/lock`](../lock)), locking across machines (use
[`dsync/lock`](../../dsync/lock)), or data that must survive power loss
(publication is atomic but not fsynced).

## Installation

```sh
go get github.com/alextanhongpin/core/sync/flock
```

## Quick start

```go
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/alextanhongpin/core/sync/flock"
)

func main() {
	dir, err := os.MkdirTemp("", "flock")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	var fs flock.FS
	name := filepath.Join(dir, "report.txt")
	for range 2 {
		b, err := fs.ReadOrWrite(name, func() ([]byte, error) {
			fmt.Println("generating")
			return []byte("expensive result"), nil
		})
		if errors.Is(err, flock.ErrLocked) {
			continue // another process is generating it; retry later
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(b))
	}
}
```

It prints `generating`, then `expensive result` twice.

## API

```go
var ErrLocked = errors.New("flock: locked")

type FS struct{} // zero value is ready to use
func (*FS) ReadOrWrite(name string, fn func() ([]byte, error)) ([]byte, error)
```

## Behavior and guarantees

- **Build constraint:** unix only (darwin, linux, freebsd, openbsd, netbsd,
  dragonfly), because it uses `flock(2)`. The package does not build elsewhere.
- **Fast path:** `ReadOrWrite` first reads `name` without taking the lock. If it
  exists it is returned. This is safe because files appear only through an
  atomic rename of a complete temporary file, so a reader never sees a partial
  file, and a held lock or missing `.lock` file does not matter for reads.
- **Slow path:** when `name` is absent, a nonblocking exclusive lock is taken on
  `name + ".lock"`. If someone else holds it the call returns `ErrLocked`
  immediately; there is no waiting, so callers choose whether and when to retry.
  The file is checked again under the lock before `fn` runs.
- `fn` runs under the lock and must not call `ReadOrWrite` for the same path.
  If `fn` fails, nothing is published and the call can be retried; temporary
  files are removed.
- All cooperating writers must use the same `name` and keep the `.lock` file in
  place (it is never deleted). The directory of `name` must be writable and on
  one filesystem for the rename.
- Existing files are returned unchanged; invalidation is up to the caller
  (delete the file).
- Locks are advisory and per open file description; they do not stop code that
  ignores them. Advisory locks over some network filesystems are unreliable.

## Testing

```sh
go vet ./...
go test -race -count=1 ./...
```
