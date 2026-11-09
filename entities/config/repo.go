package config

import (
	"fmt"

	"github.com/git-bug/git-bug/util/text"
)

// The repo section: what describes the repository.

// RepoSection holds the settings describing the repository
type RepoSection struct {
	// Description of the repository, a single line of at most
	// maxRepoDescriptionLength characters, empty by default
	Description string
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
