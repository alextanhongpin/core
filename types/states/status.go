package states

type Status int

const (
	Idle Status = iota
	Pending
	Success
	Failed
)

func (s Status) Valid() bool {
	switch s {
	case Idle, Pending, Success, Failed:
		return true
	default:
		return false
	}
}

func (s Status) String() string {
	switch s {
	case Idle:
		return "idle"
	case Pending:
		return "pending"
	case Success:
		return "success"
	case Failed:
		return "failed"
	}

	return "unknown"
}
