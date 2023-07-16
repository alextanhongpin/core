package always

// Valid checks if an entity is valid.
type Validatable interface {
	Valid() error
}

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
