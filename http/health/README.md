# health

Liveness and readiness HTTP handlers with named dependency checks.

```go
h := health.New("1.4.2")
h.AddCheck("database", health.DatabaseCheck(db.Ping))
h.AddCheck("payments", health.HTTPCheck("https://payments.internal/healthz", 2*time.Second))
h.AddCheck("cache", func() health.Check {
	if cache.Lagging() {
		return health.Check{Status: health.StatusDegraded, Message: "replica lag"}
	}
	return health.Check{Status: health.StatusHealthy}
})

mux.HandleFunc("GET /livez", h.Live)
mux.HandleFunc("GET /readyz", h.Ready)
mux.HandleFunc("GET /healthz", h.Health) // same as Ready
```

- `Live` always returns 200 with version and uptime.
- `Ready` runs every check concurrently, records each check's latency, and
  returns 200 `healthy`, 200 `degraded` (some degraded, none unhealthy), or
  503 `unhealthy` (any unhealthy or unknown status).
- `AddCheck` is safe to call while requests are being served.
- Checks receive no context: bound their own duration (as `HTTPCheck` does
  with its timeout), because `Ready` waits for all of them.
- `DatabaseCheck(ping)` and `HTTPCheck(url, timeout)` (2xx is healthy) cover
  common dependencies.

Avoid exposing `Ready` publicly: check messages may include dependency
error text.
