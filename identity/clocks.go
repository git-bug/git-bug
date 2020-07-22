package identity

import (
	"github.com/MichaelMure/git-bug/repository"
)

// ClockLoader is the repository.ClockLoader for the Identity entity
var ClockLoader = repository.ClockLoader{
	Clocks: []string{identityEditClockName},
	Witnesser: func(repo repository.ClockedRepo) error {
		editClock, err := repo.GetOrCreateClock(identityEditClockName)
		if err != nil {
			return err
		}

		for id := range ReadAllLocalIdentities(repo) {
			if id.Err != nil {
				return id.Err
			}

			for _, ver := range id.Identity.versions {
				err = editClock.Witness(ver.time)
				if err != nil {
					return err
				}
			}
		}

		return nil
	},
}
