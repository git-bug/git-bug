package bug

import (
	"errors"

	"github.com/MichaelMure/git-bug/repository"
)

type Person struct {
	Name  string
	Email string
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
