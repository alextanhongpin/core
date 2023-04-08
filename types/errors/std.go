package errors

import "errors"

// To reduce the need to import the std errors under another alias.
var (
	Is     = errors.Is
	As     = errors.As
	Unwrap = errors.Unwrap
	New    = errors.New
)
