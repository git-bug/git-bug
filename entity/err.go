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

// ErrOldFormatVersion indicate that the read data has a too old format.
type ErrOldFormatVersion struct {
	formatVersion uint
}

func NewErrOldFormatVersion(formatVersion uint) *ErrOldFormatVersion {
	return &ErrOldFormatVersion{formatVersion: formatVersion}
}

func (e ErrOldFormatVersion) Error() string {
	return fmt.Sprintf("outdated repository format %v, please use https://github.com/MichaelMure/git-bug-migration to upgrade", e.formatVersion)
}

// ErrNewFormatVersion indicate that the read data is too new for this software.
type ErrNewFormatVersion struct {
	formatVersion uint
}

func NewErrNewFormatVersion(formatVersion uint) *ErrNewFormatVersion {
	return &ErrNewFormatVersion{formatVersion: formatVersion}
}

func (e ErrNewFormatVersion) Error() string {
	return fmt.Sprintf("your version of git-bug is too old for this repository (version %v), please upgrade to the latest version", e.formatVersion)
}
