# HTTP Webhook Package

The HTTP Webhook package provides secure webhook handling with HMAC signature verification, supporting multiple secrets for key rotation and secure payload validation.

## Features

- **Signature Verification**: HMAC-SHA256 signature validation
- **Multi-Secret Support**: Key rotation with multiple verification secrets
- **Payload Verification**: Complete request body verification
- **Middleware Pattern**: Simple integration as HTTP middleware
- **Manual Verification**: Direct payload verification for custom scenarios
- **Key Rotation**: Support for seamless secret rotation
- **Cross-Package Integration**: Works with `chain`, `handler`, and logging middleware

## Quick Start

```go
package main

import (
    "net/http"
    "github.com/alextanhongpin/core/http/webhook"
)

func main() {
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Webhook received!"))
    })
    secret := []byte("your-webhook-secret")
    webhookHandler := webhook.Handler(handler, secret)
    http.ListenAndServe(":8080", webhookHandler)
}
```

## API Reference

### Middleware

#### `Handler(next http.Handler, secrets ...[]byte) http.Handler`
Creates middleware that verifies webhook signatures before passing to the handler.

```go
// Single secret
handler = webhook.Handler(handler, []byte("secret"))
// Multiple secrets for key rotation
handler = webhook.Handler(handler, 
    []byte("new-secret"),
    []byte("old-secret"),
)
```

### Manual Verification

#### `Verify(r *http.Request, secrets ...[]byte) error`
Manually verifies webhook signature and payload.

## Best Practices

- Rotate secrets regularly for security.
- Always verify payload signatures before processing.
- Integrate with logging and error handling middleware.

## Related Packages

- [`chain`](../chain/README.md): Middleware chaining
- [`handler`](../handler/README.md): Base handler utilities
- [`server`](../server/README.md): HTTP server utilities

## License

MIT
