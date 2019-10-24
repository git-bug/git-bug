package core

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/MichaelMure/git-bug/repository"
)

const (
	tokenConfigKeyPrefix = "git-bug.token"
	tokenValueKey        = "value"
	tokenTargetKey       = "target"
	tokenScopesKey       = "scopes"
)

// Token holds an API access token data
type Token struct {
	ID     string
	Value  string
	Target string
	Global bool
	Scopes []string
}

// NewToken instantiate a new token
func NewToken(value, target string, global bool, scopes []string) *Token {
	token := &Token{
		Value:  value,
		Target: target,
		Global: global,
		Scopes: scopes,
	}

	token.ID = hashToken(token)
	return token
}

// Id return full token identifier. It will compute the Id if it's empty
func (t *Token) Id() string {
	if t.ID == "" {
		t.ID = hashToken(t)
	}

	return t.ID
}

// HumanId return the truncated token id
func (t *Token) HumanId() string {
	return t.Id()[:6]
}

func hashToken(token *Token) string {
	tokenJson, err := json.Marshal(&token)
	if err != nil {
		panic(err)
	}

	sum := sha256.Sum256(tokenJson)
	return fmt.Sprintf("%x", sum)
}

// Validate ensure token important fields are valid
func (t *Token) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("missing id")
	}
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

// Kind return the type of the token as string
func (t *Token) Kind() string {
	if t.Global {
		return "global"
	}

	return "local"
}

func loadToken(repo repository.RepoConfig, id string, global bool) (*Token, error) {
	keyPrefix := fmt.Sprintf("git-bug.token.%s.", id)

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
		delete(configs, key)
		newKey := strings.TrimPrefix(key, keyPrefix)
		configs[newKey] = value
	}

	var ok bool
	token := &Token{ID: id, Global: global}

	token.Value, ok = configs[tokenValueKey]
	if !ok {
		return nil, fmt.Errorf("empty token value")
	}

	token.Target, ok = configs[tokenTargetKey]
	if !ok {
		return nil, fmt.Errorf("empty token key")
	}

	scopesString, ok := configs[tokenScopesKey]
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

// ListTokens return a map representing the stored tokens in the repo config and global config
// along with their type (global: true, local:false)
func ListTokens(repo repository.RepoConfig) (map[string]bool, error) {
	localTokens, err := listTokens(repo, false)
	if err != nil {
		return nil, err
	}

	globalTokens, err := listTokens(repo, true)
	if err != nil {
		return nil, err
	}

	tokens := map[string]bool{}
	for _, token := range localTokens {
		tokens[token] = false
	}

	for _, token := range globalTokens {
		tokens[token] = true
	}

	return tokens, nil
}

func storeToken(repo repository.RepoConfig, token *Token) error {
	storeFn := repo.StoreConfig
	if token.Global {
		storeFn = repo.StoreGlobalConfig
	}

	storeValueKey := fmt.Sprintf("git-bug.token.%s.%s", token.Id(), tokenValueKey)
	err := storeFn(storeValueKey, token.Value)
	if err != nil {
		return err
	}

	storeTargetKey := fmt.Sprintf("git-bug.token.%s.%s", token.Id(), tokenTargetKey)
	err = storeFn(storeTargetKey, token.Target)
	if err != nil {
		return err
	}

	storeScopesKey := fmt.Sprintf("git-bug.token.%s.%s", token.Id(), tokenScopesKey)
	return storeFn(storeScopesKey, strings.Join(token.Scopes, ","))
}

// StoreToken stores a token in the repo config
func StoreToken(repo repository.RepoConfig, token *Token) error {
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
