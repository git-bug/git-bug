// Package identity contains the identity data model and low-level related functions
package identity

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/lamport"
)

const identityRefPattern = "refs/identities/"
const identityRemoteRefPattern = "refs/remotes/%s/identities/"
const versionEntryName = "version"
const identityConfigKey = "git-bug.identity"

var _ Interface = &Identity{}

type Identity struct {
	id       string
	Versions []*Version
}

func NewIdentity(name string, email string) *Identity {
	return &Identity{
		Versions: []*Version{
			{
				Name:  name,
				Email: email,
				Nonce: makeNonce(20),
			},
		},
	}
}

func NewIdentityFull(name string, email string, login string, avatarUrl string) *Identity {
	return &Identity{
		Versions: []*Version{
			{
				Name:      name,
				Email:     email,
				Login:     login,
				AvatarUrl: avatarUrl,
				Nonce:     makeNonce(20),
			},
		},
	}
}

// MarshalJSON will only serialize the id
func (i *Identity) MarshalJSON() ([]byte, error) {
	return json.Marshal(&IdentityStub{
		id: i.Id(),
	})
}

// UnmarshalJSON will only read the id
// Users of this package are expected to run Load() to load
// the remaining data from the identities data in git.
func (i *Identity) UnmarshalJSON(data []byte) error {
	panic("identity should be loaded with identity.UnmarshalJSON")
}

// ReadLocal load a local Identity from the identities data available in git
func ReadLocal(repo repository.Repo, id string) (*Identity, error) {
	ref := fmt.Sprintf("%s%s", identityRefPattern, id)
	return read(repo, ref)
}

// ReadRemote load a remote Identity from the identities data available in git
func ReadRemote(repo repository.Repo, remote string, id string) (*Identity, error) {
	ref := fmt.Sprintf(identityRemoteRefPattern, remote) + id
	return read(repo, ref)
}

// read will load and parse an identity frdm git
func read(repo repository.Repo, ref string) (*Identity, error) {
	refSplit := strings.Split(ref, "/")
	id := refSplit[len(refSplit)-1]

	hashes, err := repo.ListCommits(ref)

	var versions []*Version

	// TODO: this is not perfect, it might be a command invoke error
	if err != nil {
		return nil, ErrIdentityNotExist
	}

	for _, hash := range hashes {
		entries, err := repo.ListEntries(hash)
		if err != nil {
			return nil, errors.Wrap(err, "can't list git tree entries")
		}

		if len(entries) != 1 {
			return nil, fmt.Errorf("invalid identity data at hash %s", hash)
		}

		entry := entries[0]

		if entry.Name != versionEntryName {
			return nil, fmt.Errorf("invalid identity data at hash %s", hash)
		}

		data, err := repo.ReadData(entry.Hash)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read git blob data")
		}

		var version Version
		err = json.Unmarshal(data, &version)

		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode Identity version json %s", hash)
		}

		// tag the version with the commit hash
		version.commitHash = hash

		versions = append(versions, &version)
	}

	return &Identity{
		id:       id,
		Versions: versions,
	}, nil
}

// NewFromGitUser will query the repository for user detail and
// build the corresponding Identity
func NewFromGitUser(repo repository.Repo) (*Identity, error) {
	name, err := repo.GetUserName()
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, errors.New("user name is not configured in git yet. Please use `git config --global user.name \"John Doe\"`")
	}

	email, err := repo.GetUserEmail()
	if err != nil {
		return nil, err
	}
	if email == "" {
		return nil, errors.New("user name is not configured in git yet. Please use `git config --global user.email johndoe@example.com`")
	}

	return NewIdentity(name, email), nil
}

// SetUserIdentity store the user identity's id in the git config
func SetUserIdentity(repo repository.RepoCommon, identity Identity) error {
	return repo.StoreConfig(identityConfigKey, identity.Id())
}

// GetUserIdentity read the current user identity, set with a git config entry
func GetUserIdentity(repo repository.Repo) (*Identity, error) {
	configs, err := repo.ReadConfigs(identityConfigKey)
	if err != nil {
		return nil, err
	}

	if len(configs) == 0 {
		return nil, fmt.Errorf("no identity set")
	}

	if len(configs) > 1 {
		return nil, fmt.Errorf("multiple identity config exist")
	}

	var id string
	for _, val := range configs {
		id = val
	}

	return ReadLocal(repo, id)
}

func (i *Identity) AddVersion(version *Version) {
	i.Versions = append(i.Versions, version)
}

