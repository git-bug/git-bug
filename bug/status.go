package bug

import (
	"fmt"
	"strings"
)

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

func StatusFromString(str string) (Status, error) {
	cleaned := strings.ToLower(strings.TrimSpace(str))

	switch cleaned {
	case "open":
		return OpenStatus, nil
	case "closed":
		return ClosedStatus, nil
	default:
		return 0, fmt.Errorf("unknown status")
	}
}

func (s Status) Validate() error {
	if s != OpenStatus && s != ClosedStatus {
		return fmt.Errorf("invalid")
	}

	return nil
}
