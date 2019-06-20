package github

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func Test_Importer(t *testing.T) {
	author := identity.NewIdentity("Michael Muré", "batolettre@gmail.com")
	tests := []struct {
		name string
		url  string
		bug  *bug.Snapshot
	}{
		{
			name: "simple issue",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/1",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "simple issue", "initial comment", nil),
					bug.NewAddCommentOp(author, 0, "first comment", nil),
					bug.NewAddCommentOp(author, 0, "second comment", nil),
				},
			},
		},
		{
			name: "empty issue",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/2",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "empty issue", "", nil),
				},
			},
		},
		{
			name: "complex issue",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/3",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "complex issue", "initial comment", nil),
					bug.NewLabelChangeOperation(author, 0, []bug.Label{"bug"}, []bug.Label{}),
					bug.NewLabelChangeOperation(author, 0, []bug.Label{"duplicate"}, []bug.Label{}),
					bug.NewLabelChangeOperation(author, 0, []bug.Label{}, []bug.Label{"duplicate"}),
					bug.NewAddCommentOp(author, 0, "### header\n\n**bold**\n\n_italic_\n\n> with quote\n\n`inline code`\n\n```\nmultiline code\n```\n\n- bulleted\n- list\n\n1. numbered\n1. list\n\n- [ ] task\n- [x] list\n\n@MichaelMure mention\n\n#2 reference issue\n#3 auto-reference issue\n\n![image](https://user-images.githubusercontent.com/294669/56870222-811faf80-6a0c-11e9-8f2c-f0beb686303f.png)", nil),
					bug.NewSetTitleOp(author, 0, "complex issue edited", "complex issue"),
					bug.NewSetTitleOp(author, 0, "complex issue", "complex issue edited"),
					bug.NewSetStatusOp(author, 0, bug.ClosedStatus),
					bug.NewSetStatusOp(author, 0, bug.OpenStatus),
				},
			},
		},
		{
			name: "editions",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/4",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "editions", "initial comment edited", nil),
					bug.NewEditCommentOp(author, 0, "", "erased then edited again", nil),
					bug.NewAddCommentOp(author, 0, "first comment", nil),
					bug.NewEditCommentOp(author, 0, "", "first comment edited", nil),
				},
			},
		},
		{
			name: "comment deletion",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/5",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "comment deletion", "", nil),
				},
			},
		},
		{
			name: "edition deletion",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/6",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "edition deletion", "initial comment", nil),
					bug.NewEditCommentOp(author, 0, "", "initial comment edited again", nil),
					bug.NewAddCommentOp(author, 0, "first comment", nil),
					bug.NewEditCommentOp(author, 0, "", "first comment edited again", nil),
				},
			},
		},
		{
			name: "hidden comment",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/7",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "hidden comment", "initial comment", nil),
					bug.NewAddCommentOp(author, 0, "first comment", nil),
				},
			},
		},
		{
			name: "transfered issue",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/8",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "transfered issue", "", nil),
				},
			},
		},
		{
			name: "unicode control characters",
			url:  "https://github.com/MichaelMure/git-bug-test-github-bridge/issues/10",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "unicode control characters", "u0000: \nu0001: \nu0002: \nu0003: \nu0004: \nu0005: \nu0006: \nu0007: \nu0008: \nu0009: \t\nu0010: \nu0011: \nu0012: \nu0013: \nu0014: \nu0015: \nu0016: \nu0017: \nu0018: \nu0019:", nil),
				},
			},
		},
	}

	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(t, repo)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	token := os.Getenv("GITHUB_TOKEN_PRIVATE")
	if token == "" {
		t.Skip("Env var GITHUB_TOKEN_PRIVATE missing")
	}

	importer := &githubImporter{}
	err = importer.Init(core.Configuration{
		keyOwner:   "MichaelMure",
		keyProject: "git-bug-test-github-bridge",
		keyToken:   token,
	})
	require.NoError(t, err)

	start := time.Now()

	err = importer.ImportAll(backend, time.Time{})
	require.NoError(t, err)

	fmt.Printf("test repository imported in %f seconds\n", time.Since(start).Seconds())

	require.Len(t, backend.AllBugsIds(), len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := backend.ResolveBugCreateMetadata(keyGithubUrl, tt.url)
			require.NoError(t, err)

			ops := b.Snapshot().Operations
			assert.Len(t, tt.bug.Operations, len(b.Snapshot().Operations))

			for i, op := range tt.bug.Operations {
				require.IsType(t, ops[i], op)

				switch op.(type) {
				case *bug.CreateOperation:
					assert.Equal(t, op.(*bug.CreateOperation).Title, ops[i].(*bug.CreateOperation).Title)
					assert.Equal(t, op.(*bug.CreateOperation).Message, ops[i].(*bug.CreateOperation).Message)
					assert.Equal(t, op.(*bug.CreateOperation).Author.Name(), ops[i].(*bug.CreateOperation).Author.Name())
				case *bug.SetStatusOperation:
					assert.Equal(t, op.(*bug.SetStatusOperation).Status, ops[i].(*bug.SetStatusOperation).Status)
					assert.Equal(t, op.(*bug.SetStatusOperation).Author.Name(), ops[i].(*bug.SetStatusOperation).Author.Name())
				case *bug.SetTitleOperation:
					assert.Equal(t, op.(*bug.SetTitleOperation).Was, ops[i].(*bug.SetTitleOperation).Was)
					assert.Equal(t, op.(*bug.SetTitleOperation).Title, ops[i].(*bug.SetTitleOperation).Title)
					assert.Equal(t, op.(*bug.SetTitleOperation).Author.Name(), ops[i].(*bug.SetTitleOperation).Author.Name())
				case *bug.LabelChangeOperation:
					assert.ElementsMatch(t, op.(*bug.LabelChangeOperation).Added, ops[i].(*bug.LabelChangeOperation).Added)
					assert.ElementsMatch(t, op.(*bug.LabelChangeOperation).Removed, ops[i].(*bug.LabelChangeOperation).Removed)
					assert.Equal(t, op.(*bug.LabelChangeOperation).Author.Name(), ops[i].(*bug.LabelChangeOperation).Author.Name())
				case *bug.AddCommentOperation:
					assert.Equal(t, op.(*bug.AddCommentOperation).Message, ops[i].(*bug.AddCommentOperation).Message)
					assert.Equal(t, op.(*bug.AddCommentOperation).Author.Name(), ops[i].(*bug.AddCommentOperation).Author.Name())
				case *bug.EditCommentOperation:
					assert.Equal(t, op.(*bug.EditCommentOperation).Message, ops[i].(*bug.EditCommentOperation).Message)
					assert.Equal(t, op.(*bug.EditCommentOperation).Author.Name(), ops[i].(*bug.EditCommentOperation).Author.Name())

				default:
					panic("unknown operation type")
				}
			}
		})
	}
}
