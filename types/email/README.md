# email

Check email address syntax, normalize addresses for storage, and split them
into local part and domain.

**Use it for:** rejecting malformed input and storing addresses consistently.

**Don't use it for:** proving an address exists or receives mail (send a
verification email), or detecting disposable providers.

## Installation

```sh
go get github.com/alextanhongpin/core/types
```

`email` is a package in the `types` module; import it as
`github.com/alextanhongpin/core/types/email`. Check the module's `go.mod` for the
required Go version.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/types/email"
)

func main() {
	addr := email.Normalize("  John.Doe@Example.COM ")
	fmt.Println(addr)
	fmt.Println(email.IsValid(addr))
	fmt.Println(email.LocalPart(addr), email.Domain(addr))
	fmt.Println(email.IsBusinessEmail(addr), email.IsBusinessEmail("jane@gmail.com"))
}
```

It prints:

```
john.doe@example.com
true
john.doe example.com
true false
```

## API

```go
func IsValid(s string) bool         // RFC 5322 style pattern, max 254 bytes
func IsValidBasic(s string) bool    // simple ASCII pattern, max 254 bytes
func Normalize(s string) string     // strings.ToLower(strings.TrimSpace(s))
func LocalPart(s string) string
func Domain(s string) string
func IsBusinessEmail(s string) bool
```

## Behavior and guarantees

- `IsValid` accepts quoted local parts and internationalized characters, and
  requires a domain with at least one dot. `IsValidBasic` accepts only
  `[A-Za-z0-9._%+-]@[A-Za-z0-9.-].[A-Za-z]{2,}`.
- Neither validator checks DNS, MX records, or mailbox existence.
- `Normalize` lowercases the whole address. Most providers treat the local part
  case-insensitively, but RFC 5321 does not require it.
- `LocalPart` and `Domain` split at the last `@` and return `""` when there is
  no `@` or either side is empty. They do not validate.
- `IsBusinessEmail` is true when the domain (case-insensitive) is not in a short
  fixed list of consumer providers: gmail.com, yahoo.com, hotmail.com,
  outlook.com, aol.com, icloud.com, mail.com, protonmail.com, yandex.com. It is
  false when there is no domain.
- All functions are safe for concurrent use.

## Testing

```sh
cd types
go vet ./email
go test -race -count=1 ./email
```
