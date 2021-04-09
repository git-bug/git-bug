package dag

import (
	"fmt"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
)

// ClockLoader is the repository.ClockLoader for Entity
func ClockLoader(defs ...Definition) repository.ClockLoader {
	clocks := make([]string, 0, len(defs)*2)
	for _, def := range defs {
		clocks = append(clocks, fmt.Sprintf(creationClockPattern, def.Namespace))
		clocks = append(clocks, fmt.Sprintf(editClockPattern, def.Namespace))
	}

	return repository.ClockLoader{
		Clocks: clocks,
		Witnesser: func(repo repository.ClockedRepo) error {
			// we need to actually load the identities because of the commit signature check when reading,
			// which require the full identities with crypto keys
			resolver := identity.NewCachedResolver(identity.NewSimpleResolver(repo))

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
