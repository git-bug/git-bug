// Package identity contains the identity data model and low-level related functions
package identity

import (
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/lamport"
)

const identityRefPattern = "refs/identities/"
const versionEntryName = "version"
const identityConfigKey = "git-bug.identity"

type Identity struct {
	id       string
	Versions []Version
}

func NewIdentity(name string, email string) (*Identity, error) {
	return &Identity{
		Versions: []Version{
			{
				Name:  name,
				Email: email,
				Nonce: makeNonce(20),
			},
		},
	}, nil
}

type identityJson struct {
	Id string `json:"id"`
}

// TODO: marshal/unmarshal identity + load/write from OpBase

func Read(repo repository.Repo, id string) (*Identity, error) {
	// Todo
	return &Identity{}, nil
}

// NewFromGitUser will query the repository for user detail and
// build the corresponding Identity
/*func NewFromGitUser(repo repository.Repo) (*Identity, error) {
	name, err := repo.GetUserName()
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, errors.New("User name is not configured in git yet. Please use `git config --global user.name \"John Doe\"`")
	}

	email, err := repo.GetUserEmail()
	if err != nil {
		return nil, err
	}
	if email == "" {
		return nil, errors.New("User name is not configured in git yet. Please use `git config --global user.email johndoe@example.com`")
	}

	return NewIdentity(name, email)
}*/

//
func BuildFromGit(repo repository.Repo) *Identity {
	version := Version{}

	name, err := repo.GetUserName()
	if err == nil {
		version.Name = name
	}

	email, err := repo.GetUserEmail()
	if err == nil {
		version.Email = email
	}

	return &Identity{
		Versions: []Version{
			version,
		},
	}
}

// SetIdentity store the user identity's id in the git config
func SetIdentity(repo repository.RepoCommon, identity Identity) error {
	return repo.StoreConfig(identityConfigKey, identity.Id())
}

// GetIdentity read the current user identity, set with a git config entry
func GetIdentity(repo repository.Repo) (*Identity, error) {
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

	return Read(repo, id)
}

func (i *Identity) AddVersion(version Version) {
	i.Versions = append(i.Versions, version)
}

func (i *Identity) Commit(repo repository.ClockedRepo) error {
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

func (i *Identity) LastVersion() Version {
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
	return i.LastVersion().Name
}

// Email return the last version of the email
func (i *Identity) Email() string {
	return i.LastVersion().Email
}

// Login return the last version of the login
func (i *Identity) Login() string {
	return i.LastVersion().Login
}

// Login return the last version of the Avatar URL
func (i *Identity) AvatarUrl() string {
	return i.LastVersion().AvatarUrl
}

// Login return the last version of the valid keys
func (i *Identity) Keys() []Key {
	return i.LastVersion().Keys
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

// Match tell is the Identity match the given query string
func (i *Identity) Match(query string) bool {
	query = strings.ToLower(query)

	return strings.Contains(strings.ToLower(i.Name()), query) ||
		strings.Contains(strings.ToLower(i.Login()), query)
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
