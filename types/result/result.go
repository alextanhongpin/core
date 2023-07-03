// package result wraps both data and error together.
package result

type Result[T any] struct {
	Data T
	Err  error
}

func (r *Result[T]) Unwrap() (T, error) {
	return r.Data, r.Err
}
