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
