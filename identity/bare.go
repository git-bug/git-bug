package identity

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/util/lamport"
	"github.com/MichaelMure/git-bug/util/text"
)

type Bare struct {
	name      string
	email     string
	login     string
	avatarUrl string
}

func NewBare(name string, email string) *Bare {
	return &Bare{name: name, email: email}
}

func NewBareFull(name string, email string, login string, avatarUrl string) *Bare {
	return &Bare{name: name, email: email, login: login, avatarUrl: avatarUrl}
}

type bareIdentityJson struct {
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	Login     string `json:"login,omitempty"`
	AvatarUrl string `json:"avatar_url,omitempty"`
}

func (i Bare) MarshalJSON() ([]byte, error) {
	return json.Marshal(bareIdentityJson{
		Name:      i.name,
		Email:     i.email,
		Login:     i.login,
		AvatarUrl: i.avatarUrl,
	})
}

func (i Bare) UnmarshalJSON(data []byte) error {
	aux := bareIdentityJson{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	i.name = aux.Name
	i.email = aux.Email
	i.login = aux.Login
	i.avatarUrl = aux.AvatarUrl

	return nil
}

func (i Bare) Name() string {
	return i.name
}

func (i Bare) Email() string {
	return i.email
}

func (i Bare) Login() string {
	return i.login
}

func (i Bare) AvatarUrl() string {
	return i.avatarUrl
}

func (i Bare) Keys() []Key {
	return []Key{}
}

func (i Bare) ValidKeysAtTime(time lamport.Time) []Key {
	return []Key{}
}

// DisplayName return a non-empty string to display, representing the
// identity, based on the non-empty values.
func (i Bare) DisplayName() string {
	switch {
	case i.name == "" && i.login != "":
		return i.login
	case i.name != "" && i.login == "":
		return i.name
	case i.name != "" && i.login != "":
		return fmt.Sprintf("%s (%s)", i.name, i.login)
	}

	panic("invalid person data")
}

// Match tell is the Person match the given query string
func (i Bare) Match(query string) bool {
	query = strings.ToLower(query)

	return strings.Contains(strings.ToLower(i.name), query) ||
		strings.Contains(strings.ToLower(i.login), query)
}

// Validate check if the Identity data is valid
func (i Bare) Validate() error {
	if text.Empty(i.name) && text.Empty(i.login) {
		return fmt.Errorf("either name or login should be set")
	}

	if strings.Contains(i.name, "\n") {
		return fmt.Errorf("name should be a single line")
	}

	if !text.Safe(i.name) {
		return fmt.Errorf("name is not fully printable")
	}

	if strings.Contains(i.login, "\n") {
		return fmt.Errorf("login should be a single line")
	}

	if !text.Safe(i.login) {
		return fmt.Errorf("login is not fully printable")
	}

	if strings.Contains(i.email, "\n") {
		return fmt.Errorf("email should be a single line")
	}

	if !text.Safe(i.email) {
		return fmt.Errorf("email is not fully printable")
	}

	if i.avatarUrl != "" && !text.ValidUrl(i.avatarUrl) {
		return fmt.Errorf("avatarUrl is not a valid URL")
	}

	return nil
}

func (i Bare) IsProtected() bool {
	return false
}
