package okay

import (
	"context"
	"errors"
	"fmt"
)

var (
	Invalid = errors.New("okay: no successful checks")
	Denied  = errors.New("okay: denied")
)

type Response interface {
	Unwrap() (bool, error)
}

// OK can be used for authorization, authentication, verification of context,
// as well as validation.
type OK[T any] interface {
	Check(ctx context.Context, t T) Response
}

type Func[T any] func(ctx context.Context, t T) Response

func (fn Func[T]) Check(ctx context.Context, t T) Response {
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

// All requires all checks to be successful.
func (o *Okay[T]) All(ctx context.Context, t T) Response {
	if len(o.checks) == 0 {
		return Error(Invalid)
	}

	for _, ok := range o.checks {
		res := ok.Check(ctx, t)
		if ok, _ := res.Unwrap(); !ok {
			return res
		}
	}

	return Allow(true)
}

// Any requires just one check to be successful.
func (o *Okay[T]) Any(ctx context.Context, t T) Response {
	if len(o.checks) == 0 {
		return Error(Invalid)
	}

	for _, ok := range o.checks {
		res := ok.Check(ctx, t)
		if ok, _ := res.Unwrap(); ok {
			return res
		}
	}

	return Error(Invalid)
}

// Some is just an alias to Any.
func (o *Okay[T]) Some(ctx context.Context, t T) Response {
	return o.Any(ctx, t)
}

// None requires all checks to be false.
func (o *Okay[T]) None(ctx context.Context, t T) Response {
	if len(o.checks) == 0 {
		return Error(Invalid)
	}

	for _, ok := range o.checks {
		res := ok.Check(ctx, t)
		if ok, _ := res.Unwrap(); ok {
			return Error(Denied)
		}
	}

	return Allow(true)
}

type Result struct {
	ok  bool
	err error
}

func (r *Result) Unwrap() (bool, error) {
	return r.ok, r.err
}

func Error(err error) *Result {
	return &Result{
		err: err,
	}
}

func Errorf(msg string, args ...any) *Result {
	return &Result{
		err: fmt.Errorf(msg, args...),
	}
}

func Allow(ok bool) *Result {
	return &Result{
		ok: ok,
	}
}

type deny[T any] struct{}

func (d *deny[T]) Check(ctx context.Context, t T) Response {
	return Error(Denied)
}

func Deny[T any]() *deny[T] {
	return &deny[T]{}
}

type verify[T any] struct{}

func (v *verify[T]) Check(ctx context.Context, t T) Response {
	return Allow(true)
}

func NoVerify[T any]() *verify[T] {
	return &verify[T]{}
}
