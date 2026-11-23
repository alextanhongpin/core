# jsonl

Append typed values to a [JSON Lines](https://jsonlines.org) file and read
them back as an iterator.

**Use it for:** small append-only logs, fixtures, and local caches where one
JSON value per line is enough.

**Don't use it for:** concurrent writers, indexed lookups, or crash-safe
storage; use a database.

## Installation

```sh
go get github.com/alextanhongpin/core/types/jsonl
```

The package is part of the `github.com/alextanhongpin/core/types` module;
check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/types/jsonl"
)

type event struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	f, err := jsonl.Open[event]("data/events.jsonl")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_ = f.Write(event{1, "signup"})
	_ = f.Write(event{2, "login"})

	seq, stop := f.ReadLines()
	for e := range seq {
		fmt.Println(e.ID, e.Name)
	}
	if err := stop(); err != nil {
		panic(err)
	}
}
```

On a new file it prints `1 signup` and `2 login`.

## API

```go
type File[T any] struct{ *os.File }

func Open[T any](path string) (*File[T], error)              // read/write, append, create
func OpenFile[T any](path string, flag int) (*File[T], error) // os.OpenFile flags

func (f *File[T]) Write(v T) error
func (f *File[T]) ReadLines() (iter.Seq[T], func() error)
func (f *File[T]) Remove() error

func Copy[T any](from, to *File[T]) error
func CopyFunc[T any](from, to *File[T], keep func(T) bool) error
```

## Behavior and guarantees

- `Open` and `OpenFile` create missing parent directories. New files get
  mode `0666` before the umask. Pass `os.O_TRUNC` to `OpenFile` to replace a
  file.
- `Write` encodes `v` followed by a newline at the current file offset.
  With `Open` (`O_APPEND`) that is always the end of the file; with other
  flags, a previous `ReadLines` leaves the offset where reading stopped.
- `ReadLines` seeks to the start on each iteration and decodes values until
  EOF, an error, or the loop breaks. Call the returned function after the
  loop for the seek or decode error; it is reset when a new iteration starts.
- `Copy` and `CopyFunc` read `from` and write to `to`; they must be different
  files.
- `Remove` deletes the path but does not close the file. You close files.
- A `File` is not safe for concurrent use.

## Testing

```sh
go vet ./jsonl
go test -race -count=1 ./jsonl
```
