package channel

import (
	"context"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/helper"
)

const dataKey = "data"

type Channel struct {
	client *redis.Client
}

func New(client *redis.Client) *Channel {
	return &Channel{client: client}
}

func (c *Channel) Send(ctx context.Context, key string, value []byte) error {
	_, err := c.client.XAdd(ctx, &redis.XAddArgs{
		Stream:       key,
		Values:       []string{dataKey, string(value)},
		ProducerID:   key,
		IdempotentID: fmt.Sprint(helper.DigestString(string(value))),
	}).Result()
	return err
}

func (c *Channel) Close(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *Channel) Recv(ctx context.Context, key string, block time.Duration) ([]byte, error) {
	streams, err := c.client.XRead(ctx, &redis.XReadArgs{
		Streams: []string{key, "$"},
		Count:   1,
		Block:   block,
	}).Result()
	if err != nil {
		return nil, err
	}

	for _, stream := range streams {
		for _, msg := range stream.Messages {
			return []byte(msg.Values[dataKey].(string)), nil
		}
	}

	return nil, redis.Nil
}
