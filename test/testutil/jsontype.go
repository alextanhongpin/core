package testutil

type JSONTypeInterceptor[T any] func(T) (T, error)

func (i JSONTypeInterceptor[T]) isJSONType() {}
