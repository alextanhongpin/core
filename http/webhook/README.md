# webhook

Sign and verify webhook requests with HMAC-SHA256, with key rotation and
replay-window checks.

```go
// Receiver
h := webhook.Handler(eventsHandler, newSecret, oldSecret)

// Sender
payload := webhook.NewPayload(body)
req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
payload.SignRequest(req, newSecret) // one X-Webhook-Signature per secret
```

## Scheme

| Header | Value |
| --- | --- |
| `X-Webhook-Id` | Payload ID (random UUID from `NewPayload`). |
| `X-Webhook-Timestamp` | Send time, Unix nanoseconds. |
| `X-Webhook-Signature` | base64url HMAC-SHA256 of `<id>.<timestamp>.<body>`; may repeat. |

This is a package-specific scheme. It does not verify GitHub
(`X-Hub-Signature-256`) or Stripe (`Stripe-Signature`) signatures, which sign
different content.

## Verification

`Verify(r, config)` (used by `Handler` / `HandlerWithConfig`):

1. requires the ID and timestamp headers (`ErrMissingHeader`,
   `ErrInvalidTimestamp`);
2. rejects payloads older than `MaxAge` (default 5 minutes, negative
   disables) or more than a minute in the future (`ErrExpired`);
3. reads at most `MaxBodySize` bytes (default 10 MiB, `ErrBodyTooLarge`) and
   restores the body for the next handler;
4. accepts the request if any signature matches any secret, compared in
   constant time (`ErrInvalidSignature` otherwise).

Failures go to `Config.ErrorHandler`; the default replies 413 for oversized
bodies and 401 otherwise. Payload IDs are not remembered, so a captured
request can be replayed within `MaxAge`: deduplicate on `X-Webhook-Id` if
that matters.
