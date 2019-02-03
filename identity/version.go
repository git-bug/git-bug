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
	"github.com/pkg/errors"
)

const formatVersion = 1

// Version is a complete set of information about an Identity at a point in time.
type Version struct {
	// Not serialized
	commitHash git.Hash

	// Todo: add unix timestamp for ordering with identical lamport time ?

	// The lamport time at which this version become effective
	// The reference time is the bug edition lamport clock
	Time lamport.Time

	Name      string
	Email     string
	Login     string
	AvatarUrl string

	// The set of keys valid at that time, from this version onward, until they get removed
	// in a new version. This allow to have multiple key for the same identity (e.g. one per
	// device) as well as revoke key.
	Keys []Key

	// This optional array is here to ensure a better randomness of the identity id to avoid collisions.
	// It has no functional purpose and should be ignored.
	// It is advised to fill this array if there is not enough entropy, e.g. if there is no keys.
	Nonce []byte

	// A set of arbitrary key/value to store metadata about a version or about an Identity in general.
	Metadata map[string]string
}

type VersionJSON struct {
	// Additional field to version the data
	FormatVersion uint `json:"version"`

	Time      lamport.Time      `json:"time"`
	Name      string            `json:"name"`
	Email     string            `json:"email"`
	Login     string            `json:"login"`
	AvatarUrl string            `json:"avatar_url"`
	Keys      []Key             `json:"pub_keys"`
	Nonce     []byte            `json:"nonce,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

func (v *Version) MarshalJSON() ([]byte, error) {
	return json.Marshal(VersionJSON{
		FormatVersion: formatVersion,
		Time:          v.Time,
		Name:          v.Name,
		Email:         v.Email,
		Login:         v.Login,
		AvatarUrl:     v.AvatarUrl,
		Keys:          v.Keys,
		Nonce:         v.Nonce,
		Metadata:      v.Metadata,
	})
}

func (v *Version) UnmarshalJSON(data []byte) error {
	var aux VersionJSON

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.FormatVersion != formatVersion {
		return fmt.Errorf("unknown format version %v", aux.FormatVersion)
	}

	v.Time = aux.Time
	v.Name = aux.Name
	v.Email = aux.Email
	v.Login = aux.Login
	v.AvatarUrl = aux.AvatarUrl
	v.Keys = aux.Keys
	v.Nonce = aux.Nonce
	v.Metadata = aux.Metadata

	return nil
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

	for _, k := range v.Keys {
		if err := k.Validate(); err != nil {
			return errors.Wrap(err, "invalid key")
		}
	}

	return nil
}

// Write will serialize and store the Version as a git blob and return
// its hash
func (v *Version) Write(repo repository.Repo) (git.Hash, error) {
	// make sure we don't write invalid data
	err := v.Validate()
	if err != nil {
		return "", errors.Wrap(err, "validation error")
	}

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

// SetMetadata store arbitrary metadata about a version or an Identity in general
// If the Version has been commit to git already, it won't be overwritten.
func (v *Version) SetMetadata(key string, value string) {
	if v.Metadata == nil {
		v.Metadata = make(map[string]string)
	}

	v.Metadata[key] = value
}

// GetMetadata retrieve arbitrary metadata about the Version
func (v *Version) GetMetadata(key string) (string, bool) {
	val, ok := v.Metadata[key]
	return val, ok
}

// AllMetadata return all metadata for this Identity
func (v *Version) AllMetadata() map[string]string {
	return v.Metadata
}
