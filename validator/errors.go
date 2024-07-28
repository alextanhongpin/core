package validator

import (
	"fmt"
	"strings"
)

type FieldError struct {
	Field string
	Error error
}

func Field(field string, err error) FieldError {
	return FieldError{Field: field, Error: err}
}

type Errors map[string]string

func (ve Errors) Error() string {
	errs := make([]string, 0, len(ve))
	for field, msg := range ve {
		errs = append(errs, fmt.Sprintf("%s: %s", field, msg))
	}

	return strings.Join(errs, "\n")
}

func NewErrors(fes ...FieldError) error {
	ve := make(Errors)
	for _, fe := range fes {
		if fe.Error == nil {
			continue
		}
		ve[fe.Field] = fe.Error.Error()
	}

	if len(ve) == 0 {
		return nil
	}

	return ve
}
