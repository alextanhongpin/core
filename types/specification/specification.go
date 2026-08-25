package specification

type specification[T any] interface {
	IsSatisfiedBy(T) bool
}

func And[T any](a, b specification[T]) specification[T] {
	return Func[T](func(v T) bool {
		return a.IsSatisfiedBy(v) && b.IsSatisfiedBy(v)
	})
}

func Or[T any](a, b specification[T]) specification[T] {
	return Func[T](func(v T) bool {
		return a.IsSatisfiedBy(v) || b.IsSatisfiedBy(v)
	})
}

func Xor[T any](a, b specification[T]) specification[T] {
	return Func[T](func(v T) bool {
		return a.IsSatisfiedBy(v) != b.IsSatisfiedBy(v)
	})
}

func All[T any](as ...specification[T]) specification[T] {
	return Func[T](func(v T) bool {
		for _, a := range as {
			if !a.IsSatisfiedBy(v) {
				return false
			}
		}
		return true
	})
}

func Any[T any](as ...specification[T]) specification[T] {
	return Func[T](func(v T) bool {
		for _, a := range as {
			if a.IsSatisfiedBy(v) {
				return true
			}
		}
		return false
	})
}

type Func[T any] func(T) bool

func (f Func[T]) IsSatisfiedBy(v T) bool {
	return f(v)
}
