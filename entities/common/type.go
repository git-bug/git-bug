package common

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Kind distinguishes the kind of bug entity: a plain issue or a pull-request.
// The zero value is IssueKind so pre-existing bugs deserialize correctly
// without a migration.
//
// This type is named Kind (not Type) to avoid gqlgen's autobind picking it
// up as the target for GraphQL's introspection `__Type`.
type Kind int

const (
	IssueKind Kind = iota
	PRKind
)

func (k Kind) String() string {
	switch k {
	case IssueKind:
		return "issue"
	case PRKind:
		return "pr"
	default:
		return "unknown kind"
	}
}

func KindFromString(str string) (Kind, error) {
	switch strings.ToLower(strings.TrimSpace(str)) {
	case "issue", "bug":
		return IssueKind, nil
	case "pr", "pull-request", "pullrequest":
		return PRKind, nil
	default:
		return 0, fmt.Errorf("unknown kind %q", str)
	}
}

func (k Kind) Validate() error {
	switch k {
	case IssueKind, PRKind:
		return nil
	default:
		return fmt.Errorf("invalid kind")
	}
}

func (k Kind) MarshalGQL(w io.Writer) {
	switch k {
	case IssueKind:
		_, _ = w.Write([]byte(strconv.Quote("ISSUE")))
	case PRKind:
		_, _ = w.Write([]byte(strconv.Quote("PR")))
	default:
		panic("missing case")
	}
}

func (k *Kind) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}
	switch str {
	case "ISSUE":
		*k = IssueKind
	case "PR":
		*k = PRKind
	default:
		return fmt.Errorf("%s is not a valid Kind", str)
	}
	return nil
}
