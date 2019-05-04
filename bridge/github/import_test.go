package github

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
			url:  "https://github.com/MichaelMure/git-but-test-github-bridge/issues/1",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "simple issue", "initial comment", nil),
					bug.NewAddCommentOp(author, 0, "first comment", nil),
					bug.NewAddCommentOp(author, 0, "second comment", nil)},
			},
		},
		{
			name: "empty issue",
			url:  "https://github.com/MichaelMure/git-but-test-github-bridge/issues/2",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "empty issue", "", nil),
				},
			},
		},
		{
			name: "complex issue",
			url:  "https://github.com/MichaelMure/git-but-test-github-bridge/issues/3",
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
			url:  "https://github.com/MichaelMure/git-but-test-github-bridge/issues/4",
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
			url:  "https://github.com/MichaelMure/git-but-test-github-bridge/issues/5",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "comment deletion", "", nil),
				},
			},
		},
		{
			name: "edition deletion",
			url:  "https://github.com/MichaelMure/git-but-test-github-bridge/issues/6",
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
			url:  "https://github.com/MichaelMure/git-but-test-github-bridge/issues/7",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "hidden comment", "initial comment", nil),
					bug.NewAddCommentOp(author, 0, "first comment", nil),
				},
			},
		},
		{
			name: "transfered issue",
			url:  "https://github.com/MichaelMure/git-but-test-github-bridge/issues/8",
			bug: &bug.Snapshot{
				Operations: []bug.Operation{
					bug.NewCreateOp(author, 0, "transfered issue", "", nil),
				},
			},
		},
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	repo, err := repository.NewGitRepo(cwd, bug.Witnesser)
	if err != nil {
		t.Fatal(err)
	}

	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		t.Fatal(err)
	}

	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("Env var GITHUB_TOKEN missing")
	}

	importer := &githubImporter{}
	err = importer.Init(core.Configuration{
		"user":    "MichaelMure",
		"project": "git-but-test-github-bridge",
		"token":   token,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = importer.ImportAll(backend, time.Time{})
	if err != nil {
		t.Fatal(err)
	}

	ids := backend.AllBugsIds()
	assert.Equal(t, len(ids), 8)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := backend.ResolveBugCreateMetadata(keyGithubUrl, tt.url)
			if err != nil {
				t.Fatal(err)
			}

			ops := b.Snapshot().Operations
			assert.Equal(t, len(tt.bug.Operations), len(b.Snapshot().Operations))

			for i, op := range tt.bug.Operations {
				switch op.(type) {
				case *bug.CreateOperation:
					if op2, ok := ops[i].(*bug.CreateOperation); ok {
						assert.Equal(t, op2.Title, op.(*bug.CreateOperation).Title)
						assert.Equal(t, op2.Message, op.(*bug.CreateOperation).Message)
						continue
					}
					t.Errorf("bad operation type index = %d expected = CreationOperation", i)
				case *bug.SetStatusOperation:
					if op2, ok := ops[i].(*bug.SetStatusOperation); ok {
						assert.Equal(t, op2.Status, op.(*bug.SetStatusOperation).Status)
						continue
					}
					t.Errorf("bad operation type index = %d expected = SetStatusOperation", i)
				case *bug.SetTitleOperation:
					if op2, ok := ops[i].(*bug.SetTitleOperation); ok {
						assert.Equal(t, op.(*bug.SetTitleOperation).Was, op2.Was)
						assert.Equal(t, op.(*bug.SetTitleOperation).Title, op2.Title)
						continue
					}
					t.Errorf("bad operation type index = %d expected = SetTitleOperation", i)
				case *bug.LabelChangeOperation:
					if op2, ok := ops[i].(*bug.LabelChangeOperation); ok {
						assert.ElementsMatch(t, op.(*bug.LabelChangeOperation).Added, op2.Added)
						assert.ElementsMatch(t, op.(*bug.LabelChangeOperation).Removed, op2.Removed)
						continue
					}
					t.Errorf("bad operation type index = %d expected = ChangeLabelOperation", i)
				case *bug.AddCommentOperation:
					if op2, ok := ops[i].(*bug.AddCommentOperation); ok {
						assert.Equal(t, op.(*bug.AddCommentOperation).Message, op2.Message)
						continue
					}
					t.Errorf("bad operation type index = %d expected = AddCommentOperation", i)
				case *bug.EditCommentOperation:
					if op2, ok := ops[i].(*bug.EditCommentOperation); ok {
						assert.Equal(t, op.(*bug.EditCommentOperation).Message, op2.Message)
						continue
					}
					t.Errorf("bad operation type index = %d expected = EditCommentOperation", i)
				default:
					panic("Unknown operation type")
				}
			}
		})
	}
}
