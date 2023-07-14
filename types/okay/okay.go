package okay

import (
	"context"
	"errors"
	"fmt"
)

var NotAllowed = errors.New("not allowed")

// Return generic, so that we can add custom types.
type Response interface {
	Err() error
	OK() bool
}

// Allows, Denies, None, All, Any, Authorize, Verify.
type OK[T any] interface {
	Allows(ctx context.Context, t T) Response
}

func Check[T any](ctx context.Context, t T, oks ...OK[T]) Response {
	for _, ok := range oks {
		if res := ok.Allows(ctx, t); !res.OK() {
			return res
		}
	}

	return Allow(len(oks) > 0)
}

type Func[T any] func(ctx context.Context, t T) Response

func (fn Func[T]) Allows(ctx context.Context, t T) Response {
	return fn(ctx, t)
}

type Okay[T any] struct {
	checks []OK[T]
}

func New[T any](checks ...OK[T]) *Okay[T] {
	return &Okay[T]{
		checks: checks,
	}
}

func (o *Okay[T]) Add(fns ...func(ctx context.Context, t T) Response) {
	funcs := make([]OK[T], len(fns))
	for i, fn := range fns {
		funcs[i] = Func[T](fn)
	}

	o.checks = append(o.checks, funcs...)
}

func (o *Okay[T]) AddFunc(oks ...OK[T]) {
	o.checks = append(o.checks, oks...)
}

func (o *Okay[T]) Allows(ctx context.Context, t T) Response {
	if len(o.checks) == 0 {
		return Error(NotAllowed)
	}

	return Check(ctx, t, o.checks...)
}

type response struct {
	ok  bool
	err error
}

func Error(err error) *response {
	return &response{
		err: err,
	}
}

func Errorf(msg string, args ...any) *response {
	return &response{
		err: fmt.Errorf(msg, args...),
	}
}

func (r *response) OK() bool {
	return r.ok
}

func (r *response) Err() error {
	return r.err
}

func Allow(ok bool) *response {
	return &response{
		ok: ok,
	}
}
