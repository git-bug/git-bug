package bugcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/commands/bug/testenv"
)

// Run the test executable as a fake editor without depending on a shell or an
// installed editor. The normal test process takes the m.Run path.
func TestMain(m *testing.M) {
	if os.Getenv("GIT_BUG_TEST_COMMENT_EDITOR") == "1" {
		if len(os.Args) != 2 {
			os.Exit(2)
		}
		data, err := os.ReadFile(os.Args[1])
		if err == nil {
			err = os.WriteFile(os.Getenv("GIT_BUG_TEST_EDITOR_CAPTURE"), data, 0600)
		}
		if replacement, ok := os.LookupEnv("GIT_BUG_TEST_EDITOR_REPLACEMENT"); err == nil && ok {
			err = os.WriteFile(os.Args[1], []byte(replacement), 0600)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestBugCommentEdit(t *testing.T) {
	const golden = "testdata/comment/edit"

	env, bugID, commentID := testenv.NewTestEnvAndBugWithComment(t)

	opts := bugCommentEditOptions{
		message: "this is an altered bug comment",
	}
	require.NoError(t, runBugCommentEdit(env, opts, []string{commentID.Human()}))

	require.NoError(t, runBugComment(env, []string{bugID.Human()}))
	requireCommentsEqual(t, golden, env)
}

func TestBugCommentEditWithEditor(t *testing.T) {
	for _, tc := range []struct {
		name           string
		previousEdit   string
		replacement    string
		leaveUnchanged bool
	}{
		{name: "original comment", replacement: "edited in the editor"},
		{name: "previously edited comment", previousEdit: "latest text\n\nwith another paragraph", replacement: "edited again"},
		{name: "unchanged comment", leaveUnchanged: true},
		{name: "empty message cancels"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, bugID, commentID := testenv.NewTestEnvAndBugWithComment(t)
			b, err := env.Backend.Bugs().Resolve(bugID)
			require.NoError(t, err)
			if tc.previousEdit != "" {
				_, err = b.EditComment(commentID, tc.previousEdit)
				require.NoError(t, err)
			}
			require.NoError(t, b.Commit())
			comment, err := b.Snapshot().SearchComment(commentID)
			require.NoError(t, err)
			original := comment.Message
			operationCount := len(b.Snapshot().AllOperations())

			executable, err := os.Executable()
			require.NoError(t, err)
			capture := filepath.Join(t.TempDir(), "editor-input.txt")
			t.Setenv("GIT_EDITOR", executable)
			t.Setenv("GIT_BUG_TEST_COMMENT_EDITOR", "1")
			t.Setenv("GIT_BUG_TEST_EDITOR_CAPTURE", capture)
			if !tc.leaveUnchanged {
				t.Setenv("GIT_BUG_TEST_EDITOR_REPLACEMENT", tc.replacement)
			}

			require.NoError(t, runBugCommentEdit(env, bugCommentEditOptions{}, []string{commentID.Human()}))
			input, err := os.ReadFile(capture)
			require.NoError(t, err)
			require.Contains(t, string(input), original)

			comment, err = b.Snapshot().SearchComment(commentID)
			require.NoError(t, err)
			if tc.leaveUnchanged || tc.replacement == "" {
				require.Equal(t, original, comment.Message)
			} else {
				require.Equal(t, tc.replacement, comment.Message)
			}
			if !tc.leaveUnchanged && tc.replacement == "" {
				require.Equal(t, operationCount, len(b.Snapshot().AllOperations()))
			}
		})
	}
}

func TestBugCommentEditWithoutEditor(t *testing.T) {
	for _, fromFile := range []bool{false, true} {
		name := "non-interactive without a message"
		if fromFile {
			name = "message from file"
		}
		t.Run(name, func(t *testing.T) {
			env, bugID, commentID := testenv.NewTestEnvAndBugWithComment(t)
			b, err := env.Backend.Bugs().Resolve(bugID)
			require.NoError(t, err)
			comment, err := b.Snapshot().SearchComment(commentID)
			require.NoError(t, err)
			expected := comment.Message
			opts := bugCommentEditOptions{nonInteractive: true}
			if fromFile {
				expected = "replacement from file"
				opts.messageFile = filepath.Join(t.TempDir(), "comment.txt")
				require.NoError(t, os.WriteFile(opts.messageFile, []byte(expected), 0600))
			}
			t.Setenv("GIT_EDITOR", filepath.Join(t.TempDir(), "missing-editor"))
			require.NoError(t, runBugCommentEdit(env, opts, []string{commentID.Human()}))
			comment, err = b.Snapshot().SearchComment(commentID)
			require.NoError(t, err)
			require.Equal(t, expected, comment.Message)
		})
	}
}
