// package result wraps both Data and error together. This is useful when
// passing Data through channels.
//
// Ideally, each package that needs to have their own result type should define
// their own result type.
//
// Another form of result is using interface, where instead of just wrapping
// Data, an entire struct can be returned that fulfils the result interface.
package result

import "errors"

var Zero = errors.New("result: no result")

type Result[T any] struct {
	Data T
	Err  error
}

func (r *Result[T]) Unwrap() (T, error) {
	return r.Data, r.Err
}

func New[T any]() *Result[T] {
	return &Result[T]{}
}

func OK[T any](v T) *Result[T] {
	return &Result[T]{
		Data: v,
	}
}

func Err[T any](err error) *Result[T] {
	return &Result[T]{
		Err: err,
	}
}

func All[T any](rs ...*Result[T]) ([]T, error) {
	if len(rs) == 0 {
		return nil, Zero
	}

	res := make([]T, len(rs))
	for i, r := range rs {
		v, err := r.Unwrap()
		if err != nil {
			return nil, err
		}
		res[i] = v
	}

	return res, nil
}

func Any[T any](rs ...*Result[T]) (T, error) {
	if len(rs) == 0 {
		var t T
		return t, Zero
	}

	res := make([]error, len(rs))
	for i, r := range rs {
		v, err := r.Unwrap()
		if err == nil {
			return v, nil
		}

		res[i] = err
	}

	var t T
	return t, errors.Join(res...)
}
