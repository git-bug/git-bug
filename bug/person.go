package bug

import (
	"errors"
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/text"
)

type Person struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GetUser will query the repository for user detail and build the corresponding Person
func GetUser(repo repository.Repo) (Person, error) {
	name, err := repo.GetUserName()
	if err != nil {
		return Person{}, err
	}
	if name == "" {
		return Person{}, errors.New("User name is not configured in git yet. Please use `git config --global user.name \"John Doe\"`")
	}

	email, err := repo.GetUserEmail()
	if err != nil {
		return Person{}, err
	}
	if email == "" {
		return Person{}, errors.New("User name is not configured in git yet. Please use `git config --global user.email johndoe@example.com`")
	}

	return Person{Name: name, Email: email}, nil
}

// Match tell is the Person match the given query string
func (p Person) Match(query string) bool {
	return strings.Contains(strings.ToLower(p.Name), strings.ToLower(query))
}

func (p Person) Validate() error {
	if text.Empty(p.Name) {
		return fmt.Errorf("name is not set")
	}

	if strings.Contains(p.Name, "\n") {
		return fmt.Errorf("name should be a single line")
	}

	if !text.Safe(p.Name) {
		return fmt.Errorf("name is not fully printable")
	}

	if strings.Contains(p.Email, "\n") {
		return fmt.Errorf("email should be a single line")
	}

	if !text.Safe(p.Email) {
		return fmt.Errorf("email is not fully printable")
	}

	return nil
}
