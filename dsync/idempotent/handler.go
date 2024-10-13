package idempotent

import "context"

type HandlerFunc func(ctx context.Context, req []byte) ([]byte, error)

func (fn HandlerFunc) Handle(ctx context.Context, req []byte) ([]byte, error) {
	return fn(ctx, req)
}

type Handler interface {
	Handle(ctx context.Context, req []byte) ([]byte, error)
}
