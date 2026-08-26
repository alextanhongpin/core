package multicloser

import (
	"errors"
	"io"
)

// MultiCloser joins multiple io.Closer instances into one.
type MultiCloser []io.Closer

// Close executes all closers and merges their errors.
func (mc MultiCloser) Close() error {
	var errs []error
	for _, c := range mc {
		if c == nil {
			continue
		}
		errs = append(errs, c.Close())
	}
	return errors.Join(errs...)
}
