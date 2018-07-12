package bug

import (
	"encoding/json"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util"
	"github.com/pkg/errors"
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

// Store will convert the Person to JSON and store it in the internal git datastore
// Return the git hash handle of the data
func (person *Person) Store(repo repository.Repo) (util.Hash, error) {

	data, err := json.Marshal(person)

	if err != nil {
		return "", err
	}

	return repo.StoreData(data)
}
