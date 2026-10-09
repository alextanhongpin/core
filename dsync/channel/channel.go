package channel

import (
	"context"
	"errors"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const dataKey = "data"

type Channel struct {
	client *redis.Client
}

// New borrows a client; callers retain cleanup ownership.
func New(client *redis.Client) (*Channel, error) {
	if client == nil {
		return nil, errors.New("channel: nil client")
	}
	return &Channel{client: client}, nil
}
func MustNew(client *redis.Client) *Channel {
	c, err := New(client)
	if err != nil {
		panic(err)
	}
	return c
}

func (c *Channel) Send(ctx context.Context, key string, value []byte) error {
	_, err := c.client.XAdd(ctx, &redis.XAddArgs{
		Stream: key,
		Values: []string{dataKey, string(value)},
	}).Result()
	return err
}

func (c *Channel) Close(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Message is one stream entry. Its ID can be supplied to RecvAfter to resume.
type Message struct {
	ID    string
	Value []byte
}

// Recv waits for an entry added after this call snapshots the stream cursor. It does not replay
// entries already present at that snapshot. Zero block waits indefinitely; negative block
// polls. Close deletes stored entries but does not wake blocked reads.
func (c *Channel) Recv(ctx context.Context, key string, block time.Duration) ([]byte, error) {
	msg, err := c.RecvAfter(ctx, key, "$", block)
	return msg.Value, err
}

// RecvAfter reads the first entry after id; use "0-0" to read from the beginning.
// Each caller owns its cursor. This is a broadcast log, not a consuming queue.
func (c *Channel) RecvAfter(ctx context.Context, key, id string, block time.Duration) (Message, error) {
	if err := context.Cause(ctx); err != nil {
		return Message{}, err
	}
	if key == "" || id == "" {
		return Message{}, errors.New("channel: key and cursor are required")
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
			readBlock = 100 * time.Millisecond
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
					return Message{}, errors.New("channel: entry has no string data field")
				}
				return Message{ID: msg.ID, Value: []byte(value)}, nil
			}
		}
		return Message{}, redis.Nil
	}
}
