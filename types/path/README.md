# path

Wrap filesystem paths with joining and file operations.

## Installation

```sh
go get github.com/alextanhongpin/core/types/path
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/path"
)

func main() {
	p := path.Path("data").Join("settings.json")
	fmt.Println(p.String(), p.Base(), p.Ext())
}
```

## Behavior and limits

Join uses filepath rules. OpenFile and WriteFile create parent directories. Close handles returned by OpenFile. ReadFile returns nil, nil for a missing file. Exist returns false for any stat error, including permission errors. WriteFile overwrites existing content.
