package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
)

type testEnv struct {
	env *Env
	cwd string
	out *bytes.Buffer
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	cwd := t.TempDir()

	repo, err := repository.InitGoGitRepo(cwd, gitBugNamespace)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, repo.Close())
	})

	buf := new(bytes.Buffer)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)
	t.Cleanup(func() {
		backend.Close()
	})

	return &testEnv{
		env: &Env{
			repo:    repo,
			backend: backend,
			out:     out{Writer: buf},
			err:     out{Writer: buf},
		},
		cwd: cwd,
		out: buf,
	}
}
