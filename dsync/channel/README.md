# Redis stream channels

Publish bytes to a Redis stream and read them using independent cursors.

## Installation and documentation

```sh
go get github.com/alextanhongpin/core/dsync/channel
go doc github.com/alextanhongpin/core/dsync/channel
```

[Go API reference](https://pkg.go.dev/github.com/alextanhongpin/core/dsync/channel) · [Module requirements](go.mod)

## Typical use

Use for replayable notifications where each subscriber should see the same entries. Persist the last message ID to resume reading.

Examples assume a caller-owned Redis client and a context. Create the client once
and close it after all operations finish:

```go
client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
defer client.Close()
ctx := context.Background()
```

Import `context`, `github.com/redis/go-redis/v9`, and this package's import path.

## Quick start

The snippet belongs inside a function returning `error`; application functions
such as `handleRequest` represent your own work. Import the package above and
the standard packages used in the snippet (`context`, `errors`, `fmt`, or `time`).

```go
ch, err := channel.New(client)
if err != nil {
	return err
}
if err := ch.Send(ctx, "notifications", []byte("report ready")); err != nil {
	return err
}
message, err := ch.RecvAfter(ctx, "notifications", "0-0", time.Second)
if errors.Is(err, redis.Nil) {
	return nil
} // No entry within the wait.
if err != nil {
	return err
}
fmt.Println(string(message.Value))
// Save message.ID and pass it as the next RecvAfter cursor.
```

## Behavior and configuration

New(client) returns a Channel and error; MustNew is the panicking startup helper. The Redis client is borrowed and must be closed by its owner. Instances support concurrent calls.

Send appends one entry per call, including repeated identical payloads. Recv waits only for new entries arriving after the call snapshots the stream cursor. To replay existing entries and avoid gaps between reads, use RecvAfter with a caller-owned cursor: start at "0-0", then pass each returned Message.ID to the next call. Reads do not consume entries and independent callers can receive the same values.

Zero block waits indefinitely, negative block polls, and positive block limits the overall wait. Reads poll in bounded server waits to observe context cancellation even with the default borrowed client settings. Cancellation is cooperative and remains subject to Redis connection timeouts. Close deletes the stream; it does not wake blocked reads or permanently prevent new sends. Stream trimming and retention are caller-owned.

## Expected errors

`redis.Nil` means no entry was available before the read wait ended. Invalid keys/cursors and malformed entries return errors. Redis failures and context cancellation propagate.

## Pitfalls

Recv starts with future entries only. Use RecvAfter for replay. Readers do not consume entries; manage trimming and retention separately.
