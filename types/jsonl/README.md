# jsonl

Read and append typed JSON values in files.

## Installation

```sh
go get github.com/alextanhongpin/core/types/jsonl
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/jsonl"
)

func main() {
	f, err := jsonl.Open[int]("records.jsonl")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := f.Write(42); err != nil {
		panic(err)
	}
	seq, check := f.ReadLines()
	for value := range seq {
		fmt.Println(value)
	}
	if err := check(); err != nil {
		panic(err)
	}
}
```

## Behavior and limits

Open creates parent directories and opens for read/write and append. OpenFile accepts os flags, including os.O_TRUNC when replacement is intended. ReadLines seeks to the start and returns a separate function for retrieving decoding errors; call it after iteration, including early termination. It does not close the file. Copy and CopyFunc write all or selected values to another file. Close each file explicitly. Remove deletes the path without closing the handle.
