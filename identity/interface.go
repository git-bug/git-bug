package identity

import (
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/util/lamport"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

type Interface interface {
	entity.Interface

	// Name return the last version of the name
	// Can be empty.
	Name() string

	// Email return the last version of the email
	// Can be empty.
	Email() string

	// Login return the last version of the login
	// Can be empty.
	// Warning: this login can be defined when importing from a bridge but should *not* be
	// used to identify an identity as multiple bridge with different login can map to the same
	// identity. Use the metadata system for that usage instead.
	Login() string

	// AvatarUrl return the last version of the Avatar URL
	// Can be empty.
	AvatarUrl() string

	// Keys return the last version of the valid keys
	// Can be empty.
	Keys() []*Key

	// ValidKeysAtTime return the set of keys valid at a given lamport time
	// Can be empty.
	ValidKeysAtTime(time lamport.Time) []*Key

	// DisplayName return a non-empty string to display, representing the
	// identity, based on the non-empty values.
	DisplayName() string

	// Validate check if the Identity data is valid
	Validate() error

	// IsProtected return true if the chain of git commits started to be signed.
	// If that's the case, only signed commit with a valid key for this identity can be added.
	IsProtected() bool

	// LastModificationLamportTime return the Lamport time at which the last version of the identity became valid.
	LastModificationLamport() lamport.Time

	// LastModification return the timestamp at which the last version of the identity became valid.
	LastModification() timestamp.Timestamp

	// Indicate that the in-memory state changed and need to be commit in the repository
	NeedCommit() bool
}
