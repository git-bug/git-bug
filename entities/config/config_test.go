package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/repository/repotest"
)

func TestConfig(t *testing.T) {
	clocks, rene, _ := makeTestContext(t)
	unix := time.Now().Unix()

	t.Run("validate", func(t *testing.T) {
		c, _, err := Create(clocks, rene, unix)
		require.NoError(t, err)
		_, err = LabelsSet(c, clocks, rene, unix, "bug", LabelsDefinition{})
		require.NoError(t, err)
		require.NoError(t, c.Validate())

		// no Create first
		c = NewConfig()
		_, err = LabelsSet(c, clocks, rene, unix, "bug", LabelsDefinition{})
		require.NoError(t, err)
		require.Error(t, c.Validate())

		// a second Create
		c, _, err = Create(clocks, rene, unix)
		require.NoError(t, err)
		second, err := NewCreateOp(clocks, rene, unix)
		require.NoError(t, err)
		c.Append(second)
		require.Error(t, c.Validate())
	})

	t.Run("defaults", func(t *testing.T) {
		c, _, err := Create(clocks, rene, unix)
		require.NoError(t, err)
		snap := c.Compile()

		require.Empty(t, snap.Repo.Description)

		_, ok := snap.Labels.Get("bug")
		require.False(t, ok)
		require.Equal(t, common.Label("bug").Color(), snap.Labels.Color("bug"))
		require.Empty(t, snap.Labels.Names())
		require.True(t, snap.Labels.Allowed("bug"))

		require.Len(t, snap.Operations, 1)
	})
}

