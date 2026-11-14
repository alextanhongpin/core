// Package circuitbreaker is a circuit breaker whose state is shared through
// Redis, so every instance of a service stops calling a failing dependency
// at the same time and recovers together.
//
// A breaker is identified by a key, usually one per downstream dependency
// (for example "cb:payments-api"). It moves between three states:
//
//	         failures >= FailureThreshold
//	Closed ──────────────────────────────▶ Opened
//	  ▲                                     │  ▲
//	  │ successes >= SuccessThreshold       │  │ any failure
//	  │                    OpenTimeout over ▼  │
//	  └───────────────────────────────── HalfOpen
//
// In detail:
//
//   - Closed: calls run; failures are counted.
//   - Opened: calls are rejected with ErrOpened without running.
//   - HalfOpen: calls run as probes; one failure reopens the breaker,
//     SuccessThreshold successes close it.
//
// Operators can also pin a breaker to Disabled (always run, never count) or
// ForcedOpen (always reject) with SetStatus.
//
// Minimal usage:
//
//	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	if err := circuitbreaker.Setup(ctx, client); err != nil {
//		return err
//	}
//	cb, err := circuitbreaker.New(client, circuitbreaker.Config{FailureThreshold: 5})
//	if err != nil {
//		return err
//	}
//	err = cb.Do(ctx, "cb:payments-api", func() error {
//		return callPaymentsAPI(ctx)
//	})
//	if errors.Is(err, circuitbreaker.ErrOpened) {
//		// Not called: serve a fallback or fail fast.
//	}
package circuitbreaker