// Write the identity into the Repository. In particular, this ensure that
// the Id is properly set.
func (i *Identity) Commit(repo repository.Repo) error {
	// Todo: check for mismatch between memory and commited data

	var lastCommit git.Hash = ""

	for _, v := range i.Versions {
		if v.commitHash != "" {
			lastCommit = v.commitHash
			// ignore already commited versions
			continue
		}

		blobHash, err := v.Write(repo)
		if err != nil {
			return err
		}

		// Make a git tree referencing the blob
		tree := []repository.TreeEntry{
			{ObjectType: repository.Blob, Hash: blobHash, Name: versionEntryName},
		}

		treeHash, err := repo.StoreTree(tree)
		if err != nil {
			return err
		}

		var commitHash git.Hash
		if lastCommit != "" {
			commitHash, err = repo.StoreCommitWithParent(treeHash, lastCommit)
		} else {
			commitHash, err = repo.StoreCommit(treeHash)
		}

		if err != nil {
			return err
		}

		lastCommit = commitHash

		// if it was the first commit, use the commit hash as the Identity id
		if i.id == "" {
			i.id = string(commitHash)
		}
	}

	if i.id == "" {
		panic("identity with no id")
	}

	ref := fmt.Sprintf("%s%s", identityRefPattern, i.id)
	err := repo.UpdateRef(ref, lastCommit)

	if err != nil {
		return err
	}

	return nil
}

// Validate check if the Identity data is valid
func (i *Identity) Validate() error {
	lastTime := lamport.Time(0)

	for _, v := range i.Versions {
		if err := v.Validate(); err != nil {
			return err
		}

		if v.Time < lastTime {
			return fmt.Errorf("non-chronological version (%d --> %d)", lastTime, v.Time)
		}

		lastTime = v.Time
	}

	return nil
}

func (i *Identity) lastVersion() *Version {
	if len(i.Versions) <= 0 {
		panic("no version at all")
	}

	return i.Versions[len(i.Versions)-1]
}

// Id return the Identity identifier
func (i *Identity) Id() string {
	if i.id == "" {
		// simply panic as it would be a coding error
		// (using an id of an identity not stored yet)
		panic("no id yet")
	}
	return i.id
}

// Name return the last version of the name
func (i *Identity) Name() string {
	return i.lastVersion().Name
}

// Email return the last version of the email
func (i *Identity) Email() string {
	return i.lastVersion().Email
}

// Login return the last version of the login
func (i *Identity) Login() string {
	return i.lastVersion().Login
}

// Login return the last version of the Avatar URL
func (i *Identity) AvatarUrl() string {
	return i.lastVersion().AvatarUrl
}

// Login return the last version of the valid keys
func (i *Identity) Keys() []Key {
	return i.lastVersion().Keys
}

// IsProtected return true if the chain of git commits started to be signed.
// If that's the case, only signed commit with a valid key for this identity can be added.
func (i *Identity) IsProtected() bool {
	// Todo
	return false
}

// ValidKeysAtTime return the set of keys valid at a given lamport time
func (i *Identity) ValidKeysAtTime(time lamport.Time) []Key {
	var result []Key

	for _, v := range i.Versions {
		if v.Time > time {
			return result
		}

		result = v.Keys
	}

	return result
}

// DisplayName return a non-empty string to display, representing the
// identity, based on the non-empty values.
func (i *Identity) DisplayName() string {
	switch {
	case i.Name() == "" && i.Login() != "":
		return i.Login()
	case i.Name() != "" && i.Login() == "":
		return i.Name()
	case i.Name() != "" && i.Login() != "":
		return fmt.Sprintf("%s (%s)", i.Name(), i.Login())
	}

	panic("invalid person data")
}

// SetMetadata store arbitrary metadata along the last defined Version.
// If the Version has been commit to git already, it won't be overwritten.
func (i *Identity) SetMetadata(key string, value string) {
	i.lastVersion().SetMetadata(key, value)
}

// ImmutableMetadata return all metadata for this Identity, accumulated from each Version.
// If multiple value are found, the first defined takes precedence.
func (i *Identity) ImmutableMetadata() map[string]string {
	metadata := make(map[string]string)

	for _, version := range i.Versions {
		for key, value := range version.Metadata {
			if _, has := metadata[key]; !has {
				metadata[key] = value
			}
		}
	}

	return metadata
}

// MutableMetadata return all metadata for this Identity, accumulated from each Version.
// If multiple value are found, the last defined takes precedence.
func (i *Identity) MutableMetadata() map[string]string {
	metadata := make(map[string]string)

	for _, version := range i.Versions {
		for key, value := range version.Metadata {
			metadata[key] = value
		}
	}

	return metadata
}
