// package result wraps both data and error together. This is useful when
// passing data through channels.
//
// Ideally, each package that needs to have their own result type should define
// their own result type.
//
// Another form of result is using interface, where instead of just wrapping
// data, an entire struct can be returned that fulfils the result interface.
package result

import "errors"

var Zero = errors.New("result: no result")

type Result[T any] interface {
	Unwrap() (T, error)
}

type result[T any] struct {
	data T
	err  error
}

func (r *result[T]) Unwrap() (T, error) {
	return r.data, r.err
}

func OK[T any](v T) Result[T] {
	return &result[T]{
		data: v,
	}
}

func Err[T any](err error) Result[T] {
	return &result[T]{
		err: err,
	}
}

func All[T any](rs ...result[T]) ([]T, error) {
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

func Any[T any](rs ...result[T]) (T, error) {
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
