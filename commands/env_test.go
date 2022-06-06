package commands_test

import (
	"bytes"
	"context"
	"flag"
	"io/ioutil"
	"os"
	"testing"

	"github.com/MichaelMure/git-bug/commands"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "pass -update to the test runner to update golden files")

type testEnv struct {
	cwd  string
	repo *repository.GoGitRepo
	cmd  *cobra.Command
	out  *bytes.Buffer
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	cwd, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(cwd))
	})

	repo, err := repository.InitGoGitRepo(cwd, commands.GitBugNamespace)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, repo.Close())
	})

	out := new(bytes.Buffer)
	cmd := commands.NewRootCommand()
	cmd.SetArgs([]string{})
	cmd.SetErr(out)
	cmd.SetOut(out)

	return &testEnv{
		cwd:  cwd,
		repo: repo,
		cmd:  cmd,
		out:  out,
	}
}

func (e *testEnv) Execute(t *testing.T) {
	t.Helper()

	ctx := context.WithValue(context.Background(), "cwd", e.cwd)
	require.NoError(t, e.cmd.ExecuteContext(ctx))
}
