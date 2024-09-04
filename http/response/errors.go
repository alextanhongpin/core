package response

import (
	"fmt"
	"slices"
	"strings"
)

type ValidationErrors map[string]string

func NewValidationErrors(m map[string]string) error {
	for k, v := range m {
		if v == "" {
			delete(m, k)
		}
	}
	if len(m) == 0 {
		return nil
	}

	return ValidationErrors(m)
}

func (ve ValidationErrors) Error() string {
	keys := make([]string, 0, len(ve))
	for k := range ve {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	errs := make([]string, 0, len(ve))
	for _, k := range keys {
		v := ve[k]
		errs = append(errs, fmt.Sprintf("%s: %s", k, v))
	}

	return strings.Join(errs, "\n")
}
