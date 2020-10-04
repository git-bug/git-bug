package bug

import (
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
)

// ClockLoader is the repository.ClockLoader for the Bug entity
var ClockLoader = repository.ClockLoader{
	Clocks: []string{creationClockName, editClockName},
	Witnesser: func(repo repository.ClockedRepo) error {
		// We don't care about the actual identity so an IdentityStub will do
		resolver := identity.NewStubResolver()
		for b := range ReadAllLocalWithResolver(repo, resolver) {
			if b.Err != nil {
				return b.Err
			}

			createClock, err := repo.GetOrCreateClock(creationClockName)
			if err != nil {
				return err
			}
			err = createClock.Witness(b.Bug.createTime)
			if err != nil {
				return err
			}

			editClock, err := repo.GetOrCreateClock(editClockName)
			if err != nil {
				return err
			}
			err = editClock.Witness(b.Bug.editTime)
			if err != nil {
				return err
			}
		}

		return nil
	},
}
