package entity

import (
	"fmt"
	"strings"
)

type ErrMultipleMatch struct {
	entityType string
	Matching   []Id
}

func NewErrMultipleMatch(entityType string, matching []Id) *ErrMultipleMatch {
	return &ErrMultipleMatch{entityType: entityType, Matching: matching}
}

func (e ErrMultipleMatch) Error() string {
	matching := make([]string, len(e.Matching))

	for i, match := range e.Matching {
		matching[i] = match.String()
	}

	return fmt.Sprintf("Multiple matching %s found:\n%s",
		e.entityType,
		strings.Join(matching, "\n"))
}

func IsErrMultipleMatch(err error) bool {
	_, ok := err.(*ErrMultipleMatch)
	return ok
}

type ErrInvalidFormat struct {
	version  uint
	expected uint
}

func NewErrInvalidFormat(version uint, expected uint) *ErrInvalidFormat {
	return &ErrInvalidFormat{
		version:  version,
		expected: expected,
	}
}

func NewErrUnknowFormat(expected uint) *ErrInvalidFormat {
	return &ErrInvalidFormat{
		version:  0,
		expected: expected,
	}
}

func (e ErrInvalidFormat) Error() string {
	if e.version == 0 {
		return fmt.Sprintf("unreadable data, expected format version %v", e.expected)
	}
	if e.version < e.expected {
		return fmt.Sprintf("outdated repository format %v, please use https://github.com/MichaelMure/git-bug-migration to upgrade to format version %v", e.version, e.expected)
	}
	return fmt.Sprintf("your version of git-bug is too old for this repository (format version %v, expected %v), please upgrade to the latest version", e.version, e.expected)
}
