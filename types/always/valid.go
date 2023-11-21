package always

// Validatable is an interface that represents types that can be validated.
type Validatable interface {
	// Valid performs validation on the implementing type.
	// It returns an error if the validation fails, or nil if the validation passes.
	Valid() error
}

// Valid checks if the given Validatable objects are valid.
// It iterates through each object and calls the Valid() method.
// If any object returns an error, it immediately returns that error.
// If all objects are valid, it returns nil.
func Valid(ts ...Validatable) error {
	for i := 0; i < len(ts); i++ {
		if err := ts[i].Valid(); err != nil {
			return err
		}
	}

	return nil
}

type ValidFunc func() error

func (fn ValidFunc) Valid() error {
	return fn()
}
