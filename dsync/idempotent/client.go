package idempotent

import (
	"context"
	"encoding/json"
	"reflect"
)

type client[In, Out any] interface {
	Do(ctx context.Context, key string, req In) (res Out, shared bool, err error)
}

func NewClient[In, Out any](s Store, fn func(ctx context.Context, req In) (Out, error), opts ...Option) client[In, Out] {
	client := newClient[In, Out](s, opts...)
	client.handle(fn)
	return client
}

type Client[In, Out any] struct {
	s    Store
	opts []Option
}

func newClient[In, Out any](s Store, opts ...Option) *Client[In, Out] {
	return &Client[In, Out]{
		s:    s,
		opts: opts,
	}
}

func (c *Client[In, Out]) handle(fn func(context.Context, In) (Out, error)) {
	var in In
	c.s.HandleFunc(getTypeName(in), func(ctx context.Context, b []byte) ([]byte, error) {
		var req In
		if err := json.Unmarshal(b, &req); err != nil {
			return nil, err
		}

		res, err := fn(ctx, req)
		if err != nil {
			return nil, err
		}

		return json.Marshal(res)
	})
}

func (c *Client[In, Out]) Do(ctx context.Context, key string, req In) (res Out, shared bool, err error) {
	b, err := json.Marshal(req)
	if err != nil {
		return res, shared, err
	}

	b, shared, err = c.s.Do(ctx, getTypeName(req), key, b, c.opts...)
	if err != nil {
		return
	}

	err = json.Unmarshal(b, &res)
	return
}

func getTypeName(v any) string {
	if v == nil {
		return "<nil>"
	}

	return reflect.TypeOf(v).String()
}
