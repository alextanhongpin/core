package channel

import (
	"context"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// dataKey is the stream entry field that holds the message payload.
const dataKey = "data"

// pollInterval caps each XREAD BLOCK so a blocked read notices context
// cancellation between reads, even if the Redis client does not close the
// connection on cancellation. Redis only fires blocking timeouts on its
// server tick (every 100 ms at the default hz 10), so one read lasts 100 to
// 200 ms and cancellation is noticed up to about 200 ms late.
const pollInterval = 100 * time.Millisecond

var (
	// ErrInvalidArgument is returned when a required argument such as the
	// stream key or the cursor is empty. Fix the call; retrying will not help.
	ErrInvalidArgument = errors.New("channel: invalid argument")

	// ErrMalformedEntry is returned when a stream entry was not written by
	// Send (it has no string "data" field). The returned Message still carries
	// the entry ID, so the reader can pass it to the next RecvAfter call to
	// skip the entry instead of reading it forever.
	ErrMalformedEntry = errors.New("channel: entry has no string data field")
)

// Channel appends messages to Redis streams and reads them back by cursor.
//
// A Channel holds no state besides the Redis client, so one value can be
// shared by many goroutines and used with any number of stream keys.
type Channel struct {
	client *redis.Client
}

// New returns a Channel that uses client. The client is borrowed: the
// caller keeps ownership and must close it after the Channel is no longer
// used. New returns an error if client is nil.
func New(client *redis.Client) (*Channel, error) {
	if client == nil {
		return nil, errors.New("channel: nil client")
	}
	return &Channel{client: client}, nil
}

// MustNew is like New but panics on error. It is intended for program
// start-up.
func MustNew(client *redis.Client) *Channel {
	c, err := New(client)
	if err != nil {
		panic(err)
	}
	return c
}

// Send appends value as a new entry to the stream stored at key (XADD),
// creating the stream if needed. Every call adds an entry, even if the
// payload repeats. The stream grows until Trim or Close is called.
func (c *Channel) Send(ctx context.Context, key string, value []byte) error {
	if key == "" {
		return fmt.Errorf("%w: key is required", ErrInvalidArgument)
	}
	return c.client.XAdd(ctx, &redis.XAddArgs{
		Stream: key,
		Values: []string{dataKey, string(value)},
	}).Err()
}

// Trim keeps only the newest maxLen entries of the stream at key (XTRIM
// MAXLEN) and returns how many entries were removed. Readers whose cursor
// points at a removed entry continue from the oldest remaining entry.
func (c *Channel) Trim(ctx context.Context, key string, maxLen int64) (int64, error) {
	if key == "" {
		return 0, fmt.Errorf("%w: key is required", ErrInvalidArgument)
	}
	if maxLen < 0 {
		return 0, fmt.Errorf("%w: maxLen must not be negative", ErrInvalidArgument)
	}
	return c.client.XTrimMaxLen(ctx, key, maxLen).Result()
}

// Close deletes the stream at key and all its entries (DEL). Despite its
// name it does not close the Channel or the Redis client, it does not wake
// readers that are currently blocked, and a later Send creates the stream
// again.
func (c *Channel) Close(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Message is one stream entry. Pass ID to RecvAfter to read the entry that
// follows it.
type Message struct {
	// ID is the Redis stream entry ID, for example "1700000000000-0".
	ID string
	// Value is the payload given to Send.
	Value []byte
}

// Recv waits for the next entry added after the call starts and returns
// its payload. Entries already in the stream are not returned.
//
// Because it does not return the entry ID, calling Recv in a loop can miss
// entries sent between calls. To read every entry, use RecvAfter and pass
// each returned Message.ID to the next call.
//
// See RecvAfter for the meaning of block and the errors returned.
func (c *Channel) Recv(ctx context.Context, key string, block time.Duration) ([]byte, error) {
	msg, err := c.RecvAfter(ctx, key, "$", block)
	return msg.Value, err
}

// RecvAfter returns the first entry whose ID is greater than id. Use "0-0"
// (or "0") to start from the oldest entry, "$" to wait only for entries
// added after the call starts, or a Message.ID returned earlier to resume.
//
// Reading does not remove entries: every reader sees every entry, and each
// reader is responsible for remembering its own last ID (for example in
// memory or in a database). Use Redis consumer groups instead if each entry
// should be handled by only one worker.
//
// block controls how long to wait when no entry is available yet:
//
//	block < 0   do not wait; return redis.Nil immediately
//	block == 0  wait until an entry arrives or ctx is done
//	block > 0   wait at most block, then return redis.Nil
//
// Errors: redis.Nil when the wait ended without an entry; the context's
// cause when ctx is done; ErrInvalidArgument for an empty key or id;
// ErrMalformedEntry (with Message.ID set) for an entry not written by Send;
// otherwise the Redis error.
func (c *Channel) RecvAfter(ctx context.Context, key, id string, block time.Duration) (Message, error) {
	if err := context.Cause(ctx); err != nil {
		return Message{}, err
	}
	if key == "" || id == "" {
		return Message{}, fmt.Errorf("%w: key and cursor are required", ErrInvalidArgument)
	}
	// Resolve $ once. Repeating it after each polling read could skip entries
	// arriving between reads.
	if id == "$" {
		entries, err := c.client.XRevRangeN(ctx, key, "+", "-", 1).Result()
		if err != nil {
			return Message{}, err
		}
		id = "0-0"
		if len(entries) > 0 {
			id = entries[0].ID
		}
	}
	var deadline time.Time
	if block > 0 {
		deadline = time.Now().Add(block)
	}
	for {
		if err := context.Cause(ctx); err != nil {
			return Message{}, err
		}
		readBlock := block
		if block >= 0 {
			readBlock = pollInterval
			if !deadline.IsZero() {
				remaining := time.Until(deadline)
				if remaining <= 0 {
					return Message{}, redis.Nil
				}
				readBlock = min(readBlock, remaining)
			}
			// Redis rounds BLOCK to milliseconds. Avoid truncating a short duration
			// into BLOCK 0, which would wait indefinitely.
			readBlock = max(readBlock, time.Millisecond)
		}
		streams, err := c.client.XRead(ctx, &redis.XReadArgs{Streams: []string{key, id}, Count: 1, Block: readBlock}).Result()
		if err := context.Cause(ctx); err != nil {
			return Message{}, err
		}
		if errors.Is(err, redis.Nil) && block >= 0 {
			continue
		}
		if err != nil {
			return Message{}, err
		}
		for _, stream := range streams {
			for _, msg := range stream.Messages {
				value, ok := msg.Values[dataKey].(string)
				if !ok {
					return Message{ID: msg.ID}, fmt.Errorf("%w (id %s)", ErrMalformedEntry, msg.ID)
				}
				return Message{ID: msg.ID, Value: []byte(value)}, nil
			}
		}
		return Message{}, redis.Nil
	}
}
