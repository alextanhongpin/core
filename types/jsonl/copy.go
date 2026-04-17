package jsonl

func Copy[T any](from, to *File[T]) error {
	seq, stop := from.ReadLines()
	for line := range seq {
		err := to.Write(line)
		if err != nil {
			return err
		}
	}

	return stop()
}

func CopyFunc[T any](from, to *File[T], fn func(T) bool) error {
	seq, stop := from.ReadLines()
	for line := range seq {
		if !fn(line) {
			continue
		}
		err := to.Write(line)
		if err != nil {
			return err
		}
	}

	return stop()
}
