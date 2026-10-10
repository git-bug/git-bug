// Package envtest provides an execenv.Env for tests, over a test repository,
// with in-memory input and outputs.
package envtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/repository/repotest"
)

var _ execenv.In = &TestIn{}

type TestIn struct {
	*bytes.Buffer
	forceIsTerminal bool
}

func (t *TestIn) IsTerminal() bool {
	return t.forceIsTerminal
}

func (t *TestIn) ForceIsTerminal(value bool) {
	t.forceIsTerminal = value
}

var _ execenv.Out = &TestOut{}

type TestOut struct {
	*bytes.Buffer
	forceIsTerminal bool
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

func (te *TestOut) IsTerminal() bool {
	return te.forceIsTerminal
}

func (te *TestOut) Width() int {
	return 80
}

func (te *TestOut) Raw() io.Writer {
	return te.Buffer
}

func (te *TestOut) ForceIsTerminal(value bool) {
	te.forceIsTerminal = value
}

func NewEnv(t *testing.T) *execenv.Env {
	t.Helper()
	return newEnv(t, false)
}

func NewEnvTerminal(t *testing.T) *execenv.Env {
	t.Helper()
	return newEnv(t, true)
}

func newEnv(t *testing.T, isTerminal bool) *execenv.Env {
	repo := repotest.NewGoGitRepo(t, false)

	backend, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)

	t.Cleanup(func() {
		backend.Close()
	})

	return &execenv.Env{
		Ctx:     t.Context(),
		Repo:    repo,
		Backend: backend,
		In:      &TestIn{Buffer: &bytes.Buffer{}, forceIsTerminal: isTerminal},
		Out:     &TestOut{Buffer: &bytes.Buffer{}, forceIsTerminal: isTerminal},
		Err:     &TestOut{Buffer: &bytes.Buffer{}, forceIsTerminal: isTerminal},
	}
}
