package dag

import (
	"errors"
	"fmt"

	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/lamport"
)

// The clocks of a namespace order its entities, so a time they hand out has to
// be above every time those entities already hold. A clock that is missing or
// corrupted can't be trusted for that, and is rebuilt from the entities
// themselves before it issues anything.
//
// Reading an entity witnesses its times, but only into a clock that is already
// usable: raising a missing clock from a single entity would only make it look
// complete. A clock can still be stale after another process pushed entities
// into the repository, and that's fine: those are concurrent with anything
// issued here until they are read, at which point they are witnessed.

// EnsureClocks makes sure that the clocks of a definition are usable, and
// rebuilds them from the entities of its namespace if not.
//
// Entities rebuild their clocks on their own when they need a time, so this is
// for callers that need all the clocks right before anything is written: a
// fresh clone has none, and an identity records every clock in each of its
// versions.
func EnsureClocks(def Definition, repo repository.ClockedRepo) error {
	for _, pattern := range []string{creationClockPattern, editClockPattern} {
		usable, err := clockUsable(repo, fmt.Sprintf(pattern, def.Namespace))
		if err != nil {
			return err
		}
		if !usable {
			return rebuildClocks(def, repo)
		}
	}
	return nil
}

// clockUsable tells whether a clock exists and can issue times.
func clockUsable(repo repository.ClockedRepo, name string) (bool, error) {
	clock, err := repo.GetClock(name)
	if errors.Is(err, lamport.ErrClockNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	_, err = clock.Time()
	if unusableClock(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// unusableClock tells whether an error from a clock means that it has to be
// rebuilt: it's missing, or it's corrupted.
func unusableClock(err error) bool {
	return errors.Is(err, lamport.ErrClockNotExist) || errors.Is(err, lamport.ErrClockCorrupt)
}

// witnessClock witnesses a time read from an entity. A clock that isn't usable
// is left alone: raising it from this one entity would only make it look
// complete, that's for rebuildClocks to do, from all of them.
func witnessClock(repo repository.ClockedRepo, name string, time lamport.Time) error {
	err := repo.Witness(name, time)
	if unusableClock(err) {
		return nil
	}
	return err
}

// incrementClock returns the next time of a clock, rebuilding it first if it
// isn't usable.
func incrementClock(def Definition, repo repository.ClockedRepo, pattern string) (lamport.Time, error) {
	name := fmt.Sprintf(pattern, def.Namespace)

	time, err := repo.Increment(name)
	if !unusableClock(err) {
		return time, err
	}

	err = rebuildClocks(def, repo)
	if err != nil {
		return 0, err
	}

	return repo.Increment(name)
}

// rebuildClocks creates the clocks of a definition again, from every entity of
// its namespace. The entities are only scanned, not checked: see
// readClockNoCheck.
func rebuildClocks(def Definition, repo repository.ClockedRepo) error {
	createTime, editTime, err := readAllClocksNoCheck(def, repo)
	if err != nil {
		return err
	}

	err = rebuildClock(repo, fmt.Sprintf(creationClockPattern, def.Namespace), createTime)
	if err != nil {
		return err
	}

	return rebuildClock(repo, fmt.Sprintf(editClockPattern, def.Namespace), editTime)
}

// rebuildClock creates a clock at a time found by scanning every entity, or
// raises the existing one to it. Another process can be rebuilding the same
// clock: whichever creates it, the other's witness brings it to the highest of
// the two.
func rebuildClock(repo repository.ClockedRepo, name string, time lamport.Time) error {
	// with no entity to witness, a clock starts from the beginning
	time = max(time, 1)

	clock, err := repo.GetOrCreateClock(name, time)
	if err != nil {
		return err
	}

	return clock.Witness(time)
}
