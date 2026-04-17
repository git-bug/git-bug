package common

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Type distinguishes the kind of bug entity: a plain issue or a pull-request.
// The zero value is IssueType so pre-existing bugs deserialize correctly
// without a migration.
type Type int

const (
	IssueType Type = iota
	PRType
)

func (t Type) String() string {
	switch t {
	case IssueType:
		return "issue"
	case PRType:
		return "pr"
	default:
		return "unknown type"
	}
}

func TypeFromString(str string) (Type, error) {
	switch strings.ToLower(strings.TrimSpace(str)) {
	case "issue", "bug":
		return IssueType, nil
	case "pr", "pull-request", "pullrequest":
		return PRType, nil
	default:
		return 0, fmt.Errorf("unknown type %q", str)
	}
}

func (t Type) Validate() error {
	switch t {
	case IssueType, PRType:
		return nil
	default:
		return fmt.Errorf("invalid type")
	}
}

func (t Type) MarshalGQL(w io.Writer) {
	switch t {
	case IssueType:
		_, _ = w.Write([]byte(strconv.Quote("ISSUE")))
	case PRType:
		_, _ = w.Write([]byte(strconv.Quote("PR")))
	default:
		panic("missing case")
	}
}

func (t *Type) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}
	switch str {
	case "ISSUE":
		*t = IssueType
	case "PR":
		*t = PRType
	default:
		return fmt.Errorf("%s is not a valid Type", str)
	}
	return nil
}
