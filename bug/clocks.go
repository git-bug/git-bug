package bug

import (
	"github.com/MichaelMure/git-bug/repository"
)

func Witnesser(repo *repository.GitRepo) error {
	for b := range ReadAllLocalBugs(repo) {
		if b.Err != nil {
			return b.Err
		}

		repo.CreateWitness(b.Bug.createTime)
		repo.EditWitness(b.Bug.editTime)
	}

	return nil
}
