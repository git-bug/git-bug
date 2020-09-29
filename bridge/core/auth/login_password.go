package auth

import (
	"crypto/sha256"
	"fmt"

	"github.com/MichaelMure/git-bug/entity"
)

const (
	keyringKeyLoginPasswordLogin    = "login"
	keyringKeyLoginPasswordPassword = "password"
)

var _ Credential = &LoginPassword{}

type LoginPassword struct {
	*credentialBase
	Login    string
	Password string
}

func NewLoginPassword(target, login, password string) *LoginPassword {
	return &LoginPassword{
		credentialBase: newCredentialBase(target),
		Login:          login,
		Password:       password,
	}
}

func NewLoginPasswordFromConfig(conf map[string]string) (*LoginPassword, error) {
	base, err := newCredentialBaseFromData(conf)
	if err != nil {
		return nil, err
	}

	return &LoginPassword{
		credentialBase: base,
		Login:          conf[keyringKeyLoginPasswordLogin],
		Password:       conf[keyringKeyLoginPasswordPassword],
	}, nil
}

func (lp *LoginPassword) ID() entity.Id {
	h := sha256.New()
	_, _ = h.Write(lp.salt)
	_, _ = h.Write([]byte(lp.target))
	_, _ = h.Write([]byte(lp.Login))
	_, _ = h.Write([]byte(lp.Password))
	return entity.Id(fmt.Sprintf("%x", h.Sum(nil)))
}

func (lp *LoginPassword) Kind() CredentialKind {
	return KindLoginPassword
}

func (lp *LoginPassword) Validate() error {
	err := lp.credentialBase.validate()
	if err != nil {
		return err
	}
	if lp.Login == "" {
		return fmt.Errorf("missing login")
	}
	if lp.Password == "" {
		return fmt.Errorf("missing password")
	}
	return nil
}

func (lp *LoginPassword) toConfig() map[string]string {
	return map[string]string{
		keyringKeyLoginPasswordLogin:    lp.Login,
		keyringKeyLoginPasswordPassword: lp.Password,
	}
}
