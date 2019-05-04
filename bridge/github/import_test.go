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
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/MichaelMure/git-bug/util/test"
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
					bug.NewAddCommentOp(author, 0, "second comment", nil)},
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
	}

	repo := test.CreateRepo(false)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("Env var GITHUB_TOKEN missing")
	}

	importer := &githubImporter{}
	err = importer.Init(core.Configuration{
		"user":    "MichaelMure",
		"project": "git-bug-test-github-bridge",
		"token":   token,
	})
	require.NoError(t, err)

	start := time.Now()

	err = importer.ImportAll(backend, time.Time{})
	require.NoError(t, err)

	fmt.Printf("test repository imported in %f seconds\n", time.Since(start).Seconds())

	require.Len(t, backend.AllBugsIds(), 8)

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
					assert.Equal(t, ops[i].(*bug.CreateOperation).Title, op.(*bug.CreateOperation).Title)
					assert.Equal(t, ops[i].(*bug.CreateOperation).Message, op.(*bug.CreateOperation).Message)
					assert.Equal(t, ops[i].(*bug.CreateOperation).Author.Name(), op.(*bug.CreateOperation).Author.Name())
				case *bug.SetStatusOperation:
					assert.Equal(t, ops[i].(*bug.SetStatusOperation).Status, op.(*bug.SetStatusOperation).Status)
					assert.Equal(t, ops[i].(*bug.SetStatusOperation).Author.Name(), op.(*bug.SetStatusOperation).Author.Name())
				case *bug.SetTitleOperation:
					assert.Equal(t, ops[i].(*bug.SetTitleOperation).Was, op.(*bug.SetTitleOperation).Was)
					assert.Equal(t, ops[i].(*bug.SetTitleOperation).Title, op.(*bug.SetTitleOperation).Title)
					assert.Equal(t, ops[i].(*bug.SetTitleOperation).Author.Name(), op.(*bug.SetTitleOperation).Author.Name())
				case *bug.LabelChangeOperation:
					assert.ElementsMatch(t, ops[i].(*bug.LabelChangeOperation).Added, op.(*bug.LabelChangeOperation).Added)
					assert.ElementsMatch(t, ops[i].(*bug.LabelChangeOperation).Removed, op.(*bug.LabelChangeOperation).Removed)
					assert.Equal(t, ops[i].(*bug.LabelChangeOperation).Author.Name(), op.(*bug.LabelChangeOperation).Author.Name())
				case *bug.AddCommentOperation:
					assert.Equal(t, ops[i].(*bug.AddCommentOperation).Message, op.(*bug.AddCommentOperation).Message)
					assert.Equal(t, ops[i].(*bug.AddCommentOperation).Author.Name(), op.(*bug.AddCommentOperation).Author.Name())
				case *bug.EditCommentOperation:
					assert.Equal(t, ops[i].(*bug.EditCommentOperation).Message, op.(*bug.EditCommentOperation).Message)
					assert.Equal(t, ops[i].(*bug.EditCommentOperation).Author.Name(), op.(*bug.EditCommentOperation).Author.Name())

				default:
					panic("Unknown operation type")
				}
			}
		})
	}
}
