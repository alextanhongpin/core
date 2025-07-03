# HTTP Webhook Package

The HTTP Webhook package provides secure webhook handling with HMAC signature verification, supporting multiple secrets for key rotation and secure payload validation.

## Features

- **Signature Verification**: HMAC-SHA256 signature validation
- **Multi-Secret Support**: Key rotation with multiple verification secrets
- **Payload Verification**: Complete request body verification
- **Middleware Pattern**: Simple integration as HTTP middleware
- **Manual Verification**: Direct payload verification for custom scenarios
- **Key Rotation**: Support for seamless secret rotation

## Quick Start

```go
package main

import (
    "net/http"
    
    "github.com/alextanhongpin/core/http/webhook"
)

func main() {
    // Create your handler
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Only runs for requests with valid signatures
        w.Write([]byte("Webhook received!"))
    })
    
    // Wrap with webhook verification
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

#### `NewPayload(body []byte) *Payload`

Creates a new webhook payload for verification.

```go
// Read request body
body, err := io.ReadAll(r.Body)
if err != nil {
    http.Error(w, "Failed to read body", http.StatusBadRequest)
    return
}

// Create payload
payload := webhook.NewPayload(body)
```

#### `(p *Payload) Verify(signature string, secret []byte) bool`

Verifies a signature against a payload using the provided secret.

```go
// Get signature from header
signature := r.Header.Get("X-Hub-Signature-256")

// Verify signature
if !payload.Verify(signature, secret) {
    http.Error(w, "Invalid signature", http.StatusUnauthorized)
    return
}

// Process valid webhook...
```

#### `(p *Payload) Sign(secret []byte) string`

Generates a signature for the payload using the provided secret.

```go
// Generate signature for testing or sending webhooks
signature := payload.Sign(secret)
```

## Webhook Signature Format

The package expects signatures in the standard format:

```
sha256=<hex encoded signature>
```

This is compatible with GitHub, Stripe, and many other webhook providers.

## Key Rotation Example

Smoothly transition from old to new webhook secrets:

```go
// During rotation period, accept both old and new secrets
oldSecret := []byte("old-webhook-secret")
newSecret := []byte("new-webhook-secret")

// Priority matters - check new secret first
handler = webhook.Handler(yourHandler, newSecret, oldSecret)
```

## Manual Handler Implementation

You can implement custom webhook handling logic:

```go
func CustomWebhookHandler(secret []byte) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Only accept POST requests
        if r.Method != "POST" {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }
        
        // Get signature from header
        signature := r.Header.Get("X-Hub-Signature-256")
        if signature == "" {
            http.Error(w, "No signature provided", http.StatusBadRequest)
            return
        }
        
        // Read and verify body
        body, err := io.ReadAll(r.Body)
        if err != nil {
            http.Error(w, "Failed to read body", http.StatusBadRequest)
            return
        }
        
        // Create and verify payload
        payload := webhook.NewPayload(body)
        if !payload.Verify(signature, secret) {
            http.Error(w, "Invalid signature", http.StatusUnauthorized)
            return
        }
        
        // Process webhook payload...
    }
}
```

## Testing Webhooks

The package makes it easy to test webhook handlers:

```go
func TestWebhook(t *testing.T) {
    // Create a test server
    secret := []byte("test-secret")
    ts := httptest.NewServer(webhook.Handler(yourHandler, secret))
    defer ts.Close()
    
    // Create test payload
    payload := []byte(`{"event":"test","data":123}`)
    
    // Calculate signature
    p := webhook.NewPayload(payload)
    signature := p.Sign(secret)
    
    // Create and send request
    req, _ := http.NewRequest("POST", ts.URL, bytes.NewReader(payload))
    req.Header.Set("X-Hub-Signature-256", signature)
    
    resp, err := http.DefaultClient.Do(req)
    // Check response...
}
```

## Multiple Signature Headers Support

The webhook package automatically checks common signature header locations:

- `X-Hub-Signature-256` (GitHub style)
- `X-Signature-256` (Stripe-compatible)
- `X-Webhook-Signature` (Generic style)

## Best Practices

1. **Secret Management**: Store webhook secrets securely and rotate regularly
2. **HTTPS Only**: Always use HTTPS for webhook endpoints
3. **Idempotency**: Design webhook handlers to be idempotent (handle duplicate deliveries)
4. **Key Rotation**: Use multi-secret support during key rotation periods
5. **Validation**: Validate payload format and contents after signature verification

## Security Considerations

- **Secret Length**: Use secrets of at least 32 random bytes
- **Response Content**: Don't leak information in error responses
- **Timing Attacks**: The package uses constant-time comparison to prevent timing attacks
- **Replay Protection**: Consider implementing timestamp validation for critical webhooks
