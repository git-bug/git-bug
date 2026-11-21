package gitlab

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/api/client-go"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository/repotest"
)

func TestGitlabImport(t *testing.T) {
	envToken := os.Getenv("GITLAB_API_TOKEN")
	if envToken == "" {
		t.Skip("Env var GITLAB_API_TOKEN missing")
	}

	projectID := os.Getenv("GITLAB_PROJECT_ID")
	if projectID == "" {
		t.Skip("Env var GITLAB_PROJECT_ID missing")
	}

	repo := repotest.NewGoGitRepo(t, false)

	backend, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)

	defer backend.Close()

	author, err := identity.NewIdentity(repo, "Amine Hilaly", "hilalyamine@gmail.com")
	require.NoError(t, err)

	tests := []struct {
		name string
		url  string
		bug  *bug.Snapshot
	}{
		{
			name: "simple issue",
			url:  "https://gitlab.com/git-bug/test/-/work_items/1",
			bug: &bug.Snapshot{
				Operations: []dag.Operation{
					bug.NewCreateOp(author, 0, "simple issue", "initial comment", nil),
					bug.NewAddCommentOp(author, 0, "first comment", nil),
					bug.NewAddCommentOp(author, 0, "second comment", nil),
				},
			},
		},
		{
			name: "empty issue",
			url:  "https://gitlab.com/git-bug/test/-/work_items/2",
			bug: &bug.Snapshot{
				Operations: []dag.Operation{
					bug.NewCreateOp(author, 0, "empty issue", "", nil),
				},
			},
		},
		{
			name: "complex issue",
			url:  "https://gitlab.com/git-bug/test/-/work_items/3",
			bug: &bug.Snapshot{
				Operations: []dag.Operation{
					bug.NewCreateOp(author, 0, "complex issue", "initial comment", nil),
					bug.NewAddCommentOp(author, 0, "### header\n\n**bold**\n\n_italic_\n\n> with quote\n\n`inline code`\n\n```\nmultiline code\n```\n\n- bulleted\n- list\n\n1. numbered\n1. list\n\n- [ ] task\n- [x] list\n\n@git-bug mention\n\n#2 reference issue\n#3 auto-reference issue", nil),
					bug.NewSetTitleOp(author, 0, "complex issue edited", "complex issue"),
					bug.NewSetTitleOp(author, 0, "complex issue", "complex issue edited"),
					bug.NewSetStatusOp(author, 0, common.ClosedStatus),
					bug.NewSetStatusOp(author, 0, common.OpenStatus),
					bug.NewLabelChangeOperation(author, 0, []common.Label{"bug"}, []common.Label{}),
					bug.NewLabelChangeOperation(author, 0, []common.Label{"critical"}, []common.Label{}),
					bug.NewLabelChangeOperation(author, 0, []common.Label{}, []common.Label{"critical"}),
				},
			},
		},
		{
			name: "editions",
			url:  "https://gitlab.com/git-bug/test/-/work_items/4",
			bug: &bug.Snapshot{
				Operations: []dag.Operation{
					bug.NewCreateOp(author, 0, "editions", "initial comment edited", nil),
					bug.NewAddCommentOp(author, 0, "first comment edited", nil),
				},
			},
		},
	}

	login := "test-identity"
	author.SetMetadata(metaKeyGitlabLogin, login)

	token := auth.NewToken(target, envToken)
	token.SetMetadata(auth.MetaKeyLogin, login)
	token.SetMetadata(auth.MetaKeyBaseURL, defaultBaseURL)
	err = auth.Store(repo, token)
	require.NoError(t, err)

	ctx := context.Background()

	importer := &gitlabImporter{}
	err = importer.Init(ctx, backend, core.Configuration{
		confKeyProjectID:     projectID,
		confKeyGitlabBaseUrl: defaultBaseURL,
		confKeyDefaultLogin:  login,
	})
	require.NoError(t, err)

	start := time.Now()

	events, err := importer.ImportAll(ctx, backend, time.Time{})
	require.NoError(t, err)

	for result := range events {
		require.NoError(t, result.Err)
	}

	fmt.Printf("test repository imported in %f seconds\n", time.Since(start).Seconds())

	require.Len(t, backend.Bugs().AllIds(), len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := backend.Bugs().ResolveBugCreateMetadata(t.Context(), metaKeyGitlabUrl, tt.url)
			require.NoError(t, err)

			ops := b.Snapshot().Operations
			require.Len(t, tt.bug.Operations, len(ops))

			for i, op := range tt.bug.Operations {

				require.IsType(t, ops[i], op)
				require.Equal(t, op.Author().Name(), ops[i].Author().Name())

				switch op := op.(type) {
				case *bug.CreateOperation:
					require.Equal(t, op.Title, ops[i].(*bug.CreateOperation).Title)
					require.Equal(t, op.Message, ops[i].(*bug.CreateOperation).Message)
				case *bug.SetStatusOperation:
					require.Equal(t, op.Status, ops[i].(*bug.SetStatusOperation).Status)
				case *bug.SetTitleOperation:
					require.Equal(t, op.Was, ops[i].(*bug.SetTitleOperation).Was)
					require.Equal(t, op.Title, ops[i].(*bug.SetTitleOperation).Title)
				case *bug.LabelChangeOperation:
					require.ElementsMatch(t, op.Added, ops[i].(*bug.LabelChangeOperation).Added)
					require.ElementsMatch(t, op.Removed, ops[i].(*bug.LabelChangeOperation).Removed)
				case *bug.AddCommentOperation:
					require.Equal(t, op.Message, ops[i].(*bug.AddCommentOperation).Message)
				case *bug.EditCommentOperation:
					require.Equal(t, op.Message, ops[i].(*bug.EditCommentOperation).Message)

				default:
					panic("unknown operation type")
				}
			}
		})
	}
}

