// package result wraps both data and error together. This is useful when
// passing data through channels.
//
// Ideally, each package that needs to have their own result type should define
// their own result type.
//
// Another form of Result is using interface, where instead of just wrapping
// data, an entire struct can be returned that fulfils the Result interface.
package result

type Result[T any] struct {
	Data T
	Err  error
}

func (r *Result[T]) Unwrap() (T, error) {
	return r.Data, r.Err
}
