# stringcase

Stringcase provides utilities for converting strings between different naming conventions, including:

- Kebab-case (`ToKebab`)
- Snake_case (`ToSnake`)
- CamelCase (`ToCamel`)
- PascalCase (`ToPascal`)
- Title Case (`ToTitle`)
- Space-separated words (`ToWords`, `FromKebab`, `FromSnake`)

## Features
- Handles Go initialisms (e.g., `ID`, `API`, `HTTP`) according to Go naming conventions.
- Collapses multiple separators and spaces.
- Converts from and to all major string case styles.

## Usage

```go
import "github.com/alextanhongpin/core/types/stringcase"

fmt.Println(stringcase.ToKebab("HelloWorld"))      // "hello-world"
fmt.Println(stringcase.ToSnake("HelloWorld"))      // "hello_world"
fmt.Println(stringcase.ToCamel("user_id"))         // "userID"
fmt.Println(stringcase.ToPascal("user_id"))        // "UserID"
fmt.Println(stringcase.ToTitle("api_response_id")) // "API Response ID"
```

## Supported Initialisms

The following initialisms are handled according to Go conventions:

API, ASCII, CPU, CSS, DNS, EOF, GUID, HTML, HTTP, HTTPS, ID, IP, JSON, LHS, QPS, RAM, RHS, RPC, SLA, SMTP, SQL, SSH, TCP, TLS, TTL, UDP, UI, UID, UUID, URI, URL, UTF8, VM, XML, XSRF, XSS

## Testing

Run the tests:

```
go test -v
```

## License

MIT
