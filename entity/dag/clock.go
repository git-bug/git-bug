package dag

import (
	"fmt"

	"golang.org/x/sync/errgroup"

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
			var errG errgroup.Group
			for _, def := range defs {
				errG.Go(func() error {
					return ReadAllClocksNoCheck(def, repo)
				})
			}
			return errG.Wait()
		},
	}
}