func TestSync(t *testing.T) {
	// setup returns two repositories sharing the remote "origin", with an identity
	// known to both
	setup := func(t *testing.T) (repository.Repo, repository.Repo, identity.Interface) {
		repoA, repoB, _ := repotest.SetupGoGitReposAndRemote(t)

		rene, err := identity.NewIdentity(repoA, "René Descartes", "rene@descartes.fr")
		require.NoError(t, err)
		require.NoError(t, rene.Commit(t.Context(), repoA))
		_, err = identity.Push(t.Context(), repoA, "origin")
		require.NoError(t, err)
		require.NoError(t, identity.Pull(t.Context(), repoB, "origin"))

		return repoA, repoB, rene
	}

	pull := func(t *testing.T, repo repository.Repo, author identity.Interface) entity.MergeResult {
		_, err := Fetch(t.Context(), repo, "origin")
		require.NoError(t, err)
		var results []entity.MergeResult
		for result := range MergeAll(t.Context(), repo, simpleResolvers(repo), "origin", author) {
			require.NoError(t, result.Err)
			results = append(results, result)
		}
		require.Len(t, results, 1)
		return results[0]
	}

	read := func(t *testing.T, repo repository.Repo) *Config {
		c, err := Read(t.Context(), repo)
		require.NoError(t, err)
		return c
	}

	unix := time.Now().Unix()

	t.Run("commit and read", func(t *testing.T) {
		repoA, _, rene := setup(t)

		_, err := Read(t.Context(), repoA)
		require.True(t, entity.IsErrNotFound(err))

		c, _, err := Create(repoA, rene, unix)
		require.NoError(t, err)
		_, err = RepoSetDescription(c, repoA, rene, unix, "description")
		require.NoError(t, err)
		require.NoError(t, c.Commit(t.Context(), repoA))

		commit, err := repoA.ResolveRef(t.Context(), Namespace, "config")
		require.NoError(t, err)
		require.Equal(t, c.LastCommit(), commit)

		readC := read(t, repoA)
		require.Equal(t, c.Id(), readC.Id())
		require.Equal(t, "description", readC.Compile().Repo.Description)

		// an operation records the time of the clocks, the config's included
		op, err := LabelsSet(readC, repoA, rene, unix, "bug", LabelsDefinition{Color: SetField("#f44336"), Description: ClearField[string]()})
		require.NoError(t, err)
		require.Equal(t, readC.EditLamportTime(), op.Times["config-edit"])
		require.NoError(t, readC.Commit(t.Context(), repoA))

		readC = read(t, repoA)
		definition, ok := readC.Compile().Labels.Get("bug")
		require.True(t, ok)
		require.Equal(t, LabelsDefinition{Color: SetField("#f44336")}, definition)
	})

	t.Run("concurrent edits are merged", func(t *testing.T) {
		repoA, repoB, rene := setup(t)

		cA, _, err := Create(repoA, rene, unix)
		require.NoError(t, err)
		_, err = LabelsSet(cA, repoA, rene, unix, "crash", LabelsDefinition{Color: SetField("#f44336"), Description: SetField("Program dies")})
		require.NoError(t, err)
		require.NoError(t, cA.Commit(t.Context(), repoA))
		_, err = Push(t.Context(), repoA, "origin")
		require.NoError(t, err)

		require.Equal(t, entity.MergeStatusNew, pull(t, repoB, rene).Status)
		cB := read(t, repoB)
		require.Equal(t, cA.Id(), cB.Id())

		// each side defines and allows a label, sets the description, and changes a
		// different field of the same label
		_, err = LabelsSet(cA, repoA, rene, unix, "a", LabelsDefinition{})
		require.NoError(t, err)
		_, err = LabelsAllow(cA, repoA, rene, unix, "a")
		require.NoError(t, err)
		_, err = RepoSetDescription(cA, repoA, rene, unix, "from A")
		require.NoError(t, err)
		_, err = LabelsSet(cA, repoA, rene, unix, "crash", LabelsDefinition{Color: SetField("#ff9800")})
		require.NoError(t, err)
		require.NoError(t, cA.Commit(t.Context(), repoA))
		_, err = Push(t.Context(), repoA, "origin")
		require.NoError(t, err)

		_, err = LabelsSet(cB, repoB, rene, unix, "b", LabelsDefinition{})
		require.NoError(t, err)
		_, err = LabelsAllow(cB, repoB, rene, unix, "b")
		require.NoError(t, err)
		_, err = RepoSetDescription(cB, repoB, rene, unix, "from B")
		require.NoError(t, err)
		_, err = LabelsSet(cB, repoB, rene, unix, "crash", LabelsDefinition{Description: SetField("Program exits")})
		require.NoError(t, err)
		require.NoError(t, cB.Commit(t.Context(), repoB))

		require.Equal(t, entity.MergeStatusUpdated, pull(t, repoB, rene).Status)
		_, err = Push(t.Context(), repoB, "origin")
		require.NoError(t, err)
		require.Equal(t, entity.MergeStatusUpdated, pull(t, repoA, rene).Status)

		// both labels are kept, both changes of the same label too, and both sides
		// agree on the description
		snapA := read(t, repoA).Compile()
		snapB := read(t, repoB).Compile()
		require.Equal(t, []common.Label{"a", "b", "crash"}, snapA.Labels.Names())
		require.Equal(t, []common.Label{"a", "b"}, snapA.Labels.AllowedNames())
		crash, ok := snapA.Labels.Get("crash")
		require.True(t, ok)
		require.Equal(t, LabelsDefinition{Color: SetField("#ff9800"), Description: SetField("Program exits")}, crash)
		require.Contains(t, []string{"from A", "from B"}, snapA.Repo.Description)
		require.Equal(t, snapA.Labels, snapB.Labels)
		require.Equal(t, snapA.Repo, snapB.Repo)
	})

	t.Run("another config is a conflict", func(t *testing.T) {
		repoA, repoB, rene := setup(t)

		for _, repo := range []repository.Repo{repoA, repoB} {
			c, _, err := Create(repo, rene, unix)
			require.NoError(t, err)
			_, err = LabelsSet(c, repo, rene, unix, "bug", LabelsDefinition{})
			require.NoError(t, err)
			require.NoError(t, c.Commit(t.Context(), repo))
		}
		_, err := Push(t.Context(), repoA, "origin")
		require.NoError(t, err)

		before := read(t, repoB)
		require.Equal(t, entity.MergeStatusConflict, pull(t, repoB, rene).Status)
		require.Error(t, Pull(t.Context(), repoB, simpleResolvers(repoB), "origin", rene))

		// nothing changed locally, and the remote refuses the other config
		after := read(t, repoB)
		require.Equal(t, before.Id(), after.Id())
		require.Equal(t, before.LastCommit(), after.LastCommit())
		_, err = Push(t.Context(), repoB, "origin")
		require.Error(t, err)
	})
}
