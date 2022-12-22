package execenv

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
)

type TestOut struct {
	*bytes.Buffer
}

func (te *TestOut) Printf(format string, a ...interface{}) {
	_, _ = fmt.Fprintf(te.Buffer, format, a...)
}

func (te *TestOut) Print(a ...interface{}) {
	_, _ = fmt.Fprint(te.Buffer, a...)
}

func (te *TestOut) Println(a ...interface{}) {
	_, _ = fmt.Fprintln(te.Buffer, a...)
}

func NewTestEnv(t *testing.T) *Env {
	t.Helper()

	repo := repository.CreateGoGitTestRepo(t, false)

	buf := new(bytes.Buffer)

	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)

	t.Cleanup(func() {
		backend.Close()
	})

	return &Env{
		Repo:    repo,
		Backend: backend,
		Out:     &TestOut{buf},
		Err:     &TestOut{buf},
	}
}
