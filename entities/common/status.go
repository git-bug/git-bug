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
	MergedStatus
	DraftStatus
)

func (s Status) String() string {
	switch s {
	case OpenStatus:
		return "open"
	case ClosedStatus:
		return "closed"
	case MergedStatus:
		return "merged"
	case DraftStatus:
		return "draft"
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
	case MergedStatus:
		return "merged"
	case DraftStatus:
		return "marked as draft"
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
	case "merged":
		return MergedStatus, nil
	case "draft":
		return DraftStatus, nil
	default:
		return 0, fmt.Errorf("unknown status")
	}
}

func (s Status) Validate() error {
	switch s {
	case OpenStatus, ClosedStatus, MergedStatus, DraftStatus:
		return nil
	default:
		return fmt.Errorf("invalid")
	}
}

func (s Status) MarshalGQL(w io.Writer) {
	switch s {
	case OpenStatus:
		_, _ = w.Write([]byte(strconv.Quote("OPEN")))
	case ClosedStatus:
		_, _ = w.Write([]byte(strconv.Quote("CLOSED")))
	case MergedStatus:
		_, _ = w.Write([]byte(strconv.Quote("MERGED")))
	case DraftStatus:
		_, _ = w.Write([]byte(strconv.Quote("DRAFT")))
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
	case "MERGED":
		*s = MergedStatus
	case "DRAFT":
		*s = DraftStatus
	default:
		return fmt.Errorf("%s is not a valid Status", str)
	}
	return nil
}
