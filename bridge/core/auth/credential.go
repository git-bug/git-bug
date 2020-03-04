package auth

import (
	"crypto/rand"
	"encoding/base64"
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
	configKeyTarget     = "target"
	configKeyCreateTime = "createtime"
	configKeySalt       = "salt"
	configKeyPrefixMeta = "meta."

	MetaKeyLogin    = "login"
	MetaKeyBaseURL  = "base-url"
	MetaKeyIdentity = "identity"
)

type CredentialKind string

const (
	KindToken         CredentialKind = "token"
	KindLogin         CredentialKind = "login"
	KindLoginPassword CredentialKind = "login-password"
)

var ErrCredentialNotExist = errors.New("credential doesn't exist")

func NewErrMultipleMatchCredential(matching []entity.Id) *entity.ErrMultipleMatch {
	return entity.NewErrMultipleMatch("credential", matching)
}

type Credential interface {
	ID() entity.Id
	Kind() CredentialKind
	Target() string
	CreateTime() time.Time
	Salt() []byte
	Validate() error

	Metadata() map[string]string
	GetMetadata(key string) (string, bool)
	SetMetadata(key string, value string)

	// Return all the specific properties of the credential that need to be saved into the configuration.
	// This does not include Target, Kind, CreateTime, Metadata or Salt.
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
	var err error

	switch CredentialKind(configs[configKeyKind]) {
	case KindToken:
		cred, err = NewTokenFromConfig(configs)
	case KindLogin:
		cred, err = NewLoginFromConfig(configs)
	case KindLoginPassword:
		cred, err = NewLoginPasswordFromConfig(configs)
	default:
		return nil, fmt.Errorf("unknown credential type \"%s\"", configs[configKeyKind])
	}

	if err != nil {
		return nil, fmt.Errorf("loading credential: %v", err)
	}

	return cred, nil
}

func metaFromConfig(configs map[string]string) map[string]string {
	result := make(map[string]string)
	for key, val := range configs {
		if strings.HasPrefix(key, configKeyPrefixMeta) {
			key = strings.TrimPrefix(key, configKeyPrefixMeta)
			result[key] = val
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func makeSalt() []byte {
	result := make([]byte, 16)
	_, err := rand.Read(result)
	if err != nil {
		panic(err)
	}
	return result
}

func saltFromConfig(configs map[string]string) ([]byte, error) {
	val, ok := configs[configKeySalt]
	if !ok {
		return nil, fmt.Errorf("no credential salt found")
	}
	return base64.StdEncoding.DecodeString(val)
}

// List load all existing credentials
func List(repo repository.RepoConfig, opts ...Option) ([]Credential, error) {
	rawConfigs, err := repo.GlobalConfig().ReadAll(configKeyPrefix + ".")
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`^` + configKeyPrefix + `\.([^.]+)\.([^.]+(?:\.[^.]+)*)$`)

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

	// Salt
	if len(cred.Salt()) != 16 {
		panic("credentials need to be salted")
	}
	encoded := base64.StdEncoding.EncodeToString(cred.Salt())
	err = repo.GlobalConfig().StoreString(prefix+configKeySalt, encoded)
	if err != nil {
		return err
	}

	// Metadata
	for key, val := range cred.Metadata() {
		err := repo.GlobalConfig().StoreString(prefix+configKeyPrefixMeta+key, val)
		if err != nil {
			return err
		}
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
