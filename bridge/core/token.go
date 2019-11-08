package core

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

const (
	tokenConfigKeyPrefix = "git-bug.token"
	tokenValueKey        = "value"
	tokenTargetKey       = "target"
	tokenCreateTimeKey   = "createtime"
)

// Token holds an API access token data
type Token struct {
	ID         entity.Id
	Value      string
	Target     string
	CreateTime time.Time
}

// NewToken instantiate a new token
func NewToken(value, target string) *Token {
	token := &Token{
		Value:      value,
		Target:     target,
		CreateTime: time.Now(),
	}

	token.ID = entity.Id(hashToken(token))
	return token
}

func hashToken(token *Token) string {
	sum := sha256.Sum256([]byte(token.Value))
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
	if t.CreateTime.Equal(time.Time{}) {
		return fmt.Errorf("missing creation time")
	}
	if _, ok := bridgeImpl[t.Target]; !ok {
		return fmt.Errorf("unknown target")
	}
	return nil
}

// LoadToken loads a token from repo config
func LoadToken(repo repository.RepoCommon, id string) (*Token, error) {
	keyPrefix := fmt.Sprintf("git-bug.token.%s.", id)

	// read token config pairs
	configs, err := repo.GlobalConfig().ReadAll(keyPrefix)
	if err != nil {
		return nil, err
	}

	// trim key prefix
	for key, value := range configs {
		delete(configs, key)
		newKey := strings.TrimPrefix(key, keyPrefix)
		configs[newKey] = value
	}

	token := &Token{ID: entity.Id(id)}

	var ok bool
	token.Value, ok = configs[tokenValueKey]
	if !ok {
		return nil, fmt.Errorf("empty token value")
	}

	token.Target, ok = configs[tokenTargetKey]
	if !ok {
		return nil, fmt.Errorf("empty token key")
	}

	createTime, ok := configs[tokenCreateTimeKey]
	if !ok {
		return nil, fmt.Errorf("missing createtime key")
	}

	token.CreateTime, err = dateparse.ParseLocal(createTime)
	if err != nil {
		return nil, err
	}
	return token, nil
}

// ListTokens return a map representing the stored tokens in the repo config and global config
// along with their type (global: true, local:false)
func ListTokens(repo repository.RepoCommon) ([]string, error) {
	configs, err := repo.GlobalConfig().ReadAll(tokenConfigKeyPrefix + ".")
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

// StoreToken stores a token in the repo config
func StoreToken(repo repository.RepoCommon, token *Token) error {
	storeValueKey := fmt.Sprintf("git-bug.token.%s.%s", token.ID.String(), tokenValueKey)
	err := repo.GlobalConfig().StoreString(storeValueKey, token.Value)
	if err != nil {
		return err
	}

	storeTargetKey := fmt.Sprintf("git-bug.token.%s.%s", token.ID.String(), tokenTargetKey)
	err = repo.GlobalConfig().StoreString(storeTargetKey, token.Target)
	if err != nil {
		return err
	}

	createTimeKey := fmt.Sprintf("git-bug.token.%s.%s", token.ID.String(), tokenCreateTimeKey)
	return repo.GlobalConfig().StoreString(createTimeKey, strconv.Itoa(int(token.CreateTime.Unix())))
}

// RemoveToken removes a token from the repo config
func RemoveToken(repo repository.RepoCommon, id string) error {
	keyPrefix := fmt.Sprintf("git-bug.token.%s", id)
	return repo.GlobalConfig().RemoveAll(keyPrefix)
}
