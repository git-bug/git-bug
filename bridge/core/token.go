package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/MichaelMure/git-bug/repository"
)

const (
	tokenConfigKeyPrefix = "git-bug.token"
	tokenKeyTarget       = "target"
	tokenKeyScopes       = "scopes"
)

// Token holds an API access token data
type Token struct {
	Value  string
	Target string
	Global bool
	Scopes []string
}

// NewToken instantiate a new token
func NewToken(value, target string, global bool, scopes []string) *Token {
	return &Token{
		Value:  value,
		Target: target,
		Global: global,
		Scopes: scopes,
	}
}

// Validate ensure token important fields are valid
func (t *Token) Validate() error {
	if t.Value == "" {
		return fmt.Errorf("missing value")
	}
	if t.Target == "" {
		return fmt.Errorf("missing target")
	}
	if _, ok := bridgeImpl[t.Target]; !ok {
		return fmt.Errorf("unknown target")
	}
	return nil
}

func loadToken(repo repository.RepoConfig, value string, global bool) (*Token, error) {
	keyPrefix := fmt.Sprintf("git-bug.token.%s.", value)

	readerFn := repo.ReadConfigs
	if global {
		readerFn = repo.ReadGlobalConfigs
	}

	// read token config pairs
	configs, err := readerFn(keyPrefix)
	if err != nil {
		return nil, err
	}

	// trim key prefix
	for key, value := range configs {
		newKey := strings.TrimPrefix(key, keyPrefix)
		configs[newKey] = value
		delete(configs, key)
	}

	var ok bool
	token := &Token{Value: value, Global: global}
	token.Target, ok = configs[tokenKeyTarget]
	if !ok {
		return nil, fmt.Errorf("empty token key")
	}

	scopesString, ok := configs[tokenKeyScopes]
	if !ok {
		return nil, fmt.Errorf("missing scopes config")
	}

	token.Scopes = strings.Split(scopesString, ",")
	return token, nil
}

// GetToken loads a token from repo config
func GetToken(repo repository.RepoConfig, value string) (*Token, error) {
	return loadToken(repo, value, false)
}

// GetGlobalToken loads a token from the global config
func GetGlobalToken(repo repository.RepoConfig, value string) (*Token, error) {
	return loadToken(repo, value, true)
}

func listTokens(repo repository.RepoConfig, global bool) ([]string, error) {
	readerFn := repo.ReadConfigs
	if global {
		readerFn = repo.ReadGlobalConfigs
	}

	configs, err := readerFn(tokenConfigKeyPrefix + ".")
	if err != nil {
		return nil, err
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
	storeFn := repo.StoreConfig
	if token.Global {
		storeFn = repo.StoreGlobalConfig
	}

	storeTargetKey := fmt.Sprintf("git-bug.token.%s.%s", token.Value, tokenKeyTarget)
	err := storeFn(storeTargetKey, token.Target)
	if err != nil {
		return err
	}

	storeScopesKey := fmt.Sprintf("git-bug.token.%s.%s", token.Value, tokenKeyScopes)
	return storeFn(storeScopesKey, strings.Join(token.Scopes, ","))
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
func RemoveToken(repo repository.RepoConfig, value string) error {
	keyPrefix := fmt.Sprintf("git-bug.token.%s", value)
	return repo.RmConfigs(keyPrefix)
}

// RemoveGlobalToken removes a token from the repo config
func RemoveGlobalToken(repo repository.RepoConfig, value string) error {
	keyPrefix := fmt.Sprintf("git-bug.token.%s", value)
	return repo.RmGlobalConfigs(keyPrefix)
}
