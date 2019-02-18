package identity

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
	"github.com/MichaelMure/git-bug/util/text"
)

var _ Interface = &Bare{}

// Bare is a very minimal identity, designed to be fully embedded directly along
// other data.
//
// in particular, this identity is designed to be compatible with the handling of
// identities in the early version of git-bug.
type Bare struct {
	id        string
	name      string
	email     string
	login     string
	avatarUrl string
}

func NewBare(name string, email string) *Bare {
	return &Bare{name: name, email: email}
}

func NewBareFull(name string, email string, login string, avatarUrl string) *Bare {
	return &Bare{name: name, email: email, login: login, avatarUrl: avatarUrl}
}

type bareIdentityJSON struct {
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	Login     string `json:"login,omitempty"`
	AvatarUrl string `json:"avatar_url,omitempty"`
}

func (i *Bare) MarshalJSON() ([]byte, error) {
	return json.Marshal(bareIdentityJSON{
		Name:      i.name,
		Email:     i.email,
		Login:     i.login,
		AvatarUrl: i.avatarUrl,
	})
}

func (i *Bare) UnmarshalJSON(data []byte) error {
	aux := bareIdentityJSON{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	i.name = aux.Name
	i.email = aux.Email
	i.login = aux.Login
	i.avatarUrl = aux.AvatarUrl

	return nil
}

// Id return the Identity identifier
func (i *Bare) Id() string {
	// We don't have a proper ID at hand, so let's hash all the data to get one.
	// Hopefully the

	if i.id != "" {
		return i.id
	}

	data, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}

	h := fmt.Sprintf("%x", sha256.New().Sum(data)[:16])
	i.id = string(h)

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

// Login return the last version of the login
func (i *Bare) Login() string {
	return i.login
}

// AvatarUrl return the last version of the Avatar URL
func (i *Bare) AvatarUrl() string {
	return i.avatarUrl
}

// Keys return the last version of the valid keys
func (i *Bare) Keys() []Key {
	return []Key{}
}

// ValidKeysAtTime return the set of keys valid at a given lamport time
func (i *Bare) ValidKeysAtTime(time lamport.Time) []Key {
	return []Key{}
}

// DisplayName return a non-empty string to display, representing the
// identity, based on the non-empty values.
func (i *Bare) DisplayName() string {
	switch {
	case i.name == "" && i.login != "":
		return i.login
	case i.name != "" && i.login == "":
		return i.name
	case i.name != "" && i.login != "":
		return fmt.Sprintf("%s (%s)", i.name, i.login)
	}

	panic("invalid person data")
}

// Validate check if the Identity data is valid
func (i *Bare) Validate() error {
	if text.Empty(i.name) && text.Empty(i.login) {
		return fmt.Errorf("either name or login should be set")
	}

	if strings.Contains(i.name, "\n") {
		return fmt.Errorf("name should be a single line")
	}

	if !text.Safe(i.name) {
		return fmt.Errorf("name is not fully printable")
	}

	if strings.Contains(i.login, "\n") {
		return fmt.Errorf("login should be a single line")
	}

	if !text.Safe(i.login) {
		return fmt.Errorf("login is not fully printable")
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
