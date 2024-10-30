package probs

type Event struct {
	data []any
}

func (e *Event) Add(key any, value int64) {
	e.data = append(e.data, key, value)
}

func (e *Event) Data() []any {
	return e.data
}
