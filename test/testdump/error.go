package testdump

import (
	"errors"

	"github.com/alextanhongpin/core/internal"
)

type DiffError = internal.DiffError

func AsDiffError(err error) (*DiffError, bool) {
	var diffErr *DiffError
	return diffErr, errors.As(err, &diffErr)
}
