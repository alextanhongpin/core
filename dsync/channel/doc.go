// Package channel is a small broadcast log on top of Redis streams.
//
// Send appends a payload to a stream; RecvAfter returns the first entry after
// a cursor (a stream entry ID). Reading never removes entries, so any number
// of readers can follow the same stream independently, and a reader that
// stores its last Message.ID can resume after a restart without missing
// entries. Entries stay in Redis until Trim or Close removes them.
//
// It is not a work queue: if each message must be processed by exactly one
// of several workers, use Redis consumer groups (XREADGROUP/XACK) instead.
//
// Minimal usage:
//
//	ch := channel.MustNew(client) // client is a *redis.Client you own
//	_ = ch.Send(ctx, "events", []byte("hello"))
//
//	cursor := "0-0" // start from the oldest entry
//	for {
//		msg, err := ch.RecvAfter(ctx, "events", cursor, 0) // 0 = wait forever
//		if err != nil {
//			return err
//		}
//		handle(msg.Value)
//		cursor = msg.ID // remember where to continue
//	}
package channel
