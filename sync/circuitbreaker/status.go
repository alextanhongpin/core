package circuitbreaker

type Status int64

const (
	Closed Status = iota
	Open
	HalfOpen
)

func (s Status) Int64() int64 {
	return int64(s)
}

var statusText = map[Status]string{
	Closed:   "closed",
	Open:     "open",
	HalfOpen: "half-open",
}

func (s Status) String() string {
	return statusText[s]
}

func (s Status) IsOpen() bool {
	return s == Open
}

func (s Status) IsClosed() bool {
	return s == Closed
}

func (s Status) IsHalfOpen() bool {
	return s == HalfOpen
}
