package core

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/MichaelMure/git-bug/cache"
)

const (
	tokenConfigKeyPrefix = "git-bug.token"
	tokenKeyValue        = "value"
	tokenKeyTarget       = "target"
	tokenKeyGlobal       = "global"
	tokenKeyScopes       = "scopes"
)

type Token struct {
	Name   string
	Value  string
	Target string
	Global bool
	Scopes []string
}

func NewToken(name, value, target string, global bool, scopes []string) *Token {
	return &Token{
		Name:   name,
		Value:  value,
		Target: target,
		Global: global,
		Scopes: scopes,
	}
}

func (t *Token) Valnameate() error {
	if t.Name == "" {
		return fmt.Errorf("missing token name")
	}
	if t.Value == "" {
		return fmt.Errorf("missing token value")
	}
	if t.Target == "" {
		return fmt.Errorf("missing token target")
	}
	return nil
}

func loadToken(repo *cache.RepoCache, name string, global bool) (*Token, error) {
	keyPrefix := fmt.Sprintf("git-bug.token.%s", name)
	var pairs map[string]string
	var err error
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

	result := make(Configuration, len(pairs))
	for key, value := range pairs {
		key := strings.TrimPrefix(key, keyPrefix)
		result[key] = value
	}

	var ok bool
	token := &Token{Name: name}
	token.Value, ok = result[tokenKeyValue]
	if !ok {
		return nil, fmt.Errorf("nil value")
	}

	token.Target, ok = result[tokenKeyTarget]
	if !ok {
		return nil, fmt.Errorf("nil value")
	}

	if g, ok := result[tokenKeyGlobal]; !ok {
		return nil, fmt.Errorf("nil value")
	} else if g == "true" {
		token.Global = true
	}

	scopesString, ok := result[tokenKeyScopes]
	if !ok {
		return nil, fmt.Errorf("missing scopes config")
	}

	token.Scopes = strings.Split(scopesString, ",")
	return token, nil
}

func GetToken(repo *cache.RepoCache, name string) (*Token, error) {
	return loadToken(repo, name, false)
}

func GetGlobalToken(repo *cache.RepoCache, name string) (*Token, error) {
	return loadToken(repo, name, true)
}

func listTokens(repo *cache.RepoCache, global bool) ([]string, error) {
	var configs map[string]string
	var err error
	if global {
		configs, err = repo.ReadConfigs(tokenConfigKeyPrefix + ".")
		if err != nil {
			return nil, err
		}
	} else {
		configs, err = repo.ReadGlobalConfigs(tokenConfigKeyPrefix + ".")
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

func ListTokens(repo *cache.RepoCache) ([]string, error) {
	return listTokens(repo, false)
}

func ListGlobalTokens(repo *cache.RepoCache) ([]string, error) {
	return listTokens(repo, true)
}

func storeToken(repo *cache.RepoCache, token *Token) error {
	var store func(key, value string) error
	if token.Global {
		store = repo.StoreGlobalConfig
	} else {
		store = repo.StoreConfig
	}

	var err error
	storeValueKey := fmt.Sprintf("git-bug.token.%s.%s", token.Name, tokenKeyValue)
	err = store(storeValueKey, token.Value)
	if err != nil {
		return err
	}

	storeTargetKey := fmt.Sprintf("git-bug.token.%s.%s", token.Name, tokenKeyTarget)
	err = store(storeTargetKey, token.Target)
	if err != nil {
		return err
	}

	storeGlobalKey := fmt.Sprintf("git-bug.token.%s.%s", token.Name, tokenKeyGlobal)
	err = store(storeGlobalKey, strconv.FormatBool(token.Global))
	if err != nil {
		return err
	}

	storeScopesKey := fmt.Sprintf("git-bug.token.%s.%s", token.Name, tokenKeyScopes)
	err = store(storeScopesKey, strings.Join(token.Scopes, ","))
	if err != nil {
		return err
	}

	return nil
}

func StoreToken(repo *cache.RepoCache, name, value, target string, global bool, scopes []string) error {
	return storeToken(repo, NewToken(name, value, target, global, scopes))
}

func StoreGlobalToken(repo *cache.RepoCache, name, value, target string, global bool, scopes []string) error {
	return storeToken(repo, NewToken(name, value, target, global, scopes))
}

func RemoveToken(repo *cache.RepoCache, name string) error {
	keyPrefix := fmt.Sprintf("git-bug.token.%s", name)
	return repo.RmConfigs(keyPrefix)
}

func RemoveGlobalToken(repo *cache.RepoCache, name string) error {
	keyPrefix := fmt.Sprintf("git-bug.token.%s", name)
	return repo.RmGlobalConfigs(keyPrefix)
}
