package identity

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/lamport"
	"github.com/MichaelMure/git-bug/util/text"
)

// Version is a complete set of informations about an Identity at a point in time.
type Version struct {
	// Private field so not serialized
	commitHash git.Hash

	// The lamport time at which this version become effective
	// The reference time is the bug edition lamport clock
	Time lamport.Time `json:"time"`

	Name      string `json:"name"`
	Email     string `json:"email"`
	Login     string `json:"login"`
	AvatarUrl string `json:"avatar_url"`

	// The set of keys valid at that time, from this version onward, until they get removed
	// in a new version. This allow to have multiple key for the same identity (e.g. one per
	// device) as well as revoke key.
	Keys []Key `json:"pub_keys"`

	// This optional array is here to ensure a better randomness of the identity id to avoid collisions.
	// It has no functional purpose and should be ignored.
	// It is advised to fill this array if there is not enough entropy, e.g. if there is no keys.
	Nonce []byte `json:"nonce,omitempty"`
}

func (v *Version) Validate() error {
	if text.Empty(v.Name) && text.Empty(v.Login) {
		return fmt.Errorf("either name or login should be set")
	}

	if strings.Contains(v.Name, "\n") {
		return fmt.Errorf("name should be a single line")
	}

	if !text.Safe(v.Name) {
		return fmt.Errorf("name is not fully printable")
	}

	if strings.Contains(v.Login, "\n") {
		return fmt.Errorf("login should be a single line")
	}

	if !text.Safe(v.Login) {
		return fmt.Errorf("login is not fully printable")
	}

	if strings.Contains(v.Email, "\n") {
		return fmt.Errorf("email should be a single line")
	}

	if !text.Safe(v.Email) {
		return fmt.Errorf("email is not fully printable")
	}

	if v.AvatarUrl != "" && !text.ValidUrl(v.AvatarUrl) {
		return fmt.Errorf("avatarUrl is not a valid URL")
	}

	if len(v.Nonce) > 64 {
		return fmt.Errorf("nonce is too big")
	}

	return nil
}

// Write will serialize and store the Version as a git blob and return
// its hash
func (v *Version) Write(repo repository.Repo) (git.Hash, error) {
	data, err := json.Marshal(v)

	if err != nil {
		return "", err
	}

	hash, err := repo.StoreData(data)

	if err != nil {
		return "", err
	}

	return hash, nil
}

func makeNonce(len int) []byte {
	result := make([]byte, len)
	_, err := rand.Read(result)
	if err != nil {
		panic(err)
	}
	return result
}
