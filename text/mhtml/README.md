# mhtml

Read metadata and embedded HTML from MHTML archives.

## Installation

```sh
go get github.com/alextanhongpin/core/text/mhtml
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/text/mhtml"
	"os"
)

func main() {
	f, err := os.Open("snapshot.mhtml")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	html, metadata, err := mhtml.Read(f, true)
	if err != nil {
		panic(err)
	}
	fmt.Println(metadata.Title, len(html))
}
```

## Behavior and limits

Pass false to read metadata only. Metadata includes Snapshot-Content-Location, Subject, and Date; an invalid Date returns an error. Body extraction requires multipart/related with a boundary and an HTML part. Resources and additional HTML parts are embedded as base64 data URIs using literal byte replacement. This is not HTML sanitization or a full browser resource resolver. Archives containing resources but no HTML can panic in the current implementation.
