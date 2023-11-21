package always

// Validate checks if the given objects are valid.
func Validate[T any](t T, ts ...Validator[T]) error {
	for i := 0; i < len(ts); i++ {
		if err := ts[i].Validate(t); err != nil {
			return err
		}
	}

	return nil
}

// Validator validates an entity. Note that there is no
// use of using such generic interface:
//
// type Validator[T any] interface {
// 	 Validate(T) error
// }
//
// Which only increases complexity, since the type T could
// have been kept in the struct, keeping the surface area
// smaller (like Valid).

type Validator[T any] func(T) error

func (fn Validator[T]) Validate(t T) error {
	return fn(t)
}
