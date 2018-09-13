package bug

import (
	"github.com/MichaelMure/git-bug/repository"
)

// Witnesser will read all the available Bug to recreate the different logical
// clocks
func Witnesser(repo *repository.GitRepo) error {
	for b := range ReadAllLocalBugs(repo) {
		if b.Err != nil {
			return b.Err
		}

		err := repo.CreateWitness(b.Bug.createTime)
		if err != nil {
			return err
		}

		err = repo.EditWitness(b.Bug.editTime)
		if err != nil {
			return err
		}
	}

	return nil
}
