package auth

import (
	"crypto/sha256"
	"fmt"

	"github.com/MichaelMure/git-bug/entity"
)

const (
	keyringKeyTokenValue = "value"
)

var _ Credential = &Token{}

// Token holds an API access token data
type Token struct {
	*credentialBase
	Value string
}

// NewToken instantiate a new token
func NewToken(target, value string) *Token {
	return &Token{
		credentialBase: newCredentialBase(target),
		Value:          value,
	}
}

func NewTokenFromConfig(conf map[string]string) (*Token, error) {
	base, err := newCredentialBaseFromData(conf)
	if err != nil {
		return nil, err
	}

	return &Token{
		credentialBase: base,
		Value:          conf[keyringKeyTokenValue],
	}, nil
}

func (t *Token) ID() entity.Id {
	h := sha256.New()
	_, _ = h.Write(t.salt)
	_, _ = h.Write([]byte(t.target))
	_, _ = h.Write([]byte(t.Value))
	return entity.Id(fmt.Sprintf("%x", h.Sum(nil)))
}

func (t *Token) Kind() CredentialKind {
	return KindToken
}

// Validate ensure token important fields are valid
func (t *Token) Validate() error {
	err := t.credentialBase.validate()
	if err != nil {
		return err
	}
	if t.Value == "" {
		return fmt.Errorf("missing value")
	}
	return nil
}

func (t *Token) toConfig() map[string]string {
	return map[string]string{
		keyringKeyTokenValue: t.Value,
	}
}
