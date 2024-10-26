package response

type ValidationErrors map[string]string

func NewValidationErrors(m map[string]string) error {
	err := make(map[string]string)
	for k, v := range m {
		if v == "" {
			continue
		}

		err[k] = v
	}

	if len(err) == 0 {
		return nil
	}

	return ValidationErrors(err)
}

func (ve ValidationErrors) Error() string {
	return "Validation failed"
}
