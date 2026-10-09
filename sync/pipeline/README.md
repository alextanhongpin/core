# Channel pipelines

This module requires Go 1.25 or later. Sources, transformations, flow control, and sinks compose through typed channels.

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
in := pipeline.SourceSlice(ctx, []int{1, 2, 3})
out := pipeline.Pipe(in, func(v int) (int, bool) { return v * 2, true })
values := pipeline.Collect(out)
```

Sources accept a context. SourceChan stops both receiving and forwarding on cancellation. SourceIter cannot interrupt an iterator blocked before yielding. SourceSlice reads the caller's slice without copying it; do not mutate its backing array while the source runs.

Intermediate stages are not independently cancelable. Consumers must drain every output until closed, including both Tee outputs and every FanOut output. Canceling a source cannot unblock a stage already sending downstream. When Reduce returns an error, cancel the source and Flush the remaining Reduce input to let upstream stages exit. For long-lived or externally owned inputs, arrange closure explicitly.

PipeN and Semaphore bound concurrent callbacks; results may arrive out of order. Callbacks and mutable payloads remain caller-owned and require synchronization. FanOut distributes values round-robin, so a slow output blocks later distribution. FanIn preserves each input's order, with unspecified interleaving.

Batch flushes when size is reached or a timeout elapses after the first item, and flushes remaining items when input closes. Size must be positive. Buffer capacity and worker counts must be positive. RateLimit requires a positive interval and count with at least a one-nanosecond emission interval.

Debounce emits at most once per duration and drops intervening items; it implements leading-edge suppression, not a trailing-edge debounce. Dedup retains every observed key for the pipeline lifetime. Merge combines a left and right value only when selected together; output timing and pairing depend on scheduling.

SafeClose catches a close panic; it does not make concurrent sends and closes safe. Channel closure remains the sender owner's responsibility.

Collect, Sink, Flush, and Count drain until closure. Reduce folds values and returns immediately on callback error. No stage provides an error channel or recovers callback panics.
