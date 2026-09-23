package dag

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/lamport"
)

// A clock that isn't usable is rebuilt from every entity when a time is needed,
// not raised from the first entity read.
func TestUnusableClockRebuiltOnIncrement(t *testing.T) {
	damages := map[string]func(t *testing.T, repo repository.ClockedRepo, name string){
		"corrupted": corruptClock,
		"missing":   removeClock,
	}

	for name, damage := range damages {
		t.Run(name, func(t *testing.T) {
			repo, id1, _, resolver, def := makeTestContextGoGit(t)

			e1 := wrapper(New(def))
			e1.Append(newOp1(id1, "foo"))
			require.NoError(t, e1.Commit(repo))

			// e2 holds the highest edit time
			e2 := wrapper(New(def))
			for i := 0; i < 5; i++ {
				e2.Append(newOp1(id1, "bar"))
				require.NoError(t, e2.Commit(repo))
			}

			editClock := fmt.Sprintf(editClockPattern, def.Namespace)
			damage(t, repo, editClock)

			// reading e1 alone doesn't touch the clock
			e1, err := Read(def, wrapper, repo, resolver, e1.Id())
			require.NoError(t, err)
			requireClockUnusable(t, repo, editClock)

			e1.Append(newOp1(id1, "foobar"))
			require.NoError(t, e1.Commit(repo))
			require.Greater(t, e1.EditLamportTime(), e2.EditLamportTime())
			requireClockTime(t, repo, editClock, e1.EditLamportTime())
		})
	}
}

// With no entity to witness, a rebuilt clock starts over from the beginning.
func TestUnusableClockRebuiltWithoutEntities(t *testing.T) {
	repo, id1, _, _, def := makeTestContextGoGit(t)

	createClock := fmt.Sprintf(creationClockPattern, def.Namespace)
	corruptClock(t, repo, createClock)

	e := wrapper(New(def))
	e.Append(newOp1(id1, "foo"))
	require.NoError(t, e.Commit(repo))
	require.Equal(t, lamport.Time(2), e.CreateLamportTime())
}

// EnsureClocks rebuilds whatever isn't usable, and leaves the rest alone.
func TestEnsureClocks(t *testing.T) {
	repo, id1, _, _, def := makeTestContextGoGit(t)
	createClock := fmt.Sprintf(creationClockPattern, def.Namespace)
	editClock := fmt.Sprintf(editClockPattern, def.Namespace)

	// no entity yet: the clocks start from the beginning
	require.NoError(t, EnsureClocks(def, repo))
	requireClockTime(t, repo, createClock, 1)
	requireClockTime(t, repo, editClock, 1)

	e := wrapper(New(def))
	for i := 0; i < 3; i++ {
		e.Append(newOp1(id1, "foo"))
		require.NoError(t, e.Commit(repo))
	}
	createTime, editTime := e.CreateLamportTime(), e.EditLamportTime()

	// usable clocks are left alone, even above the entities
	require.NoError(t, repo.Witness(editClock, editTime+10))
	require.NoError(t, EnsureClocks(def, repo))
	requireClockTime(t, repo, createClock, createTime)
	requireClockTime(t, repo, editClock, editTime+10)

	// a missing clock and a corrupted one come back from the entities
	removeClock(t, repo, createClock)
	corruptClock(t, repo, editClock)
	require.NoError(t, EnsureClocks(def, repo))
	requireClockTime(t, repo, createClock, createTime)
	requireClockTime(t, repo, editClock, editTime)
}

func corruptClock(t *testing.T, repo repository.ClockedRepo, name string) {
	t.Helper()

	err := util.WriteFile(repo.LocalStorage(), filepath.Join("clocks", name), []byte("garbage"), 0644)
	require.NoError(t, err)
}

func removeClock(t *testing.T, repo repository.ClockedRepo, name string) {
	t.Helper()

	err := repo.LocalStorage().Remove(filepath.Join("clocks", name))
	require.NoError(t, err)
}

func requireClockUnusable(t *testing.T, repo repository.ClockedRepo, name string) {
	t.Helper()

	clock, err := repo.GetClock(name)
	if errors.Is(err, lamport.ErrClockNotExist) {
		return
	}
	require.NoError(t, err)

	_, err = clock.Time()
	require.True(t, unusableClock(err), "clock %s is usable: %v", name, err)
}

func requireClockTime(t *testing.T, repo repository.ClockedRepo, name string, expected lamport.Time) {
	t.Helper()

	clock, err := repo.GetClock(name)
	require.NoError(t, err)

	time, err := clock.Time()
	require.NoError(t, err)
	require.Equal(t, expected, time)
}
