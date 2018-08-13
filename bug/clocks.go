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

		repo.CreateWitness(b.Bug.createTime)
		repo.EditWitness(b.Bug.editTime)
	}

	return nil
}
