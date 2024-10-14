package validator

import (
	"fmt"
	"sort"
	"strings"
)

type Errors map[string]string

func (ve Errors) Error() string {
	keys := make([]string, 0, len(ve))
	for key := range ve {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	errs := make([]string, 0, len(ve))
	for _, key := range keys {
		errs = append(errs, fmt.Sprintf("%s: %s", key, ve[key]))
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
