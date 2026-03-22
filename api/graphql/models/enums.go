package models

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// GitRefType is the kind of git reference: a branch or a tag.
type GitRefType string

const (
	// GitRefTypeBranch refers to a local branch (refs/heads/*).
	GitRefTypeBranch GitRefType = "BRANCH"
	// GitRefTypeTag refers to an annotated or lightweight tag (refs/tags/*).
	GitRefTypeTag GitRefType = "TAG"
)

func (e GitRefType) IsValid() bool {
	switch e {
	case GitRefTypeBranch, GitRefTypeTag:
		return true
	}
	return false
}

func (e GitRefType) String() string { return string(e) }

func (e *GitRefType) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}
	*e = GitRefType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid GitRefType", str)
	}
	return nil
}

func (e GitRefType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

func (e *GitRefType) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	return e.UnmarshalGQL(s)
}

func (e GitRefType) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	e.MarshalGQL(&buf)
	return buf.Bytes(), nil
}
