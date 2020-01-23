package identity

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
	"github.com/MichaelMure/git-bug/util/text"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

var _ Interface = &Bare{}
var _ entity.Interface = &Bare{}

// Bare is a very minimal identity, designed to be fully embedded directly along
// other data.
//
// in particular, this identity is designed to be compatible with the handling of
// identities in the early version of git-bug.
type Bare struct {
	id        entity.Id
	name      string
	email     string
	avatarUrl string
}

func NewBare(name string, email string) *Bare {
	return &Bare{id: entity.UnsetId, name: name, email: email}
}

func NewBareFull(name string, email string, avatarUrl string) *Bare {
	return &Bare{id: entity.UnsetId, name: name, email: email, avatarUrl: avatarUrl}
}

func deriveId(data []byte) entity.Id {
	sum := sha256.Sum256(data)
	return entity.Id(fmt.Sprintf("%x", sum))
}

type bareIdentityJSON struct {
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	Login     string `json:"login,omitempty"` // Deprecated, only kept to have the same ID when reading an old value
	AvatarUrl string `json:"avatar_url,omitempty"`
}

func (i *Bare) MarshalJSON() ([]byte, error) {
	return json.Marshal(bareIdentityJSON{
		Name:      i.name,
		Email:     i.email,
		AvatarUrl: i.avatarUrl,
	})
}

func (i *Bare) UnmarshalJSON(data []byte) error {
	// Compute the Id when loading the op from disk.
	i.id = deriveId(data)

	aux := bareIdentityJSON{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	i.name = aux.Name
	i.email = aux.Email
	i.avatarUrl = aux.AvatarUrl

	return nil
}

// Id return the Identity identifier
func (i *Bare) Id() entity.Id {
	// We don't have a proper Id at hand, so let's hash all the data to get one.

	if i.id == "" {
		// something went really wrong
		panic("identity's id not set")
	}
	if i.id == entity.UnsetId {
		// This means we are trying to get the identity identifier *before* it has been stored
		// As the Id is computed based on the actual bytes written on the disk, we are going to predict
		// those and then get the Id. This is safe as it will be the exact same code writing on disk later.

		data, err := json.Marshal(i)
		if err != nil {
			panic(err)
		}

		i.id = deriveId(data)
	}
	return i.id
}

// Name return the last version of the name
func (i *Bare) Name() string {
	return i.name
}

// Email return the last version of the email
func (i *Bare) Email() string {
	return i.email
}

// AvatarUrl return the last version of the Avatar URL
func (i *Bare) AvatarUrl() string {
	return i.avatarUrl
}

// Keys return the last version of the valid keys
func (i *Bare) Keys() []*Key {
	return nil
}

// ValidKeysAtTime return the set of keys valid at a given lamport time
func (i *Bare) ValidKeysAtTime(_ lamport.Time) []*Key {
	return nil
}

// DisplayName return a non-empty string to display, representing the
// identity, based on the non-empty values.
func (i *Bare) DisplayName() string {
	return i.name
}

// Validate check if the Identity data is valid
func (i *Bare) Validate() error {
	if text.Empty(i.name) {
		return fmt.Errorf("name is not set")
	}

	if strings.Contains(i.name, "\n") {
		return fmt.Errorf("name should be a single line")
	}

	if !text.Safe(i.name) {
		return fmt.Errorf("name is not fully printable")
	}

	if strings.Contains(i.email, "\n") {
		return fmt.Errorf("email should be a single line")
	}

	if !text.Safe(i.email) {
		return fmt.Errorf("email is not fully printable")
	}

	if i.avatarUrl != "" && !text.ValidUrl(i.avatarUrl) {
		return fmt.Errorf("avatarUrl is not a valid URL")
	}

	return nil
}

// Write the identity into the Repository. In particular, this ensure that
// the Id is properly set.
func (i *Bare) Commit(repo repository.ClockedRepo) error {
	// Nothing to do, everything is directly embedded
	return nil
}

// If needed, write the identity into the Repository. In particular, this
// ensure that the Id is properly set.
func (i *Bare) CommitAsNeeded(repo repository.ClockedRepo) error {
	// Nothing to do, everything is directly embedded
	return nil
}

// IsProtected return true if the chain of git commits started to be signed.
// If that's the case, only signed commit with a valid key for this identity can be added.
func (i *Bare) IsProtected() bool {
	return false
}

// LastModificationLamportTime return the Lamport time at which the last version of the identity became valid.
func (i *Bare) LastModificationLamport() lamport.Time {
	return 0
}

// LastModification return the timestamp at which the last version of the identity became valid.
func (i *Bare) LastModification() timestamp.Timestamp {
	return 0
}
