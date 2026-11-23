# auth

HTTP Basic and JWT bearer authentication middleware.

```go
secret := []byte("a-random-32-byte-secret-key.....")

// Issue a token (HS256 by default; always sets exp, iat, nbf).
j := auth.NewJWT(secret)
token, err := j.Sign(auth.Claims{Subject: "user@example.com"}, time.Hour)

// Optional bearer auth: requests without a token pass through; invalid
// tokens get 401. Verified claims are stored in the request context.
h := auth.BearerHandler(mux, secret)

// Require a token on a sub-tree.
protected := auth.BearerHandler(auth.RequireBearerHandler(adminMux), secret)

func profile(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsContext.Value(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	fmt.Fprintf(w, "hello, %s", claims.Subject)
}

// Basic auth: the map holds SHA-256 hashes, not passwords.
h = auth.BasicHandler(mux, map[string]string{
	"admin": auth.HashPasswordSHA256("generated-secret"),
})
```

## Contracts

- `BearerAuth` matches the `Bearer` scheme case-insensitively and reports
  `false` for an empty token.
- `Verify` accepts only the configured signing method (no `none`, no
  algorithm switching), requires an unexpired `exp` (with `ExpiryLeeway`),
  rejects future `nbf`/`iat`, and checks `iss` when `Issuer` is set. Errors
  wrap `ErrTokenExpired` or `ErrTokenInvalid`.
- `SignWithCustomClaims` sets no claims; set `ExpiresAt` yourself or `Verify`
  will reject the token.
- `DefaultBearerErrorHandler` replies 401 with `WWW-Authenticate: Bearer` for
  a missing token (`ErrMissingToken`) and `Bearer error="invalid_token"`
  otherwise. A nil `OnError` uses it.
- Basic auth compares hashes in constant time and does the same work for
  unknown usernames. SHA-256 is unsalted and fast: use it for generated,
  high-entropy secrets, not user-chosen passwords.

Context keys: `ClaimsContext` (`*Claims`, set by the bearer middleware),
`UsernameContext` (set by Basic auth), and `LoggerContext` (an optional
`*slog.Logger` the default error handler logs to).
