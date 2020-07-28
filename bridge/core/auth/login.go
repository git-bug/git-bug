package auth

import (
	"crypto/sha256"
	"fmt"

	"github.com/MichaelMure/git-bug/entity"
)

const (
	keyringKeyLoginLogin = "login"
)

var _ Credential = &Login{}

type Login struct {
	*credentialBase
	Login string
}

func NewLogin(target, login string) *Login {
	return &Login{
		credentialBase: newCredentialBase(target),
		Login:          login,
	}
}

func NewLoginFromConfig(conf map[string]string) (*Login, error) {
	base, err := newCredentialBaseFromData(conf)
	if err != nil {
		return nil, err
	}

	return &Login{
		credentialBase: base,
		Login:          conf[keyringKeyLoginLogin],
	}, nil
}

func (lp *Login) ID() entity.Id {
	h := sha256.New()
	_, _ = h.Write(lp.salt)
	_, _ = h.Write([]byte(lp.target))
	_, _ = h.Write([]byte(lp.Login))
	return entity.Id(fmt.Sprintf("%x", h.Sum(nil)))
}

func (lp *Login) Kind() CredentialKind {
	return KindLogin
}

func (lp *Login) Validate() error {
	err := lp.credentialBase.validate()
	if err != nil {
		return err
	}
	if lp.Login == "" {
		return fmt.Errorf("missing login")
	}
	return nil
}

func (lp *Login) toConfig() map[string]string {
	return map[string]string{
		keyringKeyLoginLogin: lp.Login,
	}
}
