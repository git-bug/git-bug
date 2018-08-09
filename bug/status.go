package bug

type Status int

const (
	_ Status = iota
	OpenStatus
	ClosedStatus
)

func (s Status) String() string {
	switch s {
	case OpenStatus:
		return "open"
	case ClosedStatus:
		return "closed"
	default:
		return "unknown status"
	}
}

func (s Status) Action() string {
	switch s {
	case OpenStatus:
		return "opened"
	case ClosedStatus:
		return "closed"
	default:
		return "unknown status"
	}
}
