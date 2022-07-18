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
	out *bytes.Buffer
	err *bytes.Buffer
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	repo := repository.CreateGoGitTestRepo(t, false)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)
	t.Cleanup(func() {
		backend.Close()
	})

	return &testEnv{
		env: &Env{
			repo:    repo,
			backend: backend,
			out:     out{Writer: outBuf},
			err:     out{Writer: errBuf},
		},
		out: outBuf,
		err: errBuf,
	}
}
