package config

import (
	"fmt"

	"github.com/git-bug/git-bug/util/text"
)

// The repo section: what describes the repository.

// RepoSection holds the settings describing the repository
type RepoSection struct {
	description register[string]
}

// Description returns the description of the repository, a single line of at most
// maxRepoDescriptionLength characters, empty by default.
func (s RepoSection) Description() string {
	return s.description.get()
}

const maxRepoDescriptionLength = 256

func validateRepoDescription(description string) error {
	if !text.SafeOneLine(description) {
		return fmt.Errorf("description is not a single line of printable text")
	}
	if err := validateLength(description, maxRepoDescriptionLength); err != nil {
		return fmt.Errorf("description: %w", err)
	}
	return nil
}
