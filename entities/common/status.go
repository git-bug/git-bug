package common

import (
	"fmt"
	"io"
	"strconv"
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

func (s Status) MarshalGQL(w io.Writer) {
	switch s {
	case OpenStatus:
		_, _ = w.Write([]byte(strconv.Quote("OPEN")))
	case ClosedStatus:
		_, _ = w.Write([]byte(strconv.Quote("CLOSED")))
	default:
		panic("missing case")
	}
}

func (s *Status) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}
	switch str {
	case "OPEN":
		*s = OpenStatus
	case "CLOSED":
		*s = ClosedStatus
	default:
		return fmt.Errorf("%s is not a valid Status", str)
	}
	return nil
}
