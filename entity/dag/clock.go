package dag

import (
	"fmt"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
)

// ClockLoader is the repository.ClockLoader for Entity
func ClockLoader(defs ...Definition) repository.ClockLoader {
	clocks := make([]string, len(defs)*2)
	for _, def := range defs {
		clocks = append(clocks, fmt.Sprintf(creationClockPattern, def.namespace))
		clocks = append(clocks, fmt.Sprintf(editClockPattern, def.namespace))
	}

	return repository.ClockLoader{
		Clocks: clocks,
		Witnesser: func(repo repository.ClockedRepo) error {
			// We don't care about the actual identity so an IdentityStub will do
			resolver := identity.NewStubResolver()

			for _, def := range defs {
				// we actually just need to read all entities,
				// as that will create and update the clocks
				// TODO: concurrent loading to be faster?
				for b := range ReadAll(def, repo, resolver) {
					if b.Err != nil {
						return b.Err
					}
				}
			}
			return nil
		},
	}
}
