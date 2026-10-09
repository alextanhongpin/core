# Promise

New(ctx, fn) starts one asynchronous operation. Deferred[T](ctx) creates a manually settled promise and returns only the promise. Resolve, Reject, and WithResolvers support immediate or external settlement. Await can be called repeatedly and concurrently; all callers observe the same settlement.

```go
p := promise.Deferred[int](ctx)
go func() { p.Resolve(42) }()
value, err := p.Await()
```

The first settlement wins. Abort cancels the child context supplied to New's handler, but does not wait for handler completion. Cancellation is cooperative. A handler that ignores its context may continue after Await returns. Nil rejection errors produce a fulfilled zero result; use a non-nil error to reject.

Race returns the first observed settlement. Any returns the first observed success, or joined errors if all inputs fail. All returns results in input order or fails immediately on the first observed error. AllSettled waits for every input and returns ordered status/data/error results. Empty input panics. Race, Any, and All stop and join their own waiters before returning; they do not abort input promises. When several inputs are already settled, scheduling determines which settlement is observed first.

Map is a weak cache of promises. LoadOrCreate may allocate competing deferred promises, but only one live entry is installed per key. The caller whose loaded result is false owns settlement. Weak retention permits re-creation after references disappear and GC runs. There is no LoadAndDelete method.

Channel is the reusable single-settlement primitive. Send publishes at most one value; Recv can return that value repeatedly. Close cancels unsettled receivers with its cause. Done reports whether a value was published, rather than whether cancellation occurred or Recv was called. Returned mutable values remain shared and need caller synchronization.
