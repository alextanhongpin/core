package validator

import (
	"fmt"
	"strings"
)

type Errors map[string]string

func (ve Errors) Error() string {
	errs := make([]string, 0, len(ve))
	for field, msg := range ve {
		errs = append(errs, fmt.Sprintf("%s: %s", field, msg))
	}

	return strings.Join(errs, "\n")
}

func NewErrors(m map[string]error) error {
	ve := make(Errors)
	for field, err := range m {
		if err == nil {
			continue
		}
		ve[field] = err.Error()
	}

	if len(ve) == 0 {
		return nil
	}

	return ve
}
