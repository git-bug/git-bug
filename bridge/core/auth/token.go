package auth

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

const (
	tokenValueKey = "value"
)

var _ Credential = &Token{}

// Token holds an API access token data
type Token struct {
	target     string
	createTime time.Time
	Value      string
	meta       map[string]string
}

// NewToken instantiate a new token
func NewToken(value, target string) *Token {
	return &Token{
		target:     target,
		createTime: time.Now(),
		Value:      value,
		meta:       make(map[string]string),
	}
}

func NewTokenFromConfig(conf map[string]string) *Token {
	token := &Token{}

	token.target = conf[configKeyTarget]
	if createTime, ok := conf[configKeyCreateTime]; ok {
		if t, err := repository.ParseTimestamp(createTime); err == nil {
			token.createTime = t
		}
	}

	token.Value = conf[tokenValueKey]
	token.meta = metaFromConfig(conf)

	return token
}

func (t *Token) ID() entity.Id {
	sum := sha256.Sum256([]byte(t.target + t.Value))
	return entity.Id(fmt.Sprintf("%x", sum))
}

func (t *Token) Target() string {
	return t.target
}

func (t *Token) Kind() CredentialKind {
	return KindToken
}

func (t *Token) CreateTime() time.Time {
	return t.createTime
}

// Validate ensure token important fields are valid
func (t *Token) Validate() error {
	if t.Value == "" {
		return fmt.Errorf("missing value")
	}
	if t.target == "" {
		return fmt.Errorf("missing target")
	}
	if t.createTime.IsZero() || t.createTime.Equal(time.Time{}) {
		return fmt.Errorf("missing creation time")
	}
	if !core.TargetExist(t.target) {
		return fmt.Errorf("unknown target")
	}
	return nil
}

func (t *Token) Metadata() map[string]string {
	return t.meta
}

func (t *Token) toConfig() map[string]string {
	return map[string]string{
		tokenValueKey: t.Value,
	}
}
