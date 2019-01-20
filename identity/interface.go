package identity

import (
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
)

type Interface interface {
	Id() string

	Name() string
	Email() string
	Login() string
	AvatarUrl() string

	// Keys return the last version of the valid keys
	Keys() []Key

	// ValidKeysAtTime return the set of keys valid at a given lamport time
	ValidKeysAtTime(time lamport.Time) []Key

	// DisplayName return a non-empty string to display, representing the
	// identity, based on the non-empty values.
	DisplayName() string

	// Validate check if the Identity data is valid
	Validate() error

	// Write the identity into the Repository. In particular, this ensure that
	// the Id is properly set.
	Commit(repo repository.Repo) error

	// IsProtected return true if the chain of git commits started to be signed.
	// If that's the case, only signed commit with a valid key for this identity can be added.
	IsProtected() bool
}
