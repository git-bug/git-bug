package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/MichaelMure/git-bug/repository"
)

const (
	tokenConfigKeyPrefix = "git-bug.token"
	tokenKeyValue        = "value"
	tokenKeyTarget       = "target"
	tokenKeyScopes       = "scopes"
)

// Token represent token related informations
type Token struct {
	Id     string
	Value  string
	Target string
	Global bool
	Scopes []string
}

// NewToken instantiate a new token
func NewToken(id, value, target string, global bool, scopes []string) *Token {
	return &Token{
		Id:     id,
		Value:  value,
		Target: target,
		Global: global,
		Scopes: scopes,
	}
}

// Validate ensure token important fields are valid
func (t *Token) Validate() error {
	if t.Id == "" {
		return fmt.Errorf("missing token id")
	}
	if t.Value == "" {
		return fmt.Errorf("missing token value")
	}
	if t.Target == "" {
		return fmt.Errorf("missing token target")
	}
	return nil
}

func loadToken(repo repository.RepoConfig, id string, global bool) (*Token, error) {
	keyPrefix := fmt.Sprintf("git-bug.token.%s.", id)
	var pairs map[string]string
	var err error

	// read token config pairs
	if global {
		pairs, err = repo.ReadGlobalConfigs(keyPrefix)
		if err != nil {
			return nil, err
		}
	} else {
		pairs, err = repo.ReadConfigs(keyPrefix)
		if err != nil {
			return nil, err
		}
	}

	// trim key prefix
	result := make(Configuration, len(pairs))
	for key, value := range pairs {
		key := strings.TrimPrefix(key, keyPrefix)
		result[key] = value
	}

	var ok bool
	token := &Token{Id: id, Global: global}
	token.Value, ok = result[tokenKeyValue]
	if !ok {
		return nil, fmt.Errorf("empty token value")
	}

	token.Target, ok = result[tokenKeyTarget]
	if !ok {
		return nil, fmt.Errorf("empty token key")
	}

	scopesString, ok := result[tokenKeyScopes]
	if !ok {
		return nil, fmt.Errorf("missing scopes config")
	}

	token.Scopes = strings.Split(scopesString, ",")
	return token, nil
}

// GetToken loads a token from repo config
func GetToken(repo repository.RepoConfig, id string) (*Token, error) {
	return loadToken(repo, id, false)
}

// GetGlobalToken loads a token from the global config
func GetGlobalToken(repo repository.RepoConfig, id string) (*Token, error) {
	return loadToken(repo, id, true)
}

func listTokens(repo repository.RepoConfig, global bool) ([]string, error) {
	var configs map[string]string
	var err error
	if global {
		configs, err = repo.ReadGlobalConfigs(tokenConfigKeyPrefix + ".")
		if err != nil {
			return nil, err
		}
	} else {
		configs, err = repo.ReadConfigs(tokenConfigKeyPrefix + ".")
		if err != nil {
			return nil, err
		}
	}

	re, err := regexp.Compile(tokenConfigKeyPrefix + `.([^.]+)`)
	if err != nil {
		panic(err)
	}

	set := make(map[string]interface{})

	for key := range configs {
		res := re.FindStringSubmatch(key)

		if res == nil {
			continue
		}

		set[res[1]] = nil
	}

	result := make([]string, len(set))
	i := 0
	for key := range set {
		result[i] = key
		i++
	}

	return result, nil
}

// ListTokens return the list of stored tokens in the repo config
func ListTokens(repo repository.RepoConfig) ([]string, error) {
	return listTokens(repo, false)
}

// ListGlobalTokens return the list of stored tokens in the global config
func ListGlobalTokens(repo repository.RepoConfig) ([]string, error) {
	return listTokens(repo, true)
}

func storeToken(repo repository.RepoConfig, token *Token) error {
	var store func(key, value string) error
	if token.Global {
		store = repo.StoreGlobalConfig
	} else {
		store = repo.StoreConfig
	}

	var err error
	storeValueKey := fmt.Sprintf("git-bug.token.%s.%s", token.Id, tokenKeyValue)
	err = store(storeValueKey, token.Value)
	if err != nil {
		return err
	}

	storeTargetKey := fmt.Sprintf("git-bug.token.%s.%s", token.Id, tokenKeyTarget)
	err = store(storeTargetKey, token.Target)
	if err != nil {
		return err
	}

	storeScopesKey := fmt.Sprintf("git-bug.token.%s.%s", token.Id, tokenKeyScopes)
	err = store(storeScopesKey, strings.Join(token.Scopes, ","))
	if err != nil {
		return err
	}

	return nil
}

// StoreToken stores a token in the repo config
func StoreToken(repo repository.RepoConfig, token *Token) error {
	return storeToken(repo, token)
}

// StoreGlobalToken stores a token in global config
func StoreGlobalToken(repo repository.RepoConfig, token *Token) error {
	return storeToken(repo, token)
}

// RemoveToken removes a token from the repo config
func RemoveToken(repo repository.RepoConfig, id string) error {
	keyPrefix := fmt.Sprintf("git-bug.token.%s", id)
	return repo.RmConfigs(keyPrefix)
}

// RemoveGlobalToken removes a token from the repo config
func RemoveGlobalToken(repo repository.RepoConfig, id string) error {
	keyPrefix := fmt.Sprintf("git-bug.token.%s", id)
	return repo.RmGlobalConfigs(keyPrefix)
}
