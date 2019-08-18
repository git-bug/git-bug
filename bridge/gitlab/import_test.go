package gitlab

import (
	"context"
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

func TestImport(t *testing.T) {
	author := identity.NewIdentity("Amine hilaly", "hilalyamine@gmail.com")
	tests := []struct {
		name string
		url  string
		bug  *bug.Snapshot
	}{
		{
			name: "simple issue",
			url:  "https://gitlab.com/git-bug/test/issues/1",
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
			url:  "https://gitlab.com/git-bug/test/issues/2",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "empty issue", "", nil),
				},
			},
		},
		{
			name: "complex issue",
			url:  "https://gitlab.com/git-bug/test/issues/3",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "complex issue", "initial comment", nil),
					bug.NewAddCommentOp(author, 0, "### header\n\n**bold**\n\n_italic_\n\n> with quote\n\n`inline code`\n\n```\nmultiline code\n```\n\n- bulleted\n- list\n\n1. numbered\n1. list\n\n- [ ] task\n- [x] list\n\n@MichaelMure mention\n\n#2 reference issue\n#3 auto-reference issue", nil),
					bug.NewSetTitleOp(author, 0, "complex issue edited", "complex issue"),
					bug.NewSetTitleOp(author, 0, "complex issue", "complex issue edited"),
					bug.NewSetStatusOp(author, 0, bug.ClosedStatus),
					bug.NewSetStatusOp(author, 0, bug.OpenStatus),
					bug.NewLabelChangeOperation(author, 0, []bug.Label{"bug"}, []bug.Label{}),
					bug.NewLabelChangeOperation(author, 0, []bug.Label{"critical"}, []bug.Label{}),
					bug.NewLabelChangeOperation(author, 0, []bug.Label{}, []bug.Label{"critical"}),
				},
			},
		},
		{
			name: "editions",
			url:  "https://gitlab.com/git-bug/test/issues/4",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "editions", "initial comment edited", nil),
					bug.NewAddCommentOp(author, 0, "first comment edited", nil),
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

	token := os.Getenv("GITLAB_API_TOKEN")
	if token == "" {
		t.Skip("Env var GITLAB_API_TOKEN missing")
	}

	projectID := os.Getenv("GITLAB_PROJECT_ID")
	if projectID == "" {
		t.Skip("Env var GITLAB_PROJECT_ID missing")
	}

	importer := &gitlabImporter{}
	err = importer.Init(core.Configuration{
		keyProjectID: projectID,
		keyToken:     token,
	})
	require.NoError(t, err)

	ctx := context.Background()
	start := time.Now()

	events, err := importer.ImportAll(ctx, backend, time.Time{})
	require.NoError(t, err)

	for result := range events {
		require.NoError(t, result.Err)
	}

	fmt.Printf("test repository imported in %f seconds\n", time.Since(start).Seconds())

	require.Len(t, backend.AllBugsIds(), len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := backend.ResolveBugCreateMetadata(keyGitlabUrl, tt.url)
			require.NoError(t, err)

			ops := b.Snapshot().Operations
			require.Len(t, tt.bug.Operations, len(ops))

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
