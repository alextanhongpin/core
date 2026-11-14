# channel

A broadcast message log on Redis streams: every reader sees every message, and
each reader keeps track of how far it has read.

**Use it when** several processes need the same events (cache invalidation,
"report ready" notifications, audit/event feeds) and a reader that restarts
should be able to catch up on what it missed.

**Don't use it when** each message should be handled by exactly one worker
(a job queue). Reading here never removes or acknowledges messages. Use Redis
consumer groups (`XREADGROUP` / `XACK`) for that. If you need neither replay
nor history, plain Redis Pub/Sub is simpler.

| | `channel` | Pub/Sub | Consumer groups |
|---|---|---|---|
| Every reader gets every message | yes | yes | no, one per group |
| Missed messages can be replayed | yes (until trimmed) | no | yes |
| Who tracks progress | the reader (keeps the last ID) | nobody | Redis (`XACK`) |

## Install

```sh
go get github.com/alextanhongpin/core/dsync/channel
```

API reference: [pkg.go.dev/github.com/alextanhongpin/core/dsync/channel](https://pkg.go.dev/github.com/alextanhongpin/core/dsync/channel)

## Quick start

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/alextanhongpin/core/dsync/channel"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close()

	ch, err := channel.New(client)
	if err != nil {
		log.Fatal(err)
	}

	// Publish. Each Send appends one message to the stream "orders".
	for _, event := range []string{"created", "paid", "shipped"} {
		if err := ch.Send(ctx, "orders", []byte(event)); err != nil {
			log.Fatal(err)
		}
	}

	// Read everything from the beginning. Save cursor (e.g. in your
	// database) to continue from the same place after a restart.
	cursor := "0-0"
	for {
		msg, err := ch.RecvAfter(ctx, "orders", cursor, -1) // -1 = don't wait
		if errors.Is(err, redis.Nil) {
			break // no more messages right now
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(msg.Value))
		cursor = msg.ID
	}

	// Keep only the newest 1000 messages so the stream doesn't grow forever.
	if _, err := ch.Trim(ctx, "orders", 1000); err != nil {
		log.Fatal(err)
	}
}
```

For a long-running subscriber, pass `0` as the wait (block until a message
arrives) and stop it by cancelling `ctx`.

## How it works

Each key is one Redis stream. Each message is one stream entry with a single
field, `data`, holding your bytes. Redis gives every entry an increasing ID
such as `1700000000000-0`.

```
            Send (XADD)
                 |
                 v
 stream "orders":  [1-0 created] [2-0 paid] [3-0 shipped]
                        ^                         ^
 reader A cursor="1-0" -+   RecvAfter -> "paid"   |
 reader B cursor="3-0" ---------------------------+  RecvAfter waits for the next Send
```

- `RecvAfter(ctx, key, cursor, wait)` returns the first entry with an ID
  greater than `cursor` (`XREAD COUNT 1`). Readers don't change the stream, so
  A and B above are independent.
- The cursor is just the last `Message.ID` you processed. Start with `"0-0"`
  (everything still stored) or `"$"` (only messages sent from now on).
- `Recv(ctx, key, wait)` is `RecvAfter` with `"$"` that returns only the
  payload. Because you don't get the ID, calling it in a loop can miss
  messages sent between calls; use `RecvAfter` when that matters.
- Waiting is done with short `XREAD BLOCK` calls of at most 100 ms, so a
  cancelled `ctx` stops a read within about 100 ms.
- `Trim` (`XTRIM MAXLEN`) drops the oldest entries; `Close` (`DEL`) deletes
  the whole stream.

### The `wait` argument

| Value | Behaviour when no message is available |
|---|---|
| negative (e.g. `-1`) | return `redis.Nil` immediately |
| `0` | wait until a message arrives or `ctx` is done |
| positive (e.g. `5*time.Second`) | wait up to that long, then return `redis.Nil` |

## API at a glance

| Function | What it does |
|---|---|
| `New(client)` / `MustNew(client)` | create a Channel; the client stays yours to close |
| `Send(ctx, key, value)` | append a message |
| `RecvAfter(ctx, key, cursor, wait)` | next message after `cursor`, with its ID |
| `Recv(ctx, key, wait)` | next message sent after this call, payload only |
| `Trim(ctx, key, maxLen)` | keep only the newest `maxLen` messages |
| `Close(ctx, key)` | delete the stream and all its messages |

There is no configuration. A `Channel` is safe for concurrent use.

## Errors

| Error | Meaning | What to do |
|---|---|---|
| `redis.Nil` | the wait ended with no new message | normal; try again later, keep the same cursor |
| `context.Canceled` / `DeadlineExceeded` (or the context's cause) | `ctx` is done | stop reading |
| `channel.ErrInvalidArgument` | empty key or cursor, or negative `maxLen` | fix the call |
| `channel.ErrMalformedEntry` | the entry wasn't written by `Send` (no `data` field) | log it and continue with `cursor = msg.ID`, which is still set |
| any other error | Redis/network failure or an invalid cursor string | retry or report |

Check errors with `errors.Is`.

## Requirements

- Redis 5.0+ (streams). Works with any `*redis.Client` from
  `github.com/redis/go-redis/v9`.

## Caveats

- **Nothing is deleted automatically.** Call `Trim` periodically (or after
  each `Send`) or the stream grows without limit.
- **Trimming can drop unread messages.** A reader whose cursor is older than
  the oldest kept entry silently continues from the oldest kept entry.
- **`Close` deletes data.** It does not close the client, does not wake
  blocked readers, and a later `Send` creates the stream again.
- **Readers store their own cursor.** If you lose it, restart from `"0-0"`
  (replay everything stored) or `"$"` (skip to new messages).
- Delivery is at-least-once only if you save the cursor *after* handling the
  message.

## Running the tests

Tests start a Redis container through Docker:

```sh
cd dsync/channel
go test -race ./...
```
