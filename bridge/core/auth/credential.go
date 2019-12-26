package auth

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

const (
	configKeyPrefix     = "git-bug.auth"
	configKeyKind       = "kind"
	configKeyUserId     = "userid"
	configKeyTarget     = "target"
	configKeyCreateTime = "createtime"
)

type CredentialKind string

const (
	KindToken         CredentialKind = "token"
	KindLoginPassword CredentialKind = "login-password"
)

var ErrCredentialNotExist = errors.New("credential doesn't exist")

func NewErrMultipleMatchCredential(matching []entity.Id) *entity.ErrMultipleMatch {
	return entity.NewErrMultipleMatch("credential", matching)
}

// Special Id to mark a credential as being associated to the default user, whoever it might be.
// The intended use is for the bridge configuration, to be able to create and store a credential
// with no identities created yet, and then select one with `git-bug user adopt`
const DefaultUserId = entity.Id("default-user")

type Credential interface {
	ID() entity.Id
	UserId() entity.Id
	updateUserId(id entity.Id)
	Target() string
	Kind() CredentialKind
	CreateTime() time.Time
	Validate() error

	// Return all the specific properties of the credential that need to be saved into the configuration.
	// This does not include Target, User, Kind and CreateTime.
	toConfig() map[string]string
}

// Load loads a credential from the repo config
func LoadWithId(repo repository.RepoConfig, id entity.Id) (Credential, error) {
	keyPrefix := fmt.Sprintf("%s.%s.", configKeyPrefix, id)

	// read token config pairs
	rawconfigs, err := repo.GlobalConfig().ReadAll(keyPrefix)
	if err != nil {
		// Not exactly right due to the limitation of ReadAll()
		return nil, ErrCredentialNotExist
	}

	return loadFromConfig(rawconfigs, id)
}

// LoadWithPrefix load a credential from the repo config with a prefix
func LoadWithPrefix(repo repository.RepoConfig, prefix string) (Credential, error) {
	creds, err := List(repo)
	if err != nil {
		return nil, err
	}

	// preallocate but empty
	matching := make([]Credential, 0, 5)

	for _, cred := range creds {
		if cred.ID().HasPrefix(prefix) {
			matching = append(matching, cred)
		}
	}

	if len(matching) > 1 {
		ids := make([]entity.Id, len(matching))
		for i, cred := range matching {
			ids[i] = cred.ID()
		}
		return nil, NewErrMultipleMatchCredential(ids)
	}

	if len(matching) == 0 {
		return nil, ErrCredentialNotExist
	}

	return matching[0], nil
}

// loadFromConfig is a helper to construct a Credential from the set of git configs
func loadFromConfig(rawConfigs map[string]string, id entity.Id) (Credential, error) {
	keyPrefix := fmt.Sprintf("%s.%s.", configKeyPrefix, id)

	// trim key prefix
	configs := make(map[string]string)
	for key, value := range rawConfigs {
		newKey := strings.TrimPrefix(key, keyPrefix)
		configs[newKey] = value
	}

	var cred Credential

	switch CredentialKind(configs[configKeyKind]) {
	case KindToken:
		cred = NewTokenFromConfig(configs)
	case KindLoginPassword:
	default:
		return nil, fmt.Errorf("unknown credential type %s", configs[configKeyKind])
	}

	return cred, nil
}

// List load all existing credentials
func List(repo repository.RepoConfig, opts ...Option) ([]Credential, error) {
	rawConfigs, err := repo.GlobalConfig().ReadAll(configKeyPrefix + ".")
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(configKeyPrefix + `.([^.]+).([^.]+)`)
	if err != nil {
		panic(err)
	}

	mapped := make(map[string]map[string]string)

	for key, val := range rawConfigs {
		res := re.FindStringSubmatch(key)
		if res == nil {
			continue
		}
		if mapped[res[1]] == nil {
			mapped[res[1]] = make(map[string]string)
		}
		mapped[res[1]][res[2]] = val
	}

	matcher := matcher(opts)

	var credentials []Credential
	for id, kvs := range mapped {
		cred, err := loadFromConfig(kvs, entity.Id(id))
		if err != nil {
			return nil, err
		}
		if matcher.Match(cred) {
			credentials = append(credentials, cred)
		}
	}

	return credentials, nil
}

// IdExist return whether a credential id exist or not
func IdExist(repo repository.RepoConfig, id entity.Id) bool {
	_, err := LoadWithId(repo, id)
	return err == nil
}

// PrefixExist return whether a credential id prefix exist or not
func PrefixExist(repo repository.RepoConfig, prefix string) bool {
	_, err := LoadWithPrefix(repo, prefix)
	return err == nil
}

// Store stores a credential in the global git config
func Store(repo repository.RepoConfig, cred Credential) error {
	confs := cred.toConfig()

	prefix := fmt.Sprintf("%s.%s.", configKeyPrefix, cred.ID())

	// Kind
	err := repo.GlobalConfig().StoreString(prefix+configKeyKind, string(cred.Kind()))
	if err != nil {
		return err
	}

	// UserId
	err = repo.GlobalConfig().StoreString(prefix+configKeyUserId, cred.UserId().String())
	if err != nil {
		return err
	}

	// Target
	err = repo.GlobalConfig().StoreString(prefix+configKeyTarget, cred.Target())
	if err != nil {
		return err
	}

	// CreateTime
	err = repo.GlobalConfig().StoreTimestamp(prefix+configKeyCreateTime, cred.CreateTime())
	if err != nil {
		return err
	}

	// Custom
	for key, val := range confs {
		err := repo.GlobalConfig().StoreString(prefix+key, val)
		if err != nil {
			return err
		}
	}

	return nil
}

// Remove removes a credential from the global git config
func Remove(repo repository.RepoConfig, id entity.Id) error {
	keyPrefix := fmt.Sprintf("%s.%s", configKeyPrefix, id)
	return repo.GlobalConfig().RemoveAll(keyPrefix)
}

// ReplaceDefaultUser update all the credential attributed to the temporary "default user"
// with a real user Id
func ReplaceDefaultUser(repo repository.RepoConfig, id entity.Id) error {
	list, err := List(repo, WithUserId(DefaultUserId))
	if err != nil {
		return err
	}

	for _, cred := range list {
		cred.updateUserId(id)
		err = Store(repo, cred)
		if err != nil {
			return err
		}
	}

	return nil
}

/*
 * Sorting
 */

type ById []Credential

func (b ById) Len() int {
	return len(b)
}

func (b ById) Less(i, j int) bool {
	return b[i].ID() < b[j].ID()
}

func (b ById) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
