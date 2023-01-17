package execenv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
)

var _ Out = &TestOut{}

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

func (te *TestOut) PrintJSON(v interface{}) error {
	raw, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return err
	}
	te.Println(string(raw))
	return nil
}

func (te *TestOut) Raw() io.Writer {
	return te.Buffer
}

func NewTestEnv(t *testing.T) *Env {
	t.Helper()

	repo := repository.CreateGoGitTestRepo(t, false)

	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)

	t.Cleanup(func() {
		backend.Close()
	})

	return &Env{
		Repo:    repo,
		Backend: backend,
		Out:     &TestOut{&bytes.Buffer{}},
		Err:     &TestOut{&bytes.Buffer{}},
	}
}
