package identity

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
	"github.com/MichaelMure/git-bug/util/text"
)

// 1: original format
const formatVersion = 1

// Version is a complete set of information about an Identity at a point in time.
type Version struct {
	// The lamport time at which this version become effective
	// The reference time is the bug edition lamport clock
	// It must be the first field in this struct due to https://github.com/golang/go/issues/599
	//
	// TODO: BREAKING CHANGE - this need to actually be one edition lamport time **per entity**
	// This is not a problem right now but will be when more entities are added (pull-request, config ...)
	time     lamport.Time
	unixTime int64

	name      string
	email     string // as defined in git or from a bridge when importing the identity
	login     string // from a bridge when importing the identity
	avatarURL string

	// The set of keys valid at that time, from this version onward, until they get removed
	// in a new version. This allow to have multiple key for the same identity (e.g. one per
	// device) as well as revoke key.
	keys []*Key

	// This optional array is here to ensure a better randomness of the identity id to avoid collisions.
	// It has no functional purpose and should be ignored.
	// It is advised to fill this array if there is not enough entropy, e.g. if there is no keys.
	nonce []byte

	// A set of arbitrary key/value to store metadata about a version or about an Identity in general.
	metadata map[string]string

	// Not serialized
	commitHash repository.Hash
}

type VersionJSON struct {
	// Additional field to version the data
	FormatVersion uint `json:"version"`

	Time      lamport.Time      `json:"time"`
	UnixTime  int64             `json:"unix_time"`
	Name      string            `json:"name,omitempty"`
	Email     string            `json:"email,omitempty"`
	Login     string            `json:"login,omitempty"`
	AvatarUrl string            `json:"avatar_url,omitempty"`
	Keys      []*Key            `json:"pub_keys,omitempty"`
	Nonce     []byte            `json:"nonce,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Make a deep copy
func (v *Version) Clone() *Version {
	clone := &Version{
		name:      v.name,
		email:     v.email,
		avatarURL: v.avatarURL,
		keys:      make([]*Key, len(v.keys)),
	}

	for i, key := range v.keys {
		clone.keys[i] = key.Clone()
	}

	return clone
}

func (v *Version) MarshalJSON() ([]byte, error) {
	return json.Marshal(VersionJSON{
		FormatVersion: formatVersion,
		Time:          v.time,
		UnixTime:      v.unixTime,
		Name:          v.name,
		Email:         v.email,
		Login:         v.login,
		AvatarUrl:     v.avatarURL,
		Keys:          v.keys,
		Nonce:         v.nonce,
		Metadata:      v.metadata,
	})
}

func (v *Version) UnmarshalJSON(data []byte) error {
	var aux VersionJSON

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.FormatVersion < formatVersion {
		return entity.NewErrOldFormatVersion(aux.FormatVersion)
	}
	if aux.FormatVersion > formatVersion {
		return entity.NewErrNewFormatVersion(aux.FormatVersion)
	}

	v.time = aux.Time
	v.unixTime = aux.UnixTime
	v.name = aux.Name
	v.email = aux.Email
	v.login = aux.Login
	v.avatarURL = aux.AvatarUrl
	v.keys = aux.Keys
	v.nonce = aux.Nonce
	v.metadata = aux.Metadata

	return nil
}

func (v *Version) Validate() error {
	// time must be set after a commit
	if v.commitHash != "" && v.unixTime == 0 {
		return fmt.Errorf("unix time not set")
	}
	if v.commitHash != "" && v.time == 0 {
		return fmt.Errorf("lamport time not set")
	}

	if text.Empty(v.name) && text.Empty(v.login) {
		return fmt.Errorf("either name or login should be set")
	}

	if strings.Contains(v.name, "\n") {
		return fmt.Errorf("name should be a single line")
	}

	if !text.Safe(v.name) {
		return fmt.Errorf("name is not fully printable")
	}

	if strings.Contains(v.login, "\n") {
		return fmt.Errorf("login should be a single line")
	}

	if !text.Safe(v.login) {
		return fmt.Errorf("login is not fully printable")
	}

	if strings.Contains(v.email, "\n") {
		return fmt.Errorf("email should be a single line")
	}

	if !text.Safe(v.email) {
		return fmt.Errorf("email is not fully printable")
	}

	if v.avatarURL != "" && !text.ValidUrl(v.avatarURL) {
		return fmt.Errorf("avatarUrl is not a valid URL")
	}

	if len(v.nonce) > 64 {
		return fmt.Errorf("nonce is too big")
	}

	for _, k := range v.keys {
		if err := k.Validate(); err != nil {
			return errors.Wrap(err, "invalid key")
		}
	}

	return nil
}

// Write will serialize and store the Version as a git blob and return
// its hash
func (v *Version) Write(repo repository.Repo) (repository.Hash, error) {
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
	if v.metadata == nil {
		v.metadata = make(map[string]string)
	}

	v.metadata[key] = value
}

// GetMetadata retrieve arbitrary metadata about the Version
func (v *Version) GetMetadata(key string) (string, bool) {
	val, ok := v.metadata[key]
	return val, ok
}

// AllMetadata return all metadata for this Version
func (v *Version) AllMetadata() map[string]string {
	return v.metadata
}
