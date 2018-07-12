package bug

import (
	"github.com/MichaelMure/git-bug/repository"
	"github.com/pkg/errors"
)

type Person struct {
	Name  string
	Email string
}

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
