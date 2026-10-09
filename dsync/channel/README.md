# Redis stream channels

New(client) returns a Channel and error; MustNew is the panicking startup helper. The Redis client is borrowed and must be closed by its owner. Instances support concurrent calls.

Send appends one entry per call, including repeated identical payloads. Recv waits only for new entries arriving after the call snapshots the stream cursor. To replay existing entries and avoid gaps between reads, use RecvAfter with a caller-owned cursor: start at "0-0", then pass each returned Message.ID to the next call. Reads do not consume entries and independent callers can receive the same values.

Zero block waits indefinitely, negative block polls, and positive block limits the overall wait. Reads poll in bounded server waits to observe context cancellation even with the default borrowed client settings. Cancellation is cooperative and remains subject to Redis connection timeouts. Close deletes the stream; it does not wake blocked reads or permanently prevent new sends. Stream trimming and retention are caller-owned.
