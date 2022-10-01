package commands

import (
	"bytes"
	"testing"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/stretchr/testify/require"
)

type testEnv struct {
	env *Env
	out *bytes.Buffer
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	repo := repository.CreateGoGitTestRepo(t, false)

	buf := new(bytes.Buffer)

	backend, stderr := cache.NewTestRepoCache(t, repo)
	t.Cleanup(func() {
		backend.Close()

		require.Empty(t, stderr.String())
	})

	return &testEnv{
		env: &Env{
			repo:    repo,
			backend: backend,
			out:     out{Writer: buf},
			err:     out{Writer: buf},
		},
		out: buf,
	}
}
