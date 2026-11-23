package jsonl

// Copy writes every value of from to to. from and to must be different
// files.
func Copy[T any](from, to *File[T]) error {
	return CopyFunc(from, to, func(T) bool { return true })
}

// CopyFunc writes the values of from for which keep returns true to to.
// from and to must be different files.
func CopyFunc[T any](from, to *File[T], keep func(T) bool) error {
	seq, stop := from.ReadLines()
	for v := range seq {
		if !keep(v) {
			continue
		}
		if err := to.Write(v); err != nil {
			return err
		}
	}
	return stop()
}
