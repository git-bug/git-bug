package bugcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/commands/bug/testenv"
	"github.com/git-bug/git-bug/commands/execenv"
)

func TestBugNew(t *testing.T) {
	writeMessageFile := func(t *testing.T, content string) string {
		t.Helper()

		path := filepath.Join(t.TempDir(), "message.txt")
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
		return path
	}

	requireSingleBug := func(t *testing.T, env *execenv.Env, title string, message string) {
		t.Helper()

		ids := env.Backend.Bugs().AllIds()
		require.Len(t, ids, 1)

		b, err := env.Backend.Bugs().Resolve(ids[0])
		require.NoError(t, err)

		snap := b.Snapshot()
		require.Equal(t, title, snap.Title)
		require.Equal(t, message, snap.Comments[0].Message)
	}

	t.Run("title and message", func(t *testing.T) {
		env, _ := testenv.NewTestEnvAndUser(t)

		err := runBugNew(env, bugNewOptions{
			nonInteractive: true,
			message:        "message",
			title:          "title",
		})
		require.NoError(t, err)
		require.Regexp(t, "^[0-9A-Fa-f]{7} created\n$", env.Out.String())
	})

	t.Run("file", func(t *testing.T) {
		env, _ := testenv.NewTestEnvAndUser(t)

		err := runBugNew(env, bugNewOptions{
			nonInteractive: true,
			messageFile:    writeMessageFile(t, "file title\n\nfile message\n"),
		})
		require.NoError(t, err)

		requireSingleBug(t, env, "file title", "file message")
	})

	t.Run("title and file", func(t *testing.T) {
		env, _ := testenv.NewTestEnvAndUser(t)

		err := runBugNew(env, bugNewOptions{
			nonInteractive: true,
			title:          "title",
			messageFile:    writeMessageFile(t, "first line\n\nfile message\n"),
		})
		require.NoError(t, err)

		requireSingleBug(t, env, "title", "first line\n\nfile message")
	})
}