func countNonMetadataOps(b *cache.BugCache) int {
	count := 0
	for _, op := range b.Snapshot().Operations {
		if _, ok := op.(dag.OperationDoesntChangeSnapshot); !ok {
			count++
		}
	}
	return count
}

func TestGitlabImporterCrossBridgeAndDeduplicateExportedOps(t *testing.T) {
	repo := repotest.NewGoGitRepo(t, false)
	backend, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	author, err := backend.Identities().NewRaw(ctx, "Amine Hilaly", "hilalyamine@gmail.com", "ahilaly", "", nil, map[string]string{
		metaKeyGitlabId:    "42",
		metaKeyGitlabLogin: "ahilaly",
	})
	require.NoError(t, err)
	require.NoError(t, backend.SetUserIdentity(ctx, author))

	// Create bug originally imported from GitHub (origin: github) and exported to GitLab
	const issueGitlabID = "7"
	const issueGitlabURL = "https://gitlab.com/git-bug/test/-/issues/7"
	const projectID = "12345"
	b, createOp, err := backend.Bugs().NewRaw(
		author,
		100,
		"initial title",
		"initial body",
		nil,
		map[string]string{
			core.MetaKeyOrigin: "github",
		},
	)
	require.NoError(t, err)
	_, err = b.SetMetadata(ctx, createOp.Id(), map[string]string{
		metaKeyGitlabId:      issueGitlabID,
		metaKeyGitlabUrl:     issueGitlabURL,
		metaKeyGitlabProject: projectID,
		metaKeyGitlabBaseUrl: defaultBaseURL,
	})
	require.NoError(t, err)

	// Simulate exported operations tagged with issue ID (metaKeyGitlabId: issueGitlabID)
	closeOp, err := b.CloseRaw(author, 101, nil)
	require.NoError(t, err)
	require.NoError(t, markOperationAsExported(ctx, b, closeOp.Id(), issueGitlabID, ""))

	titleOp, err := b.SetTitleRaw(author, 102, "renamed title", nil)
	require.NoError(t, err)
	require.NoError(t, markOperationAsExported(ctx, b, titleOp.Id(), issueGitlabID, ""))

	_, labelAddOp, err := b.ChangeLabelsRaw(author, 103, []string{"bug", "core"}, nil, nil)
	require.NoError(t, err)
	require.NoError(t, markOperationAsExported(ctx, b, labelAddOp.Id(), issueGitlabID, ""))

	_, labelRmOp, err := b.ChangeLabelsRaw(author, 104, nil, []string{"core"}, nil)
	require.NoError(t, err)
	require.NoError(t, markOperationAsExported(ctx, b, labelRmOp.Id(), issueGitlabID, ""))

	require.NoError(t, b.Commit(ctx))
	initialNonMetaOps := countNonMetadataOps(b)

	out := make(chan core.ImportResult, 32)
	importer := &gitlabImporter{
		conf: core.Configuration{
			confKeyProjectID:     projectID,
			confKeyGitlabBaseUrl: defaultBaseURL,
		},
		out: out,
	}

	now := time.Unix(200, 0)
	glIssue := &gitlab.Issue{
		IID:         7,
		Title:       "renamed title",
		Description: "initial body",
		WebURL:      issueGitlabURL,
		CreatedAt:   &now,
		Author:      &gitlab.IssueAuthor{ID: 42},
	}

	// 1. ensureIssue must resolve the existing bug even though origin == "github"
	resolvedBug, err := importer.ensureIssue(ctx, backend, glIssue)
	require.NoError(t, err)
	require.Equal(t, b.Id(), resolvedBug.Id())

	// 2. Echoed events from the export above should be matched to the exported operations
	echoedClose := StateEvent{gitlab.StateEvent{
		ID:        901,
		State:     "closed",
		CreatedAt: &now,
		User:      &gitlab.BasicUser{ID: 42},
	}}
	require.NoError(t, importer.ensureIssueEvent(ctx, backend, b, glIssue, echoedClose))
	var echoedTitleNote gitlab.Note
	echoedTitleNote.ID = 902
	echoedTitleNote.System = true
	echoedTitleNote.Body = "changed title from **initial title** to **renamed title**"
	echoedTitleNote.CreatedAt = &now
	echoedTitleNote.UpdatedAt = &now
	echoedTitleNote.Author.ID = 42
	require.NoError(t, importer.ensureIssueEvent(ctx, backend, b, glIssue, NoteEvent{echoedTitleNote}))
	var echoedAddLabelBug gitlab.LabelEvent
	echoedAddLabelBug.ID = 903
	echoedAddLabelBug.Action = "add"
	echoedAddLabelBug.CreatedAt = &now
	echoedAddLabelBug.Label.Name = "bug"
	echoedAddLabelBug.User.ID = 42
	require.NoError(t, importer.ensureIssueEvent(ctx, backend, b, glIssue, LabelEvent{echoedAddLabelBug}))
	var echoedAddLabelCore gitlab.LabelEvent
	echoedAddLabelCore.ID = 9031
	echoedAddLabelCore.Action = "add"
	echoedAddLabelCore.CreatedAt = &now
	echoedAddLabelCore.Label.Name = "core"
	echoedAddLabelCore.User.ID = 42
	require.NoError(t, importer.ensureIssueEvent(ctx, backend, b, glIssue, LabelEvent{echoedAddLabelCore}))
	var echoedRmLabel gitlab.LabelEvent
	echoedRmLabel.ID = 904
	echoedRmLabel.Action = "remove"
	echoedRmLabel.CreatedAt = &now
	echoedRmLabel.Label.Name = "core"
	echoedRmLabel.User.ID = 42
	require.NoError(t, importer.ensureIssueEvent(ctx, backend, b, glIssue, LabelEvent{echoedRmLabel}))
	require.Equal(t, initialNonMetaOps, countNonMetadataOps(b))

	// 3. Subsequent legitimate remote events (reopen -> close, new title, re-add core) must be imported
	reopenEvt := StateEvent{gitlab.StateEvent{
		ID:        905,
		State:     "reopened",
		CreatedAt: &now,
		User:      &gitlab.BasicUser{ID: 42},
	}}
	require.NoError(t, importer.ensureIssueEvent(ctx, backend, b, glIssue, reopenEvt))
	require.Equal(t, initialNonMetaOps+1, countNonMetadataOps(b))
	require.Equal(t, common.OpenStatus, b.Snapshot().Status)

	// Re-importing the earlier echoed close event (ID 901) must still be skipped!
	require.NoError(t, importer.ensureIssueEvent(ctx, backend, b, glIssue, echoedClose))
	require.Equal(t, initialNonMetaOps+1, countNonMetadataOps(b))
	require.Equal(t, common.OpenStatus, b.Snapshot().Status)

	closeEvt2 := StateEvent{gitlab.StateEvent{
		ID:        906,
		State:     "closed",
		CreatedAt: &now,
		User:      &gitlab.BasicUser{ID: 42},
	}}
	require.NoError(t, importer.ensureIssueEvent(ctx, backend, b, glIssue, closeEvt2))
	require.Equal(t, initialNonMetaOps+2, countNonMetadataOps(b))
	require.Equal(t, common.ClosedStatus, b.Snapshot().Status)

	var titleNote gitlab.Note
	titleNote.ID = 907
	titleNote.System = true
	titleNote.Body = "changed title from **renamed title** to **third title**"
	titleNote.CreatedAt = &now
	titleNote.UpdatedAt = &now
	titleNote.Author.ID = 42
	require.NoError(t, importer.ensureIssueEvent(ctx, backend, b, glIssue, NoteEvent{titleNote}))
	require.Equal(t, initialNonMetaOps+3, countNonMetadataOps(b))
	require.Equal(t, "third title", b.Snapshot().Title)

	// Re-importing the earlier echoed title event (ID 902) must still be skipped!
	require.NoError(t, importer.ensureIssueEvent(ctx, backend, b, glIssue, NoteEvent{echoedTitleNote}))
	require.Equal(t, initialNonMetaOps+3, countNonMetadataOps(b))
	require.Equal(t, "third title", b.Snapshot().Title)

	var addLabelEvt gitlab.LabelEvent
	addLabelEvt.ID = 908
	addLabelEvt.Action = "add"
	addLabelEvt.CreatedAt = &now
	addLabelEvt.Label.Name = "core"
	addLabelEvt.User.ID = 42
	require.NoError(t, importer.ensureIssueEvent(ctx, backend, b, glIssue, LabelEvent{addLabelEvt}))
	require.Equal(t, initialNonMetaOps+4, countNonMetadataOps(b))

	// Re-importing the earlier echoed label removal event (ID 904) must still be skipped!
	require.NoError(t, importer.ensureIssueEvent(ctx, backend, b, glIssue, LabelEvent{echoedRmLabel}))
	require.Equal(t, initialNonMetaOps+4, countNonMetadataOps(b))

	// Re-importing the exact same label event by ID must be deduplicated via errResolve == nil
	require.NoError(t, importer.ensureIssueEvent(ctx, backend, b, glIssue, LabelEvent{addLabelEvt}))
	require.Equal(t, initialNonMetaOps+4, countNonMetadataOps(b))

	// 4. Simulate a fresh ImportAll run (new matchedExported map) replaying the entire remote timeline in chronological order
	importer.matchedExported = make(map[string]string)
	fullTimeline := []Event{
		echoedClose,
		NoteEvent{echoedTitleNote},
		LabelEvent{echoedAddLabelBug},
		LabelEvent{echoedAddLabelCore},
		LabelEvent{echoedRmLabel},
		reopenEvt,
		closeEvt2,
		NoteEvent{titleNote},
		LabelEvent{addLabelEvt},
	}
	for _, ev := range fullTimeline {
		require.NoError(t, importer.ensureIssueEvent(ctx, backend, b, glIssue, ev))
	}
	require.Equal(t, initialNonMetaOps+4, countNonMetadataOps(b))

	// 5. Verify EventComment does not fail with "Multiple matching operation found" when
	// both AddCommentOperation and an exported EditCommentOperation share metaKeyGitlabId == noteID.
	commentCombinedID, commentOp, err := b.AddCommentRaw(author, 105, "comment 1", nil, nil)
	require.NoError(t, err)
	const noteGitlabID = "555"
	require.NoError(t, markOperationAsExported(ctx, b, commentOp.Id(), noteGitlabID, ""))

	editCommentOp, err := b.EditCommentRaw(author, 106, commentCombinedID, "comment 1 edited", nil)
	require.NoError(t, err)
	require.NoError(t, markOperationAsExported(ctx, b, editCommentOp.Id(), noteGitlabID, ""))

	opsAfterLocalCommentEdit := countNonMetadataOps(b)
	var commentNote gitlab.Note
	commentNote.ID = 555
	commentNote.System = false
	commentNote.Body = "comment 1 edited"
	commentNote.CreatedAt = &now
	commentNote.UpdatedAt = &now
	commentNote.Author.ID = 42
	require.NoError(t, importer.ensureIssueEvent(ctx, backend, b, glIssue, NoteEvent{commentNote}))
	require.Equal(t, opsAfterLocalCommentEdit, countNonMetadataOps(b))
}
