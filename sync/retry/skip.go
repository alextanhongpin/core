package retry

import "errors"

type SkipError struct {
	err error
}

func Skip(err error) error {
	return &SkipError{err: err}
}

func (e *SkipError) Is(err error) bool {
	return errors.Is(err, e)
}

func (e *SkipError) Error() string {
	return e.err.Error()
}

func (e *SkipError) Unwrap() error {
	return e.err
}
