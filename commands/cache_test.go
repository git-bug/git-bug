package commands

import (
	"bytes"
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/commands/bug/testenv"
	"github.com/git-bug/git-bug/entities/bug"
)

func TestCacheRebuildCommand(t *testing.T) {
	env, bugId := testenv.NewTestEnvAndBug(t)

	root := env.Repo.LocalStorage().Root() // $REPO/.git/git-bug
	t.Chdir(filepath.Dir(filepath.Dir(root)))

	require.NoError(t, env.Backend.Close())
	env.Backend = nil

	// removed outside of git-bug
	require.NoError(t, env.Repo.RemoveRef("refs/"+bug.Namespace+"/"+bugId.String()))

	cmd := newCacheCommand(env)
	cmd.SetArgs([]string{"rebuild"})
	require.NoError(t, cmd.Execute())
	require.Empty(t, env.Out.String())
	require.Nil(t, env.Backend)

	backend, err := cache.NewRepoCacheNoEvents(env.Repo)
	require.NoError(t, err)
	t.Cleanup(func() { _ = backend.Close() })
	require.False(t, slices.Contains(backend.Bugs().AllIds(), bugId))
}

func TestCacheRebuildCommandArgs(t *testing.T) {
	env, _ := testenv.NewTestEnvAndUser(t)
	backend := env.Backend

	cmd := newCacheCommand(env)
	cmd.SetArgs([]string{"rebuild", "extra"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	require.ErrorContains(t, err, `unknown command "extra"`)

	// the pre-run never replaced the backend
	require.Same(t, backend, env.Backend)
}

func TestRootHelpMaintenanceGroup(t *testing.T) {
	root := NewRootCommand(context.Background(), "")

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"--help"})
	require.NoError(t, root.Execute())

	require.Regexp(t, `(?m)^Maintenance\n  cache +Manage the local cache\n  wipe +Wipe git-bug from the git repository\n`, out.String())
}
