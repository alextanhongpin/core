package states

type Status int

const (
	NotStarted Status = iota
	Pending
	Success
	Failed
)

func (s Status) Valid() bool {
	switch s {
	case NotStarted, Pending, Success, Failed:
		return true
	default:
		return false
	}
}

func (s Status) String() string {
	switch s {
	case NotStarted:
		return "not started"
	case Pending:
		return "pending"
	case Success:
		return "success"
	case Failed:
		return "failed"
	}

	return "unknown"
}
