package internal

import "context"

// CommandHandler returns an error.
type CommandHandler interface {
	Exec(ctx context.Context) error
}

// QueryHandler returns a value and an error.
type QueryHandler[T any] interface {
	Exec(ctx context.Context) (T, error)
}

// RequestReplyHandler accepts a request, and returns a
// value and an error.
type RequestReplyHandler[T, U any] interface {
	Exec(ctx context.Context, t T) (U, error)
}

// RequestHandler accepts a request, and returns an error.
type RequestHandler[T any] interface {
	Exec(ctx context.Context, t T) error
}

// SilentHandler is similar like CommandHandler, but
// returns no error.
type SilentHandler interface {
	Exec(ctx context.Context)
}

// SilentRequestHandler is similar like RequestHandler,
// but returns no error.
type SilentRequestHandler[T any] interface {
	Exec(ctx context.Context, t T)
}

type CommandHandlerFunc func(ctx context.Context) error

func (h CommandHandlerFunc) Exec(ctx context.Context) error {
	return h(ctx)
}

type QueryHandlerFunc[T any] func(ctx context.Context) (T, error)

func (h QueryHandlerFunc[T]) Exec(ctx context.Context) (T, error) {
	return h(ctx)
}

type RequestReplyHandlerFunc[T, U any] func(ctx context.Context, v T) (U, error)

func (h RequestReplyHandlerFunc[T, U]) Exec(ctx context.Context, v T) (U, error) {
	return h(ctx, v)
}

func (h RequestReplyHandlerFunc[T, U]) ToQueryHandler(ctx context.Context, v T) QueryHandler[U] {
	return QueryHandlerFunc[U](func(ctx context.Context) (U, error) {
		return h.Exec(ctx, v)
	})
}

type RequestHandlerFunc[T any] func(ctx context.Context, v T) error

func (h RequestHandlerFunc[T]) Exec(ctx context.Context, v T) error {
	return h(ctx, v)
}

func (h RequestHandlerFunc[T]) ToCommandHandler(ctx context.Context, v T) CommandHandler {
	return CommandHandlerFunc(func(ctx context.Context) error {
		return h.Exec(ctx, v)
	})
}
