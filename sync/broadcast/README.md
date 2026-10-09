# Broadcast

`New[T]()` starts a serial dispatcher and returns a broadcast and an idempotent, concurrent-safe stop function. Each subscriber receives values in dispatcher order. Values are shared without copying; callers own synchronization for mutable payloads.

```go
b, stop := broadcast.New[int]()
defer stop()
b.Go(func(value int) { fmt.Println(value) })
b.Send(42)
```

`Send` waits for dispatcher acceptance, which happens before delivery to subscribers. It does not confirm delivery or callback completion. Slow or unread subscribers block later delivery, sends, and registrations. Subscribers must continue receiving until shutdown; there is no unsubscribe operation.

`stop` interrupts pending delivery, closes subscriber channels, and waits for running worker callbacks. Accepted values may be discarded during shutdown. Callbacks must finish for stop to return and must not call stop, Send, Chan, or Go on the same instance, because those operations can wait for the callback itself.

After stop, Send and Go return without work, and Chan returns nil. Do not range over that nil channel. The zero value is not usable; construct instances with New.
